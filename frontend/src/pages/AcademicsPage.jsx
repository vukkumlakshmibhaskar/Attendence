import { useState } from "react";

import { Modal } from "../components/Modal.jsx";
import { SimpleTable } from "../components/SimpleTable.jsx";

export function AcademicsPage({ classes, subjects, assignments, teachers, canManage, onCreateClass, onCreateSubject, onCreateAssignment }) {
  const [modal, setModal] = useState("");
  const [classForm, setClassForm] = useState({ name: "", section: "", grade_level: "" });
  const [subjectForm, setSubjectForm] = useState({ name: "", code: "" });
  const [assignmentForm, setAssignmentForm] = useState({ teacher_id: "", class_id: "", subject_id: "" });

  return (
    <div className="page">
      <header className="page-heading">
        <h1>Academics</h1>
        {canManage && (
          <div className="actions">
            <button type="button" onClick={() => setModal("class")}>Create class</button>
            <button type="button" onClick={() => setModal("subject")}>Create subject</button>
            <button type="button" onClick={() => setModal("assignment")}>Assign teacher</button>
          </div>
        )}
      </header>
      <section className="split-grid">
        <div className="panel">
          <h2>Classes</h2>
          <SimpleTable columns={[{ key: "name", label: "Class" }, { key: "section", label: "Section" }, { key: "grade_level", label: "Grade" }]} rows={classes} />
        </div>
        <div className="panel">
          <h2>Subjects</h2>
          <SimpleTable columns={[{ key: "name", label: "Subject" }, { key: "code", label: "Code" }]} rows={subjects} />
        </div>
      </section>
      <section className="panel">
        <h2>Teacher-class-subject assignments</h2>
        <SimpleTable
          columns={[{ key: "teacher_id", label: "Teacher" }, { key: "class_id", label: "Class" }, { key: "subject_id", label: "Subject" }]}
          rows={assignments}
        />
      </section>
      {modal === "class" && (
        <Modal title="Create class" onClose={() => setModal("")}>
          <div className="form-grid">
            <input placeholder="Class" value={classForm.name} onChange={(event) => setClassForm({ ...classForm, name: event.target.value })} />
            <input placeholder="Section" value={classForm.section} onChange={(event) => setClassForm({ ...classForm, section: event.target.value })} />
            <input placeholder="Grade level" value={classForm.grade_level} onChange={(event) => setClassForm({ ...classForm, grade_level: event.target.value })} />
            <button type="button" onClick={async () => { await onCreateClass(classForm); setModal(""); }}>Save class</button>
          </div>
        </Modal>
      )}
      {modal === "subject" && (
        <Modal title="Create subject" onClose={() => setModal("")}>
          <div className="form-grid">
            <input placeholder="Subject" value={subjectForm.name} onChange={(event) => setSubjectForm({ ...subjectForm, name: event.target.value })} />
            <input placeholder="Code" value={subjectForm.code} onChange={(event) => setSubjectForm({ ...subjectForm, code: event.target.value })} />
            <button type="button" onClick={async () => { await onCreateSubject(subjectForm); setModal(""); }}>Save subject</button>
          </div>
        </Modal>
      )}
      {modal === "assignment" && (
        <Modal title="Assign teacher" onClose={() => setModal("")}>
          <div className="form-grid">
            <select value={assignmentForm.teacher_id} onChange={(event) => setAssignmentForm({ ...assignmentForm, teacher_id: event.target.value })}>
              <option value="">Teacher</option>
              {teachers.map((item) => <option value={item.id} key={item.id}>{item.name}</option>)}
            </select>
            <select value={assignmentForm.class_id} onChange={(event) => setAssignmentForm({ ...assignmentForm, class_id: event.target.value })}>
              <option value="">Class</option>
              {classes.map((item) => <option value={item.id} key={item.id}>{item.name} {item.section}</option>)}
            </select>
            <select value={assignmentForm.subject_id} onChange={(event) => setAssignmentForm({ ...assignmentForm, subject_id: event.target.value })}>
              <option value="">Subject</option>
              {subjects.map((item) => <option value={item.id} key={item.id}>{item.name}</option>)}
            </select>
            <button type="button" onClick={async () => { await onCreateAssignment(assignmentForm); setModal(""); }}>Save assignment</button>
          </div>
        </Modal>
      )}
    </div>
  );
}
