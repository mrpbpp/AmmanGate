"use client";

import { useRouter } from "next/navigation";

export function StatusCard({
  title,
  value,
  color,
  icon,
  onClick,
}: {
  title: string;
  value: string | number;
  color: "primary" | "success" | "danger" | "warning" | "info";
  icon: string;
  onClick?: () => void;
}) {
  const colorClasses = {
    primary: "border-blue-500 bg-blue-900/20 hover:bg-blue-900/30",
    success: "border-emerald-500 bg-emerald-900/20 hover:bg-emerald-900/30",
    danger: "border-red-500 bg-red-900/20 hover:bg-red-900/30",
    warning: "border-yellow-500 bg-yellow-900/20 hover:bg-yellow-900/30",
    info: "border-slate-500 bg-slate-700/50 hover:bg-slate-700",
  };

  const handleClick = () => {
    if (onClick) {
      onClick();
    }
  };

  return (
    <div
      onClick={handleClick}
      className={`card border-l-4 cursor-pointer transition-all ${colorClasses[color]}`}
    >
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm font-medium text-slate-400">{title}</p>
          <p className="text-2xl font-bold text-white mt-1">{value}</p>
        </div>
        <div className="text-4xl">{icon}</div>
      </div>
    </div>
  );
}
