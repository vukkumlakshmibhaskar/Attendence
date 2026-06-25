# Human Face Detector Training

This folder trains a lightweight face detector from the local dataset at:

`C:\Users\lakshmibhaskar.v\Downloads\archive (1)`

The dataset is a one-class YOLO detection dataset:

- `images/train`
- `images/val`
- `labels/train`
- `labels/val`
- class `0 = human_face`

This improves face localization for enrollment and live recognition. It does not train identity recognition, drowsiness, gaze, liveness, emotion, or ANPR plate recognition because those tasks require separate labeled datasets.

## Quick CPU Smoke Training

```powershell
py -3.12 -m pip install --user ultralytics --no-deps
py -3.12 .\models\face_detector\train_face_detector.py --quick
```

The quick run trains one epoch on 5 percent of the data and skips full validation/plots so we can verify the pipeline without locking the CPU for a long time.

## Full CPU Training

```powershell
py -3.12 .\models\face_detector\train_face_detector.py --epochs 40 --imgsz 320 --batch 8
```

## Export Existing Checkpoint

```powershell
py -3.12 .\models\face_detector\train_face_detector.py --export-only --weights .\models\face_detector\runs\human_face_detector\weights\best.pt
```

Output files:

- `models/face_detector/human_face_detector.onnx`
- `models/face_detector/human_face_detector.optimized.onnx`
- `models/face_detector/human_face_detector.int8.onnx` only when ONNX Runtime quantization succeeds
- training checkpoints under `models/face_detector/runs/human_face_detector/weights`

Use `human_face_detector.optimized.onnx` for CPU runtime in this local environment. The current ONNX Runtime build rejects INT8 quantization for this YOLO graph, so any failed INT8 attempt is archived as `human_face_detector.int8.unsupported.onnx` and should not be deployed.

Keep the `.pt` checkpoints only for future retraining/export.
