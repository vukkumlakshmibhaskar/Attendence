import argparse
import shutil
from pathlib import Path


ROOT = Path(__file__).resolve().parent
DATA_YAML = ROOT / "data.yaml"
RUNS_DIR = ROOT / "runs"
EXPORT_ONNX = ROOT / "human_face_detector.onnx"
EXPORT_INT8_ONNX = ROOT / "human_face_detector.int8.onnx"
EXPORT_OPTIMIZED_ONNX = ROOT / "human_face_detector.optimized.onnx"


def read_dataset_config() -> dict[str, str]:
    if not DATA_YAML.exists():
        raise FileNotFoundError(f"Dataset config not found: {DATA_YAML}")

    config: dict[str, str] = {}
    for raw_line in DATA_YAML.read_text(encoding="utf-8").splitlines():
        line = raw_line.split("#", 1)[0].strip()
        if not line or ":" not in line:
            continue
        key, value = line.split(":", 1)
        key = key.strip()
        value = value.strip().strip("'\"")
        if key in {"path", "train", "val"}:
            config[key] = value
    return config


def dataset_root(config: dict[str, str]) -> Path:
    root = Path(config.get("path", "."))
    if not root.is_absolute():
        root = DATA_YAML.parent / root
    return root


def dataset_entry(config: dict[str, str], key: str, default: str) -> Path:
    entry = Path(config.get(key, default))
    if not entry.is_absolute():
        entry = dataset_root(config) / entry
    return entry


def label_dir_for(image_dir: Path) -> Path:
    parts = list(image_dir.parts)
    for index in range(len(parts) - 1, -1, -1):
        if parts[index].lower() == "images":
            parts[index] = "labels"
            return Path(*parts)
    return image_dir.parent.parent / "labels" / image_dir.name


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Train and export a CPU-friendly human face detector from the local YOLO dataset."
    )
    parser.add_argument("--model", default="yolov8n.pt", help="Ultralytics base model to fine-tune.")
    parser.add_argument("--epochs", type=int, default=40, help="Training epochs for a normal run.")
    parser.add_argument("--imgsz", type=int, default=320, help="Image size. 320/416 are good CPU targets.")
    parser.add_argument("--batch", type=int, default=8, help="Batch size for CPU training.")
    parser.add_argument("--workers", type=int, default=0, help="Data-loader workers. Keep 0 on Windows.")
    parser.add_argument("--device", default="cpu", help="Use cpu for this project.")
    parser.add_argument("--fraction", type=float, default=1.0, help="Training data fraction for experiments.")
    parser.add_argument("--quick", action="store_true", help="Fast smoke training: 1 epoch on 5 percent data.")
    parser.add_argument("--no-quantize", action="store_true", help="Skip ONNX Runtime dynamic INT8 export.")
    parser.add_argument("--export-only", action="store_true", help="Skip training and export a saved checkpoint.")
    parser.add_argument(
        "--weights",
        default="",
        help="Checkpoint path for --export-only. Defaults to runs/human_face_detector/weights/best.pt.",
    )
    return parser.parse_args()


def ensure_dataset_exists() -> None:
    config = read_dataset_config()
    train_images = dataset_entry(config, "train", "images/train")
    val_images = dataset_entry(config, "val", "images/val")
    required = [
        train_images,
        val_images,
        label_dir_for(train_images),
        label_dir_for(val_images),
    ]
    missing = [str(path) for path in required if not path.exists()]
    if missing:
        raise FileNotFoundError("Dataset is missing required folders: " + ", ".join(missing))


def validation_image_paths(limit: int) -> list[Path]:
    config = read_dataset_config()
    image_dir = dataset_entry(config, "val", "images/val")
    paths: list[Path] = []
    for pattern in ("*.jpg", "*.jpeg", "*.png"):
        paths.extend(image_dir.rglob(pattern))
    return paths[:limit]


def copy_exported_onnx(exported_path: str | Path) -> Path:
    exported = Path(exported_path)
    if not exported.exists():
        raise FileNotFoundError(f"Ultralytics export did not create an ONNX file: {exported}")
    shutil.copy2(exported, EXPORT_ONNX)
    return EXPORT_ONNX


