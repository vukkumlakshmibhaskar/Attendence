import { useState } from "react";

import { Modal } from "../components/Modal.jsx";
import { SimpleTable } from "../components/SimpleTable.jsx";

export function AttendancePage({
  sessions,
  events,
  cameras,
  classes,
  subjects,
  teachers,
  students,
  canManage,
  onCreateSession,
  onCreateCamera,
  onSimulate,
  onCorrect,
}) {
  const [modal, setModal] = useState("");
  const [sessionForm, setSessionForm] = useState({ title: "", class_id: "", subject_id: "", teacher_id: "", start_time: "", end_time: "", late_threshold_minutes: 10 });
  const [cameraForm, setCameraForm] = useState({ name: "", room: "", rtsp_url: "", class_id: "", health_status: "unknown" });
  const [simulation, setSimulation] = useState({ session_id: "", student_id: "", status: "present", confidence: 0.88 });
  const [correction, setCorrection] = useState(null);

  return (
    <div className="page">
      <header className="page-heading">
        <h1>Attendance</h1>
        {canManage && (
          <div className="actions">
            <button type="button" onClick={() => setModal("session")}>Create session</button>
            <button type="button" onClick={() => setModal("simulate")}>Recognition test</button>
            <button type="button" onClick={() => setModal("camera")}>Add camera</button>
          </div>
        )}
      </header>
      <section className="panel">
        <h2>Attendance sessions</h2>
        <SimpleTable columns={[{ key: "title", label: "Title" }, { key: "status", label: "Status" }, { key: "start_time", label: "Start" }, { key: "end_time", label: "End" }]} rows={sessions} />
      </section>
      <section className="panel">
        <h2>Attendance events</h2>
        <SimpleTable
          columns={[
            { key: "student_id", label: "Student" },
            { key: "status", label: "Status" },
            { key: "confidence", label: "Confidence" },
            { key: "source", label: "Source" },
            { key: "actions", label: "", render: (row) => canManage ? <button type="button" onClick={() => setCorrection(row)}>Correct</button> : "" },
          ]}
          rows={events}
        />
      </section>
      <section className="panel">
        <h2>Cameras</h2>
        <SimpleTable columns={[{ key: "name", label: "Name" }, { key: "room", label: "Room" }, { key: "rtsp_url", label: "RTSP URL" }, { key: "health_status", label: "Health" }]} rows={cameras} />
      </section>
      {modal === "session" && (
        <Modal title="Create attendance session" onClose={() => setModal("")}>
          <div className="form-grid">
            <input placeholder="Title" value={sessionForm.title} onChange={(event) => setSessionForm({ ...sessionForm, title: event.target.value })} />
            <select value={sessionForm.class_id} onChange={(event) => setSessionForm({ ...sessionForm, class_id: event.target.value })}>
              <option value="">Class</option>
              {classes.map((item) => <option value={item.id} key={item.id}>{item.name} {item.section}</option>)}
            </select>
            <select value={sessionForm.subject_id} onChange={(event) => setSessionForm({ ...sessionForm, subject_id: event.target.value })}>
              <option value="">Subject</option>
              {subjects.map((item) => <option value={item.id} key={item.id}>{item.name}</option>)}
            </select>
            <select value={sessionForm.teacher_id} onChange={(event) => setSessionForm({ ...sessionForm, teacher_id: event.target.value })}>
              <option value="">Teacher</option>
              {teachers.map((item) => <option value={item.id} key={item.id}>{item.name}</option>)}
            </select>
            <input type="datetime-local" value={sessionForm.start_time} onChange={(event) => setSessionForm({ ...sessionForm, start_time: event.target.value })} />
            <input type="datetime-local" value={sessionForm.end_time} onChange={(event) => setSessionForm({ ...sessionForm, end_time: event.target.value })} />
            <input type="number" value={sessionForm.late_threshold_minutes} onChange={(event) => setSessionForm({ ...sessionForm, late_threshold_minutes: Number(event.target.value) })} />
            <button type="button" onClick={async () => { await onCreateSession(sessionForm); setModal(""); }}>Save session</button>
          </div>
        </Modal>
      )}
      {modal === "simulate" && (
        <Modal title="Recognition test / simulation" onClose={() => setModal("")}>
          <div className="form-grid">
            <select value={simulation.session_id} onChange={(event) => setSimulation({ ...simulation, session_id: event.target.value })}>
              <option value="">Session</option>
              {sessions.map((item) => <option value={item.id} key={item.id}>{item.title}</option>)}
            </select>
            <select value={simulation.student_id} onChange={(event) => setSimulation({ ...simulation, student_id: event.target.value })}>
              <option value="">Student</option>
              {students.map((item) => <option value={item.id} key={item.id}>{item.name}</option>)}
            </select>
            <select value={simulation.status} onChange={(event) => setSimulation({ ...simulation, status: event.target.value })}>
              <option value="present">Present</option>
              <option value="late">Late</option>
              <option value="review">Review</option>
              <option value="absent">Absent</option>
            </select>
            <button type="button" onClick={async () => { await onSimulate(simulation); setModal(""); }}>Run test</button>
          </div>
        </Modal>
      )}
      {modal === "camera" && (
        <Modal title="Add camera" onClose={() => setModal("")}>
          <div className="form-grid">
            <input placeholder="Name" value={cameraForm.name} onChange={(event) => setCameraForm({ ...cameraForm, name: event.target.value })} />
            <input placeholder="Room" value={cameraForm.room} onChange={(event) => setCameraForm({ ...cameraForm, room: event.target.value })} />
            <input placeholder="RTSP URL" value={cameraForm.rtsp_url} onChange={(event) => setCameraForm({ ...cameraForm, rtsp_url: event.target.value })} />
            <select value={cameraForm.class_id} onChange={(event) => setCameraForm({ ...cameraForm, class_id: event.target.value })}>
              <option value="">Class</option>
              {classes.map((item) => <option value={item.id} key={item.id}>{item.name} {item.section}</option>)}
            </select>
            <button type="button" onClick={async () => { await onCreateCamera(cameraForm); setModal(""); }}>Save camera</button>
          </div>
        </Modal>
      )}
      {correction && (
        <Modal title="Manual correction" onClose={() => setCorrection(null)}>
          <div className="form-grid">
            <select value={correction.corrected_status || correction.status} onChange={(event) => setCorrection({ ...correction, corrected_status: event.target.value })}>
              <option value="present">Present</option>
              <option value="late">Late</option>
              <option value="review">Review</option>
              <option value="absent">Absent</option>
            </select>
            <input placeholder="Reason" value={correction.reason || ""} onChange={(event) => setCorrection({ ...correction, reason: event.target.value })} />
            <button type="button" onClick={async () => { await onCorrect(correction.id, { corrected_status: correction.corrected_status || correction.status, reason: correction.reason || "" }); setCorrection(null); }}>Save correction</button>
          </div>
        </Modal>
      )}
    </div>
  );
}
