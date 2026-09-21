import argparse
import json
import os
import sys
from typing import Optional, Tuple

import cv2
import numpy as np

BASE_DIR = os.path.dirname(os.path.abspath(__file__))
MODELS_DIR = os.path.join(BASE_DIR, "models")
YUNET_PATH = os.path.join(MODELS_DIR, "face_detection_yunet_2023mar.onnx")
SFACE_PATH = os.path.join(MODELS_DIR, "face_recognition_sface_2021dec.onnx")
MODEL_JSON = os.path.join(BASE_DIR, "face_verifier_pins.json")

def load_models():
    if not os.path.exists(YUNET_PATH) or not os.path.exists(SFACE_PATH):
        print("[ERROR] Deep face models not found in backend/models/.")
        sys.exit(1)

    detector = cv2.FaceDetectorYN_create(YUNET_PATH, "", (320, 320), score_threshold=0.6, nms_threshold=0.3)
    recognizer = cv2.FaceRecognizerSF_create(SFACE_PATH, "")

    verifier_config = {}
    if os.path.exists(MODEL_JSON):
        with open(MODEL_JSON, "r") as f:
            verifier_config = json.load(f)
            print(f"[MODEL] Loaded '{os.path.basename(MODEL_JSON)}' (Trained Accuracy: {verifier_config.get('test_accuracy', 'N/A')}%)")
    else:
        print("[WARN] face_verifier_pins.json not found, using default SFace threshold (0.363).")

    return detector, recognizer, verifier_config

def get_embedding(img_path: str, detector: cv2.FaceDetectorYN, recognizer: cv2.FaceRecognizerSF) -> Tuple[Optional[np.ndarray], Optional[dict]]:
    img = cv2.imread(img_path)
    if img is None:
        print(f"[ERROR] Could not read image: {img_path}")
        return None, None

    h, w, _ = img.shape
    detector.setInputSize((w, h))
    _, faces = detector.detect(img)

    if faces is None or len(faces) == 0:
        return None, None

    best_face = max(faces, key=lambda f: f[2] * f[3])
    aligned = recognizer.alignCrop(img, best_face)
    feat = recognizer.feature(aligned).flatten().astype(np.float64)
    norm = np.linalg.norm(feat)
    if norm > 1e-9:
        feat = feat / norm

    box = {"x": int(best_face[0]), "y": int(best_face[1]), "width": int(best_face[2]), "height": int(best_face[3])}
    return feat, box

def verify_pair(img1_path: str, img2_path: str):
    detector, recognizer, config = load_models()

    print(f"\nComparing:")
    print(f"  Photo 1: {img1_path}")
    print(f"  Photo 2: {img2_path}")
    print("-" * 55)

    emb1, box1 = get_embedding(img1_path, detector, recognizer)
    if emb1 is None:
        print(f"[FAILED] No face detected in Photo 1: {img1_path}")
        return

    emb2, box2 = get_embedding(img2_path, detector, recognizer)
    if emb2 is None:
        print(f"[FAILED] No face detected in Photo 2: {img2_path}")
        return

    # Metrics
    cos_sim = float(np.dot(emb1, emb2))
    l2_dist = float(np.linalg.norm(emb1 - emb2))
    threshold = float(config.get("optimal_threshold", 0.277))

    # Logistic calibration
    confidence = 0.0
    if "weights" in config and "bias" in config:
        diff = np.abs(emb1 - emb2)
        feat = np.concatenate([[cos_sim, l2_dist], diff])
        weights = np.array(config["weights"])
        bias = float(config["bias"])
        if len(feat) == len(weights):
            linear = np.dot(feat, weights) + bias
            prob = float(1.0 / (1.0 + np.exp(-np.clip(linear, -25.0, 25.0))))
            confidence = prob

    if confidence == 0.0:
        # Fallback to cosine scaling
        if cos_sim >= threshold:
            confidence = 0.70 + 0.30 * min(1.0, (cos_sim - threshold) / (1.0 - threshold + 1e-9))
        else:
            confidence = max(0.0, 0.50 * (cos_sim / max(threshold, 1e-9)))

    is_match = (cos_sim >= threshold) or (confidence >= 0.50)

    print(f"  Cosine Similarity:  {cos_sim:.4f}  (Threshold: {threshold:.3f})")
    print(f"  Euclidean Distance: {l2_dist:.4f}")
    print(f"  Model Confidence:   {confidence * 100.0:.2f}%")
    print("-" * 55)

    if is_match:
        print("  [RESULT] -> MATCH! Both photos belong to the SAME PERSON.")
    else:
        print("  [RESULT] -> NO MATCH. Photos belong to DIFFERENT PEOPLE.")
    print("-" * 55 + "\n")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Verify if two face photos belong to the same person using the trained model.")
    parser.add_argument("--img1", type=str, required=True, help="Path to first face photo")
    parser.add_argument("--img2", type=str, required=True, help="Path to second face photo")
    args = parser.parse_args()

    verify_pair(args.img1, args.img2)
