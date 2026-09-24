const HISTORY_LIMIT = 4;
const MIN_STABLE_SAMPLES = 3;
const FRONT_MAX_YAW = 0.18;
// This landmark ratio is not an angle and has no universal zero.
// Accept a steady frontal reference; subsequent poses use its measured baseline.
const FRONT_MIN_PITCH = -0.60;
const FRONT_MAX_PITCH = 0.35;
const SIDE_MIN_YAW = 0.18;
const SIDE_MAX_PITCH = 0.22;
const SIDE_MAX_ROLL = 0.30;
const VERTICAL_MIN_PITCH = 0.12;
const VERTICAL_MAX_YAW = 0.20;
const VERTICAL_MAX_ROLL = 0.30;

export function evaluateEnrollmentPose(poseKey, metrics, capturedMetrics) {
  if (!metrics || ![metrics.yawScore, metrics.pitchScore, metrics.roll].every(Number.isFinite)) {
    return { valid: false, message: "Keep your face visible to the camera" };
  }
  const front = capturedMetrics.front;
  const yawBase = front?.yawScore ?? 0;
  const pitchBase = front?.pitchScore ?? 0;
  const yawDelta = metrics.yawScore - yawBase;
  const pitchDelta = metrics.pitchScore - pitchBase;
  const roll = Math.abs(metrics.roll ?? 0);

  if (poseKey === "front") {
    if (Math.abs(metrics.yawScore) > FRONT_MAX_YAW) {
      return { valid: false, message: "Center your face and look straight" };
    }
    if (metrics.pitchScore < FRONT_MIN_PITCH || metrics.pitchScore > FRONT_MAX_PITCH) {
      return { valid: false, message: "Keep your chin level for the front pose" };
    }
    if (roll > 0.30) {
      return { valid: false, message: "Keep your head upright and look at the camera" };
    }
    return {
      valid: true,
      message: "Look at the camera and hold still to set your starting position",
    };
  }

  if (!front) {
    return { valid: false, message: "Capture the front pose first" };
  }

  if (poseKey === "left") {
    if (yawDelta < -0.12) {
      return { valid: false, message: "That is the right side. Turn your face to your left" };
    }
    if (yawDelta < SIDE_MIN_YAW) {
      return { valid: false, message: "Turn your face more clearly to your left" };
    }
    if (Math.abs(pitchDelta) > SIDE_MAX_PITCH) {
      return { valid: false, message: "Keep your chin level while turning left" };
    }
    if (roll > SIDE_MAX_ROLL) {
      return { valid: false, message: "Keep your head upright while turning left" };
    }
    return {
      valid: true,
      message: "Left side confirmed. Hold steady",
    };
  }

  if (poseKey === "right") {
    if (yawDelta > 0.12) {
      return { valid: false, message: "That is the left side. Turn your face to your right" };
    }
    if (yawDelta > -SIDE_MIN_YAW) {
      return { valid: false, message: "Turn your face more clearly to your right" };
    }
    if (Math.abs(pitchDelta) > SIDE_MAX_PITCH) {
      return { valid: false, message: "Keep your chin level while turning right" };
    }
    if (roll > SIDE_MAX_ROLL) {
      return { valid: false, message: "Keep your head upright while turning right" };
    }
    return {
      valid: true,
      message: "Right side confirmed. Hold steady",
    };
  }

  if (poseKey === "look_up") {
    if (pitchDelta > 0.1) {
      return { valid: false, message: "That is looking down. Raise your chin for look up" };
    }
    if (pitchDelta > -VERTICAL_MIN_PITCH) {
      return { valid: false, message: "Raise your chin more clearly" };
    }
    if (Math.abs(yawDelta) > VERTICAL_MAX_YAW) {
      return { valid: false, message: "Face forward while looking up" };
    }
    if (roll > VERTICAL_MAX_ROLL) {
      return { valid: false, message: "Keep your head upright while looking up" };
    }
    return {
      valid: true,
      message: "Look up confirmed. Hold steady",
    };
  }

  if (poseKey === "look_down") {
    if (pitchDelta < -0.1) {
      return { valid: false, message: "That is looking up. Lower your chin for look down" };
    }
    if (pitchDelta < VERTICAL_MIN_PITCH) {
      return { valid: false, message: "Lower your chin more clearly" };
    }
    if (Math.abs(yawDelta) > VERTICAL_MAX_YAW) {
      return { valid: false, message: "Face forward while looking down" };
    }
    if (roll > VERTICAL_MAX_ROLL) {
      return { valid: false, message: "Keep your head upright while looking down" };
    }
    return {
      valid: true,
      message: "Look down confirmed. Hold steady",
    };
  }

  return { valid: false, message: "Unknown pose" };
}

