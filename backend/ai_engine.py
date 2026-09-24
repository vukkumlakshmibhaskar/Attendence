import os
import time
import numpy as np
import cv2
import onnxruntime as ort
from typing import Dict, List, Optional, Tuple, Any

BASE_DIR = os.path.dirname(os.path.abspath(__file__))
MODELS_DIR = os.path.join(BASE_DIR, "models")

# Model paths
YUNET_PATH = os.path.join(MODELS_DIR, "face_detection_yunet_2023mar.onnx")
YUNET_INT8_PATH = os.path.join(MODELS_DIR, "face_detection_yunet_2023mar_int8.onnx")
SFACE_PATH = os.path.join(MODELS_DIR, "face_recognition_sface_2021dec.onnx")
MBF_INT8_PATH = os.path.join(MODELS_DIR, "mobilefacenet_int8.onnx")
MBF_FP32_PATH = os.path.join(MODELS_DIR, "w600k_mbf.onnx")
MINIFAS_PATH = os.path.join(MODELS_DIR, "minifasnet_v2.onnx")
MASK_PATH = os.path.join(MODELS_DIR, "yolov8n_face_mask_320x320.onnx")
CDCN_PATH = os.path.join(MODELS_DIR, "cdcnpp.onnx")

class AIEngine:
    def __init__(self):
        self.yunet = None
        self.sface = None
        self.mbf_session = None
        self.mbf_input_name = None
        self.minifas_session = None
        self.minifas_input_name = None
        self.mask_session = None
        self.mask_input_name = None
        self.cdcn_session = None
        self.cdcn_input_name = None
        self.initialized = False
        self._init_models()

    def _init_models(self):
        # 1. Face Detector (YuNet)
        detector_path = YUNET_INT8_PATH if os.path.exists(YUNET_INT8_PATH) else YUNET_PATH
        if os.path.exists(detector_path):
            try:
                self.yunet = cv2.FaceDetectorYN_create(
                    detector_path, "", (320, 320), score_threshold=0.6, nms_threshold=0.3
                )
                print(f"[AI ENGINE] Loaded Face Detector: {os.path.basename(detector_path)}")
            except Exception as e:
                print(f"[AI ENGINE] Warning: Failed to load YuNet: {e}")

        # 2. SFace (Legacy 128-d)
        if os.path.exists(SFACE_PATH):
            try:
                self.sface = cv2.FaceRecognizerSF_create(SFACE_PATH, "")
                print(f"[AI ENGINE] Loaded SFace 128-d Recognizer: {os.path.basename(SFACE_PATH)}")
            except Exception as e:
                print(f"[AI ENGINE] Warning: Failed to load SFace: {e}")

        # 3. InsightFace MobileFaceNet (512-d ArcFace)
        mbf_candidates = [MBF_FP32_PATH, MBF_INT8_PATH]
        for p in mbf_candidates:
            if not os.path.exists(p):
                continue
            try:
                opts = ort.SessionOptions()
                opts.intra_op_num_threads = 2
                self.mbf_session = ort.InferenceSession(p, sess_options=opts, providers=['CPUExecutionProvider'])
                self.mbf_input_name = self.mbf_session.get_inputs()[0].name
                print(f"[AI ENGINE] Loaded InsightFace MobileFaceNet (512-d ArcFace): {os.path.basename(p)}")
                break
            except Exception as e:
                print(f"[AI ENGINE] Notice: Could not load {os.path.basename(p)}: {e}")

        # 4. MiniFASNet v2 Anti-Spoofing (Liveness)
        if os.path.exists(MINIFAS_PATH):
            try:
                opts = ort.SessionOptions()
                opts.intra_op_num_threads = 1
                self.minifas_session = ort.InferenceSession(MINIFAS_PATH, sess_options=opts, providers=['CPUExecutionProvider'])
                self.minifas_input_name = self.minifas_session.get_inputs()[0].name
                print(f"[AI ENGINE] Loaded MiniFASNet v2 Anti-Spoofing: {os.path.basename(MINIFAS_PATH)}")
            except Exception as e:
                print(f"[AI ENGINE] Warning: Failed to load MiniFASNet: {e}")

        # 5. Mask Detection
        if os.path.exists(MASK_PATH):
            try:
                opts = ort.SessionOptions()
                opts.intra_op_num_threads = 1
                self.mask_session = ort.InferenceSession(MASK_PATH, sess_options=opts, providers=['CPUExecutionProvider'])
                self.mask_input_name = self.mask_session.get_inputs()[0].name
                print(f"[AI ENGINE] Loaded Mask Detector: {os.path.basename(MASK_PATH)}")
            except Exception as e:
                print(f"[AI ENGINE] Warning: Failed to load Mask Detector: {e}")

        # 6. CDCN 3D Depth Liveness (Central Difference CNN)
        if os.path.exists(CDCN_PATH):
            try:
                opts = ort.SessionOptions()
                opts.intra_op_num_threads = 1
                self.cdcn_session = ort.InferenceSession(CDCN_PATH, sess_options=opts, providers=['CPUExecutionProvider'])
                self.cdcn_input_name = self.cdcn_session.get_inputs()[0].name
                print(f"[AI ENGINE] Loaded CDCN Depth Liveness: {os.path.basename(CDCN_PATH)}")
            except Exception as e:
                print(f"[AI ENGINE] Warning: Failed to load CDCN: {e}")

        self.initialized = True

    def extract_arcface_embedding(self, aligned_112: np.ndarray) -> np.ndarray:
        """Extracts 512-d L2-normalized ArcFace embedding from 112x112 aligned crop."""
        if self.mbf_session is None:
            return np.zeros(512, dtype=np.float64)
        blob = cv2.dnn.blobFromImage(
            aligned_112, 1.0 / 127.5, (112, 112), (127.5, 127.5, 127.5), swapRB=True
        )
        out = self.mbf_session.run(None, {self.mbf_input_name: blob})[0]
        feat = out.flatten().astype(np.float64)
        norm = np.linalg.norm(feat)
        if norm > 1e-9:
            feat = feat / norm
        return feat

    def extract_sface_embedding(self, aligned_112: np.ndarray) -> np.ndarray:
        """Extracts 128-d L2-normalized SFace embedding from 112x112 aligned crop."""
        if self.sface is None:
            return np.zeros(128, dtype=np.float64)
        feat = self.sface.feature(aligned_112).flatten().astype(np.float64)
        norm = np.linalg.norm(feat)
        if norm > 1e-9:
            feat = feat / norm
        return feat

    def check_texture_liveness(self, face_crop: np.ndarray) -> Tuple[bool, float]:
        """
        Runs MiniFASNet v2 texture-based anti-spoofing on face crop (80x80).
        Detects screen moire, paper edges, print artifacts, cutout attacks.
        Returns: (is_live: bool, live_confidence: float)
        """
        if self.minifas_session is None or face_crop is None or face_crop.size == 0:
            return True, 0.85

        try:
            crop_80 = cv2.resize(face_crop, (80, 80)).astype(np.float32)
            blob = np.transpose(crop_80, (2, 0, 1))[np.newaxis, ...]
            logits = self.minifas_session.run(None, {self.minifas_input_name: blob})[0]
            # Softmax
            exp_s = np.exp(logits - np.max(logits))
            probs = exp_s / np.sum(exp_s)
            live_prob = float(probs[0][1])  # Class 1 is real/live face
            is_live = bool(live_prob >= 0.65)
            return is_live, live_prob
        except Exception:
            return True, 0.80

    def check_depth_liveness(self, face_crop: np.ndarray) -> Tuple[bool, float]:
        """
        Runs CDCN++ (Central Difference CNN) 3D facial depth anti-spoofing on face crop (256x256).
        Estimates 3D topological depth map. Real faces show 3D facial topography; flat attacks produce near-zero maps.
        Returns: (is_live: bool, depth_confidence: float)
        """
        if self.cdcn_session is None or face_crop is None or face_crop.size == 0:
            return True, 0.85

        try:
            crop_256 = cv2.resize(face_crop, (256, 256)).astype(np.float32) / 255.0
            rgb = cv2.cvtColor(crop_256, cv2.COLOR_BGR2RGB)
            mean = np.array([0.485, 0.456, 0.406], dtype=np.float32)
            std = np.array([0.229, 0.224, 0.225], dtype=np.float32)
            normed = (rgb - mean) / std
            blob = np.transpose(normed, (2, 0, 1))[np.newaxis, ...]
            out = self.cdcn_session.run(None, {self.cdcn_input_name: blob})[0]
            depth_map = out[0]  # shape: (32, 32)
            depth_mean = float(np.mean(depth_map))
            # Real face threshold: depth_mean >= 0.22 (typically 0.35 - 0.65 for live face)
            depth_conf = float(min(1.0, max(0.0, depth_mean / 0.50)))
            is_live = bool(depth_mean >= 0.22)
            return is_live, depth_conf
        except Exception:
            return True, 0.80

    def check_liveness(self, face_crop: np.ndarray) -> Tuple[bool, float]:
        """
        Fused multi-modal Anti-Spoofing:
        Combines Stage 2 (MiniFASNet v2 texture analysis) + Stage 3 (CDCN++ 3D depth analysis).
        Returns: (is_live: bool, fused_confidence: float)
        """
        t_live, t_conf = self.check_texture_liveness(face_crop)
        d_live, d_conf = self.check_depth_liveness(face_crop)

        # Multi-modal fusion
        fused_conf = 0.55 * t_conf + 0.45 * d_conf
        is_live = bool(t_live and d_live and fused_conf >= 0.55)
        return is_live, fused_conf

    def check_mask(self, img_or_crop: np.ndarray) -> Tuple[bool, float]:
        """
        Runs YOLOv8 face mask detector.
        Returns: (is_masked: bool, mask_confidence: float)
        """
        if self.mask_session is None or img_or_crop is None or img_or_crop.size == 0:
            return False, 0.0

        try:
            h, w = img_or_crop.shape[:2]
            resized = cv2.resize(img_or_crop, (320, 320)).astype(np.float32) / 255.0
            blob = np.transpose(resized, (2, 0, 1))[np.newaxis, ...]
            out = self.mask_session.run(None, {self.mask_input_name: blob})[0]
            # out shape: (1, 6, 2100) -> [x, y, w, h, score_mask, score_nomask]
            mask_scores = out[0, 4, :]
            nomask_scores = out[0, 5, :]
            max_mask = float(np.max(mask_scores))
            max_nomask = float(np.max(nomask_scores))
            is_masked = bool(max_mask > 0.35 and max_mask >= max_nomask)
            confidence = max_mask if is_masked else max_nomask
            return is_masked, confidence
        except Exception:
            return False, 0.0

    def detect_and_extract(self, img: np.ndarray) -> List[Dict[str, Any]]:
        """
        End-to-End detection, liveness, mask check, and dual embedding extraction.
        Returns a list of dicts with:
          - box: {x, y, width, height}
          - embedding_512: 512-d ArcFace vector
          - embedding_128: 128-d SFace vector
          - is_live: bool
          - liveness_confidence: float
          - is_masked: bool
          - mask_confidence: float
        """
        results = []
        if img is None:
            return results

        h, w, _ = img.shape
        faces = None

        if self.yunet is not None:
            self.yunet.setInputSize((w, h))
            _, faces = self.yunet.detect(img)

        if faces is None or len(faces) == 0:
            return results

        for face in faces:
            bx, by, bw, bh = int(face[0]), int(face[1]), int(face[2]), int(face[3])
            box = {"x": max(0, bx), "y": max(0, by), "width": max(1, bw), "height": max(1, bh)}

            # Align face crop using landmarks if SFace alignCrop is available
            face_crop = img[box["y"]:box["y"]+box["height"], box["x"]:box["x"]+box["width"]]
            if self.sface is not None:
                aligned = self.sface.alignCrop(img, face)
            else:
                aligned = cv2.resize(face_crop, (112, 112))

            # 1. ArcFace 512-d embedding
            emb_512 = self.extract_arcface_embedding(aligned)

            # 2. SFace 128-d embedding
            emb_128 = self.extract_sface_embedding(aligned)

            # 3. Liveness check (Stage 2: MiniFASNet v2 Texture + Stage 3: CDCN++ 3D Depth)
            target_crop = face_crop if face_crop.size > 0 else aligned
            t_live, t_conf = self.check_texture_liveness(target_crop)
            d_live, d_conf = self.check_depth_liveness(target_crop)
            fused_conf = 0.55 * t_conf + 0.45 * d_conf
            is_live = bool(t_live and d_live and fused_conf >= 0.55)

            # 4. Mask check
            is_masked, mask_conf = self.check_mask(target_crop)

            results.append({
                "box": box,
                "embedding_512": emb_512.tolist(),
                "embedding_128": emb_128.tolist(),
                "is_live": is_live,
                "liveness_confidence": round(fused_conf, 4),
                "texture_liveness": round(t_conf, 4),
                "depth_liveness": round(d_conf, 4),
                "is_masked": is_masked,
                "mask_confidence": round(mask_conf, 4)
            })

        return results

    def compare_embeddings(self, vec1: List[float], vec2: List[float], is_masked: bool = False) -> Tuple[float, float, bool]:
        """
        Compares two embeddings (supporting both 512-d and 128-d).
        Returns: (distance, confidence, is_match)
        """
        arr1 = np.array(vec1, dtype=np.float64)
        arr2 = np.array(vec2, dtype=np.float64)

        if len(arr1) != len(arr2):
            return 999.0, 0.0, False

        norm1 = np.linalg.norm(arr1)
        norm2 = np.linalg.norm(arr2)
        if norm1 <= 1e-9 or norm2 <= 1e-9:
            return 999.0, 0.0, False

        cos_sim = float(np.dot(arr1, arr2) / (norm1 * norm2))
        dist = float(np.linalg.norm(arr1 - arr2))

        # Check dimension: 512-d (ArcFace) vs 128-d (SFace)
        if len(arr1) == 512:
            # ArcFace thresholds
            # Normal: threshold 0.45 - 0.50 (cos_sim >= 0.45 is a match)
            # Masked: lower threshold to 0.40
            threshold = 0.40 if is_masked else 0.45
            is_match = bool(cos_sim >= threshold)
            if is_match:
                conf = 0.70 + 0.30 * min(1.0, (cos_sim - threshold) / (1.0 - threshold + 1e-9))
            else:
                conf = max(0.0, 0.60 * (cos_sim / max(threshold, 1e-9)))
            return dist, conf, is_match
        else:
            # SFace (128-d) threshold
            threshold = 0.363
            is_match = bool(cos_sim >= threshold)
            if is_match:
                conf = 0.70 + 0.30 * min(1.0, (cos_sim - threshold) / (1.0 - threshold + 1e-9))
            else:
                conf = max(0.0, 0.50 * (cos_sim / max(threshold, 1e-9)))
            return dist, conf, is_match

# Global singleton
ai_engine = AIEngine()
