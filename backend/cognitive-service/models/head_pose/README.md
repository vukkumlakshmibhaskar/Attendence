# Head Pose Training

This trains a small CPU-friendly head-pose estimator from the KARA 2018 annotations.

The current local folder is:

`C:\Users\lakshmibhaskar.v\Downloads\kara2018_headposeannotations\kara2018_headposeannotations`

That archive contains annotation CSV files only. It does not currently contain the matching `person*.jpg` image files, so a real visual head-pose model cannot be trained from this folder alone.

## Validate Dataset

```powershell
py -3.12 .\models\head_pose\train_head_pose.py --validate-only
```

The script writes:

- `models/head_pose/kara_dataset_report.json`

## Train When Images Are Available

Put the matching KARA image files beside the CSVs, under an `images` folder, or pass the image directory explicitly:

```powershell
py -3.12 .\models\head_pose\train_head_pose.py --images-dir "C:\path\to\kara\images" --epochs 30 --batch 16
```

Fast smoke run:

```powershell
py -3.12 .\models\head_pose\train_head_pose.py --images-dir "C:\path\to\kara\images" --quick
```

Output files:

- `models/head_pose/head_pose_estimator.pt`
- `models/head_pose/head_pose_estimator.onnx`
- `models/head_pose/head_pose_estimator.optimized.onnx`

The ONNX model output is `[tilt_degrees, pan_degrees]`.
