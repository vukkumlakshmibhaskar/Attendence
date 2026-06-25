from __future__ import annotations

import argparse
import json
import random
from pathlib import Path

import numpy as np
import onnx
import onnxruntime as ort
from onnx import TensorProto, helper, numpy_helper
from PIL import Image, ImageOps
from sklearn.linear_model import SGDClassifier
from sklearn.metrics import accuracy_score, classification_report, confusion_matrix
from sklearn.preprocessing import StandardScaler


LABELS = {"awake": 0, "sleepy": 1}
OUTPUT_LABELS = ["attentive", "tired", "distracted"]
IMAGE_EXTENSIONS = {".png", ".jpg", ".jpeg", ".bmp", ".webp"}


def collect_samples(dataset: Path, split: str) -> list[tuple[Path, int]]:
    samples: list[tuple[Path, int]] = []
    split_root = dataset / split
    if not split_root.exists():
        raise FileNotFoundError(f"missing split folder: {split_root}")
    for label, class_id in LABELS.items():
        class_root = split_root / label
        if not class_root.exists():
            raise FileNotFoundError(f"missing class folder: {class_root}")
        for path in class_root.rglob("*"):
            if path.is_file() and path.suffix.lower() in IMAGE_EXTENSIONS:
                samples.append((path, class_id))
    return samples


def extract_features(path: Path) -> np.ndarray:
    image = Image.open(path)
    image = ImageOps.exif_transpose(image).convert("L").resize((64, 64), Image.Resampling.BILINEAR)
    gray = np.asarray(image, dtype=np.float32) / 255.0
    pooled = gray.reshape(32, 2, 32, 2).mean(axis=(1, 3))
    return pooled.reshape(-1).astype(np.float32)


def batch_features(samples: list[tuple[Path, int]], batch_size: int):
    for start in range(0, len(samples), batch_size):
        batch = samples[start : start + batch_size]
        x = np.empty((len(batch), 1024), dtype=np.float32)
        y = np.empty((len(batch),), dtype=np.int32)
        for index, (path, label) in enumerate(batch):
            x[index] = extract_features(path)
            y[index] = label
        yield x, y


def evaluate(model: SGDClassifier, scaler: StandardScaler, samples: list[tuple[Path, int]], batch_size: int):
    truth: list[int] = []
    predicted: list[int] = []
    for x, y in batch_features(samples, batch_size):
        x_scaled = scaler.transform(x)
        pred = model.predict(x_scaled)
        truth.extend(y.tolist())
        predicted.extend(pred.tolist())
    report = classification_report(
        truth,
        predicted,
        target_names=["awake", "sleepy"],
        output_dict=True,
        zero_division=0,
    )
    return {
        "accuracy": float(accuracy_score(truth, predicted)),
        "confusion_matrix": confusion_matrix(truth, predicted).tolist(),
        "classification_report": report,
    }


def export_onnx(path: Path, model: SGDClassifier, scaler: StandardScaler) -> None:
    coef = model.coef_[0].astype(np.float32)
    intercept = np.float32(model.intercept_[0])
    mean = scaler.mean_.astype(np.float32)
    scale = scaler.scale_.astype(np.float32)
    scale[scale == 0] = 1.0

    weights = np.zeros((1024, 3), dtype=np.float32)
    weights[:, 1] = coef
    bias = np.asarray([0.0, float(intercept), -8.0], dtype=np.float32)

    input_tensor = helper.make_tensor_value_info("image_uint8", TensorProto.UINT8, [1, 3, 64, 64])
    output_tensor = helper.make_tensor_value_info("probabilities", TensorProto.FLOAT, [1, 3])

    initializers = [
        numpy_helper.from_array(np.asarray(255.0, dtype=np.float32), name="scale_255"),
        numpy_helper.from_array(mean.reshape(1, 1024), name="feature_mean"),
        numpy_helper.from_array(scale.reshape(1, 1024), name="feature_scale"),
        numpy_helper.from_array(weights, name="linear_weights"),
        numpy_helper.from_array(bias, name="linear_bias"),
    ]

    nodes = [
        helper.make_node("Cast", ["image_uint8"], ["image_float"], to=TensorProto.FLOAT),
        helper.make_node("Div", ["image_float", "scale_255"], ["image_norm"]),
        helper.make_node("ReduceMean", ["image_norm"], ["gray"], axes=[1], keepdims=1),
        helper.make_node("AveragePool", ["gray"], ["pooled"], kernel_shape=[2, 2], strides=[2, 2]),
        helper.make_node("Flatten", ["pooled"], ["features"], axis=1),
        helper.make_node("Sub", ["features", "feature_mean"], ["centered"]),
        helper.make_node("Div", ["centered", "feature_scale"], ["standardized"]),
        helper.make_node("Gemm", ["standardized", "linear_weights", "linear_bias"], ["logits"]),
        helper.make_node("Softmax", ["logits"], ["probabilities"], axis=1),
    ]

    graph = helper.make_graph(nodes, path.stem, [input_tensor], [output_tensor], initializers)
    onnx_model = helper.make_model(
        graph,
        producer_name="classroom-attendance-eye-state-trainer",
        opset_imports=[helper.make_opsetid("", 13)],
    )
    onnx_model.ir_version = 8
    onnx.checker.check_model(onnx_model)
    path.parent.mkdir(parents=True, exist_ok=True)
    onnx.save(onnx_model, path)


