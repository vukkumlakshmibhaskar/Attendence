from __future__ import annotations

import argparse
import csv
import json
import random
import shutil
from dataclasses import dataclass
from pathlib import Path


ROOT = Path(__file__).resolve().parent
DEFAULT_DATASET = Path(r"C:\Users\lakshmibhaskar.v\Downloads\kara2018_headposeannotations\kara2018_headposeannotations")
REPORT_PATH = ROOT / "kara_dataset_report.json"
ONNX_PATH = ROOT / "head_pose_estimator.onnx"
OPTIMIZED_ONNX_PATH = ROOT / "head_pose_estimator.optimized.onnx"


@dataclass(frozen=True)
class PoseSample:
    sample_id: str
    image_path: Path
    tilt: float
    pan: float
    glasses: str


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Train a lightweight CPU head-pose estimator and export ONNX.")
    parser.add_argument("--dataset-root", type=Path, default=DEFAULT_DATASET)
    parser.add_argument("--images-dir", type=Path, default=None)
    parser.add_argument("--epochs", type=int, default=30)
    parser.add_argument("--batch", type=int, default=16)
    parser.add_argument("--imgsz", type=int, default=96)
    parser.add_argument("--lr", type=float, default=1e-3)
    parser.add_argument("--quick", action="store_true", help="Train only 3 epochs for a fast smoke run.")
    parser.add_argument("--validate-only", action="store_true", help="Only check labels/images and write a report.")
    return parser.parse_args()


def read_ground_truth(dataset_root: Path) -> list[dict[str, str]]:
    csv_path = dataset_root / "headpose_groundtruth.csv"
    if not csv_path.exists():
        raise FileNotFoundError(f"Missing headpose_groundtruth.csv: {csv_path}")

    with csv_path.open("r", encoding="utf-8-sig", newline="") as handle:
        return list(csv.DictReader(handle, delimiter=";"))


def build_image_index(search_roots: list[Path]) -> dict[str, Path]:
    index: dict[str, Path] = {}
    for root in search_roots:
        if not root.exists():
            continue
        for path in root.rglob("*"):
            if path.is_file() and path.suffix.lower() in {".jpg", ".jpeg", ".png"}:
                index.setdefault(path.name.lower(), path)
    return index


def load_samples(dataset_root: Path, images_dir: Path | None) -> tuple[list[PoseSample], dict[str, object]]:
    rows = read_ground_truth(dataset_root)
    search_roots = [images_dir] if images_dir else [
        dataset_root / "images",
        dataset_root,
        dataset_root.parent,
    ]
    image_index = build_image_index([root for root in search_roots if root is not None])

    samples: list[PoseSample] = []
    missing: list[str] = []
    for row in rows:
        sample_id = row["SampleID"].strip()
        image_path = image_index.get(sample_id.lower())
        if not image_path:
            missing.append(sample_id)
            continue
        samples.append(
            PoseSample(
                sample_id=sample_id,
                image_path=image_path,
                tilt=float(row["Tilt"]),
                pan=float(row["Pan"]),
                glasses=row["Glasses"].strip().lower(),
            )
        )

    report = {
        "dataset_root": str(dataset_root),
        "images_dir": str(images_dir) if images_dir else "",
        "ground_truth_rows": len(rows),
        "matched_images": len(samples),
        "missing_images": len(missing),
        "missing_examples": missing[:20],
        "tilt_values": sorted({float(row["Tilt"]) for row in rows}),
        "pan_values": sorted({float(row["Pan"]) for row in rows}),
    }
    REPORT_PATH.parent.mkdir(parents=True, exist_ok=True)
    REPORT_PATH.write_text(json.dumps(report, indent=2), encoding="utf-8")
    return samples, report


