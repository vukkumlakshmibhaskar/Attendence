export function StatCard({ label, value, icon: Icon, tone = "violet" }) {
  return (
    <div className={`stat-card tone-${tone}`}>
      {Icon && (
        <span className="stat-icon">
          <Icon size={20} />
        </span>
      )}
      <div>
        <span>{label}</span>
        <strong>{value ?? 0}</strong>
      </div>
    </div>
  );
}