def validate_onnx(path: Path, samples: list[tuple[Path, int]], limit: int = 256) -> dict[str, float]:
    session = ort.InferenceSession(str(path), providers=["CPUExecutionProvider"])
    input_name = session.get_inputs()[0].name
    chosen = samples[:limit]
    correct = 0
    for image_path, label in chosen:
        image = Image.open(image_path)
        image = ImageOps.exif_transpose(image).convert("RGB").resize((64, 64), Image.Resampling.BILINEAR)
        arr = np.asarray(image, dtype=np.uint8)
        tensor = np.transpose(arr, (2, 0, 1))[None, ...]
        probs = session.run(None, {input_name: tensor})[0][0]
        prediction = int(np.argmax(probs[:2]))
        if prediction == label:
            correct += 1
    return {
        "onnx_checked_samples": len(chosen),
        "onnx_checked_accuracy": float(correct / max(len(chosen), 1)),
    }


def main() -> None:
    parser = argparse.ArgumentParser(description="Train a CPU ONNX eye-state model for awake/sleepy detection.")
    parser.add_argument("--dataset", required=True, type=Path)
    parser.add_argument("--output", default=Path("models/emotion_eye_state.onnx"), type=Path)
    parser.add_argument("--report", default=Path("models/emotion_eye_state_report.json"), type=Path)
    parser.add_argument("--batch-size", type=int, default=2048)
    parser.add_argument("--epochs", type=int, default=5)
    parser.add_argument("--seed", type=int, default=42)
    args = parser.parse_args()

    rng = random.Random(args.seed)
    train_samples = collect_samples(args.dataset, "train")
    val_samples = collect_samples(args.dataset, "val")
    test_samples = collect_samples(args.dataset, "test")
    rng.shuffle(train_samples)
    rng.shuffle(val_samples)
    rng.shuffle(test_samples)

    print(f"train={len(train_samples)} val={len(val_samples)} test={len(test_samples)}")

    scaler = StandardScaler()
    for x, _y in batch_features(train_samples, args.batch_size):
        scaler.partial_fit(x)

    model = SGDClassifier(
        loss="log_loss",
        alpha=0.0001,
        penalty="l2",
        random_state=args.seed,
        learning_rate="optimal",
        average=True,
    )

    classes = np.asarray([0, 1], dtype=np.int32)
    for epoch in range(args.epochs):
        rng.shuffle(train_samples)
        for x, y in batch_features(train_samples, args.batch_size):
            model.partial_fit(scaler.transform(x), y, classes=classes)
        val_metrics = evaluate(model, scaler, val_samples, args.batch_size)
        print(f"epoch={epoch + 1} val_accuracy={val_metrics['accuracy']:.4f}")

    val_metrics = evaluate(model, scaler, val_samples, args.batch_size)
    test_metrics = evaluate(model, scaler, test_samples, args.batch_size)
    export_onnx(args.output, model, scaler)
    onnx_metrics = validate_onnx(args.output, test_samples)

    report = {
        "dataset": str(args.dataset),
        "task": "eye_state_awake_sleepy",
        "output_labels": OUTPUT_LABELS,
        "train_samples": len(train_samples),
        "val_samples": len(val_samples),
        "test_samples": len(test_samples),
        "epochs": args.epochs,
        "feature_shape": "grayscale_32x32_average_pool",
        "model": "standardized_linear_logistic_sgd",
        "val": val_metrics,
        "test": test_metrics,
        "onnx_validation": onnx_metrics,
    }
    args.report.parent.mkdir(parents=True, exist_ok=True)
    args.report.write_text(json.dumps(report, indent=2), encoding="utf-8")
    print(f"saved_model={args.output}")
    print(f"saved_report={args.report}")
    print(json.dumps(report, indent=2))


if __name__ == "__main__":
    main()
