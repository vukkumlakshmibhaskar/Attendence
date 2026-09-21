import argparse
import json
import os
import random
import sys
import time
import urllib.request
from pathlib import Path
from typing import Dict, List, Optional, Tuple

import cv2
import numpy as np

BASE_DIR = os.path.dirname(os.path.abspath(__file__))
MODELS_DIR = os.path.join(BASE_DIR, "models")
os.makedirs(MODELS_DIR, exist_ok=True)

YUNET_URL = "https://github.com/opencv/opencv_zoo/raw/main/models/face_detection_yunet/face_detection_yunet_2023mar.onnx"
SFACE_URL = "https://github.com/opencv/opencv_zoo/raw/main/models/face_recognition_sface/face_recognition_sface_2021dec.onnx"

YUNET_PATH = os.path.join(MODELS_DIR, "face_detection_yunet_2023mar.onnx")
SFACE_PATH = os.path.join(MODELS_DIR, "face_recognition_sface_2021dec.onnx")

def ensure_deep_models():
    """Ensure OpenCV YuNet Face Detector and SFace Deep Recognizer are available."""
    if not os.path.exists(YUNET_PATH) or os.path.getsize(YUNET_PATH) < 100000:
        print(f"[DOWNLOAD] Downloading YuNet Face Detector to {YUNET_PATH}...")
        urllib.request.urlretrieve(YUNET_URL, YUNET_PATH)
    if not os.path.exists(SFACE_PATH) or os.path.getsize(SFACE_PATH) < 10000000:
        print(f"[DOWNLOAD] Downloading SFace Deep Recognizer to {SFACE_PATH}...")
        urllib.request.urlretrieve(SFACE_URL, SFACE_PATH)

def init_detectors() -> Tuple[cv2.FaceDetectorYN, cv2.FaceRecognizerSF]:
    ensure_deep_models()
    detector = cv2.FaceDetectorYN_create(YUNET_PATH, "", (320, 320), score_threshold=0.6, nms_threshold=0.3)
    recognizer = cv2.FaceRecognizerSF_create(SFACE_PATH, "")
    return detector, recognizer

