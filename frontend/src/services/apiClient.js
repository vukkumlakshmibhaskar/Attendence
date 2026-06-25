import { API_URL } from "../config/env.js";

const TOKEN_KEY = "attendance_saas_token";

export function getToken() {
  return window.localStorage.getItem(TOKEN_KEY);
}

export function setToken(token) {
  if (token) {
    window.localStorage.setItem(TOKEN_KEY, token);
  } else {
    window.localStorage.removeItem(TOKEN_KEY);
  }
}

export async function apiRequest(path, options = {}) {
  const headers = {
    "Content-Type": "application/json",
    ...(options.headers || {}),
  };
  const token = getToken();
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }

  const response = await fetch(`${API_URL}${path}`, {
    ...options,
    headers,
  });
  const text = await response.text();
  const contentType = response.headers.get("content-type") || "";
  let payload = null;
  if (text && contentType.includes("application/json")) {
    payload = JSON.parse(text);
  }
  if (!response.ok) {
    throw new Error(payload?.error || text || `Request failed with ${response.status}`);
  }
  if (text && !contentType.includes("application/json")) {
    throw new Error(`Expected JSON response but received: ${text.slice(0, 160)}`);
  }
  return payload;
}

export function postJSON(path, body) {
  return apiRequest(path, {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function deleteJSON(path) {
  return apiRequest(path, {
    method: "DELETE",
  });
}
