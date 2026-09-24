import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { evaluateEnrollmentFrame, rememberPoseMetrics, isPoseHistoryStable, validateEnrollmentDetection } from "../utils/enrollmentPose.js";
import { getFacePoseAnalyzer } from "../services/facePoseAnalyzer.js";
import { drawFullFrameToCanvas, mapFrameBoxToElementStyle } from "../utils/faceCrop.js";

const POSE_CHECK_INTERVAL_MS = 160;
const BACKEND_DETECT_INTERVAL_MS = 350;
const STABLE_HOLD_MS = 520;
const RETRY_DELAY_MS = 2000;

const poses = [
  { key: "front", label: "Front", instruction: "Look straight at the camera" },
  { key: "left", label: "Left side", instruction: "Turn gently to your left; keep your face inside the camera" },
  { key: "right", label: "Right side", instruction: "Turn gently to your right; keep your face inside the camera" },
  { key: "look_up", label: "Look up", instruction: "Raise your chin gently" },
  { key: "look_down", label: "Look down", instruction: "Lower your chin from the previous pose" },
];

export function useEnrollmentCamera(onCapture, onDetectFrame) {
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
  const detectInProgressRef = useRef(false);
  const lastDetectAtRef = useRef(0);
  const backendDetectionRef = useRef({ box: null, frameSize: null, count: 0 });
  const runIdRef = useRef(0);
  const startingRef = useRef(false);
  const retryAfterRef = useRef(0);
  const [holdProgress, setHoldProgress] = useState(0);
  const [saving, setSaving] = useState(false);
  const [starting, setStarting] = useState(false);
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
  const [backendFaceBox, setBackendFaceBox] = useState(null);
  const [backendFrameSize, setBackendFrameSize] = useState(null);
  const [backendFaceCount, setBackendFaceCount] = useState(0);

  const complete = useMemo(() => Object.keys(captured).length >= poses.length, [captured]);
  const pose = poses[Math.min(poseIndex, poses.length - 1)];

  const clearPoseInterval = useCallback(() => {
    if (poseIntervalRef.current) {
      window.clearInterval(poseIntervalRef.current);
      poseIntervalRef.current = null;
    }
  }, []);

  const stop = useCallback(() => {
    runIdRef.current += 1;
    clearPoseInterval();
    streamRef.current?.getTracks().forEach((track) => track.stop());
    streamRef.current = null;
    stableSinceRef.current = 0;
    metricHistoryRef.current = [];
    historyPoseRef.current = "";
    backendDetectionRef.current = { box: null, frameSize: null, count: 0 };
    setBackendFaceBox(null);
    setBackendFaceCount(0);
    setHoldProgress(0);
    setRunning(false);
  }, [clearPoseInterval]);

  const detectBackendFace = useCallback(async () => {
    const video = videoRef.current;
    const canvas = canvasRef.current;
    if (!video || !canvas || !onDetectFrame || video.readyState < 2 || detectInProgressRef.current) {
      return;
    }

    const now = Date.now();
    if (now - lastDetectAtRef.current < BACKEND_DETECT_INTERVAL_MS) {
      return;
    }
    lastDetectAtRef.current = now;
    detectInProgressRef.current = true;
    const runId = runIdRef.current;

    try {
      const frameSize = drawFullFrameToCanvas(canvas, video, 640);
      const response = await onDetectFrame({
        session_id: "",
        image_base64: canvas.toDataURL("image/jpeg", 0.82),
        captured_at: new Date().toISOString(),
      });
      if (runId !== runIdRef.current) return;
      const results = Array.isArray(response?.results) ? response.results : [];
      const box = results[0]?.box || null;
      const count = Number(response?.faces_detected || results.length || 0);
      backendDetectionRef.current = { box, frameSize, count, sampledAt: now };
      setBackendFaceBox(box);
      setBackendFrameSize(frameSize);
      setBackendFaceCount(count);
    } catch (err) {
      if (runId !== runIdRef.current) return;
      backendDetectionRef.current = { box: null, frameSize: null, count: 0, error: `Face check failed: ${err.message}` };
      setBackendFaceBox(null);
      setBackendFrameSize(null);
      setBackendFaceCount(0);
    } finally {
      detectInProgressRef.current = false;
    }
  }, [onDetectFrame]);

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

    const detection = validateEnrollmentDetection(backendDetectionRef.current);
    if (!detection.valid) {
      setStatus(detection.message);
      return;
    }
    const runId = runIdRef.current;
    // Freeze pose measurements before the network request; never calibrate from a later frame.
    const captureMetrics = { ...latestAnalysisRef.current.metrics };
    captureInProgressRef.current = true;
    setSaving(true);
    setError("");
    setStatus(`Saving ${currentPose.label}`);
    try {
      if (backendDetectionRef.current.count !== 1 || !backendDetectionRef.current.box) {
        throw new Error("Enrollment needs exactly one backend-detected face");
      }
      drawFullFrameToCanvas(canvas, video, 960);
      const imageBase64 = canvas.toDataURL("image/jpeg", 0.84);
      await onCapture({ pose: currentPose.key, imageBase64 });

      if (runId !== runIdRef.current) return;
      capturedMetricsRef.current[currentPose.key] = captureMetrics;
      setHoldProgress(0);
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
      if (runId !== runIdRef.current) return;
      stableSinceRef.current = 0;
      metricHistoryRef.current = [];
      setHoldProgress(0);
      retryAfterRef.current = Date.now() + RETRY_DELAY_MS;
      setError(err.message);
      setStatus("Could not save this pose. Hold still to retry");
    } finally {
      captureInProgressRef.current = false;
      setSaving(false);
    }
  }, [captured, onCapture, poseIndex, stop]);

  const start = useCallback(async () => {
    if (startingRef.current || streamRef.current || complete) return;
    startingRef.current = true;
    setStarting(true);
    const runId = ++runIdRef.current;
    lastDetectAtRef.current = 0;
    retryAfterRef.current = 0;
    setError("");
    clearPoseInterval();
    stableSinceRef.current = 0;
    metricHistoryRef.current = [];
    historyPoseRef.current = "";
    backendDetectionRef.current = { box: null, frameSize: null, count: 0 };
    setBackendFaceBox(null);
    setBackendFrameSize(null);
    setBackendFaceCount(0);
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
      if (runId !== runIdRef.current) {
        stream.getTracks().forEach((track) => track.stop());
        return;
      }
      streamRef.current = stream;
      if (videoRef.current) {
        videoRef.current.srcObject = stream;
        await videoRef.current.play();
      }
      setStatus("Loading face pose model");
      const analyzer = await getFacePoseAnalyzer();
      if (runId !== runIdRef.current) return;
      poseAnalyzerRef.current = analyzer;
      setRunning(true);
      poseAnalyzerRef.current.reset?.();
      setStatus(`Hold ${pose.label} pose`);
    } catch (err) {
      if (runId !== runIdRef.current) return;
      setError(err.message);
      setStatus("Could not start enrollment. Please try starting the camera again");
      stop();
    } finally {
      startingRef.current = false;
      setStarting(false);
    }
  }, [clearPoseInterval, complete, pose.label, stop]);

  useEffect(() => {
    if (autoStartedRef.current || complete) return;
    // Defer startup so React StrictMode's setup/cleanup check cannot orphan a camera request.
    const timer = window.setTimeout(() => {
      autoStartedRef.current = true;
      start();
    }, 0);
    return () => window.clearTimeout(timer);
  }, [complete, start]);

  useEffect(() => {
    if (!running || complete) return undefined;

    const checkPose = () => {
      const video = videoRef.current;
      const analyzer = poseAnalyzerRef.current;
      if (!video || !analyzer || video.readyState < 2 || captureInProgressRef.current) {
        return;
      }
      if (Date.now() < retryAfterRef.current) return;
      detectBackendFace();

      let analysis;
      try {
        analysis = analyzer.analyze(video);
      } catch (err) {
        stableSinceRef.current = 0;
        setHoldProgress(0);
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
      const poseState = evaluateEnrollmentFrame(
        pose.key, analysis, backendDetectionRef.current, capturedMetricsRef.current,
      );
      latestAnalysisRef.current = poseState;
      setPoseCheck(poseState);

      if (!poseState.valid) {
        stableSinceRef.current = 0;
        setHoldProgress(0);
        metricHistoryRef.current = [];
        setStatus(poseState.message);
        return;
      }

      const history = rememberPoseMetrics(metricHistoryRef, historyPoseRef, pose.key, poseState);
      if (!isPoseHistoryStable(history)) {
        stableSinceRef.current = 0;
        setHoldProgress(0);
        setStatus(`Correct ${pose.label} pose. Hold steady`);
        return;
      }

      const now = Date.now();
      if (!stableSinceRef.current) {
        stableSinceRef.current = now;
      }
      const stableFor = now - stableSinceRef.current;
      setHoldProgress(Math.min(100, Math.round(stableFor / STABLE_HOLD_MS * 100)));
      setStatus(pose.key === "front" ? "Hold still. Setting your starting position" : "Hold still. Capturing automatically");

      if (stableFor >= STABLE_HOLD_MS) {
        capture();
      }
    };

    clearPoseInterval();
    checkPose();
    poseIntervalRef.current = window.setInterval(checkPose, POSE_CHECK_INTERVAL_MS);
    return clearPoseInterval;
  }, [capture, clearPoseInterval, complete, detectBackendFace, pose.key, pose.label, running]);

  useEffect(() => stop, [stop]);

  const faceBoxStyle = backendFaceBox ? mapFrameBoxToElementStyle(backendFaceBox, backendFrameSize, videoRef.current) : undefined;

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
    backendFaceCount,
    holdProgress,
    saving,
    starting,
    faceBoxStyle,
    start,
    stop,
  };
}
