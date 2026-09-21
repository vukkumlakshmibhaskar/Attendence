from __future__ import annotations

import argparse
import json
from pathlib import Path

import numpy as np
import onnx
import onnxruntime as ort
import scipy.io as sio
from onnx import TensorProto, helper, numpy_helper
from PIL import Image
from sklearn.linear_model import SGDClassifier
from sklearn.metrics import accuracy_score, classification_report, confusion_matrix
from sklearn.preprocessing import StandardScaler


LABELS = ["screen", "away"]


def gaze_angle(gaze: np.ndarray) -> np.ndarray:
    theta = np.arcsin(np.clip(-gaze[:, 1], -1.0, 1.0))
    phi = np.arctan2(-gaze[:, 0], -gaze[:, 2])
    return np.sqrt(theta * theta + phi * phi)


def image_features(images: np.ndarray) -> np.ndarray:
    values = images.astype(np.float32)
    if values.max() > 2:
        values = values / 255.0
    count = values.shape[0]
    flattened = np.empty((count, 1024), dtype=np.float32)
    for index, image in enumerate(values):
        pil = Image.fromarray(np.uint8(np.clip(image * 255.0, 0, 255)), mode="L")
        pil = pil.resize((32, 32), Image.Resampling.BILINEAR)
        flattened[index] = np.asarray(pil, dtype=np.float32).reshape(-1) / 255.0
    return flattened


def iter_mat_batches(dataset: Path, participants: list[str], center_threshold: float):
    root = dataset / "Data" / "Normalized"
    for participant in participants:
        for mat_path in sorted((root / participant).glob("*.mat")):
            mat = sio.loadmat(mat_path, squeeze_me=True, struct_as_record=False)
            data = mat["data"]
            for side in ("left", "right"):
                side_data = getattr(data, side)
                images = np.asarray(side_data.image)
                gaze = np.asarray(side_data.gaze, dtype=np.float32)
                if images.ndim != 3 or gaze.ndim != 2:
                    continue
                x = image_features(images)
                angles = gaze_angle(gaze)
                y = (angles > center_threshold).astype(np.int32)
                yield x, y


def split_participants(dataset: Path):
    root = dataset / "Data" / "Normalized"
    participants = sorted(path.name for path in root.glob("p*") if path.is_dir())
    if len(participants) < 5:
        raise FileNotFoundError(f"not enough MPIIGaze participant folders under {root}")
    return participants[:-4], participants[-4:-2], participants[-2:]


def count_samples(dataset: Path, participants: list[str], threshold: float) -> tuple[int, int]:
    total = 0
    away = 0
    for _x, y in iter_mat_batches(dataset, participants, threshold):
        total += int(y.size)
        away += int(y.sum())
    return total, away


def evaluate(model: SGDClassifier, scaler: StandardScaler, dataset: Path, participants: list[str], threshold: float):
    truth = []
    predicted = []
    for x, y in iter_mat_batches(dataset, participants, threshold):
        pred = model.predict(scaler.transform(x))
        truth.extend(y.tolist())
        predicted.extend(pred.tolist())
    return {
        "accuracy": float(accuracy_score(truth, predicted)),
        "confusion_matrix": confusion_matrix(truth, predicted).tolist(),
        "classification_report": classification_report(
            truth,
            predicted,
            target_names=LABELS,
            output_dict=True,
            zero_division=0,
        ),
    }


def export_onnx(path: Path, model: SGDClassifier, scaler: StandardScaler) -> None:
    coef = model.coef_[0].astype(np.float32)
    intercept = np.float32(model.intercept_[0])
    mean = scaler.mean_.astype(np.float32)
    scale = scaler.scale_.astype(np.float32)
    scale[scale == 0] = 1.0

    weights = np.zeros((1024, 2), dtype=np.float32)
    weights[:, 1] = coef
    bias = np.asarray([0.0, float(intercept)], dtype=np.float32)

    input_tensor = helper.make_tensor_value_info("image_uint8", TensorProto.UINT8, [1, 3, 64, 64])
    output_tensor = helper.make_tensor_value_info("probabilities", TensorProto.FLOAT, [1, 2])
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
        producer_name="classroom-attendance-mpiigaze-trainer",
        opset_imports=[helper.make_opsetid("", 13)],
    )
    onnx_model.ir_version = 8
    onnx.checker.check_model(onnx_model)
    path.parent.mkdir(parents=True, exist_ok=True)
    onnx.save(onnx_model, path)


