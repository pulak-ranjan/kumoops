import React, { useEffect, useState } from "react";
import {
  Globe,
  Mail,
  Cpu,
  MemoryStick,
  Server,
  Activity,
  Sparkles
} from "lucide-react";
import { getDashboardStats, getAIInsights } from "../api";
import { cn } from "../lib/utils";

export default function Dashboard() {
  const [stats, setStats] = useState(null);
  const [insight, setInsight] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    (async () => {
      try {
        const s = await getDashboardStats();
        setStats(s);
      } catch (err) {
        setError("Failed to load stats");
      }
    })();
  }, []);

  const getAI = async () => {
    setLoading(true);
    setInsight("");
    try {
      const res = await getAIInsights();
      setInsight(res.analysis || res.insight);
    } catch (err) {
      setInsight("Error: " + err.message);
    } finally {
      setLoading(false);
    }
  };

  if (!stats) return <div className="p-8 text-muted-foreground flex justify-center">Loading dashboard...</div>;

  return (
    <div className="space-y-8">
      <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
        <div>
          <h2 className="text-3xl font-bold tracking-tight text-foreground">Dashboard</h2>
          <p className="text-muted-foreground mt-1">Overview of your email infrastructure.</p>
        </div>
        <button
          onClick={getAI}
          disabled={loading}
          className="bg-primary hover:bg-primary/90 text-primary-foreground px-4 py-2 rounded-md text-sm font-medium flex items-center gap-2 transition-colors shadow-sm"
        >
          {loading ? <Sparkles className="w-4 h-4 animate-spin" /> : <Sparkles className="w-4 h-4" />}
          {loading ? "Analyzing..." : "AI Log Analysis"}
        </button>
      </div>

      {insight && (
        <div className="bg-card border border-primary/20 p-6 rounded-xl shadow-sm relative overflow-hidden">
          <div className="absolute top-0 left-0 w-1 h-full bg-primary" />
          <h3 className="font-semibold text-primary mb-2 flex items-center gap-2">
            <Sparkles className="w-4 h-4" /> AI Insight
          </h3>
          <div className="text-sm text-foreground whitespace-pre-wrap leading-relaxed">
            {insight}
          </div>
        </div>
      )}

      {/* Main Stats */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard label="Total Domains" value={stats.domains} icon={Globe} color="text-blue-500" bg="bg-blue-500/10" />
        <StatCard label="Active Senders" value={stats.senders} icon={Mail} color="text-emerald-500" bg="bg-emerald-500/10" />
        <StatCard label="CPU Load" value={stats.cpu_load} icon={Cpu} color="text-orange-500" bg="bg-orange-500/10" />
        <StatCard label="RAM Usage" value={stats.ram_usage} icon={MemoryStick} color="text-purple-500" bg="bg-purple-500/10" />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Service Status */}
        <div className="bg-card border rounded-xl p-6 shadow-sm">
          <h3 className="text-lg font-semibold text-foreground mb-4 flex items-center gap-2">
            <Activity className="w-5 h-5 text-muted-foreground" />
            Service Status
          </h3>
          <div className="space-y-3">
            <ServiceRow name="KumoMTA" status={stats.kumo_status} />
            <ServiceRow name="Dovecot" status={stats.dovecot_status} />
            <ServiceRow name="Fail2Ban" status={stats.f2b_status} />
          </div>
        </div>

        {/* Open Ports */}
        <div className="bg-card border rounded-xl p-6 shadow-sm">
          <h3 className="text-lg font-semibold text-foreground mb-4 flex items-center gap-2">
            <Server className="w-5 h-5 text-muted-foreground" />
            Open Ports
          </h3>
          <div className="flex flex-wrap gap-2">
            {stats.open_ports ? (
              stats.open_ports.split(", ").map(port => (
                <span key={port} className="bg-secondary text-secondary-foreground px-3 py-1.5 rounded-md text-sm font-mono border font-medium">
                  {port}
                </span>
              ))
            ) : (
              <span className="text-muted-foreground text-sm">Scanning...</span>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function StatCard({ label, value, icon: Icon, color, bg }) {
  return (
    <div className="bg-card border rounded-xl p-6 shadow-sm hover:shadow-md transition-shadow">
      <div className="flex justify-between items-start">
        <div>
          <p className="text-sm font-medium text-muted-foreground">{label}</p>
          <p className="text-2xl font-bold mt-2 text-foreground">{value}</p>
        </div>
        <div className={cn("p-2.5 rounded-lg", bg ?? "bg-secondary", color)}>
          <Icon className="w-5 h-5" />
        </div>
      </div>
    </div>
  );
}

function ServiceRow({ name, status }) {
  const isActive = status === "active";
  return (
    <div className="flex items-center justify-between p-3 rounded-lg border bg-muted/40">
      <span className="font-medium text-foreground">{name}</span>
      <div className="flex items-center gap-2">
        <span className={cn(
          "text-xs font-semibold px-2.5 py-1 rounded-full capitalize",
          isActive ? "bg-green-500/10 text-green-600 dark:text-green-400" : "bg-red-500/10 text-red-600 dark:text-red-400"
        )}>
          {status || "Unknown"}
        </span>
        <div className={cn("w-2 h-2 rounded-full", isActive ? "bg-green-500 animate-pulse" : "bg-red-500")} />
      </div>
    </div>
  );
}
