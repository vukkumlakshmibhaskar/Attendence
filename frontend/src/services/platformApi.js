import { apiRequest, deleteJSON, postJSON } from "./apiClient.js";

export const platformApi = {
  dashboard: () => apiRequest("/api/dashboard"),

  teachers: () => apiRequest("/api/teachers"),
  createTeacher: (payload) => postJSON("/api/teachers", payload),
  deleteTeacher: (id) => deleteJSON(`/api/teachers/${id}`),

  students: () => apiRequest("/api/students"),
  createStudent: (payload) => postJSON("/api/students", payload),
  deleteStudent: (id) => deleteJSON(`/api/students/${id}`),

  classes: () => apiRequest("/api/academics/classes"),
  createClass: (payload) => postJSON("/api/academics/classes", payload),
  subjects: () => apiRequest("/api/academics/subjects"),
  createSubject: (payload) => postJSON("/api/academics/subjects", payload),
  assignments: () => apiRequest("/api/academics/assignments"),
  createAssignment: (payload) => postJSON("/api/academics/assignments", payload),

  sessions: () => apiRequest("/api/attendance/sessions"),
  createSession: (payload) => postJSON("/api/attendance/sessions", payload),
  events: () => apiRequest("/api/attendance/events"),
  processFrame: (payload) => postJSON("/api/frames", payload),
  identifyFrame: (payload) => postJSON("/api/recognition/identify", payload),
  simulateRecognition: (payload) => postJSON("/api/attendance/simulate", payload),
  correctEvent: (id, payload) => postJSON(`/api/attendance/events/${id}/corrections`, payload),

  cameras: () => apiRequest("/api/cameras"),
  createCamera: (payload) => postJSON("/api/cameras", payload),

  detectEnrollmentFrame: (payload) => postJSON("/api/enrollments/detect", payload),
  enrollStudent: (payload) => postJSON("/api/enrollments/students", payload),
  deleteStudentEnrollment: (id) => deleteJSON(`/api/enrollments/students/${id}`),
  enrollTeacher: (payload) => postJSON("/api/enrollments/teachers", payload),
  deleteTeacherEnrollment: (id) => deleteJSON(`/api/enrollments/teachers/${id}`),
  reloadFaceCache: () => postJSON("/api/face-cache/reload", {}),
};
