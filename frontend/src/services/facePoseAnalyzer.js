import { FaceLandmarker, FilesetResolver } from "@mediapipe/tasks-vision";

const VISION_WASM_URL = "https://cdn.jsdelivr.net/npm/@mediapipe/tasks-vision@0.10.35/wasm";
const FACE_LANDMARKER_MODEL_URL =
  "https://storage.googleapis.com/mediapipe-models/face_landmarker/face_landmarker/float16/latest/face_landmarker.task";

let analyzerPromise;
let previousBox = null;

const FACE_OVAL_LANDMARKS = [
  10, 338, 297, 332, 284, 251, 389, 356, 454, 323, 361, 288, 397, 365, 379, 378, 400, 377, 152, 148,
  176, 149, 150, 136, 172, 58, 132, 93, 234, 127, 162, 21, 54, 103, 67, 109,
];

export function getFacePoseAnalyzer() {
  if (!analyzerPromise) {
    analyzerPromise = createAnalyzer().catch((error) => {
      analyzerPromise = undefined;
      throw error;
    });
  }
  return analyzerPromise;
}

async function createAnalyzer() {
  const vision = await FilesetResolver.forVisionTasks(VISION_WASM_URL);
  const landmarker = await FaceLandmarker.createFromOptions(vision, {
    baseOptions: {
      modelAssetPath: FACE_LANDMARKER_MODEL_URL,
      delegate: "CPU",
    },
    runningMode: "VIDEO",
    numFaces: 1,
    minFaceDetectionConfidence: 0.35,
    minFacePresenceConfidence: 0.35,
    minTrackingConfidence: 0.35,
    outputFacialTransformationMatrixes: true,
  });

  return {
    analyze(video) {
      const result = landmarker.detectForVideo(video, performance.now());
      return buildPoseState(result);
    },
    reset() {
      previousBox = null;
    },
  };
}

function buildPoseState(result) {
  const landmarks = result?.faceLandmarks?.[0];
  if (!landmarks || landmarks.length < 363) {
    previousBox = null;
    return {
      faceDetected: false,
      valid: false,
      message: "Center one face inside the camera",
      box: null,
      metrics: null,
    };
  }

  const box = smoothBox(faceTrackingBox(landmarks));
  const metrics = estimatePoseMetrics(landmarks, result?.facialTransformationMatrixes?.[0]);
  // This model supplies pose measurements. Enrollment framing uses the same
  // backend box that is drawn on the video; the estimated landmark box drifts
  // with head rotation and must not reject otherwise visible side poses.
  return {
    faceDetected: true,
    valid: true,
    message: "Face aligned",
    box,
    metrics,
  };
}

function estimatePoseMetrics(landmarks, matrix) {
  const leftEye = averagePoints([landmarks[33], landmarks[133]]);
  const rightEye = averagePoints([landmarks[263], landmarks[362]]);
  const nose = landmarks[1] || landmarks[4];
  const chin = landmarks[152];
  const forehead = landmarks[10];
  const eyeCenter = averagePoints([leftEye, rightEye]);
  const eyeDistance = Math.max(distance(leftEye, rightEye), 0.001);
  const faceHeight = Math.max(distance(forehead, chin), 0.001);
  const landmarkYaw = (nose.x - eyeCenter.x) / eyeDistance;
  const landmarkPitch = ((nose.y - eyeCenter.y) / faceHeight - 0.32) * 2.2;
  const landmarkRoll = Math.atan2(rightEye.y - leftEye.y, rightEye.x - leftEye.x);
  const matrixAngles = matrixToEuler(matrix);

  return {
    yaw: matrixAngles?.yaw ?? landmarkYaw,
    pitch: matrixAngles?.pitch ?? landmarkPitch,
    roll: matrixAngles?.roll ?? landmarkRoll,
    yawScore: landmarkYaw,
    pitchScore: landmarkPitch,
    faceWidth: boundingBox(landmarks).width,
  };
}

function smoothBox(box) {
  if (!previousBox) {
    previousBox = box;
    return box;
  }

  const centerX = box.left + box.width / 2;
  const previousCenterX = previousBox.left + previousBox.width / 2;
  const centerY = box.top + box.height / 2;
  const previousCenterY = previousBox.top + previousBox.height / 2;
  const jump = Math.hypot(centerX - previousCenterX, centerY - previousCenterY);
  const widthChange = Math.abs(box.width - previousBox.width) / Math.max(previousBox.width, 0.001);
  const heightChange = Math.abs(box.height - previousBox.height) / Math.max(previousBox.height, 0.001);
  if (jump > 0.12 || widthChange > 0.28 || heightChange > 0.28) {
    previousBox = box;
    return box;
  }

  const alpha = 0.62;
  previousBox = {
    left: previousBox.left * (1 - alpha) + box.left * alpha,
    top: previousBox.top * (1 - alpha) + box.top * alpha,
    width: previousBox.width * (1 - alpha) + box.width * alpha,
    height: previousBox.height * (1 - alpha) + box.height * alpha,
  };
  return previousBox;
}

function faceTrackingBox(landmarks) {
  return landmarkAnchoredFaceBox(landmarks);
}

