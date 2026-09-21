from __future__ import annotations

import os
from dataclasses import dataclass
from pathlib import Path


def _as_bool(value: str | None, default: bool) -> bool:
    if value is None:
        return default
    return value.strip().lower() in {"1", "true", "yes", "y", "on"}


@dataclass(frozen=True)
class Settings:
    model_dir: Path
    emotion_model_path: Path
    gaze_model_path: Path
    liveness_model_path: Path
    allow_heuristic_fallback: bool
    max_upload_bytes: int
    service_version: str


def load_settings() -> Settings:
    model_dir = Path(os.getenv("COGNITIVE_MODEL_DIR", "models"))
    trained_emotion = model_dir / "emotion_eye_state.onnx"
    default_emotion = trained_emotion if trained_emotion.exists() else model_dir / "emotion_int8_dummy.onnx"
    trained_gaze = model_dir / "gaze_mpiigaze.onnx"
    default_gaze = trained_gaze if trained_gaze.exists() else model_dir / "gaze_int8_dummy.onnx"
    return Settings(
        model_dir=model_dir,
        emotion_model_path=Path(
            os.getenv("EMOTION_MODEL_PATH", str(default_emotion))
        ),
        gaze_model_path=Path(
            os.getenv("GAZE_MODEL_PATH", str(default_gaze))
        ),
        liveness_model_path=Path(
            os.getenv("LIVENESS_MODEL_PATH", str(model_dir / "liveness_int8_dummy.onnx"))
        ),
        allow_heuristic_fallback=_as_bool(
            os.getenv("COGNITIVE_ALLOW_HEURISTIC_FALLBACK"), True
        ),
        max_upload_bytes=int(os.getenv("COGNITIVE_MAX_UPLOAD_BYTES", "1048576")),
        service_version=os.getenv("COGNITIVE_SERVICE_VERSION", "dev-cpu-onnx-1"),
    )
