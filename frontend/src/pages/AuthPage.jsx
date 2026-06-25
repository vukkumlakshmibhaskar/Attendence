import { useState } from "react";

export function AuthPage({ setupComplete, onSetup, onLogin, error }) {
  const isSetup = !setupComplete;
  const [form, setForm] = useState({
    organization_name: "",
    admin_name: "",
    email: "",
    password: "",
  });

  const submit = async (event) => {
    event.preventDefault();
    if (isSetup) {
      await onSetup(form);
    } else {
      await onLogin({ email: form.email, password: form.password });
    }
  };

  return (
    <main className="auth-shell">
      <form className="auth-panel" onSubmit={submit}>
        <h1>{isSetup ? "First-time admin setup" : "Sign in"}</h1>
        {isSetup && (
          <>
            <label>Organization</label>
            <input value={form.organization_name} onChange={(event) => setForm({ ...form, organization_name: event.target.value })} />
            <label>Admin name</label>
            <input value={form.admin_name} onChange={(event) => setForm({ ...form, admin_name: event.target.value })} />
          </>
        )}
        <label>Email</label>
        <input type="email" value={form.email} onChange={(event) => setForm({ ...form, email: event.target.value })} />
        <label>Password</label>
        <input type="password" value={form.password} onChange={(event) => setForm({ ...form, password: event.target.value })} />
        {error && <p className="error-line">{error}</p>}
        <button type="submit">{isSetup ? "Create admin account" : "Login"}</button>
      </form>
    </main>
  );
}
