import { BookOpen, CalendarCheck, GraduationCap, LayoutDashboard, ScanFace, UsersRound } from "lucide-react";

const navItems = [
  { key: "dashboard", label: "Dashboard", icon: LayoutDashboard },
  { key: "attendance", label: "Attendance", icon: CalendarCheck },
  { key: "teachers", label: "Teachers", icon: UsersRound },
  { key: "students", label: "Students", icon: GraduationCap },
  { key: "academics", label: "Academics", icon: BookOpen },
  { key: "recognition", label: "Live Recognition", icon: ScanFace },
];

export function Navbar({ active, onChange, session, onLogout }) {
  return (
    <aside className="nav-shell">
      <div className="brand">
        <strong>AttendIQ</strong>
        <small>{session?.organization?.name || "Organization"}</small>
      </div>
      <nav>
        {navItems.map((item) => {
          const Icon = item.icon;
          return (
            <button
              className={active === item.key ? "nav-item active" : "nav-item"}
              key={item.key}
              type="button"
              onClick={() => onChange(item.key)}
            >
              <Icon size={18} />
              {item.label}
            </button>
          );
        })}
      </nav>
      <div className="account-box">
        <strong>{session?.user?.name}</strong>
        <small>{session?.user?.role}</small>
        <button type="button" onClick={onLogout}>Sign out</button>
      </div>
    </aside>
  );
}
