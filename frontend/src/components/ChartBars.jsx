export function ChartBars({ title, data = [] }) {
  const rows = Array.isArray(data) ? data : [];
  const max = Math.max(1, ...rows.map((item) => Number(item.value || 0)));
  return (
    <section className="chart-panel">
      <h3>{title}</h3>
      <div className="bar-list">
        {rows.length === 0 ? (
          <p>No data</p>
        ) : (
          rows.map((item) => (
            <div className="bar-row" key={item.label}>
              <span>{item.label || "Unassigned"}</span>
              <div>
                <i style={{ width: `${(Number(item.value || 0) / max) * 100}%` }} />
              </div>
              <strong>{item.value}</strong>
            </div>
          ))
        )}
      </div>
    </section>
  );
}
