from __future__ import annotations

import importlib
import os
import sys
from io import BytesIO
from pathlib import Path

from fastapi.testclient import TestClient
from PIL import Image


ROOT = Path(__file__).resolve().parents[1]
if str(ROOT) not in sys.path:
    sys.path.insert(0, str(ROOT))


def _jpeg_bytes() -> bytes:
    image = Image.new("RGB", (96, 96), (110, 130, 150))
    buffer = BytesIO()
    image.save(buffer, format="JPEG", quality=85)
    return buffer.getvalue()


def test_analyze_endpoint_with_cpu_fallback(monkeypatch) -> None:
    monkeypatch.setenv("COGNITIVE_MODEL_DIR", str(ROOT / "missing-models"))
    monkeypatch.setenv("COGNITIVE_ALLOW_HEURISTIC_FALLBACK", "true")

    module = importlib.import_module("app.main")
    client = TestClient(module.app)

    response = client.post(
        "/analyze",
        files={"file": ("face.jpg", _jpeg_bytes(), "image/jpeg")},
    )

    assert response.status_code == 200
    body = response.json()
    assert body["model_mode"] in {"onnx", "heuristic-fallback"}
    assert body["emotion"]["label"] in {"attentive", "tired", "distracted"}
    assert body["gaze"]["label"] in {"screen", "away"}
    assert isinstance(body["liveness"]["is_live"], bool)
