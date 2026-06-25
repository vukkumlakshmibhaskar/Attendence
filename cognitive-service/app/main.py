from __future__ import annotations

from typing import Any

from fastapi import FastAPI, File, HTTPException, UploadFile
from pydantic import BaseModel, Field

from app.config import load_settings
from app.model_runner import CognitiveAnalyzer


class LabelResult(BaseModel):
    label: str
    confidence: float = Field(ge=0.0, le=1.0)
    probabilities: list[float] = []


class LivenessResult(BaseModel):
    is_live: bool
    confidence: float = Field(ge=0.0, le=1.0)
    probabilities: list[float] = []


class AnalysisResponse(BaseModel):
    emotion: LabelResult
    gaze: LabelResult
    liveness: LivenessResult
    processing_ms: int
    service_version: str
    model_mode: str
    features: dict[str, float] | None = None


settings = load_settings()
analyzer = CognitiveAnalyzer(settings)

app = FastAPI(
    title="Classroom Cognitive Analysis Service",
    version=settings.service_version,
    description="CPU-only FastAPI service backed by ONNX Runtime.",
)


@app.get("/health")
def health() -> dict[str, Any]:
    return analyzer.health()


@app.post("/analyze", response_model=AnalysisResponse)
async def analyze(file: UploadFile = File(...)) -> dict[str, Any]:
    if file.content_type and not file.content_type.startswith("image/"):
        raise HTTPException(status_code=415, detail="Only image uploads are supported")

    payload = await file.read()
    if not payload:
        raise HTTPException(status_code=400, detail="Empty image upload")
    if len(payload) > settings.max_upload_bytes:
        raise HTTPException(status_code=413, detail="Image upload is too large")

    try:
        return analyzer.analyze(payload)
    except Exception as exc:
        raise HTTPException(status_code=422, detail=f"Could not analyze image: {exc}") from exc
