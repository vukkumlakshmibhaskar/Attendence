import { apiRequest, postJSON } from "./apiClient.js";

export const authApi = {
  setupStatus: () => apiRequest("/api/auth/setup-status"),
  setup: (payload) => postJSON("/api/auth/setup", payload),
  login: (payload) => postJSON("/api/auth/login", payload),
  me: () => apiRequest("/api/auth/me"),
};
