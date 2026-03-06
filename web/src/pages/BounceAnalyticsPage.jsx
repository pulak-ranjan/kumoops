import React, { useState, useEffect } from 'react';
import {
  RefreshCw, AlertCircle, AlertTriangle, CheckCircle2, Ban,
  TrendingDown, Mail, Clock, ShieldAlert, Wifi, Lock
} from 'lucide-react';
import { cn } from '../lib/utils';

const CATEGORY_META = {
  hard: { label: 'Hard Bounce', color: 'bg-red-500', textColor: 'text-red-600 dark:text-red-400', icon: Ban },
  soft: { label: 'Soft Bounce', color: 'bg-amber-500', textColor: 'text-amber-600 dark:text-amber-400', icon: Clock },
  spam: { label: 'Spam Rejection', color: 'bg-orange-500', textColor: 'text-orange-600 dark:text-orange-400', icon: ShieldAlert },
  rate_limit: { label: 'Rate Limited', color: 'bg-yellow-500', textColor: 'text-yellow-600 dark:text-yellow-400', icon: TrendingDown },
  quota: { label: 'Mailbox Full', color: 'bg-purple-500', textColor: 'text-purple-600 dark:text-purple-400', icon: Mail },
  tls: { label: 'TLS Failure', color: 'bg-blue-500', textColor: 'text-blue-600 dark:text-blue-400', icon: Lock },
  dns: { label: 'DNS Failure', color: 'bg-gray-500', textColor: 'text-gray-600 dark:text-gray-400', icon: Wifi },
  auth: { label: 'Auth Failure', color: 'bg-rose-500', textColor: 'text-rose-600 dark:text-rose-400', icon: AlertCircle },
  unknown: { label: 'Unknown', color: 'bg-slate-400', textColor: 'text-slate-500', icon: AlertTriangle },
};

