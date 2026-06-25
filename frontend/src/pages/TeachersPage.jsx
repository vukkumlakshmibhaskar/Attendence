import { useState } from "react";

import { EnrollmentModal } from "../components/EnrollmentModal.jsx";
import { Modal } from "../components/Modal.jsx";
import { SimpleTable } from "../components/SimpleTable.jsx";

export function TeachersPage({ rows, canManage, onCreate, onDelete, onDeleteEnrollment, onRefresh }) {
  const [showAdd, setShowAdd] = useState(false);
  const [enrolling, setEnrolling] = useState(null);
  const [actionError, setActionError] = useState("");
  const [form, setForm] = useState({ name: "", email: "", password: "", status: "active" });

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

  const deleteTeacher = async (row) => {
    setActionError("");
    const confirmed = window.confirm(`Delete teacher ${row.name}? This also removes their account, assignments, sessions, and face enrollment.`);
    if (!confirmed) return;
    try {
      await onDelete(row.id);
    } catch (err) {
      setActionError(err.message);
    }
  };

  const columns = [
    { key: "name", label: "Name" },
    { key: "email", label: "Email" },
    { key: "face_enrollment_status", label: "Face enrollment" },
    { key: "face_pose_count", label: "Pose count" },
    { key: "status", label: "Account status" },
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
          <button type="button" className="danger" onClick={() => deleteTeacher(row)}>
            Delete
          </button>
        </div>
      ) : "",
    },
  ];

  return (
    <div className="page">
      <header className="page-heading">
        <h1>Teachers</h1>
        {canManage && <button type="button" onClick={() => setShowAdd(true)}>Add teacher</button>}
      </header>
      {actionError && <div className="error-line">{actionError}</div>}
      <SimpleTable columns={columns} rows={rows} />
      {showAdd && (
        <Modal title="Add teacher" onClose={() => setShowAdd(false)}>
          <div className="form-grid">
            <input placeholder="Name" value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} />
            <input placeholder="Email" value={form.email} onChange={(event) => setForm({ ...form, email: event.target.value })} />
            <input placeholder="Password" type="password" value={form.password} onChange={(event) => setForm({ ...form, password: event.target.value })} />
            <select value={form.status} onChange={(event) => setForm({ ...form, status: event.target.value })}>
              <option value="active">active</option>
              <option value="inactive">inactive</option>
            </select>
            <button type="button" onClick={async () => { await onCreate(form); setShowAdd(false); }}>Save teacher</button>
          </div>
        </Modal>
      )}
      {enrolling && (
        <EnrollmentModal type="teacher" entity={enrolling} onClose={() => setEnrolling(null)} onDone={onRefresh} />
      )}
    </div>
  );
}
