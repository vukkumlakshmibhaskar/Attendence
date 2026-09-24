import { useCallback, useEffect, useRef, useState } from "react";

import { SAMPLE_INTERVAL_MS } from "../config/env.js";
import { drawFullFrameToCanvas } from "../utils/faceCrop.js";

export function useFrameSampler({
  sessionId,
  onFrame,
  requireSession = false,
  intervalMs = SAMPLE_INTERVAL_MS,
  clearOnStop = false,
  clearOnError = false,
}) {
  const videoRef = useRef(null);
  const canvasRef = useRef(null);
  const streamRef = useRef(null);
  const inFlightRef = useRef(false);
  const cacheSizeRef = useRef(0);
  const [isRunning, setIsRunning] = useState(false);
  const [status, setStatus] = useState("Camera stopped");
  const [error, setError] = useState("");
  const [latest, setLatest] = useState(null);
  const [latestFrameSize, setLatestFrameSize] = useState(null);
  const [latestFaceBox, setLatestFaceBox] = useState(null);
  const [samplesSent, setSamplesSent] = useState(0);

  const stop = useCallback(() => {
    streamRef.current?.getTracks().forEach((track) => track.stop());
    streamRef.current = null;
    setIsRunning(false);
    setStatus("Camera stopped");
    if (clearOnStop) {
      setLatest(null);
      setLatestFrameSize(null);
      setLatestFaceBox(null);
      setSamplesSent(0);
      setError("");
      cacheSizeRef.current = 0;
    }
  }, [clearOnStop]);

  const start = useCallback(async () => {
    setError("");
    setSamplesSent(0);
    setLatest(null);
    setLatestFrameSize(null);
    setLatestFaceBox(null);
    cacheSizeRef.current = 0;
    if (requireSession && !sessionId) {
      setError("Select an attendance session first.");
      return;
    }
    try {
      const stream = await navigator.mediaDevices.getUserMedia({
        audio: false,
        video: { facingMode: "user", width: { ideal: 960 }, height: { ideal: 540 } },
      });
      streamRef.current = stream;
      if (videoRef.current) {
        videoRef.current.srcObject = stream;
        await videoRef.current.play();
      }
      setIsRunning(true);
      setStatus(`Sampling one frame every ${Math.round(intervalMs / 100) / 10} seconds`);
    } catch (err) {
      setError(err.message);
      setStatus("Camera permission failed");
    }
  }, [intervalMs, requireSession, sessionId]);

  const capture = useCallback(async () => {
    const video = videoRef.current;
    const canvas = canvasRef.current;
    if (!video || !canvas || (requireSession && !sessionId) || inFlightRef.current || video.readyState < 2) {
      return;
    }

    let frameSize;
    try {
      frameSize = drawFullFrameToCanvas(canvas, video, 640);
      setLatestFaceBox(null);
    } catch (err) {
      setStatus("Camera frame is not ready");
      return;
    }
    setLatestFrameSize(frameSize);
    const imageBase64 = canvas.toDataURL("image/jpeg", 0.82);

    inFlightRef.current = true;
    setStatus("Processing frame");
    try {
      const response = await onFrame({
        session_id: sessionId || "",
        image_base64: imageBase64,
        captured_at: new Date().toISOString(),
      });
      cacheSizeRef.current = response?.cache_size || 0;
      setLatest(response);
      setSamplesSent((count) => count + 1);
      setStatus("Waiting for next sample");
    } catch (err) {
      setError(err.message);
      if (clearOnError) {
        setLatest(null);
        setLatestFrameSize(null);
        setLatestFaceBox(null);
      }
      setStatus("Frame skipped");
    } finally {
      inFlightRef.current = false;
    }
  }, [clearOnError, onFrame, requireSession, sessionId]);

  useEffect(() => {
    if (!isRunning) return undefined;
    const firstSample = window.setTimeout(capture, Math.min(500, intervalMs));
    const interval = window.setInterval(capture, intervalMs);
    return () => {
      window.clearTimeout(firstSample);
      window.clearInterval(interval);
    };
  }, [capture, intervalMs, isRunning]);

  useEffect(() => stop, [stop]);

  return {
    videoRef,
    canvasRef,
    isRunning,
    status,
    error,
    latest,
    latestFrameSize,
    latestFaceBox,
    samplesSent,
    start,
    stop,
  };
}
