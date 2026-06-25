import { useState } from "react";

import { EnrollmentModal } from "../components/EnrollmentModal.jsx";
import { Modal } from "../components/Modal.jsx";
import { SimpleTable } from "../components/SimpleTable.jsx";

export function StudentsPage({ rows, classes, canManage, onCreate, onDelete, onDeleteEnrollment, onRefresh }) {
  const [showAdd, setShowAdd] = useState(false);
  const [enrolling, setEnrolling] = useState(null);
  const [actionError, setActionError] = useState("");
  const [form, setForm] = useState({ full_name: "", student_id: "", class_id: "", consent_status: "pending", status: "active" });

  const startEnrollment = async (row) => {
    setActionError("");
    if (row.face_pose_count > 0) {
      const confirmed = window.confirm(`Re-enroll ${row.name}? Existing face enrollment data will be deleted first.`);
      if (!confirmed) return;
      try {
        await onDeleteEnrollment(row.id);
      } catch (err) {
        setActionError(err.message);
        return;
      }
    }
    setEnrolling(row);
  };

  const deleteEnrollment = async (row) => {
    setActionError("");
    const confirmed = window.confirm(`Delete face enrollment for ${row.name}?`);
    if (!confirmed) return;
    try {
      await onDeleteEnrollment(row.id);
    } catch (err) {
      setActionError(err.message);
    }
  };

  const deleteStudent = async (row) => {
    setActionError("");
    const confirmed = window.confirm(`Delete student ${row.name}? This also removes their face enrollment and related attendance records.`);
    if (!confirmed) return;
    try {
      await onDelete(row.id);
    } catch (err) {
      setActionError(err.message);
    }
  };

  const columns = [
    { key: "name", label: "Name" },
    { key: "student_id", label: "Student ID" },
    { key: "class_name", label: "Class" },
    { key: "section", label: "Section" },
    { key: "consent_status", label: "Consent" },
    { key: "face_pose_count", label: "Pose count" },
    {
      key: "actions",
      label: "",
      render: (row) => canManage ? (
        <div className="row-actions">
          <button type="button" onClick={() => startEnrollment(row)}>
            {row.face_pose_count > 0 ? "Re-enroll" : "Enroll face"}
          </button>
          <button type="button" className="ghost bordered" onClick={() => deleteEnrollment(row)} disabled={row.face_pose_count === 0}>
            Delete enrollment
          </button>
          <button type="button" className="danger" onClick={() => deleteStudent(row)}>
            Delete
          </button>
        </div>
      ) : "",
    },
  ];

  return (
    <div className="page">
      <header className="page-heading">
        <h1>Students</h1>
        {canManage && <button type="button" onClick={() => setShowAdd(true)}>Add student</button>}
      </header>
      {actionError && <div className="error-line">{actionError}</div>}
      <SimpleTable columns={columns} rows={rows} />
      {showAdd && (
        <Modal title="Add student" onClose={() => setShowAdd(false)}>
          <div className="form-grid">
            <input placeholder="Name" value={form.full_name} onChange={(event) => setForm({ ...form, full_name: event.target.value })} />
            <input placeholder="Student ID" value={form.student_id} onChange={(event) => setForm({ ...form, student_id: event.target.value })} />
            <select value={form.class_id} onChange={(event) => setForm({ ...form, class_id: event.target.value })}>
              <option value="">Select class</option>
              {classes.map((item) => <option value={item.id} key={item.id}>{item.name} {item.section}</option>)}
            </select>
            <select value={form.consent_status} onChange={(event) => setForm({ ...form, consent_status: event.target.value })}>
              <option value="granted">granted</option>
              <option value="pending">pending</option>
              <option value="revoked">revoked</option>
            </select>
            <button type="button" onClick={async () => { await onCreate(form); setShowAdd(false); }}>Save student</button>
          </div>
        </Modal>
      )}
      {enrolling && (
        <EnrollmentModal type="student" entity={enrolling} onClose={() => setEnrolling(null)} onDone={onRefresh} />
      )}
    </div>
  );
}
