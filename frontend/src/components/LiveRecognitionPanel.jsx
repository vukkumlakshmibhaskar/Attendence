import { Camera, Pause, Play } from "lucide-react";
import { useCallback } from "react";

import { RECOGNITION_INTERVAL_MS } from "../config/env.js";
import { useFrameSampler } from "../hooks/useFrameSampler.js";
import { mapNormalizedBoxToElementStyle } from "../utils/faceCrop.js";

export function LiveRecognitionPanel({ onIdentifyFrame }) {
  const onFrame = useCallback((payload) => onIdentifyFrame({ ...payload, session_id: "" }), [onIdentifyFrame]);
  const sampler = useFrameSampler({
    onFrame,
    intervalMs: RECOGNITION_INTERVAL_MS,
    clearOnStop: true,
    clearOnError: true,
  });
  const latest = sampler.latest;
  const results = Array.isArray(latest?.results) ? latest.results : [];
  const currentRecognized = Number(latest?.faces_recognized || 0) > 0;
  const matchedFace = currentRecognized ? results.find((result) => result.student_id || result.teacher_id) : null;
  const visibleBox = sampler.latestFaceBox || matchedFace?.box || results[0]?.box || null;
  const liveBoxStyle = sampler.latestFaceBox
    ? mapNormalizedBoxToElementStyle(sampler.latestFaceBox, sampler.videoRef.current)
    : visibleBox
    ? mapFrameBoxToElementStyle(visibleBox, sampler.latestFrameSize, sampler.videoRef.current)
    : undefined;
  const currentMessage = statusMessage(latest, sampler.samplesSent);

  return (
    <section className="panel live-panel">
      <div className="section-heading">
        <div>
          <h2>Live face recognition</h2>
          <p>Open camera to identify an enrolled face.</p>
        </div>
      </div>

      <div className="live-grid">
        <div className="live-camera">
          <video ref={sampler.videoRef} muted playsInline />
          <canvas ref={sampler.canvasRef} />
          {!sampler.isRunning && (
            <div className="video-placeholder">
              <Camera size={28} />
              <span>Open camera and show an enrolled face</span>
            </div>
          )}
          {liveBoxStyle && (
            <div
              className={matchedFace ? "face-box live-box valid" : "face-box live-box unknown"}
              style={liveBoxStyle}
            />
          )}
          <div className="camera-instruction">
            <strong>{sampler.status}</strong>
            <span>{currentMessage}</span>
          </div>
        </div>

        <div className="live-results">
          <div className="actions">
            <button type="button" onClick={sampler.start} disabled={sampler.isRunning}>
              <Play size={16} /> Start
            </button>
            <button type="button" className="ghost bordered" onClick={sampler.stop} disabled={!sampler.isRunning}>
              <Pause size={16} /> Stop
            </button>
          </div>
          {sampler.error && <div className="error-line">{sampler.error}</div>}
          <div className="metric-list">
            <span>Faces detected <strong>{latest?.faces_detected ?? 0}</strong></span>
            <span>Recognized <strong>{latest?.faces_recognized ?? 0}</strong></span>
            <span>Unknown <strong>{latest?.unknown_count ?? 0}</strong></span>
            <span>Cache size <strong>{latest?.cache_size ?? 0}</strong></span>
          </div>
          <section className={matchedFace ? "match-card active" : "match-card"}>
            <h3>Matched face details</h3>
            {matchedFace ? (
              <div className="match-fields">
                <span>Name <strong>{matchedFace.full_name || "Unknown"}</strong></span>
                <span>Type <strong>{matchedFace.person_type || "student"}</strong></span>
                <span>ID <strong>{matchedFace.external_student_id || matchedFace.email || matchedFace.teacher_id || "Not set"}</strong></span>
                <span>Face ID <strong>{matchedFace.matched_face_id || matchedFace.student_id || matchedFace.teacher_id}</strong></span>
                <span>Face source <strong>{matchedFace.matched_face_source || "recognition cache"}</strong></span>
                <span>Confidence <strong>{Math.round((matchedFace.recognition_confidence || 0) * 100)}%</strong></span>
                <span>Status <strong>{matchedFace.attendance_status}</strong></span>
              </div>
            ) : (
              <p className="muted">{currentMessage}</p>
            )}
          </section>
          <div className="result-list">
            {results.map((result, index) => (
              <article className="result-item" key={`${result.student_id || "unknown"}-${index}`}>
                <strong>{result.full_name || "Unknown face"}</strong>
                <span>{result.attendance_status}</span>
                {result.person_type && <small>Type: {result.person_type}</small>}
                {result.matched_face_id && <small>Face ID: {result.matched_face_id}</small>}
                {result.cognitive && (
                  <small>
                    {result.cognitive.emotion?.label || "unknown"} / {result.cognitive.gaze?.label || "unknown"} / {result.cognitive.liveness?.is_live ? "live" : "not live"}
                  </small>
                )}
                {result.cognitive_error && <small>{result.cognitive_error}</small>}
              </article>
            ))}
            {results.length === 0 && <p className="muted">No processed faces yet.</p>}
          </div>
        </div>
      </div>
    </section>
  );
}

function statusMessage(latest, samplesSent) {
  if (!latest) {
    return samplesSent > 0 ? "No current match" : "Temporary recognition only";
  }
  if ((latest.faces_detected || 0) === 0) {
    return "No face detected in the current frame";
  }
  if ((latest.faces_recognized || 0) === 0) {
    return "Unknown face. No enrolled match found";
  }
  return "Current frame matched an enrolled face";
}

function mapFrameBoxToElementStyle(box, frameSize, video) {
  const element = video?.parentElement;
  const rect = element?.getBoundingClientRect();
  const frameWidth = frameSize?.width || video?.videoWidth || 0;
  const frameHeight = frameSize?.height || video?.videoHeight || 0;
  if (!rect?.width || !rect?.height || !frameWidth || !frameHeight) {
    return {
      left: `${(box.x / Math.max(frameWidth, 1)) * 100}%`,
      top: `${(box.y / Math.max(frameHeight, 1)) * 100}%`,
      width: `${(box.width / Math.max(frameWidth, 1)) * 100}%`,
      height: `${(box.height / Math.max(frameHeight, 1)) * 100}%`,
    };
  }

  const sourceAspect = frameWidth / frameHeight;
  const elementAspect = rect.width / rect.height;
  let renderedWidth = rect.width;
  let renderedHeight = rect.height;
  let offsetX = 0;
  let offsetY = 0;

  if (sourceAspect > elementAspect) {
    renderedHeight = rect.width / sourceAspect;
    offsetY = (rect.height - renderedHeight) / 2;
  } else {
    renderedWidth = rect.height * sourceAspect;
    offsetX = (rect.width - renderedWidth) / 2;
  }

  const left = clamp01((offsetX + (box.x / frameWidth) * renderedWidth) / rect.width);
  const top = clamp01((offsetY + (box.y / frameHeight) * renderedHeight) / rect.height);
  const right = clamp01((offsetX + ((box.x + box.width) / frameWidth) * renderedWidth) / rect.width);
  const bottom = clamp01((offsetY + ((box.y + box.height) / frameHeight) * renderedHeight) / rect.height);

  return {
    left: `${left * 100}%`,
    top: `${top * 100}%`,
    width: `${Math.max(0.02, right - left) * 100}%`,
    height: `${Math.max(0.02, bottom - top) * 100}%`,
  };
}

function clamp01(value) {
  return Math.min(1, Math.max(0, value));
}