export function rememberPoseMetrics(historyRef, poseRef, poseKey, analysis) {
  if (poseRef.current !== poseKey) {
    poseRef.current = poseKey;
    historyRef.current = [];
  }

  const box = analysis.box || {};
  historyRef.current = [
    ...historyRef.current,
    {
      yaw: analysis.metrics.yawScore,
      pitch: analysis.metrics.pitchScore,
      centerX: (box.left || 0) + (box.width || 0) / 2,
      centerY: (box.top || 0) + (box.height || 0) / 2,
      width: box.width || 0,
    },
  ].slice(-HISTORY_LIMIT);
  return historyRef.current;
}

export function isPoseHistoryStable(history) {
  if (history.length < MIN_STABLE_SAMPLES) return false;
  return (
    standardDeviation(history.map((item) => item.yaw)) <= 0.075 &&
    standardDeviation(history.map((item) => item.pitch)) <= 0.085 &&
    standardDeviation(history.map((item) => item.centerX)) <= 0.045 &&
    standardDeviation(history.map((item) => item.centerY)) <= 0.05 &&
    standardDeviation(history.map((item) => item.width)) <= 0.045
  );
}

function standardDeviation(values) {
  const mean = values.reduce((sum, value) => sum + value, 0) / values.length;
  const variance = values.reduce((sum, value) => sum + (value - mean) ** 2, 0) / values.length;
  return Math.sqrt(variance);
}


export function validateEnrollmentDetection(detection, now = Date.now()) {
  if (detection.error) return { valid: false, message: detection.error };
  if (!detection.sampledAt || now - detection.sampledAt > 1800) {
    return { valid: false, message: "Checking your face. Please hold still" };
  }
  if (detection.count > 1) return { valid: false, message: "Keep only one face in the camera" };
  if (detection.count !== 1 || !detection.box) return { valid: false, message: "Bring your face into view" };
  return { valid: true, message: "" };
}

// Frame coordinates come from the detector used by the visible overlay.
// A turned face can be narrower and off-centre without being cut off.
export function evaluateEnrollmentFrame(poseKey, analysis, detection, capturedMetrics, now = Date.now()) {
  const detected = validateEnrollmentDetection(detection, now);
  const state = { ...analysis, valid: false, box: null };
  if (!detected.valid) return { ...state, message: detected.message };

  const { width: frameWidth, height: frameHeight } = detection.frameSize || {};
  const { x, y, width, height } = detection.box;
  if (![frameWidth, frameHeight, x, y, width, height].every(Number.isFinite) ||
      frameWidth <= 0 || frameHeight <= 0 || width <= 0 || height <= 0) {
    return { ...state, message: "Waiting for a clear camera frame" };
  }
  const box = { left: x / frameWidth, top: y / frameHeight, width: width / frameWidth, height: height / frameHeight };
  state.box = box;
  if (width < 40 || height < 64 || box.width < 0.07 || box.height < 0.20) {
    return { ...state, message: "Move a little closer so your face is clearer" };
  }
  if (box.width > 0.80 || box.height > 0.92) {
    return { ...state, message: "Move slightly back so your whole face is visible" };
  }
  const margin = 0.005;
  if (box.left <= margin || box.top <= margin ||
      box.left + box.width >= 1 - margin || box.top + box.height >= 1 - margin) {
    return { ...state, message: "Move slightly toward the centre so your whole face stays in view" };
  }
  if (!analysis?.faceDetected || !analysis?.valid || !analysis?.metrics) {
    return { ...state, message: poseKey === "left" || poseKey === "right"
      ? "Turn slightly back toward the camera until your eyes and nose are visible"
      : "Look toward the camera so your eyes and nose are visible" };
  }
  return { ...state, ...evaluateEnrollmentPose(poseKey, analysis.metrics, capturedMetrics) };
}