def train(samples: list[PoseSample], args: argparse.Namespace) -> Path:
    import cv2
    import numpy as np
    import torch
    from torch import nn
    from torch.utils.data import DataLoader, Dataset

    class HeadPoseDataset(Dataset):
        def __init__(self, items: list[PoseSample]) -> None:
            self.items = items

        def __len__(self) -> int:
            return len(self.items)

        def __getitem__(self, index: int) -> tuple[torch.Tensor, torch.Tensor]:
            item = self.items[index]
            image = cv2.imread(str(item.image_path))
            if image is None:
                raise RuntimeError(f"Could not read image: {item.image_path}")
            image = cv2.resize(image, (args.imgsz, args.imgsz), interpolation=cv2.INTER_AREA)
            image = cv2.cvtColor(image, cv2.COLOR_BGR2RGB).astype("float32") / 255.0
            image = np.transpose(image, (2, 0, 1))
            label = np.array([item.tilt / 90.0, item.pan / 90.0], dtype="float32")
            return torch.from_numpy(image), torch.from_numpy(label)

    class TinyHeadPoseNet(nn.Module):
        def __init__(self) -> None:
            super().__init__()
            self.features = nn.Sequential(
                nn.Conv2d(3, 16, 3, stride=2, padding=1),
                nn.BatchNorm2d(16),
                nn.ReLU(inplace=True),
                nn.Conv2d(16, 32, 3, stride=2, padding=1),
                nn.BatchNorm2d(32),
                nn.ReLU(inplace=True),
                nn.Conv2d(32, 64, 3, stride=2, padding=1),
                nn.BatchNorm2d(64),
                nn.ReLU(inplace=True),
                nn.Conv2d(64, 96, 3, stride=2, padding=1),
                nn.BatchNorm2d(96),
                nn.ReLU(inplace=True),
                nn.AdaptiveAvgPool2d(1),
            )
            self.regressor = nn.Sequential(nn.Flatten(), nn.Dropout(0.1), nn.Linear(96, 2), nn.Tanh())

        def forward(self, x: torch.Tensor) -> torch.Tensor:
            return self.regressor(self.features(x))

    class ExportWrapper(nn.Module):
        def __init__(self, model: nn.Module) -> None:
            super().__init__()
            self.model = model

        def forward(self, x: torch.Tensor) -> torch.Tensor:
            return self.model(x) * 90.0

    random.Random(42).shuffle(samples)
    split = max(1, int(len(samples) * 0.8))
    train_samples = samples[:split]
    val_samples = samples[split:] or samples[:1]

    device = torch.device("cpu")
    epochs = 3 if args.quick else args.epochs
    model = TinyHeadPoseNet().to(device)
    optimizer = torch.optim.AdamW(model.parameters(), lr=args.lr, weight_decay=1e-4)
    loss_fn = nn.SmoothL1Loss()
    train_loader = DataLoader(HeadPoseDataset(train_samples), batch_size=args.batch, shuffle=True, num_workers=0)
    val_loader = DataLoader(HeadPoseDataset(val_samples), batch_size=args.batch, shuffle=False, num_workers=0)

    best_mae = float("inf")
    checkpoint = ROOT / "head_pose_estimator.pt"
    for epoch in range(1, epochs + 1):
        model.train()
        total_loss = 0.0
        for images, labels in train_loader:
            images = images.to(device)
            labels = labels.to(device)
            optimizer.zero_grad(set_to_none=True)
            loss = loss_fn(model(images), labels)
            loss.backward()
            optimizer.step()
            total_loss += float(loss.item()) * images.size(0)

        model.eval()
        errors: list[torch.Tensor] = []
        with torch.no_grad():
            for images, labels in val_loader:
                prediction = model(images.to(device)).cpu() * 90.0
                target = labels * 90.0
                errors.append(torch.abs(prediction - target))
        mae = torch.cat(errors).mean(dim=0)
        mean_mae = float(mae.mean().item())
        print(
            f"epoch {epoch}/{epochs} loss={total_loss / len(train_samples):.4f} "
            f"tilt_mae={mae[0].item():.2f} pan_mae={mae[1].item():.2f}"
        )
        if mean_mae < best_mae:
            best_mae = mean_mae
            torch.save(model.state_dict(), checkpoint)

    model.load_state_dict(torch.load(checkpoint, map_location=device))
    model.eval()
    wrapper = ExportWrapper(model).eval()
    dummy = torch.zeros(1, 3, args.imgsz, args.imgsz, dtype=torch.float32)
    torch.onnx.export(
        wrapper,
        dummy,
        ONNX_PATH,
        input_names=["image"],
        output_names=["tilt_pan_degrees"],
        opset_version=12,
        dynamic_axes={"image": {0: "batch"}, "tilt_pan_degrees": {0: "batch"}},
    )
    return ONNX_PATH


def optimize_onnx(model_path: Path) -> Path | None:
    try:
        import onnxruntime as ort
    except Exception as exc:
        print(f"ONNX optimization skipped: {exc}")
        return None

    try:
        options = ort.SessionOptions()
        options.graph_optimization_level = ort.GraphOptimizationLevel.ORT_ENABLE_ALL
        options.optimized_model_filepath = str(OPTIMIZED_ONNX_PATH)
        ort.InferenceSession(str(model_path), sess_options=options, providers=["CPUExecutionProvider"])
        return OPTIMIZED_ONNX_PATH
    except Exception as exc:
        print(f"ONNX optimization skipped: {exc}")
        return None


def main() -> None:
    args = parse_args()
    samples, report = load_samples(args.dataset_root, args.images_dir)
    print(json.dumps(report, indent=2))

    if args.validate_only:
        return
    if not samples:
        raise SystemExit(
            "No KARA image files were found. Place the matching person*.jpg files beside the CSVs "
            "or pass --images-dir before training."
        )

    ONNX_PATH.parent.mkdir(parents=True, exist_ok=True)
    onnx_path = train(samples, args)
    print(f"ONNX head-pose model exported: {onnx_path}")
    optimized_path = optimize_onnx(onnx_path)
    if optimized_path:
        print(f"ONNX Runtime optimized model exported: {optimized_path}")

    if (ROOT / "__pycache__").exists():
        shutil.rmtree(ROOT / "__pycache__", ignore_errors=True)


if __name__ == "__main__":
    main()
