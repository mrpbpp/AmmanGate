"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { SidebarLayout } from "../components/SidebarLayout";
import { getSession } from "@/lib/auth";

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const router = useRouter();
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    // Check authentication
    getSession().then((session) => {
      if (!session) {
        router.push("/login");
      } else {
        setIsAuthenticated(true);
      }
      setIsLoading(false);
    });
  }, [router]);

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-slate-900">
        <div className="text-white">Loading...</div>
      </div>
    );
  }

  if (!isAuthenticated) {
    return null; // Will redirect
  }

  return <SidebarLayout>{children}</SidebarLayout>;
}