def extract_deep_embedding(img_path: str, detector: cv2.FaceDetectorYN, recognizer: cv2.FaceRecognizerSF) -> Optional[np.ndarray]:
    """Detects face, aligns with 5 facial landmarks, and extracts 128-d L2-normalized embedding."""
    img = cv2.imread(img_path)
    if img is None:
        return None

    h, w, _ = img.shape
    detector.setInputSize((w, h))
    _, faces = detector.detect(img)

    if faces is None or len(faces) == 0:
        # Fallback: center crop if detector misses
        min_dim = min(h, w)
        cy, cx = h // 2, w // 2
        y0, x0 = max(0, cy - min_dim // 2), max(0, cx - min_dim // 2)
        crop = img[y0:y0+min_dim, x0:x0+min_dim]
        detector.setInputSize((min_dim, min_dim))
        _, faces = detector.detect(crop)
        if faces is None or len(faces) == 0:
            return None
        face_aligned = recognizer.alignCrop(crop, faces[0])
    else:
        # Pick the largest detected face box
        best_face = max(faces, key=lambda f: f[2] * f[3])
        face_aligned = recognizer.alignCrop(img, best_face)

    raw_feat = recognizer.feature(face_aligned).flatten().astype(np.float64)
    norm = np.linalg.norm(raw_feat)
    if norm > 1e-9:
        raw_feat = raw_feat / norm
    return raw_feat

def train_face_verifier(
    dataset_dir: str,
    output_model: str,
    max_people: int = 105,
    max_images_per_person: int = 20,
    max_pairs: int = 20000,
    epochs: int = 300,
    lr: float = 0.5
):
    print("=" * 70)
    print("  Deep Face Verification Trainer (YuNet + SFace CNN)")
    print(f"  Dataset:       {dataset_dir}")
    print(f"  Output Model:  {output_model}")
    print(f"  Classes Limit: {max_people} | Max Images/Person: {max_images_per_person}")
    print(f"  Epochs:        {epochs} | Learning Rate: {lr}")
    print("=" * 70)

    detector, recognizer = init_detectors()

    print("\n[1/4] Scanning dataset structure...")
    valid_exts = {".jpg", ".jpeg", ".png", ".bmp", ".webp"}
    people: Dict[str, List[str]] = {}

    entries = sorted(os.listdir(dataset_dir))
    for entry in entries:
        full_path = os.path.join(dataset_dir, entry)
        if os.path.isdir(full_path):
            images = [os.path.join(full_path, f) for f in os.listdir(full_path) if Path(f).suffix.lower() in valid_exts]
            if len(images) >= 2:
                random.shuffle(images)
                people[entry] = images[:max_images_per_person]
                if len(people) >= max_people:
                    break

    total_images = sum(len(v) for v in people.values())
    print(f"      Selected {len(people)} subjects with {total_images} total face images.")
    if len(people) < 2:
        print("[ERROR] Found fewer than 2 subjects with images in dataset.")
        return

    print("\n[2/4] Detecting faces & extracting 128-d deep facial embeddings...")
    embeddings: Dict[str, np.ndarray] = {}
    count = 0
    t0 = time.time()

    for person, paths in people.items():
        for p in paths:
            emb = extract_deep_embedding(p, detector, recognizer)
            if emb is not None:
                embeddings[p] = emb
            count += 1
            if count % 100 == 0 or count == total_images:
                pct = (count / total_images) * 100
                sys.stdout.write(f"\r      Progress: {count}/{total_images} ({pct:.1f}%) processed | {len(embeddings)} valid faces")
                sys.stdout.flush()

    duration = time.time() - t0
    print(f"\n      Extracted {len(embeddings)} deep embeddings in {duration:.1f}s ({len(embeddings)/max(1, duration):.1f} fps).")

    print("\n[3/4] Generating balanced verification pairs (same-person vs different-person)...")
    pos_pairs = []
    for person, paths in people.items():
        valid_paths = [p for p in paths if p in embeddings]
        for i in range(len(valid_paths)):
            for j in range(i + 1, len(valid_paths)):
                pos_pairs.append((valid_paths[i], valid_paths[j]))

    random.shuffle(pos_pairs)
    pos_pairs = pos_pairs[:max_pairs]

    neg_pairs = []
    all_people = list(people.keys())
    while len(neg_pairs) < len(pos_pairs):
        p1, p2 = random.sample(all_people, 2)
        valid1 = [p for p in people[p1] if p in embeddings]
        valid2 = [p for p in people[p2] if p in embeddings]
        if valid1 and valid2:
            neg_pairs.append((random.choice(valid1), random.choice(valid2)))

    print(f"      Total pairs: {len(pos_pairs) + len(neg_pairs)} ({len(pos_pairs)} positive, {len(neg_pairs)} negative)")

    # Compute cosine similarities and distances
    pos_sims = [float(np.dot(embeddings[p1], embeddings[p2])) for p1, p2 in pos_pairs]
    neg_sims = [float(np.dot(embeddings[p1], embeddings[p2])) for p1, p2 in neg_pairs]

    # Find optimal cosine decision threshold
    best_th = 0.363
    best_th_acc = 0.0
    for candidate_th in np.linspace(0.10, 0.70, 201):
        c_acc = (sum(1 for s in pos_sims if s >= candidate_th) + sum(1 for s in neg_sims if s < candidate_th)) / (len(pos_sims) + len(neg_sims)) * 100.0
        if c_acc > best_th_acc:
            best_th_acc = c_acc
            best_th = float(candidate_th)

    print(f"      Cosine Metric Analysis:")
    print(f"        Same-person average similarity:      {np.mean(pos_sims):.4f} (std: {np.std(pos_sims):.4f})")
    print(f"        Different-person average similarity: {np.mean(neg_sims):.4f} (std: {np.std(neg_sims):.4f})")
    print(f"        Direct Cosine Accuracy:              {best_th_acc:.2f}% (at threshold = {best_th:.3f})")

    # Build feature dataset for calibrated logistic verifier: [cos_sim, l2_dist, diff_vec]
    X, y = [], []
    for p1, p2 in pos_pairs:
        v1, v2 = embeddings[p1], embeddings[p2]
        cos_sim = float(np.dot(v1, v2))
        l2_dist = float(np.linalg.norm(v1 - v2))
        diff = np.abs(v1 - v2)
        feat = np.concatenate([[cos_sim, l2_dist], diff])
        X.append(feat)
        y.append(1.0)

    for p1, p2 in neg_pairs:
        v1, v2 = embeddings[p1], embeddings[p2]
        cos_sim = float(np.dot(v1, v2))
        l2_dist = float(np.linalg.norm(v1 - v2))
        diff = np.abs(v1 - v2)
        feat = np.concatenate([[cos_sim, l2_dist], diff])
        X.append(feat)
        y.append(0.0)

    X = np.array(X, dtype=np.float64)
    y = np.array(y, dtype=np.float64)

    # Train / Test split (85% train, 15% test)
    indices = np.arange(len(X))
    np.random.shuffle(indices)
    split_idx = int(0.85 * len(X))
    train_idx, test_idx = indices[:split_idx], indices[split_idx:]

    X_train, y_train = X[train_idx], y[train_idx]
    X_test, y_test = X[test_idx], y[test_idx]

    print(f"\n[4/4] Training Calibrated Logistic Verifier ({epochs} epochs, lr={lr})...")
    weights = np.zeros(X.shape[1], dtype=np.float64)
    # Initialize bias and cosine weight for fast gradient convergence
    weights[0] = 5.0   # positive weight for cosine similarity
    weights[1] = -3.0  # negative weight for L2 distance
    bias = -1.0

    for epoch in range(1, epochs + 1):
        linear = np.dot(X_train, weights) + bias
        preds = 1.0 / (1.0 + np.exp(-np.clip(linear, -25.0, 25.0)))
        errors = preds - y_train

        dw = np.dot(X_train.T, errors) / len(X_train)
        db = np.sum(errors) / len(X_train)

        weights -= lr * dw
        bias -= lr * db

        if epoch % 50 == 0 or epoch == epochs:
            loss = float(-np.mean(y_train * np.log(np.clip(preds, 1e-7, 1 - 1e-7)) + (1.0 - y_train) * np.log(np.clip(1.0 - preds, 1e-7, 1 - 1e-7))))
            train_acc = float(np.mean((preds >= 0.5) == y_train) * 100.0)
            print(f"      Epoch {epoch:3d}/{epochs} | Loss: {loss:.4f} | Train Acc: {train_acc:.2f}%")

    # Evaluate on held-out test set
    test_linear = np.dot(X_test, weights) + bias
    test_probs = 1.0 / (1.0 + np.exp(-np.clip(test_linear, -25.0, 25.0)))
    test_preds = test_probs >= 0.5
    accuracy = float(np.mean(test_preds == y_test) * 100.0)

    tp = np.sum((test_preds == 1) & (y_test == 1))
    fp = np.sum((test_preds == 1) & (y_test == 0))
    tn = np.sum((test_preds == 0) & (y_test == 0))
    fn = np.sum((test_preds == 0) & (y_test == 1))
    precision = float((tp / (tp + fp)) * 100.0) if (tp + fp) > 0 else 0.0
    recall = float((tp / (tp + fn)) * 100.0) if (tp + fn) > 0 else 0.0

    print("\n" + "=" * 50)
    print(f"  Test Accuracy:     {accuracy:.2f}%")
    print(f"  Precision:         {precision:.2f}%")
    print(f"  Recall:            {recall:.2f}%")
    print(f"  Optimal Cosine Th: {best_th:.3f} ({best_th_acc:.2f}% raw)")
    print("=" * 50)

    model_data = {
        "model_type": "sface_deep_verifier_128d",
        "optimal_threshold": round(best_th, 3),
        "raw_cosine_accuracy": round(best_th_acc, 2),
        "test_accuracy": round(accuracy, 2),
        "precision": round(precision, 2),
        "recall": round(recall, 2),
        "weights": weights.tolist(),
        "bias": float(bias),
        "dataset_name": os.path.basename(dataset_dir),
        "epochs": epochs,
        "learning_rate": lr,
        "input_features": X.shape[1],
        "trained_at": time.strftime("%Y-%m-%d %H:%M:%S")
    }

    with open(output_model, "w") as f:
        json.dump(model_data, f, indent=2)

    print(f"\n[SUCCESS] Calibrated deep verifier model saved to: {output_model}\n")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Deep Face Verifier Trainer (YuNet + SFace CNN)")
    parser.add_argument("--dataset", type=str, required=True, help="Path to face dataset folder")
    parser.add_argument("--output", type=str, default="face_verifier_pins.json", help="Output model JSON path")
    parser.add_argument("--max-people", type=int, default=105, help="Number of people classes to use (default: 105)")
    parser.add_argument("--max-images", type=int, default=20, help="Max images per person (default: 20)")
    parser.add_argument("--max-pairs", type=int, default=20000, help="Max training pairs (default: 20000)")
    parser.add_argument("--epochs", type=int, default=300, help="Number of training epochs (default: 300)")
    parser.add_argument("--lr", type=float, default=0.5, help="Learning rate (default: 0.5)")
    args = parser.parse_args()

    train_face_verifier(
        args.dataset,
        args.output,
        max_people=args.max_people,
        max_images_per_person=args.max_images,
        max_pairs=args.max_pairs,
        epochs=args.epochs,
        lr=args.lr
    )
