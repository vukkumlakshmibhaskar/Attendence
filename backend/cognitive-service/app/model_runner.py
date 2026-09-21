from __future__ import annotations

import time
from pathlib import Path
from typing import Any

import numpy as np
import onnxruntime as ort
from PIL import Image

from app.config import Settings
from app.image_utils import image_features, load_rgb_image, to_onnx_uint8_tensor


def _softmax(values: np.ndarray) -> np.ndarray:
    values = values.astype(np.float32)
    values = values - values.max(axis=-1, keepdims=True)
    exp = np.exp(values)
    return exp / exp.sum(axis=-1, keepdims=True)


class OnnxClassifier:
    def __init__(self, model_path: Path, labels: list[str]) -> None:
        self.model_path = model_path
        self.labels = labels
        self.session: ort.InferenceSession | None = None
        self.input_name: str | None = None

        if model_path.exists():
            options = ort.SessionOptions()
            options.intra_op_num_threads = 1
            options.inter_op_num_threads = 1
            options.graph_optimization_level = ort.GraphOptimizationLevel.ORT_ENABLE_ALL
            self.session = ort.InferenceSession(
                str(model_path),
                sess_options=options,
                providers=["CPUExecutionProvider"],
            )
            self.input_name = self.session.get_inputs()[0].name

    @property
    def loaded(self) -> bool:
        return self.session is not None and self.input_name is not None

    def predict(self, tensor: np.ndarray) -> tuple[str, float, list[float]]:
        if self.session is None or self.input_name is None:
            raise RuntimeError(f"ONNX model not loaded: {self.model_path}")

        output = self.session.run(None, {self.input_name: tensor})[0]
        output = np.asarray(output, dtype=np.float32)
        if output.ndim == 1:
            output = output[None, :]

        row = output[0]
        if np.any(row < 0) or not np.isclose(float(row.sum()), 1.0, atol=1e-3):
            row = _softmax(row[None, :])[0]

        idx = int(np.argmax(row))
        return self.labels[idx], float(row[idx]), [float(x) for x in row.tolist()]


class CognitiveAnalyzer:
    def __init__(self, settings: Settings) -> None:
        self.settings = settings
        self.emotion = OnnxClassifier(
            settings.emotion_model_path, ["attentive", "tired", "distracted"]
        )
        self.gaze = OnnxClassifier(settings.gaze_model_path, ["screen", "away"])
        self.liveness = OnnxClassifier(
            settings.liveness_model_path, ["spoof", "live"]
        )

        self.using_onnx = self.emotion.loaded and self.gaze.loaded and self.liveness.loaded
        if not self.using_onnx and not settings.allow_heuristic_fallback:
            missing = [
                str(model.model_path)
                for model in (self.emotion, self.gaze, self.liveness)
                if not model.loaded
            ]
            raise RuntimeError(f"Missing ONNX model(s): {', '.join(missing)}")

    def health(self) -> dict[str, Any]:
        return {
            "status": "ok",
            "runtime": "onnxruntime",
            "providers": ort.get_available_providers(),
            "active_provider": "CPUExecutionProvider",
            "model_mode": "onnx" if self.using_onnx else "heuristic-fallback",
            "service_version": self.settings.service_version,
            "models": {
                "emotion": str(self.settings.emotion_model_path),
                "gaze": str(self.settings.gaze_model_path),
                "liveness": str(self.settings.liveness_model_path),
            },
        }

    def analyze(self, image_bytes: bytes) -> dict[str, Any]:
        started = time.perf_counter()
        image = load_rgb_image(image_bytes)

        if self.using_onnx:
            result = self._analyze_onnx(image)
            result["model_mode"] = "onnx"
        else:
            result = self._analyze_heuristic(image)
            result["model_mode"] = "heuristic-fallback"

        result["processing_ms"] = int((time.perf_counter() - started) * 1000)
        result["service_version"] = self.settings.service_version
        return result

    def _analyze_onnx(self, image: Image.Image) -> dict[str, Any]:
        tensor = to_onnx_uint8_tensor(image)

        emotion_label, emotion_conf, emotion_probs = self.emotion.predict(tensor)
        gaze_label, gaze_conf, gaze_probs = self.gaze.predict(tensor)
        live_label, live_conf, liveness_probs = self.liveness.predict(tensor)

        return {
            "emotion": {
                "label": emotion_label,
                "confidence": emotion_conf,
                "probabilities": emotion_probs,
            },
            "gaze": {
                "label": gaze_label,
                "confidence": gaze_conf,
                "probabilities": gaze_probs,
            },
            "liveness": {
                "is_live": live_label == "live",
                "confidence": live_conf,
                "probabilities": liveness_probs,
            },
        }

    def _analyze_heuristic(self, image: Image.Image) -> dict[str, Any]:
        features = image_features(image)
        brightness = features["brightness"]
        contrast = features["contrast"]
        sharpness = features["sharpness"]
        center_bias = features["center_bias"]

        if brightness < 0.24 or sharpness < 0.012:
            emotion_label = "tired"
            emotion_conf = 0.62
        elif abs(center_bias) > 0.08:
            emotion_label = "distracted"
            emotion_conf = 0.58
        else:
            emotion_label = "attentive"
            emotion_conf = 0.66

        gaze_label = "screen" if abs(center_bias) <= 0.09 else "away"
        gaze_conf = 0.64 if gaze_label == "screen" else 0.59

        is_live = 0.03 <= sharpness <= 1.0 and 0.08 <= brightness <= 0.95 and contrast > 0.02
        liveness_conf = 0.70 if is_live else 0.74

        return {
            "emotion": {
                "label": emotion_label,
                "confidence": emotion_conf,
                "probabilities": [],
            },
            "gaze": {
                "label": gaze_label,
                "confidence": gaze_conf,
                "probabilities": [],
            },
            "liveness": {
                "is_live": is_live,
                "confidence": liveness_conf,
                "probabilities": [],
            },
            "features": features,
        }