export default function BounceAnalyticsPage() {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [lines, setLines] = useState(2000);

  const token = localStorage.getItem('kumoui_token');
  const headers = { Authorization: `Bearer ${token}` };

  const fetchData = async () => {
    setLoading(true);
    try {
      const res = await fetch(`/api/bounce-analytics?lines=${lines}`, { headers });
      if (res.status === 401) { window.location.href = '/login'; return; }
      if (res.ok) setData(await res.json());
    } catch (e) { console.error(e); }
    setLoading(false);
  };

  useEffect(() => { fetchData(); }, [lines]);

  const summary = data?.summary || {};
  const byType = data?.by_type || {};
  const topErrors = data?.top_errors || [];

  const totalBounces = (byType.hard || 0) + (byType.soft || 0);
  const totalClassified = Object.values(byType).reduce((a, b) => a + b, 0);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Bounce Analytics</h1>
          <p className="text-muted-foreground">Classify and analyze delivery failures from KumoMTA logs.</p>
        </div>
        <div className="flex gap-2">
          <select
            value={lines}
            onChange={e => setLines(+e.target.value)}
            className="h-10 rounded-md border bg-background px-3 py-2 text-sm focus:ring-2 focus:ring-ring"
          >
            <option value={1000}>Last 1,000 lines</option>
            <option value={2000}>Last 2,000 lines</option>
            <option value={5000}>Last 5,000 lines</option>
          </select>
          <button
            onClick={fetchData}
            className="flex items-center gap-2 h-10 px-4 rounded-md bg-secondary text-secondary-foreground hover:bg-secondary/80 text-sm font-medium transition-colors"
          >
            <RefreshCw className={cn("w-4 h-4", loading && "animate-spin")} />
            Refresh
          </button>
        </div>
      </div>

      {/* Summary Stats */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        {[
          { label: 'Total Sent', value: summary.total_sent?.toLocaleString() || '0', icon: Mail, color: 'text-blue-500' },
          { label: 'Hard Bounces', value: byType.hard || 0, icon: Ban, color: 'text-red-500' },
          { label: 'Soft Bounces', value: byType.soft || 0, icon: Clock, color: 'text-amber-500' },
          { label: 'Bounce Rate', value: `${(summary.bounce_rate || 0).toFixed(1)}%`, icon: TrendingDown, color: summary.bounce_rate > 5 ? 'text-red-500' : 'text-green-500' },
        ].map(({ label, value, icon: Icon, color }) => (
          <div key={label} className="rounded-xl border bg-card p-4 shadow-sm">
            <div className="flex items-center gap-2 mb-2">
              <Icon className={cn("w-4 h-4", color)} />
              <span className="text-sm text-muted-foreground font-medium">{label}</span>
            </div>
            <div className={cn("text-2xl font-bold", color)}>{value}</div>
          </div>
        ))}
      </div>

      {/* Bounce Classification Chart */}
      <div className="rounded-xl border bg-card p-6 shadow-sm">
        <h2 className="text-lg font-semibold mb-4">Bounce Classification</h2>
        {totalClassified === 0 ? (
          <div className="flex flex-col items-center justify-center py-10 text-muted-foreground">
            <CheckCircle2 className="w-10 h-10 mb-2 text-green-500" />
            <p className="font-medium">No bounce events detected in recent logs</p>
          </div>
        ) : (
          <div className="space-y-3">
            {Object.entries(byType)
              .filter(([, v]) => v > 0)
              .sort(([, a], [, b]) => b - a)
              .map(([key, count]) => {
                const meta = CATEGORY_META[key] || CATEGORY_META.unknown;
                const pct = totalClassified > 0 ? (count / totalClassified * 100).toFixed(1) : 0;
                const Icon = meta.icon;
                return (
                  <div key={key} className="flex items-center gap-3">
                    <div className="w-36 flex items-center gap-2 shrink-0">
                      <Icon className={cn("w-4 h-4", meta.textColor)} />
                      <span className="text-sm font-medium">{meta.label}</span>
                    </div>
                    <div className="flex-1 bg-muted rounded-full h-2.5 overflow-hidden">
                      <div
                        className={cn("h-full rounded-full transition-all", meta.color)}
                        style={{ width: `${pct}%` }}
                      />
                    </div>
                    <span className="w-20 text-right text-sm font-mono text-muted-foreground">
                      {count} ({pct}%)
                    </span>
                  </div>
                );
              })}
          </div>
        )}
      </div>

      {/* Top Error Messages */}
      <div className="rounded-xl border bg-card shadow-sm">
        <div className="p-4 border-b flex items-center justify-between">
          <h2 className="text-lg font-semibold">Top Error Patterns</h2>
          <span className="text-xs text-muted-foreground">{topErrors.length} unique errors</span>
        </div>
        {topErrors.length === 0 ? (
          <div className="p-8 text-center text-muted-foreground">
            <CheckCircle2 className="w-8 h-8 mx-auto mb-2 text-green-500" />
            No bounce errors detected
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b bg-muted/40">
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">Error</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">Category</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">Type</th>
                  <th className="px-4 py-3 text-right text-xs font-medium text-muted-foreground uppercase tracking-wider">Count</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {topErrors.map((e, i) => {
                  const meta = CATEGORY_META[e.category] || CATEGORY_META.unknown;
                  const Icon = meta.icon;
                  return (
                    <tr key={i} className="hover:bg-muted/30 transition-colors">
                      <td className="px-4 py-3">
                        <code className="text-xs font-mono text-muted-foreground break-all">{e.error}</code>
                      </td>
                      <td className="px-4 py-3">
                        <span className={cn(
                          "inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium",
                          meta.textColor, "bg-opacity-10 border",
                          e.category === 'hard' ? 'bg-red-50 border-red-200 dark:bg-red-950 dark:border-red-800' :
                          e.category === 'spam' ? 'bg-orange-50 border-orange-200 dark:bg-orange-950 dark:border-orange-800' :
                          'bg-muted border-border'
                        )}>
                          <Icon className="w-3 h-3" />
                          {meta.label}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <span className={cn(
                          "px-2 py-0.5 rounded-full text-xs font-medium",
                          e.type === 'hard' ? 'bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300' :
                          e.type === 'soft' ? 'bg-amber-100 text-amber-700 dark:bg-amber-900 dark:text-amber-300' :
                          'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400'
                        )}>
                          {e.type}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-right font-mono text-sm font-semibold">{e.count}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Additional Stats Row */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="rounded-xl border bg-card p-4">
          <div className="flex items-center gap-2 mb-1">
            <ShieldAlert className="w-4 h-4 text-orange-500" />
            <span className="text-sm font-medium">Spam Rejections</span>
          </div>
          <div className="text-2xl font-bold text-orange-500">{byType.spam || 0}</div>
          <p className="text-xs text-muted-foreground mt-1">Emails rejected as spam by ISPs</p>
        </div>
        <div className="rounded-xl border bg-card p-4">
          <div className="flex items-center gap-2 mb-1">
            <TrendingDown className="w-4 h-4 text-yellow-500" />
            <span className="text-sm font-medium">Rate Limited</span>
          </div>
          <div className="text-2xl font-bold text-yellow-500">{byType.rate_limit || 0}</div>
          <p className="text-xs text-muted-foreground mt-1">Temporarily deferred by ISP throttles</p>
        </div>
        <div className="rounded-xl border bg-card p-4">
          <div className="flex items-center gap-2 mb-1">
            <CheckCircle2 className="w-4 h-4 text-green-500" />
            <span className="text-sm font-medium">Total Delivered</span>
          </div>
          <div className="text-2xl font-bold text-green-500">{summary.delivered || 0}</div>
          <p className="text-xs text-muted-foreground mt-1">Successfully delivered from logs</p>
        </div>
      </div>
    </div>
  );
}
