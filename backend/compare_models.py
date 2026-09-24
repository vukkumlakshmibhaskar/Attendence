import argparse
import os
import sys
import time
from pathlib import Path
from typing import Dict, List, Tuple
import cv2
import numpy as np

# Import our AI Engine
from ai_engine import ai_engine, BASE_DIR

def compare_two_images(img1_path: str, img2_path: str):
    print("=" * 70)
    print("      FACE PIPELINE COMPARISON BENCHMARK: SFACE vs INSIGHTFACE ARCFACE")
    print(f"  Image 1: {img1_path}")
    print(f"  Image 2: {img2_path}")
    print("=" * 70)

    if not os.path.exists(img1_path):
        print(f"[ERROR] Image 1 not found: {img1_path}")
        return
    if not os.path.exists(img2_path):
        print(f"[ERROR] Image 2 not found: {img2_path}")
        return

    im1 = cv2.imread(img1_path)
    im2 = cv2.imread(img2_path)

    # 1. Pipeline Analysis for Image 1
    t0 = time.perf_counter()
    res1 = ai_engine.detect_and_extract(im1)
    ms1 = (time.perf_counter() - t0) * 1000

    if not res1:
        print("[FAIL] No face detected in Image 1.")
        return

    # 2. Pipeline Analysis for Image 2
    t0 = time.perf_counter()
    res2 = ai_engine.detect_and_extract(im2)
    ms2 = (time.perf_counter() - t0) * 1000

    if not res2:
        print("[FAIL] No face detected in Image 2.")
        return

    f1 = res1[0]
    f2 = res2[0]

    # Model 1: Legacy SFace (128-d)
    sface_v1 = f1["embedding_128"]
    sface_v2 = f2["embedding_128"]
    sface_dist, sface_conf, sface_match = ai_engine.compare_embeddings(sface_v1, sface_v2)
    sface_cos = float(np.dot(sface_v1, sface_v2))

    # Model 2: InsightFace ArcFace (512-d)
    arc_v1 = f1["embedding_512"]
    arc_v2 = f2["embedding_512"]
    is_masked = f1["is_masked"] or f2["is_masked"]
    arc_dist, arc_conf, arc_match = ai_engine.compare_embeddings(arc_v1, arc_v2, is_masked=is_masked)
    arc_cos = float(np.dot(arc_v1, arc_v2))

    print("\n[1] IMAGE ATTRIBUTES & MULTI-MODAL LIVENESS:")
    print(f"  • Image 1 Anti-Spoofing: {'LIVE (Real)' if f1['is_live'] else 'SPOOF (Fake/Photo)'} (Fused Conf: {f1['liveness_confidence']*100:.1f}%)")
    print(f"      - Texture Analysis (MiniFASNet v2): {f1.get('texture_liveness', 0)*100:.1f}%")
    print(f"      - 3D Depth Analysis (CDCN++):       {f1.get('depth_liveness', 0)*100:.1f}%")
    print(f"      - Mask Detection (YOLOv8n):         {'MASKED' if f1['is_masked'] else 'NO MASK'} ({f1['mask_confidence']*100:.1f}%)")
    print(f"  • Image 2 Anti-Spoofing: {'LIVE (Real)' if f2['is_live'] else 'SPOOF (Fake/Photo)'} (Fused Conf: {f2['liveness_confidence']*100:.1f}%)")
    print(f"      - Texture Analysis (MiniFASNet v2): {f2.get('texture_liveness', 0)*100:.1f}%")
    print(f"      - 3D Depth Analysis (CDCN++):       {f2.get('depth_liveness', 0)*100:.1f}%")
    print(f"      - Mask Detection (YOLOv8n):         {'MASKED' if f2['is_masked'] else 'NO MASK'} ({f2['mask_confidence']*100:.1f}%)")

    print("\n[2] ACCURACY & SIMILARITY COMPARISON:")
    print(f"  {'Metric / Model':<28} | {'Legacy SFace (128-d)':<22} | {'InsightFace ArcFace (512-d)':<25}")
    print("  " + "-" * 78)
    print(f"  {'Embedding Dimension':<28} | {'128 dimensions':<22} | {'512 dimensions (4x richer)':<25}")
    print(f"  {'Cosine Similarity':<28} | {sface_cos:<22.4f} | {arc_cos:<25.4f}")
    print(f"  {'Euclidean Distance':<28} | {sface_dist:<22.4f} | {arc_dist:<25.4f}")
    print(f"  {'Match Confidence':<28} | {sface_conf*100:<21.2f}% | {arc_conf*100:<24.2f}%")
    print(f"  {'Match Decision':<28} | {'SAME PERSON' if sface_match else 'DIFFERENT':<22} | {'SAME PERSON' if arc_match else 'DIFFERENT':<25}")

    print("\n[3] LATENCY & THROUGHPUT (CPU):")
    print(f"  • Image 1 Full Pipeline (Detect + Liveness + Mask + Embed): {ms1:.1f} ms")
    print(f"  • Image 2 Full Pipeline (Detect + Liveness + Mask + Embed): {ms2:.1f} ms")
    print("=" * 70 + "\n")

def benchmark_dataset(folder_path: str):
    print("=" * 70)
    print(f"  BENCHMARKING IMAGES IN: {folder_path}")
    print("=" * 70)
    
    valid_exts = {".jpg", ".jpeg", ".png", ".webp"}
    files = [os.path.join(folder_path, f) for f in sorted(os.listdir(folder_path)) if Path(f).suffix.lower() in valid_exts]

    if not files:
        print("[ERROR] No image files found.")
        return

    print(f"Found {len(files)} images to evaluate.\n")
    print(f"{'Filename':<24} | {'Face Detect':<11} | {'Liveness (Texture+Depth)':<24} | {'Mask Detection':<16} | {'Speed (ms)'}")
    print("-" * 92)

    for f in files:
        fname = os.path.basename(f)
        im = cv2.imread(f)
        if im is None:
            continue
        t0 = time.perf_counter()
        res = ai_engine.detect_and_extract(im)
        elapsed = (time.perf_counter() - t0) * 1000

        if res:
            first = res[0]
            live_str = f"{'LIVE' if first['is_live'] else 'SPOOF'} ({first['liveness_confidence']*100:.1f}%)"
            mask_str = f"{'MASKED' if first['is_masked'] else 'NO MASK'} ({first['mask_confidence']*100:.1f}%)"
            print(f"{fname:<24} | {'YES (' + str(len(res)) + ')':<11} | {live_str:<24} | {mask_str:<16} | {elapsed:.1f} ms")
        else:
            print(f"{fname:<24} | {'NO':<11} | {'N/A':<24} | {'N/A':<16} | {elapsed:.1f} ms")

    print("=" * 88 + "\n")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Compare SFace vs InsightFace ArcFace accuracy and liveness.")
    parser.add_argument("--img1", type=str, help="First face image")
    parser.add_argument("--img2", type=str, help="Second face image")
    parser.add_argument("--dataset", type=str, help="Folder containing images to benchmark")
    args = parser.parse_args()

    if args.img1 and args.img2:
        compare_two_images(args.img1, args.img2)
    elif args.dataset:
        benchmark_dataset(args.dataset)
    else:
        # Default test using built-in test_data
        test_dir = os.path.join(BASE_DIR, "test_data")
        if os.path.exists(test_dir):
            benchmark_dataset(test_dir)
            t1 = os.path.join(test_dir, "t1.jpg")
            lena = os.path.join(test_dir, "lena.jpg")
            if os.path.exists(t1) and os.path.exists(lena):
                compare_two_images(t1, lena)
        else:
            parser.print_help()