function boundingBox(landmarks) {
  let minX = 1;
  let minY = 1;
  let maxX = 0;
  let maxY = 0;
  for (const point of landmarks) {
    minX = Math.min(minX, point.x);
    minY = Math.min(minY, point.y);
    maxX = Math.max(maxX, point.x);
    maxY = Math.max(maxY, point.y);
  }
  return {
    left: clamp01(minX),
    top: clamp01(minY),
    width: clamp01(maxX - minX),
    height: clamp01(maxY - minY),
  };
}

function robustBoundingBox(landmarks) {
  if (landmarks.length < 8) {
    return boundingBox(landmarks);
  }

  const xs = landmarks.map((point) => point.x).sort((a, b) => a - b);
  const ys = landmarks.map((point) => point.y).sort((a, b) => a - b);
  const trim = Math.max(1, Math.floor(landmarks.length * 0.08));
  const minIndex = trim;
  const maxIndex = Math.max(minIndex, landmarks.length - trim - 1);
  const minX = xs[minIndex];
  const maxX = xs[maxIndex];
  const minY = ys[minIndex];
  const maxY = ys[maxIndex];

  return {
    left: clamp01(minX),
    top: clamp01(minY),
    width: clamp01(maxX - minX),
    height: clamp01(maxY - minY),
  };
}

function landmarkAnchoredFaceBox(landmarks) {
  const leftEye = averagePoints([landmarks[33], landmarks[133]].filter(Boolean));
  const rightEye = averagePoints([landmarks[263], landmarks[362]].filter(Boolean));
  const nose = landmarks[1] || landmarks[4];
  const forehead = landmarks[10];
  const chin = landmarks[152];
  const mouthLeft = landmarks[61];
  const mouthRight = landmarks[291];

  if (!leftEye || !rightEye || !nose || !forehead || !chin) {
    const ovalPoints = FACE_OVAL_LANDMARKS.map((index) => landmarks[index]).filter(Boolean);
    const box = robustBoundingBox(ovalPoints.length ? ovalPoints : landmarks);
    return expandBox(box, 0.08, 0.16, 0.08, 0.08);
  }

  const eyeCenter = averagePoints([leftEye, rightEye]);
  const mouthCenter = mouthLeft && mouthRight ? averagePoints([mouthLeft, mouthRight]) : nose;
  const eyeDistance = Math.max(distance(leftEye, rightEye), 0.001);
  const faceHeight = Math.max(distance(forehead, chin), eyeDistance * 2.2);
  const centerX = weightedAverage(
    [
      [eyeCenter.x, 0.35],
      [nose.x, 0.4],
      [mouthCenter.x, 0.25],
    ],
  );
  const centerY = weightedAverage(
    [
      [forehead.y, 0.18],
      [eyeCenter.y, 0.2],
      [nose.y, 0.24],
      [mouthCenter.y, 0.16],
      [chin.y, 0.22],
    ],
  );
  const width = Math.max(eyeDistance * 2.55, faceHeight * 0.58);
  const height = Math.max(faceHeight * 1.12, width * 1.18);

  return {
    left: clamp01(centerX - width / 2),
    top: clamp01(centerY - height / 2),
    width: clamp01(width),
    height: clamp01(height),
  };
}

function expandBox(box, leftRatio, topRatio, rightRatio, bottomRatio) {
  const left = box.left - box.width * leftRatio;
  const top = box.top - box.height * topRatio;
  const right = box.left + box.width * (1 + rightRatio);
  const bottom = box.top + box.height * (1 + bottomRatio);
  const clampedLeft = clamp01(left);
  const clampedTop = clamp01(top);
  const clampedRight = clamp01(right);
  const clampedBottom = clamp01(bottom);
  return {
    left: clampedLeft,
    top: clampedTop,
    width: Math.max(0.001, clampedRight - clampedLeft),
    height: Math.max(0.001, clampedBottom - clampedTop),
  };
}

function matrixToEuler(matrix) {
  const data = matrix?.data || matrix;
  if (!data || data.length < 16) return null;

  const r00 = data[0];
  const r10 = data[4];
  const r11 = data[5];
  const r12 = data[6];
  const r20 = data[8];
  const r21 = data[9];
  const r22 = data[10];
  const sy = Math.sqrt(r00 * r00 + r10 * r10);
  const singular = sy < 1e-6;

  if (singular) {
    return {
      pitch: Math.atan2(-r12, r11),
      yaw: Math.atan2(-r20, sy),
      roll: 0,
    };
  }

  return {
    pitch: Math.atan2(r21, r22),
    yaw: Math.atan2(-r20, sy),
    roll: Math.atan2(r10, r00),
  };
}

function averagePoints(points) {
  if (!points.length) return null;
  const total = points.reduce(
    (sum, point) => ({ x: sum.x + point.x, y: sum.y + point.y, z: sum.z + (point.z || 0) }),
    { x: 0, y: 0, z: 0 },
  );
  return {
    x: total.x / points.length,
    y: total.y / points.length,
    z: total.z / points.length,
  };
}

function weightedAverage(values) {
  const totalWeight = values.reduce((sum, [, weight]) => sum + weight, 0);
  return values.reduce((sum, [value, weight]) => sum + value * weight, 0) / totalWeight;
}

function distance(a, b) {
  return Math.hypot(a.x - b.x, a.y - b.y);
}

function clamp01(value) {
  return Math.min(1, Math.max(0, value));
}
