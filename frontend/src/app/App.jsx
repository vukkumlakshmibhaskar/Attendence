import { useCallback, useEffect, useMemo, useState } from "react";

import { Navbar } from "../components/Navbar.jsx";
import { useAuth } from "../hooks/useAuth.js";
import { platformApi } from "../services/platformApi.js";
import { AuthPage } from "../pages/AuthPage.jsx";
import { AcademicsPage } from "../pages/AcademicsPage.jsx";
import { AttendancePage } from "../pages/AttendancePage.jsx";
import { DashboardPage } from "../pages/DashboardPage.jsx";
import { LiveRecognitionPage } from "../pages/LiveRecognitionPage.jsx";
import { StudentsPage } from "../pages/StudentsPage.jsx";
import { TeachersPage } from "../pages/TeachersPage.jsx";

export default function App() {
  const auth = useAuth();
  const [active, setActive] = useState("dashboard");
  const [error, setError] = useState("");
  const [data, setData] = useState(emptyData);

  const canManage = auth.session?.user?.role === "admin";

  const load = useCallback(async () => {
    if (!auth.session) return;
    setError("");
    try {
      const [
        dashboard,
        teachers,
        students,
        classes,
        subjects,
        assignments,
        sessions,
        events,
        cameras,
      ] = await Promise.all([
        platformApi.dashboard(),
        platformApi.teachers(),
        platformApi.students(),
        platformApi.classes(),
        platformApi.subjects(),
        platformApi.assignments(),
        platformApi.sessions(),
        platformApi.events(),
        platformApi.cameras(),
      ]);
      setData({
        dashboard: normalizeDashboard(dashboard),
        teachers: ensureArray(teachers),
        students: ensureArray(students),
        classes: ensureArray(classes),
        subjects: ensureArray(subjects),
        assignments: ensureArray(assignments),
        sessions: ensureArray(sessions),
        events: ensureArray(events),
        cameras: ensureArray(cameras),
      });
    } catch (err) {
      setError(err.message);
    }
  }, [auth.session]);

  useEffect(() => {
    load();
  }, [load]);

  const actions = useMemo(
    () => ({
      createTeacher: async (payload) => {
        await platformApi.createTeacher(payload);
        await load();
      },
      deleteTeacher: async (id) => {
        await platformApi.deleteTeacher(id);
        await platformApi.reloadFaceCache();
        await load();
      },
      deleteTeacherEnrollment: async (id) => {
        await platformApi.deleteTeacherEnrollment(id);
        await platformApi.reloadFaceCache();
        await load();
      },
      createStudent: async (payload) => {
        await platformApi.createStudent(payload);
        await load();
      },
      deleteStudent: async (id) => {
        await platformApi.deleteStudent(id);
        await platformApi.reloadFaceCache();
        await load();
      },
      deleteStudentEnrollment: async (id) => {
        await platformApi.deleteStudentEnrollment(id);
        await platformApi.reloadFaceCache();
        await load();
      },
      createClass: async (payload) => {
        await platformApi.createClass(payload);
        await load();
      },
      createSubject: async (payload) => {
        await platformApi.createSubject(payload);
        await load();
      },
      createAssignment: async (payload) => {
        await platformApi.createAssignment(payload);
        await load();
      },
      createSession: async (payload) => {
        await platformApi.createSession(payload);
        await load();
      },
      createCamera: async (payload) => {
        await platformApi.createCamera(payload);
        await load();
      },
      simulate: async (payload) => {
        await platformApi.simulateRecognition(payload);
        await load();
      },
      processFrame: async (payload) => platformApi.processFrame(payload),
      identifyFrame: async (payload) => platformApi.identifyFrame(payload),
      correct: async (id, payload) => {
        await platformApi.correctEvent(id, payload);
        await load();
      },
    }),
    [load],
  );

  if (auth.loading) {
    return <main className="loading">Loading</main>;
  }

  if (!auth.session) {
    return (
      <AuthPage
        setupComplete={auth.setupComplete}
        onSetup={auth.setup}
        onLogin={auth.login}
        onShowLogin={auth.showLogin}
        error={auth.error}
      />
    );
  }

  return (
    <div className="workspace">
      <Navbar active={active} onChange={setActive} session={auth.session} onLogout={auth.logout} />
      <main className="workspace-main">
        {error && <div className="error-line">{error}</div>}
        {active === "dashboard" && <DashboardPage data={data.dashboard} />}
        {active === "teachers" && (
          <TeachersPage
            rows={data.teachers}
            canManage={canManage}
            onCreate={actions.createTeacher}
            onDelete={actions.deleteTeacher}
            onDeleteEnrollment={actions.deleteTeacherEnrollment}
            onRefresh={load}
          />
        )}
        {active === "students" && (
          <StudentsPage
            rows={data.students}
            classes={data.classes}
            canManage={canManage}
            onCreate={actions.createStudent}
            onDelete={actions.deleteStudent}
            onDeleteEnrollment={actions.deleteStudentEnrollment}
            onRefresh={load}
          />
        )}
        {active === "academics" && (
          <AcademicsPage
            classes={data.classes}
            subjects={data.subjects}
            assignments={data.assignments}
            teachers={data.teachers}
            canManage={canManage}
            onCreateClass={actions.createClass}
            onCreateSubject={actions.createSubject}
            onCreateAssignment={actions.createAssignment}
          />
        )}
        {active === "attendance" && (
          <AttendancePage
            sessions={data.sessions}
            events={data.events}
            cameras={data.cameras}
            classes={data.classes}
            subjects={data.subjects}
            teachers={data.teachers}
            students={data.students}
            canManage={canManage}
            onCreateSession={actions.createSession}
            onCreateCamera={actions.createCamera}
            onSimulate={actions.simulate}
            onCorrect={actions.correct}
          />
        )}
        {active === "recognition" && (
          <LiveRecognitionPage onIdentifyFrame={actions.identifyFrame} />
        )}
      </main>
    </div>
  );
}

const emptyData = {
  dashboard: null,
  teachers: [],
  students: [],
  classes: [],
  subjects: [],
  assignments: [],
  sessions: [],
  events: [],
  cameras: [],
};

function ensureArray(value) {
  return Array.isArray(value) ? value : [];
}

function normalizeDashboard(value) {
  if (!value) return null;
  return {
    ...value,
    weekly_classroom_operations: ensureArray(value.weekly_classroom_operations),
    attendance_mix: ensureArray(value.attendance_mix),
    students_by_class: ensureArray(value.students_by_class),
    subject_coverage: ensureArray(value.subject_coverage),
    active_sessions: ensureArray(value.active_sessions),
    teacher_workload: ensureArray(value.teacher_workload),
  };
}


