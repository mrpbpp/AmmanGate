export function SeverityBadge({ severity }: { severity: number }) {
  const getSeverityInfo = (s: number) => {
    if (s >= 9) return { label: "Critical", className: "severity-critical" };
    if (s >= 7) return { label: "High", className: "severity-high" };
    if (s >= 5) return { label: "Medium", className: "severity-medium" };
    if (s >= 3) return { label: "Low", className: "severity-low" };
    return { label: "Info", className: "severity-info" };
  };

  const { label, className } = getSeverityInfo(severity);

  return (
    <span className={`px-2 py-1 rounded-md text-xs font-medium border ${className}`}>
      {label}
    </span>
  );
}
