import argparse
import os
import sys
import numpy as np
import cv2
from typing import Optional, Tuple

from ai_engine import ai_engine

def verify_pair(img1_path: str, img2_path: str):
    print("=" * 72)
    print("  AI Face Verification & Anti-Spoofing Evaluator")
    print(f"  Photo 1: {img1_path}")
    print(f"  Photo 2: {img2_path}")
    print("=" * 72)

    if not os.path.exists(img1_path) or not os.path.exists(img2_path):
        print("[ERROR] One or both image files do not exist.")
        return

    im1 = cv2.imread(img1_path)
    im2 = cv2.imread(img2_path)

    res1 = ai_engine.detect_and_extract(im1)
    if not res1:
        print(f"[FAILED] No face detected in Photo 1: {img1_path}")
        return

    res2 = ai_engine.detect_and_extract(im2)
    if not res2:
        print(f"[FAILED] No face detected in Photo 2: {img2_path}")
        return

    f1, f2 = res1[0], res2[0]

    # 1. Anti-Spoofing (MiniFASNet v2)
    print("\n[LIVENESS & ANTI-SPOOFING ASSESSMENT]")
    p1_live = "GENUINE LIVE FACE" if f1["is_live"] else "SPOOF / SCREEN REPLAY / PRINT DETECTED"
    p2_live = "GENUINE LIVE FACE" if f2["is_live"] else "SPOOF / SCREEN REPLAY / PRINT DETECTED"
    print(f"  • Photo 1: {p1_live} (Fused: {f1['liveness_confidence']*100:.1f}%, Texture: {f1.get('texture_liveness', 0)*100:.1f}%, Depth: {f1.get('depth_liveness', 0)*100:.1f}%)")
    print(f"  • Photo 2: {p2_live} (Fused: {f2['liveness_confidence']*100:.1f}%, Texture: {f2.get('texture_liveness', 0)*100:.1f}%, Depth: {f2.get('depth_liveness', 0)*100:.1f}%)")

    # 2. Mask status
    if f1["is_masked"] or f2["is_masked"]:
        print(f"  • Mask Status: Face mask detected (Photo 1: {f1['is_masked']}, Photo 2: {f2['is_masked']})")

    # 3. Model Accuracy Comparison
    # A. InsightFace MobileFaceNet (512-d ArcFace)
    arc_v1 = f1["embedding_512"]
    arc_v2 = f2["embedding_512"]
    is_masked = f1["is_masked"] or f2["is_masked"]
    arc_dist, arc_conf, arc_match = ai_engine.compare_embeddings(arc_v1, arc_v2, is_masked=is_masked)
    arc_sim = float(np.dot(arc_v1, arc_v2))

    # B. Legacy SFace (128-d)
    sface_v1 = f1["embedding_128"]
    sface_v2 = f2["embedding_128"]
    sface_dist, sface_conf, sface_match = ai_engine.compare_embeddings(sface_v1, sface_v2)
    sface_sim = float(np.dot(sface_v1, sface_v2))

    print("\n[MODEL COMPARISON METRICS]")
    print(f"  {'Model':<30} | {'Cosine Sim':<12} | {'Confidence':<12} | {'Decision'}")
    print("  " + "-" * 70)
    print(f"  {'InsightFace ArcFace (512-d)':<30} | {arc_sim:<12.4f} | {arc_conf*100:<11.2f}% | {'MATCH (Same Person)' if arc_match else 'NO MATCH (Different)'}")
    print(f"  {'Legacy SFace (128-d)':<30} | {sface_sim:<12.4f} | {sface_conf*100:<11.2f}% | {'MATCH (Same Person)' if sface_match else 'NO MATCH (Different)'}")

    print("\n" + "=" * 72)
    if arc_match:
        print("  [FINAL RESULT] -> MATCH! Verified as the SAME PERSON.")
    else:
        print("  [FINAL RESULT] -> NO MATCH. Verified as DIFFERENT PEOPLE.")
    print("=" * 72 + "\n")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Verify if two face photos belong to the same person with Anti-Spoofing and ArcFace.")
    parser.add_argument("--img1", type=str, required=True, help="Path to first face photo")
    parser.add_argument("--img2", type=str, required=True, help="Path to second face photo")
    args = parser.parse_args()

    verify_pair(args.img1, args.img2)
