import { BookMarked, CalendarDays, GraduationCap, LibraryBig, UserRoundCheck, UsersRound } from "lucide-react";

import { DashboardGraphs } from "../components/DashboardGraphs.jsx";
import { StatCard } from "../components/StatCard.jsx";

export function DashboardPage({ data }) {
  const activeSessions = Array.isArray(data?.active_sessions) ? data.active_sessions : [];

  return (
    <div className="page">
      <header className="page-heading">
        <div>
          <h1>Dashboard</h1>
          <p>Education management overview</p>
        </div>
      </header>
      <section className="stat-grid">
        <StatCard label="Total students" value={data?.total_students} icon={GraduationCap} tone="violet" />
        <StatCard label="Total teachers" value={data?.total_teachers} icon={UsersRound} tone="teal" />
        <StatCard label="Classes" value={data?.classes} icon={LibraryBig} tone="blue" />
        <StatCard label="Subjects" value={data?.subjects} icon={BookMarked} tone="amber" />
        <StatCard label="Today sessions" value={data?.today_sessions} icon={CalendarDays} tone="violet" />
        <StatCard label="Review queue" value={data?.review_queue} icon={UserRoundCheck} tone="teal" />
      </section>
      <DashboardGraphs data={data} />
      <section className="panel">
        <h2>Active session summary</h2>
        {activeSessions.map((session) => (
          <p key={session.id}>{session.title} - {session.status}</p>
        ))}
        {activeSessions.length === 0 && <p className="muted">No active sessions.</p>}
      </section>
    </div>
  );
}
