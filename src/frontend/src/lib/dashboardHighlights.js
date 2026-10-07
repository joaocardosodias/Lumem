export function dashboardHighlights(summary, predictions = []) {
  const stations = [...(summary?.critical_stations || [])].sort((a, b) => b.doors_at_risk - a.doors_at_risk)
  const prioritized = predictions.filter((item) => item.will_fail)
  return {
    stations,
    maxRisk: Math.max(...stations.map((item) => item.doors_at_risk), 1),
    topStation: stations[0],
    prioritized,
    visiblePredictions: prioritized.slice(0, 5),
  }
}
