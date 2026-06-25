# CPU Model Strategy

This project should not train one large classroom-only face model. The better pattern, used by production face-analysis demos, is a pipeline:

1. Detect face.
2. Align face with landmarks/head pose.
3. Generate a normalized embedding.
4. Compare the embedding with enrolled gallery embeddings.
5. Reject weak matches with a verifier threshold.
6. Run separate lightweight models for head pose, gaze, emotion, and liveness.

## Reference Model Logic

### OpenVINO Face Recognition

OpenVINO face recognition demos use a multi-model pipeline: face detection, landmarks, and face re-identification against a gallery. The `face-reidentification-retail-0095` model is a lightweight embedding network based on a MobileNetV2-style backbone with inverted residual blocks, squeeze-excitation attention, PReLU activations, global depthwise pooling, and a final 1x1 convolution embedding head.

Use this idea in the project:

- Store one or more embeddings per enrolled pose.
- Use cosine/distance matching against the in-memory gallery.
- Use a trained verifier or strict threshold before accepting a match.
- Keep unknown faces unknown; never keep stale old matches.

### ArcFace / InsightFace Logic

Modern strong face-recognition systems usually train embeddings with an angular-margin loss such as ArcFace. The important idea is not to classify only known people; the model learns an embedding space where same-person faces form tight clusters and different people are separated by an angular margin.

Use this idea in the project:

- Treat attendance recognition as open-set verification, not closed-set classification.
- Enrollment creates a gallery.
- Recognition asks: "Is this face close enough to one enrolled identity?"
- Re-enrollment replaces old gallery embeddings.

Do not use InsightFace pretrained model files in a commercial product unless the model license is cleared. Their code is MIT, but their released pretrained model weights/data are non-commercial research only.

### Head Pose

OpenVINO `head-pose-estimation-adas-0001` is a lightweight CNN that regresses yaw, pitch, and roll in degrees. The project should use this style for strict pose enrollment:

- Front: yaw near 0, pitch near 0.
- Left/right: signed yaw must match the requested direction.
- Look up/down: signed pitch must match the requested direction.
- Hold for several stable samples before snapshot.

### Gaze

OpenVINO `gaze-estimation-adas-0002` uses left eye crop, right eye crop, and head-pose angles as inputs, then outputs a 3D gaze vector. The project should treat gaze as a separate inference stage after face/landmark/head-pose extraction.

### Emotion

OpenVINO `emotions-recognition-retail-0003` is a compact fully convolutional model for five emotions and is validated on a subset of AffectNet. For this project, emotion should remain a lightweight secondary signal, not part of attendance identity matching.

### Liveness

Liveness should be a separate anti-spoof model. It should block or mark review, not decide identity. For CPU deployment, use small ONNX/OpenVINO anti-spoofing models and validate them with local camera conditions.

## What Is Already Implemented

- The Go backend keeps enrolled vectors in memory.
- Live recognition does transient matching only and does not save live recognition data.
- Re-enrollment and enrollment deletion are supported.
- A CPU-trained pair verifier has been added:
  - Model: `go-backend/models/face_verifier_lfw.json`
  - Report: `go-backend/models/face_verifier_lfw_report.json`
  - Trainer: `go-backend/tools/train_face_verifier.py`
- The verifier was trained from the local LFW dataset using same/different pairs and is loaded by the Go matcher.

## Recommended Next Implementation

The current mock/hash descriptor is useful for UI flow but is not production biometric recognition. The next reliable step is:

1. Add a CPU ONNX/OpenVINO embedding model.
2. Generate real embeddings during enrollment.
3. Store embeddings per pose in the existing enrollment tables.
4. Replace mock/hash descriptor matching in live recognition.
5. Re-enroll all teachers/students after the embedding model changes.

## Source References

- OpenVINO Model Zoo: https://docs.openvino.ai/2023.3/model_zoo.html
- OpenVINO face recognition demo: https://docs.openvino.ai/2023.3/omz_demos_face_recognition_demo_python.html
- OpenVINO face re-identification model: https://docs.openvino.ai/2023.3/omz_models_model_face_reidentification_retail_0095.html
- OpenVINO head pose model: https://docs.openvino.ai/2023.3/omz_models_model_head_pose_estimation_adas_0001.html
- OpenVINO gaze model: https://docs.openvino.ai/2023.3/omz_models_model_gaze_estimation_adas_0002.html
- OpenVINO emotion model: https://docs.openvino.ai/2023.3/omz_models_model_emotions_recognition_retail_0003.html
- ArcFace paper: https://openaccess.thecvf.com/content_CVPR_2019/papers/Deng_ArcFace_Additive_Angular_Margin_Loss_for_Deep_Face_Recognition_CVPR_2019_paper.pdf
- InsightFace license note: https://github.com/deepinsight/insightface