def validate_onnx(path: Path, dataset: Path, participants: list[str], threshold: float, limit: int = 512):
    session = ort.InferenceSession(str(path), providers=["CPUExecutionProvider"])
    input_name = session.get_inputs()[0].name
    total = 0
    correct = 0
    for x, y in iter_mat_batches(dataset, participants, threshold):
        for row, label in zip(x, y):
            image = Image.fromarray(np.uint8(np.clip(row.reshape(32, 32) * 255.0, 0, 255)), mode="L")
            image = image.resize((64, 64), Image.Resampling.BILINEAR).convert("RGB")
            tensor = np.transpose(np.asarray(image, dtype=np.uint8), (2, 0, 1))[None, ...]
            probs = session.run(None, {input_name: tensor})[0][0]
            if int(np.argmax(probs)) == int(label):
                correct += 1
            total += 1
            if total >= limit:
                return {"onnx_checked_samples": total, "onnx_checked_accuracy": float(correct / total)}
    return {"onnx_checked_samples": total, "onnx_checked_accuracy": float(correct / max(total, 1))}


def main() -> None:
    parser = argparse.ArgumentParser(description="Train a CPU ONNX screen-vs-away gaze proxy from MPIIGaze.")
    parser.add_argument("--dataset", required=True, type=Path)
    parser.add_argument("--output", default=Path("models/gaze_mpiigaze.onnx"), type=Path)
    parser.add_argument("--report", default=Path("models/gaze_mpiigaze_report.json"), type=Path)
    parser.add_argument("--center-threshold", type=float, default=0.22)
    parser.add_argument("--epochs", type=int, default=3)
    args = parser.parse_args()

    train_people, val_people, test_people = split_participants(args.dataset)
    print(f"train_people={train_people} val_people={val_people} test_people={test_people}")

    scaler = StandardScaler()
    for x, _y in iter_mat_batches(args.dataset, train_people, args.center_threshold):
        scaler.partial_fit(x)

    model = SGDClassifier(loss="log_loss", alpha=0.0001, penalty="l2", random_state=42, average=True)
    classes = np.asarray([0, 1], dtype=np.int32)
    for epoch in range(args.epochs):
        for x, y in iter_mat_batches(args.dataset, train_people, args.center_threshold):
            model.partial_fit(scaler.transform(x), y, classes=classes)
        val_metrics = evaluate(model, scaler, args.dataset, val_people, args.center_threshold)
        print(f"epoch={epoch + 1} val_accuracy={val_metrics['accuracy']:.4f}")

    val_metrics = evaluate(model, scaler, args.dataset, val_people, args.center_threshold)
    test_metrics = evaluate(model, scaler, args.dataset, test_people, args.center_threshold)
    export_onnx(args.output, model, scaler)
    onnx_metrics = validate_onnx(args.output, args.dataset, test_people, args.center_threshold)
    train_total, train_away = count_samples(args.dataset, train_people, args.center_threshold)
    val_total, val_away = count_samples(args.dataset, val_people, args.center_threshold)
    test_total, test_away = count_samples(args.dataset, test_people, args.center_threshold)

    report = {
        "dataset": str(args.dataset),
        "task": "mpiigaze_screen_vs_away_proxy",
        "labels": LABELS,
        "center_threshold_radians": args.center_threshold,
        "note": "MPIIGaze contains on-screen gaze targets. The 'away' class here means far from camera/screen center, not true off-screen distraction.",
        "train_people": train_people,
        "val_people": val_people,
        "test_people": test_people,
        "train_samples": train_total,
        "train_away_samples": train_away,
        "val_samples": val_total,
        "val_away_samples": val_away,
        "test_samples": test_total,
        "test_away_samples": test_away,
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
