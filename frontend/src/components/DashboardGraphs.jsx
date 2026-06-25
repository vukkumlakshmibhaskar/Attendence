const palette = ["#6C63FF", "#38B2AC", "#FFB86C", "#5B8DEF", "#A16EFF", "#6B7280"];

export function DashboardGraphs({ data }) {
  return (
    <section className="dashboard-graphs">
      <TrendAreaChart title="Weekly classroom operations" data={data?.weekly_classroom_operations} />
      <DonutChart title="Attendance mix" data={data?.attendance_mix} />
      <HorizontalBars title="Students by class" data={data?.students_by_class} />
      <CoverageMeters title="Subject coverage" data={data?.subject_coverage} />
      <ColumnChart title="Teacher workload" data={data?.teacher_workload} />
    </section>
  );
}

function TrendAreaChart({ title, data = [] }) {
  const rows = safeRows(data);
  const width = 420;
  const height = 220;
  const padX = 30;
  const padY = 24;
  const max = Math.max(1, ...rows.map((item) => item.value));
  const points = rows.map((item, index) => {
    const x = rows.length <= 1 ? width / 2 : padX + (index * (width - padX * 2)) / (rows.length - 1);
    const y = height - padY - (item.value / max) * (height - padY * 2);
    return { ...item, x, y };
  });
  const line = points.map((point, index) => `${index === 0 ? "M" : "L"} ${point.x} ${point.y}`).join(" ");
  const area =
    points.length > 0
      ? `${line} L ${points[points.length - 1].x} ${height - padY} L ${points[0].x} ${height - padY} Z`
      : "";

  return (
    <article className="chart-panel graph-card graph-card-wide">
      <GraphHeader title={title} value={rows.reduce((sum, item) => sum + item.value, 0)} label="total actions" />
      {rows.length === 0 ? (
        <EmptyGraph />
      ) : (
        <svg className="trend-chart" viewBox={`0 0 ${width} ${height}`} role="img" aria-label={title}>
          <path className="trend-area" d={area} />
          <path className="trend-line" d={line} />
          {points.map((point) => (
            <g key={point.label}>
              <circle cx={point.x} cy={point.y} r="5" />
              <text x={point.x} y={height - 4} textAnchor="middle">{shortLabel(point.label)}</text>
            </g>
          ))}
        </svg>
      )}
    </article>
  );
}

function DonutChart({ title, data = [] }) {
  const rows = safeRows(data);
  const total = rows.reduce((sum, item) => sum + item.value, 0);
  const gradient = buildConicGradient(rows, total);

  return (
    <article className="chart-panel graph-card">
      <GraphHeader title={title} value={total} label="events" />
      {rows.length === 0 ? (
        <EmptyGraph />
      ) : (
        <div className="donut-layout">
          <div className="donut-chart" style={{ background: gradient }}>
            <span>{total}</span>
          </div>
          <div className="legend-list">
            {rows.map((item, index) => (
              <span key={item.label}>
                <i style={{ background: palette[index % palette.length] }} />
                {item.label || "Unassigned"}
                <strong>{Math.round((item.value / Math.max(total, 1)) * 100)}%</strong>
              </span>
            ))}
          </div>
        </div>
      )}
    </article>
  );
}

function HorizontalBars({ title, data = [] }) {
  const rows = safeRows(data);
  const max = Math.max(1, ...rows.map((item) => item.value));
  return (
    <article className="chart-panel graph-card">
      <GraphHeader title={title} value={rows.length} label="classes" />
      <div className="soft-bar-list">
        {rows.length === 0 ? (
          <EmptyGraph />
        ) : (
          rows.map((item, index) => (
            <div className="soft-bar-row" key={item.label}>
              <span>{item.label || "Unassigned"}</span>
              <div>
                <i
                  style={{
                    width: `${(item.value / max) * 100}%`,
                    background: palette[index % palette.length],
                  }}
                />
              </div>
              <strong>{item.value}</strong>
            </div>
          ))
        )}
      </div>
    </article>
  );
}

function CoverageMeters({ title, data = [] }) {
  const rows = safeRows(data);
  const max = Math.max(1, ...rows.map((item) => item.value));
  return (
    <article className="chart-panel graph-card">
      <GraphHeader title={title} value={rows.reduce((sum, item) => sum + item.value, 0)} label="assignments" />
      <div className="coverage-list">
        {rows.length === 0 ? (
          <EmptyGraph />
        ) : (
          rows.map((item, index) => (
            <div className="coverage-item" key={item.label}>
              <div>
                <strong>{item.label || "Unassigned"}</strong>
                <span>{item.value} mapped</span>
              </div>
              <meter min="0" max={max} value={item.value} style={{ "--meter-color": palette[index % palette.length] }} />
            </div>
          ))
        )}
      </div>
    </article>
  );
}

function ColumnChart({ title, data = [] }) {
  const rows = safeRows(data);
  const max = Math.max(1, ...rows.map((item) => item.value));
  return (
    <article className="chart-panel graph-card graph-card-wide">
      <GraphHeader title={title} value={rows.reduce((sum, item) => sum + item.value, 0)} label="loads" />
      {rows.length === 0 ? (
        <EmptyGraph />
      ) : (
        <div className="column-chart">
          {rows.map((item, index) => (
            <div className="column-item" key={item.label}>
              <div>
                <i
                  style={{
                    height: `${Math.max(8, (item.value / max) * 100)}%`,
                    background: palette[index % palette.length],
                  }}
                />
              </div>
              <span>{shortLabel(item.label)}</span>
            </div>
          ))}
        </div>
      )}
    </article>
  );
}

function GraphHeader({ title, value, label }) {
  return (
    <div className="graph-heading">
      <div>
        <h3>{title}</h3>
        <span>{label}</span>
      </div>
      <strong>{value}</strong>
    </div>
  );
}

function EmptyGraph() {
  return <p className="muted empty-graph">No graph data yet.</p>;
}

function safeRows(data) {
  return (Array.isArray(data) ? data : [])
    .map((item) => ({
      label: item.label || "Unassigned",
      value: Number(item.value || 0),
    }))
    .filter((item) => item.value >= 0);
}

function buildConicGradient(rows, total) {
  if (total <= 0) return "#D3DAE4";
  let start = 0;
  const segments = rows.map((item, index) => {
    const size = (item.value / total) * 360;
    const end = start + size;
    const segment = `${palette[index % palette.length]} ${start}deg ${end}deg`;
    start = end;
    return segment;
  });
  return `conic-gradient(${segments.join(", ")})`;
}

function shortLabel(label = "") {
  if (label.length <= 10) return label;
  return `${label.slice(0, 9)}...`;
}
