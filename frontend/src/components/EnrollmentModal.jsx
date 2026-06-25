import { useCallback, useEffect, useRef, useState } from "react";

import { useEnrollmentCamera } from "../hooks/useEnrollmentCamera.js";
import { platformApi } from "../services/platformApi.js";
import { Modal } from "./Modal.jsx";

export function EnrollmentModal({ type, entity, onClose, onDone }) {
  const [messages, setMessages] = useState([]);
  const [cacheStatus, setCacheStatus] = useState("");
  const completionHandledRef = useRef(false);
  const onCapture = useCallback(
    async ({ pose, imageBase64 }) => {
      const payload = { entity_id: entity.id, pose, image_base64: imageBase64 };
      const response =
        type === "teacher" ? await platformApi.enrollTeacher(payload) : await platformApi.enrollStudent(payload);
      setMessages((prev) => [
        ...prev,
        `${response.pose}: ${response.model_mode}${response.is_demo ? " (demo)" : ""}, ${response.variant_count || 0} scale variants`,
      ]);
    },
    [entity.id, type],
  );
  const camera = useEnrollmentCamera(onCapture);

  useEffect(() => {
    if (!camera.complete || completionHandledRef.current) return;
    completionHandledRef.current = true;
    async function refreshAfterEnrollment() {
      try {
        setCacheStatus("Updating recognition cache");
        await platformApi.reloadFaceCache();
        await onDone();
        setCacheStatus("Recognition cache updated");
      } catch (err) {
        setCacheStatus(err.message);
      }
    }
    refreshAfterEnrollment();
  }, [camera.complete, onDone]);

  return (
    <Modal title={`Face enrollment: ${entity.name || entity.full_name}`} onClose={onClose}>
      <div className="enrollment-grid">
        <div className="enrollment-camera">
          <video ref={camera.videoRef} muted playsInline />
          <canvas ref={camera.canvasRef} />
          {camera.faceBoxStyle && (
            <div
              className={camera.poseCheck.valid ? "face-box valid" : "face-box"}
              style={camera.faceBoxStyle}
            />
          )}
          {!camera.running && <div className="video-placeholder">Enrollment camera</div>}
          <div className="camera-instruction">
            <strong>{camera.pose.label}</strong>
            <span>{camera.poseCheck.message || camera.pose.instruction}</span>
          </div>
        </div>
        <div className="pose-list">
          {camera.poses.map((pose) => (
            <div className={camera.captured[pose.key] ? "pose done" : "pose"} key={pose.key}>
              <strong>{pose.label}</strong>
              <span>{camera.captured[pose.key] ? "Captured" : pose.instruction}</span>
            </div>
          ))}
          <p>{camera.status}</p>
          {camera.poseCheck.metrics && (
            <small className={camera.poseCheck.valid ? "pose-signal valid" : "pose-signal"}>
              Pose signal: yaw {camera.poseCheck.metrics.yawScore.toFixed(2)}, pitch{" "}
              {camera.poseCheck.metrics.pitchScore.toFixed(2)}
            </small>
          )}
          {camera.error && <div className="error-line">{camera.error}</div>}
          <div className="actions">
            <button type="button" onClick={camera.start} disabled={camera.running || camera.complete}>Start camera</button>
            <button
              type="button"
              onClick={() => {
                onDone();
                onClose();
              }}
              disabled={!camera.complete}
            >
              {camera.complete ? "Close" : "Done"}
            </button>
          </div>
          {cacheStatus && <small>{cacheStatus}</small>}
          {messages.map((message) => (
            <small key={message}>{message}</small>
          ))}
        </div>
      </div>
    </Modal>
  );
}
