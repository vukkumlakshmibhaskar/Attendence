import { useCallback, useEffect, useState } from "react";

import { authApi } from "../services/authApi.js";
import { setToken } from "../services/apiClient.js";

export function useAuth() {
  const [session, setSession] = useState(null);
  const [setupComplete, setSetupComplete] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const refresh = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const setup = await authApi.setupStatus();
      setSetupComplete(setup.setup_complete);
      if (setup.setup_complete) {
        try {
          const me = await authApi.me();
          setSession(me);
        } catch {
          setSession(null);
        }
      }
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const setup = async (payload) => {
    setError("");
    try {
      const response = await authApi.setup(payload);
      setToken(response.token);
      setSession(response);
      setSetupComplete(true);
    } catch (err) {
      if (err.message.includes("Admin already registered")) {
        setSetupComplete(true);
        setError("Admin is already set up. Please sign in.");
        return;
      }
      setError(err.message);
    }
  };

  const login = async (payload) => {
    setError("");
    try {
      const response = await authApi.login(payload);
      setToken(response.token);
      setSession(response);
    } catch (err) {
      setError(err.message);
    }
  };

  const showLogin = () => {
    setError("");
    setSetupComplete(true);
  };

  const logout = () => {
    setToken("");
    setSession(null);
  };

  return {
    session,
    setupComplete,
    loading,
    error,
    setup,
    login,
    showLogin,
    logout,
    refresh,
  };
}
