import argparse
import csv
import itertools
import json
import random
import re
from dataclasses import dataclass
from pathlib import Path

import numpy as np
from PIL import Image
from sklearn.linear_model import LogisticRegression
from sklearn.metrics import accuracy_score, precision_recall_fscore_support, roc_auc_score
from sklearn.model_selection import train_test_split
from sklearn.preprocessing import StandardScaler


IMAGE_RE = re.compile(r"^(?P<name>.+)_(?P<num>\d{4})\.jpg$", re.IGNORECASE)


@dataclass(frozen=True)
class ImageKey:
    name: str
    num: int


def descriptor_from_image(path: Path) -> np.ndarray:
    image = Image.open(path).convert("RGB")
    arr = np.asarray(image, dtype=np.float32) / 255.0
    height, width = arr.shape[:2]
    if width <= 0 or height <= 0:
        raise ValueError(f"invalid image size for {path}")

    x0 = width // 5
    x1 = width - width // 5
    y0 = height // 12
    y1 = height - height // 10
    roi = arr[y0:y1, x0:x1]
    if roi.size == 0:
        roi = arr

    gray = 0.299 * roi[:, :, 0] + 0.587 * roi[:, :, 1] + 0.114 * roi[:, :, 2]
    image_small = Image.fromarray(np.uint8(np.clip(gray * 255.0, 0, 255)), mode="L")
    image_small = image_small.resize((16, 8), resample=Image.Resampling.BOX)
    vector = np.asarray(image_small, dtype=np.float32).reshape(-1) / 255.0
    vector = vector - float(vector.mean())
    std = float(vector.std())
    if std > 1e-6:
        vector = vector / std
    norm = float(np.linalg.norm(vector))
    if norm > 1e-8:
        vector = vector / norm
    return vector.astype(np.float32)


def find_lfw_root(dataset: Path) -> Path:
    direct = dataset / "lfw-deepfunneled" / "lfw-deepfunneled"
    if direct.exists():
        return direct
    candidates = [item for item in dataset.rglob("*") if item.is_dir() and any(item.glob("*_0001.jpg"))]
    if not candidates:
        raise FileNotFoundError(f"could not find LFW image folders under {dataset}")
    return max(candidates, key=lambda path: len(list(path.glob("*/*.jpg"))))


def load_descriptors(image_root: Path):
    descriptors = {}
    people = {}
    for path in image_root.rglob("*.jpg"):
        match = IMAGE_RE.match(path.name)
        if not match:
            continue
        key = ImageKey(match.group("name"), int(match.group("num")))
        descriptors[key] = descriptor_from_image(path)
        people.setdefault(key.name, []).append(key)
    for values in people.values():
        values.sort(key=lambda item: item.num)
    return descriptors, people


def add_pair(pairs, seen, descriptors, first: ImageKey, second: ImageKey, label: int, source: str):
    if first not in descriptors or second not in descriptors or first == second:
        return 0
    a, b = sorted((first, second), key=lambda item: (item.name, item.num))
    dedupe_key = (a.name, a.num, b.name, b.num, label)
    if dedupe_key in seen:
        return 0
    seen.add(dedupe_key)
    pairs.append((a, b, label, source))
    return 1


