from __future__ import annotations

from io import BytesIO

import numpy as np
from PIL import Image, ImageOps

try:
    import cv2
except Exception:  # pragma: no cover - fallback exists for minimal installs
    cv2 = None


def load_rgb_image(image_bytes: bytes) -> Image.Image:
    image = Image.open(BytesIO(image_bytes))
    image = ImageOps.exif_transpose(image)
    return image.convert("RGB")


def to_onnx_uint8_tensor(image: Image.Image, size: int = 64) -> np.ndarray:
    resized = image.resize((size, size), Image.Resampling.BILINEAR)
    array = np.asarray(resized, dtype=np.uint8)
    return np.transpose(array, (2, 0, 1))[None, ...]


def image_features(image: Image.Image) -> dict[str, float]:
    gray = np.asarray(image.convert("L").resize((96, 96)), dtype=np.float32)
    brightness = float(gray.mean() / 255.0)
    contrast = float(gray.std() / 255.0)

    if cv2 is not None:
        sharpness = float(cv2.Laplacian(gray, cv2.CV_32F).var() / 10000.0)
    else:
        gx = np.diff(gray, axis=1)
        gy = np.diff(gray, axis=0)
        sharpness = float((gx.var() + gy.var()) / 10000.0)

    left = gray[:, :32].mean()
    center = gray[:, 32:64].mean()
    right = gray[:, 64:].mean()
    center_bias = float((center - ((left + right) / 2.0)) / 255.0)

    return {
        "brightness": max(0.0, min(1.0, brightness)),
        "contrast": max(0.0, min(1.0, contrast)),
        "sharpness": max(0.0, min(1.0, sharpness)),
        "center_bias": max(-1.0, min(1.0, center_bias)),
    }
