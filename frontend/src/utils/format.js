export function formatPercent(value) {
  if (typeof value !== "number" || Number.isNaN(value)) {
    return "--";
  }
  return `${Math.round(value * 100)}%`;
}

export function cognitiveLabel(result) {
  const cognitive = result?.cognitive;
  if (!cognitive) {
    return {
      emotion: "--",
      gaze: "--",
      live: "--",
    };
  }
  return {
    emotion: `${cognitive.emotion?.label ?? "--"} ${formatPercent(cognitive.emotion?.confidence)}`,
    gaze: `${cognitive.gaze?.label ?? "--"} ${formatPercent(cognitive.gaze?.confidence)}`,
    live: cognitive.liveness?.is_live ? "Live" : "Check",
  };
}