def load_csv_pairs(dataset: Path, descriptors):
    pairs = []
    seen = set()
    same_files = ["matchpairsDevTrain.csv", "matchpairsDevTest.csv", "pairs.csv"]
    different_files = ["mismatchpairsDevTrain.csv", "mismatchpairsDevTest.csv"]

    for filename in same_files:
        path = dataset / filename
        if not path.exists():
            continue
        with path.open("r", encoding="utf-8-sig", newline="") as handle:
            reader = csv.DictReader(handle)
            for row in reader:
                name = (row.get("name") or "").strip()
                n1 = parse_int(row.get("imagenum1"))
                n2 = parse_int(row.get("imagenum2"))
                if name and n1 and n2:
                    add_pair(pairs, seen, descriptors, ImageKey(name, n1), ImageKey(name, n2), 1, filename)

    for filename in different_files:
        path = dataset / filename
        if not path.exists():
            continue
        with path.open("r", encoding="utf-8-sig", newline="") as handle:
            reader = csv.reader(handle)
            header = next(reader, None)
            for row in reader:
                if len(row) < 4:
                    continue
                name1 = row[0].strip()
                n1 = parse_int(row[1])
                name2 = row[2].strip()
                n2 = parse_int(row[3])
                if name1 and name2 and n1 and n2:
                    add_pair(pairs, seen, descriptors, ImageKey(name1, n1), ImageKey(name2, n2), 0, filename)
    return pairs, seen


def parse_int(value):
    try:
        parsed = int(str(value).strip())
        return parsed if parsed > 0 else None
    except Exception:
        return None


def add_generated_pairs(pairs, seen, descriptors, people, max_positive, max_negative, rng):
    positive_added = 0
    for keys in people.values():
        if len(keys) < 2:
            continue
        for first, second in zip(keys, keys[1:]):
            positive_added += add_pair(pairs, seen, descriptors, first, second, 1, "generated_adjacent_positive")

    positive_candidates = []
    for keys in people.values():
        if len(keys) < 2:
            continue
        positive_candidates.extend(itertools.combinations(keys, 2))
    rng.shuffle(positive_candidates)
    for first, second in positive_candidates:
        if positive_added >= max_positive:
            break
        positive_added += add_pair(pairs, seen, descriptors, first, second, 1, "generated_positive")

    all_people = [name for name, keys in people.items() if keys]
    negative_added = 0
    attempts = 0
    max_attempts = max_negative * 20
    while negative_added < max_negative and attempts < max_attempts and len(all_people) >= 2:
        attempts += 1
        name1, name2 = rng.sample(all_people, 2)
        first = rng.choice(people[name1])
        second = rng.choice(people[name2])
        negative_added += add_pair(pairs, seen, descriptors, first, second, 0, "generated_negative")


def pair_features(a: np.ndarray, b: np.ndarray) -> np.ndarray:
    diff = np.abs(a - b)
    euclidean = np.linalg.norm(a - b)
    cosine_distance = 1.0 - float(np.dot(a, b))
    return np.concatenate([diff, np.array([euclidean, cosine_distance], dtype=np.float32)])


def build_matrix(pairs, descriptors):
    x = np.empty((len(pairs), 130), dtype=np.float32)
    y = np.empty((len(pairs),), dtype=np.int32)
    for index, (first, second, label, _source) in enumerate(pairs):
        x[index] = pair_features(descriptors[first], descriptors[second])
        y[index] = label
    return x, y


def choose_threshold(y_true, probabilities, target_precision):
    best = None
    for threshold in np.linspace(0.05, 0.99, 189):
        predicted = (probabilities >= threshold).astype(np.int32)
        precision, recall, f1, _ = precision_recall_fscore_support(
            y_true,
            predicted,
            average="binary",
            zero_division=0,
        )
        false_positive_rate = false_positive_rate_for(y_true, predicted)
        candidate = {
            "threshold": float(threshold),
            "precision": float(precision),
            "recall": float(recall),
            "f1": float(f1),
            "false_positive_rate": float(false_positive_rate),
        }
        if precision >= target_precision:
            score = (recall, -false_positive_rate, f1)
        else:
            score = (precision - target_precision, recall * 0.5, -false_positive_rate)
        if best is None or score > best[0]:
            best = (score, candidate)
    return best[1]


def false_positive_rate_for(y_true, predicted):
    negatives = y_true == 0
    if not np.any(negatives):
        return 0.0
    return float(np.mean(predicted[negatives] == 1))


