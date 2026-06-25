import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { getFacePoseAnalyzer } from "../services/facePoseAnalyzer.js";
import { drawFaceCropToCanvas, mapNormalizedBoxToElementStyle } from "../utils/faceCrop.js";

const POSE_CHECK_INTERVAL_MS = 160;
const STABLE_HOLD_MS = 520;
const HISTORY_LIMIT = 4;
const MIN_STABLE_SAMPLES = 3;

const FRONT_MAX_YAW = 0.1;
const FRONT_MAX_PITCH = 0.12;
const SIDE_MIN_YAW = 0.28;
const SIDE_MAX_PITCH = 0.14;
const SIDE_MAX_ROLL = 0.24;
const VERTICAL_MIN_PITCH = 0.22;
const VERTICAL_MAX_YAW = 0.14;
const VERTICAL_MAX_ROLL = 0.24;

const poses = [
  { key: "front", label: "Front", instruction: "Look straight at the camera" },
  { key: "left", label: "Left side", instruction: "Turn your face to your left" },
  { key: "right", label: "Right side", instruction: "Turn to the opposite side" },
  { key: "look_up", label: "Look up", instruction: "Raise your chin gently" },
  { key: "look_down", label: "Look down", instruction: "Lower your chin from the previous pose" },
];

export function useEnrollmentCamera(onCapture) {
  const videoRef = useRef(null);
  const canvasRef = useRef(null);
  const streamRef = useRef(null);
  const poseIntervalRef = useRef(null);
  const poseAnalyzerRef = useRef(null);
  const stableSinceRef = useRef(0);
  const capturedMetricsRef = useRef({});
  const latestAnalysisRef = useRef(null);
  const metricHistoryRef = useRef([]);
  const historyPoseRef = useRef("");
  const captureInProgressRef = useRef(false);
  const autoStartedRef = useRef(false);
  const [poseIndex, setPoseIndex] = useState(0);
  const [captured, setCaptured] = useState({});
  const [running, setRunning] = useState(false);
  const [status, setStatus] = useState("Opening camera");
  const [error, setError] = useState("");
  const [poseCheck, setPoseCheck] = useState({
    faceDetected: false,
    valid: false,
    message: "Loading face pose model",
    box: null,
    metrics: null,
  });

  const complete = useMemo(() => Object.keys(captured).length >= poses.length, [captured]);
  const pose = poses[Math.min(poseIndex, poses.length - 1)];

  const clearPoseInterval = useCallback(() => {
    if (poseIntervalRef.current) {
      window.clearInterval(poseIntervalRef.current);
      poseIntervalRef.current = null;
    }
  }, []);

  const stop = useCallback(() => {
    clearPoseInterval();
    streamRef.current?.getTracks().forEach((track) => track.stop());
    streamRef.current = null;
    stableSinceRef.current = 0;
    metricHistoryRef.current = [];
    historyPoseRef.current = "";
    setRunning(false);
  }, [clearPoseInterval]);

  const capture = useCallback(async () => {
    const video = videoRef.current;
    const canvas = canvasRef.current;
    const currentPose = poses[poseIndex];
    if (!video || !canvas || !currentPose || captured[currentPose.key] || captureInProgressRef.current) {
      return;
    }
    if (video.readyState < 2 || !video.videoWidth || !video.videoHeight) {
      setStatus("Waiting for camera frame");
      return;
    }

    captureInProgressRef.current = true;
    setError("");
    setStatus(`Saving ${currentPose.label}`);
    try {
      const box = latestAnalysisRef.current?.box;
      if (!box) {
        throw new Error("Face box is not ready");
      }
      drawFaceCropToCanvas(canvas, video, box, { outputSize: 320, expansion: 1.9 });
      const imageBase64 = canvas.toDataURL("image/jpeg", 0.84);
      await onCapture({ pose: currentPose.key, imageBase64 });

      if (latestAnalysisRef.current?.metrics) {
        capturedMetricsRef.current[currentPose.key] = latestAnalysisRef.current.metrics;
      }
      setCaptured((prev) => ({ ...prev, [currentPose.key]: true }));
      stableSinceRef.current = 0;
      metricHistoryRef.current = [];
      historyPoseRef.current = "";
      if (poseIndex >= poses.length - 1) {
        setStatus("Enrollment complete");
        stop();
      } else {
        setPoseIndex((index) => index + 1);
        setStatus(`${currentPose.label} saved`);
      }
    } catch (err) {
      setError(err.message);
      setStatus("Capture failed, retrying automatically");
    } finally {
      captureInProgressRef.current = false;
    }
  }, [captured, onCapture, poseIndex, stop]);

  const start = useCallback(async () => {
    setError("");
    clearPoseInterval();
    stableSinceRef.current = 0;
    metricHistoryRef.current = [];
    historyPoseRef.current = "";
    setPoseCheck({
      faceDetected: false,
      valid: false,
      message: "Loading face pose model",
      box: null,
      metrics: null,
    });
    try {
      const stream = await navigator.mediaDevices.getUserMedia({
        audio: false,
        video: { width: { ideal: 960 }, height: { ideal: 540 }, facingMode: "user" },
      });
      streamRef.current = stream;
      if (videoRef.current) {
        videoRef.current.srcObject = stream;
        await videoRef.current.play();
      }
      setRunning(true);
      setStatus("Loading face pose model");
      poseAnalyzerRef.current = await getFacePoseAnalyzer();
      poseAnalyzerRef.current.reset?.();
      setStatus(`Hold ${pose.label} pose`);
    } catch (err) {
      setError(err.message);
      setStatus("Camera permission failed");
      stop();
    }
  }, [clearPoseInterval, pose.label, stop]);

  useEffect(() => {
    if (autoStartedRef.current || complete) return;
    autoStartedRef.current = true;
    start();
  }, [complete, start]);

  useEffect(() => {
    if (!running || complete) return undefined;

    const checkPose = () => {
      const video = videoRef.current;
      const analyzer = poseAnalyzerRef.current;
      if (!video || !analyzer || video.readyState < 2 || captureInProgressRef.current) {
        return;
      }

      let analysis;
      try {
        analysis = analyzer.analyze(video);
      } catch (err) {
        stableSinceRef.current = 0;
        setPoseCheck({
          faceDetected: false,
          valid: false,
          message: "Face pose model is warming up",
          box: null,
          metrics: null,
        });
        setStatus(err.message || "Face pose model is warming up");
        return;
      }
      if (!analysis.valid) {
        latestAnalysisRef.current = analysis;
        setPoseCheck(analysis);
        stableSinceRef.current = 0;
        metricHistoryRef.current = [];
        setStatus(analysis.message);
        return;
      }

      const poseValidation = evaluateEnrollmentPose(pose.key, analysis.metrics, capturedMetricsRef.current);
      const poseState = {
        ...analysis,
        valid: poseValidation.valid,
        message: poseValidation.message,
      };
      latestAnalysisRef.current = poseState;
      setPoseCheck(poseState);

      if (!poseState.valid) {
        stableSinceRef.current = 0;
        metricHistoryRef.current = [];
        setStatus(poseState.message);
        return;
      }

      const history = rememberPoseMetrics(metricHistoryRef, historyPoseRef, pose.key, analysis);
      if (!isPoseHistoryStable(history)) {
        stableSinceRef.current = 0;
        setStatus(`Correct ${pose.label} pose. Hold steady`);
        return;
      }

      const now = Date.now();
      if (!stableSinceRef.current) {
        stableSinceRef.current = now;
      }
      const stableFor = now - stableSinceRef.current;
      const remaining = Math.max(0, STABLE_HOLD_MS - stableFor);
      setStatus(`Correct ${pose.label} pose. Hold ${Math.ceil(remaining / 1000)}s`);

      if (stableFor >= STABLE_HOLD_MS) {
        capture();
      }
    };

    clearPoseInterval();
    checkPose();
    poseIntervalRef.current = window.setInterval(checkPose, POSE_CHECK_INTERVAL_MS);
    return clearPoseInterval;
  }, [capture, clearPoseInterval, complete, pose.key, pose.label, running]);

  useEffect(() => stop, [stop]);

  const faceBoxStyle = poseCheck.box ? mapNormalizedBoxToElementStyle(poseCheck.box, videoRef.current) : undefined;

  return {
    poses,
    pose,
    poseIndex,
    captured,
    complete,
    videoRef,
    canvasRef,
    running,
    status,
    error,
    poseCheck,
    faceBoxStyle,
    start,
    stop,
  };
}

function evaluateEnrollmentPose(poseKey, metrics, capturedMetrics) {
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
    if (Math.abs(metrics.pitchScore) > FRONT_MAX_PITCH) {
      return { valid: false, message: "Keep your chin level for the front pose" };
    }
    return {
      valid: true,
      message: "Look straight at the camera",
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
    if (Math.abs(yawDelta) < SIDE_MIN_YAW) {
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
    if (Math.abs(pitchDelta) < VERTICAL_MIN_PITCH) {
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

function rememberPoseMetrics(historyRef, poseRef, poseKey, analysis) {
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

function isPoseHistoryStable(history) {
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

function clamp01(value) {
  return Math.min(1, Math.max(0, value));
}