def quantize_onnx(source: Path) -> Path | None:
    try:
        import cv2
        import numpy as np
        import onnxruntime as ort
        from onnxruntime.quantization import (
            CalibrationDataReader,
            CalibrationMethod,
            QuantFormat,
            QuantType,
            quantize_static,
        )
    except Exception as exc:
        print(f"INT8 quantization skipped because onnxruntime quantization is unavailable: {exc}")
        return None

    class FaceCalibrationReader(CalibrationDataReader):
        def __init__(self, input_name: str, limit: int = 48) -> None:
            self.input_name = input_name
            self.image_paths = validation_image_paths(limit)
            self.index = 0

        def get_next(self) -> dict[str, np.ndarray] | None:
            if self.index >= len(self.image_paths):
                return None
            path = self.image_paths[self.index]
            self.index += 1
            image = cv2.imread(str(path))
            if image is None:
                return self.get_next()
            image = cv2.resize(image, (320, 320), interpolation=cv2.INTER_LINEAR)
            image = image[:, :, ::-1].transpose(2, 0, 1).astype("float32") / 255.0
            return {self.input_name: image[None]}

    try:
        session = ort.InferenceSession(str(source), providers=["CPUExecutionProvider"])
        input_name = session.get_inputs()[0].name
        if EXPORT_INT8_ONNX.exists():
            EXPORT_INT8_ONNX.replace(ROOT / "human_face_detector.int8.unsupported.onnx")
        quantize_static(
            model_input=str(source),
            model_output=str(EXPORT_INT8_ONNX),
            calibration_data_reader=FaceCalibrationReader(input_name),
            quant_format=QuantFormat.QDQ,
            activation_type=QuantType.QUInt8,
            weight_type=QuantType.QInt8,
            calibrate_method=CalibrationMethod.MinMax,
            per_channel=False,
        )
        return EXPORT_INT8_ONNX
    except Exception as exc:
        print(f"INT8 quantization skipped because ONNX Runtime rejected the model: {exc}")
        return None


def optimize_onnx(source: Path) -> Path | None:
    try:
        import onnxruntime as ort
    except Exception as exc:
        print(f"ONNX optimization skipped because onnxruntime is unavailable: {exc}")
        return None

    try:
        options = ort.SessionOptions()
        options.graph_optimization_level = ort.GraphOptimizationLevel.ORT_ENABLE_ALL
        options.optimized_model_filepath = str(EXPORT_OPTIMIZED_ONNX)
        ort.InferenceSession(str(source), sess_options=options, providers=["CPUExecutionProvider"])
        return EXPORT_OPTIMIZED_ONNX
    except Exception as exc:
        print(f"ONNX optimization skipped because ONNX Runtime rejected the model: {exc}")
        return None


def export_checkpoint(weights: Path, imgsz: int, device: str, quantize: bool) -> None:
    if not weights.exists():
        raise FileNotFoundError(f"Checkpoint not found: {weights}")

    from ultralytics import YOLO

    trained_model = YOLO(str(weights))
    exported = trained_model.export(
        format="onnx",
        imgsz=imgsz,
        device=device,
        opset=12,
        simplify=False,
        dynamic=False,
    )
    onnx_path = copy_exported_onnx(exported)
    print(f"ONNX detector exported: {onnx_path}")

    optimized_path = optimize_onnx(onnx_path)
    if optimized_path:
        print(f"ONNX Runtime optimized detector exported: {optimized_path}")

    if quantize:
        int8_path = quantize_onnx(onnx_path)
        if int8_path:
            print(f"INT8 ONNX detector exported: {int8_path}")


def main() -> None:
    args = parse_args()
    ensure_dataset_exists()

    if args.quick:
        args.epochs = 1
        args.batch = min(args.batch, 4)
        args.fraction = min(args.fraction, 0.05)
        args.imgsz = min(args.imgsz, 320)

    if args.device.lower() != "cpu":
        raise ValueError("This project is CPU-first. Use --device cpu.")

    default_best = RUNS_DIR / "human_face_detector" / "weights" / "best.pt"
    if args.export_only:
        export_checkpoint(Path(args.weights) if args.weights else default_best, args.imgsz, args.device, not args.no_quantize)
        return

    from ultralytics import YOLO
    model = YOLO(args.model)
    model.train(
        data=str(DATA_YAML),
        epochs=args.epochs,
        imgsz=args.imgsz,
        batch=args.batch,
        workers=args.workers,
        device=args.device,
        project=str(RUNS_DIR),
        name="human_face_detector",
        exist_ok=True,
        patience=8,
        fraction=args.fraction,
        cache=False,
        val=not args.quick,
        plots=not args.quick,
        verbose=True,
    )

    best_weights = RUNS_DIR / "human_face_detector" / "weights" / "best.pt"
    last_weights = RUNS_DIR / "human_face_detector" / "weights" / "last.pt"
    weights = best_weights if best_weights.exists() else last_weights
    if not weights.exists():
        raise FileNotFoundError("Training finished but no best.pt or last.pt was produced.")

    export_checkpoint(weights, args.imgsz, args.device, not args.no_quantize)


if __name__ == "__main__":
    main()