def main():
    parser = argparse.ArgumentParser(description="Train a CPU face-pair verifier from an LFW-style dataset.")
    parser.add_argument("--dataset", required=True, type=Path)
    parser.add_argument("--output", default=Path("models/face_verifier_lfw.json"), type=Path)
    parser.add_argument("--report", default=Path("models/face_verifier_lfw_report.json"), type=Path)
    parser.add_argument("--max-generated-positive", type=int, default=80000)
    parser.add_argument("--max-generated-negative", type=int, default=90000)
    parser.add_argument("--target-precision", type=float, default=0.985)
    parser.add_argument("--seed", type=int, default=42)
    args = parser.parse_args()

    rng = random.Random(args.seed)
    image_root = find_lfw_root(args.dataset)
    print(f"image_root={image_root}")
    descriptors, people = load_descriptors(image_root)
    print(f"images={len(descriptors)} people={len(people)}")

    pairs, seen = load_csv_pairs(args.dataset, descriptors)
    print(f"csv_pairs={len(pairs)}")
    add_generated_pairs(
        pairs,
        seen,
        descriptors,
        people,
        args.max_generated_positive,
        args.max_generated_negative,
        rng,
    )
    rng.shuffle(pairs)
    labels = [label for *_rest, label, _source in pairs]
    positives = int(sum(labels))
    negatives = len(labels) - positives
    print(f"training_pairs={len(pairs)} positives={positives} negatives={negatives}")

    x, y = build_matrix(pairs, descriptors)
    x_train, x_valid, y_train, y_valid = train_test_split(
        x,
        y,
        test_size=0.22,
        random_state=args.seed,
        stratify=y,
    )

    scaler = StandardScaler()
    x_train_scaled = scaler.fit_transform(x_train)
    x_valid_scaled = scaler.transform(x_valid)
    model = LogisticRegression(max_iter=1200, class_weight="balanced", solver="lbfgs")
    model.fit(x_train_scaled, y_train)

    probabilities = model.predict_proba(x_valid_scaled)[:, 1]
    threshold_metrics = choose_threshold(y_valid, probabilities, args.target_precision)
    predicted = (probabilities >= threshold_metrics["threshold"]).astype(np.int32)
    precision, recall, f1, _ = precision_recall_fscore_support(
        y_valid,
        predicted,
        average="binary",
        zero_division=0,
    )
    accuracy = accuracy_score(y_valid, predicted)
    auc = roc_auc_score(y_valid, probabilities)

    coef = model.coef_[0].astype(np.float64)
    scale = scaler.scale_.astype(np.float64)
    mean = scaler.mean_.astype(np.float64)
    linear_weights = coef / scale
    linear_intercept = float(model.intercept_[0] - np.sum((coef * mean) / scale))

    output = {
        "version": 1,
        "model_type": "logistic_pair_verifier",
        "descriptor": "cpu_hash_grid_16x8_l2",
        "feature_count": int(linear_weights.shape[0]),
        "weights": [float(value) for value in linear_weights],
        "intercept": linear_intercept,
        "threshold": float(threshold_metrics["threshold"]),
        "target_precision": float(args.target_precision),
        "training": {
            "dataset": str(args.dataset),
            "image_root": str(image_root),
            "images": int(len(descriptors)),
            "people": int(len(people)),
            "pairs": int(len(pairs)),
            "positive_pairs": positives,
            "negative_pairs": negatives,
            "validation_pairs": int(len(y_valid)),
            "accuracy": float(accuracy),
            "precision": float(precision),
            "recall": float(recall),
            "f1": float(f1),
            "roc_auc": float(auc),
            "false_positive_rate": float(false_positive_rate_for(y_valid, predicted)),
        },
    }

    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.report.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(output, indent=2), encoding="utf-8")
    args.report.write_text(json.dumps(output["training"], indent=2), encoding="utf-8")
    print(f"saved_model={args.output}")
    print(f"saved_report={args.report}")
    print(json.dumps(output["training"], indent=2))


if __name__ == "__main__":
    main()
