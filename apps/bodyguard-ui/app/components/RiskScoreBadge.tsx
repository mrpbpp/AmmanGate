export function RiskScoreBadge({ score }: { score: number }) {
  const getRiskInfo = (s: number) => {
    if (s >= 80) return { label: "Critical", className: "risk-critical" };
    if (s >= 60) return { label: "High", className: "risk-high" };
    if (s >= 40) return { label: "Medium", className: "risk-medium" };
    if (s >= 20) return { label: "Low", className: "risk-low" };
    return { label: "Safe", className: "risk-low" };
  };

  const { label, className } = getRiskInfo(score);

  return (
    <span className={`text-xs font-medium ${className}`}>
      {label} ({score})
    </span>
  );
}
