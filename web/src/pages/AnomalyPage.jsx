import React, { useState, useEffect, useCallback } from 'react';
import {
  AlertTriangle, CheckCircle2, RefreshCw, ShieldAlert, Zap,
  Clock, TrendingDown, TrendingUp, Activity
} from 'lucide-react';
import { cn } from '../lib/utils';

const API = '/api';

const ANOMALY_TYPE_META = {
  bounce_spike:     { label: 'Bounce Spike',     icon: TrendingDown, color: 'text-red-600 dark:text-red-400',     bg: 'bg-red-50 dark:bg-red-900/20' },
  deferral_spike:   { label: 'Deferral Spike',   icon: TrendingDown, color: 'text-orange-600 dark:text-orange-400', bg: 'bg-orange-50 dark:bg-orange-900/20' },
  acceptance_drop:  { label: 'Acceptance Drop',  icon: TrendingDown, color: 'text-amber-600 dark:text-amber-400',  bg: 'bg-amber-50 dark:bg-amber-900/20' },
  complaint_spike:  { label: 'Complaint Spike',  icon: ShieldAlert,  color: 'text-purple-600 dark:text-purple-400', bg: 'bg-purple-50 dark:bg-purple-900/20' },
};

const SEVERITY_META = {
  critical: { cls: 'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-400', label: 'Critical' },
  warning:  { cls: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/40 dark:text-yellow-400', label: 'Warning' },
};

function fmt(ts) {
  if (!ts) return '—';
  return new Date(ts).toLocaleString();
}

function pct(v) {
  if (v == null || isNaN(v)) return '—';
  return `${parseFloat(v).toFixed(2)}%`;
}

export default function AnomalyPage() {
  const [active, setActive] = useState([]);
  const [history, setHistory] = useState([]);
  const [throttleLogs, setThrottleLogs] = useState([]);
  const [tab, setTab] = useState('active');
  const [loading, setLoading] = useState(false);
  const [resolving, setResolving] = useState(null);
  const [runningThrottle, setRunningThrottle] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [activeRes, histRes, logsRes] = await Promise.all([
        fetch(`${API}/anomalies/active`),
        fetch(`${API}/anomalies?days=30&limit=100`),
        fetch(`${API}/throttle/logs?days=7&limit=100`),
      ]);
      if (activeRes.ok) setActive(await activeRes.json());
      if (histRes.ok) setHistory(await histRes.json());
      if (logsRes.ok) setThrottleLogs(await logsRes.json());
    } catch {}
    setLoading(false);
  }, []);

  useEffect(() => { load(); }, [load]);

  const resolveAnomaly = async (id) => {
    setResolving(id);
    try {
      await fetch(`${API}/anomalies/${id}/resolve`, { method: 'POST' });
      await load();
    } catch {}
    setResolving(null);
  };

  const runThrottle = async () => {
    setRunningThrottle(true);
    try {
      await fetch(`${API}/throttle/run`, { method: 'POST' });
      setTimeout(() => { load(); setRunningThrottle(false); }, 2000);
    } catch { setRunningThrottle(false); }
  };

  const AnomalyRow = ({ ev, showResolve }) => {
    const meta = ANOMALY_TYPE_META[ev.type] || { label: ev.type, icon: AlertTriangle, color: 'text-muted-foreground', bg: 'bg-muted/30' };
    const sevMeta = SEVERITY_META[ev.severity] || { cls: 'bg-muted text-muted-foreground', label: ev.severity };
    const Icon = meta.icon;
    return (
      <tr className="border-b border-border last:border-0 hover:bg-muted/20">
        <td className="px-4 py-3">
          <div className={cn('inline-flex items-center gap-1.5 px-2 py-1 rounded text-xs font-semibold', meta.bg, meta.color)}>
            <Icon className="w-3 h-3" />
            {meta.label}
          </div>
        </td>
        <td className="px-4 py-3">
          <span className={cn('px-2 py-0.5 rounded text-xs font-semibold', sevMeta.cls)}>{sevMeta.label}</span>
        </td>
        <td className="px-4 py-3 text-sm">{ev.isp || '—'}</td>
        <td className="px-4 py-3 text-sm">{ev.domain || '—'}</td>
        <td className="px-4 py-3 text-right text-sm font-mono">{pct(ev.metric_value)}</td>
        <td className="px-4 py-3 text-right text-sm font-mono">{pct(ev.threshold)}</td>
        <td className="px-4 py-3 text-sm text-muted-foreground truncate max-w-xs">{ev.action_taken || '—'}</td>
        <td className="px-4 py-3 text-sm text-muted-foreground">{fmt(ev.detected_at)}</td>
        {showResolve && (
          <td className="px-4 py-3">
            {ev.resolved_at ? (
              <span className="flex items-center gap-1 text-xs text-green-600 dark:text-green-400">
                <CheckCircle2 className="w-3 h-3" />
                {ev.auto_healed ? 'Auto-healed' : 'Resolved'}
              </span>
            ) : (
              <button
                onClick={() => resolveAnomaly(ev.id)}
                disabled={resolving === ev.id}
                className="text-xs px-2 py-1 rounded bg-primary text-primary-foreground disabled:opacity-50"
              >
                {resolving === ev.id ? '...' : 'Resolve'}
              </button>
            )}
          </td>
        )}
      </tr>
    );
  };

  const TableHeader = ({ showResolve }) => (
    <thead>
      <tr className="border-b border-border bg-muted/30">
        <th className="text-left px-4 py-3 font-medium text-muted-foreground">Type</th>
        <th className="text-left px-4 py-3 font-medium text-muted-foreground">Severity</th>
        <th className="text-left px-4 py-3 font-medium text-muted-foreground">ISP</th>
        <th className="text-left px-4 py-3 font-medium text-muted-foreground">Domain</th>
        <th className="text-right px-4 py-3 font-medium text-muted-foreground">Value</th>
        <th className="text-right px-4 py-3 font-medium text-muted-foreground">Threshold</th>
        <th className="text-left px-4 py-3 font-medium text-muted-foreground">Action</th>
        <th className="text-left px-4 py-3 font-medium text-muted-foreground">Detected</th>
        {showResolve && <th className="text-left px-4 py-3 font-medium text-muted-foreground">Status</th>}
      </tr>
    </thead>
  );

  return (
    <div className="space-y-6 p-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <Activity className="w-6 h-6 text-primary" />
            Anomaly Detection & Adaptive Throttling
          </h1>
          <p className="text-muted-foreground text-sm mt-1">
            Real-time delivery anomaly monitoring with automated self-healing. Adaptive throttling adjusts per-ISP rates automatically.
          </p>
        </div>
        <div className="flex gap-2">
          <button
            onClick={load}
            className="flex items-center gap-2 px-4 py-2 rounded-lg border border-border text-sm font-medium hover:bg-muted"
          >
            <RefreshCw className={cn('w-4 h-4', loading && 'animate-spin')} />
            Refresh
          </button>
          <button
            onClick={runThrottle}
            disabled={runningThrottle}
            className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground text-sm font-medium disabled:opacity-50"
          >
            <Zap className="w-4 h-4" />
            {runningThrottle ? 'Running...' : 'Run Throttle Cycle'}
          </button>
        </div>
      </div>

      {/* Summary cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div className="bg-card border border-border rounded-xl p-4">
          <div className="flex items-center gap-2 text-red-600 dark:text-red-400 mb-1">
            <AlertTriangle className="w-4 h-4" />
            <span className="text-xs font-semibold uppercase tracking-wide">Active Anomalies</span>
          </div>
          <p className="text-3xl font-bold">{(active || []).length}</p>
        </div>
        <div className="bg-card border border-border rounded-xl p-4">
          <div className="flex items-center gap-2 text-orange-600 dark:text-orange-400 mb-1">
            <ShieldAlert className="w-4 h-4" />
            <span className="text-xs font-semibold uppercase tracking-wide">Critical</span>
          </div>
          <p className="text-3xl font-bold">{(active || []).filter(e => e.severity === 'critical').length}</p>
        </div>
        <div className="bg-card border border-border rounded-xl p-4">
          <div className="flex items-center gap-2 text-green-600 dark:text-green-400 mb-1">
            <CheckCircle2 className="w-4 h-4" />
            <span className="text-xs font-semibold uppercase tracking-wide">Auto-Healed (30d)</span>
          </div>
          <p className="text-3xl font-bold">{(history || []).filter(e => e.auto_healed).length}</p>
        </div>
        <div className="bg-card border border-border rounded-xl p-4">
          <div className="flex items-center gap-2 text-blue-600 dark:text-blue-400 mb-1">
            <Zap className="w-4 h-4" />
            <span className="text-xs font-semibold uppercase tracking-wide">Throttle Adjustments (7d)</span>
          </div>
          <p className="text-3xl font-bold">{(throttleLogs || []).length}</p>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 border-b border-border">
        {[
          { id: 'active', label: 'Active Anomalies' },
          { id: 'history', label: 'History (30d)' },
          { id: 'throttle', label: 'Throttle Log' },
        ].map(t => (
          <button
            key={t.id}
            onClick={() => setTab(t.id)}
            className={cn('px-4 py-2 text-sm font-medium border-b-2 -mb-px transition',
              tab === t.id ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground')}
          >
            {t.label}
            {t.id === 'active' && active.length > 0 && (
              <span className="ml-2 px-1.5 py-0.5 rounded-full bg-red-500 text-white text-xs">{active.length}</span>
            )}
          </button>
        ))}
      </div>

      {/* Active anomalies */}
      {tab === 'active' && (
        <div className="overflow-x-auto rounded-xl border border-border">
          <table className="w-full text-sm">
            <TableHeader showResolve />
            <tbody>
              {(active || []).length === 0 && (
                <tr><td colSpan={9} className="text-center py-10 text-muted-foreground">
                  <CheckCircle2 className="w-8 h-8 mx-auto mb-2 text-green-500" />
                  No active anomalies — system healthy
                </td></tr>
              )}
              {(active || []).map(ev => <AnomalyRow key={ev.id} ev={ev} showResolve />)}
            </tbody>
          </table>
        </div>
      )}

      {/* History */}
      {tab === 'history' && (
        <div className="overflow-x-auto rounded-xl border border-border">
          <table className="w-full text-sm">
            <TableHeader showResolve />
            <tbody>
              {(history || []).length === 0 && (
                <tr><td colSpan={9} className="text-center py-10 text-muted-foreground">No anomaly history in the last 30 days</td></tr>
              )}
              {(history || []).map(ev => <AnomalyRow key={ev.id} ev={ev} showResolve />)}
            </tbody>
          </table>
        </div>
      )}

      {/* Throttle Log */}
      {tab === 'throttle' && (
        <div className="overflow-x-auto rounded-xl border border-border">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border bg-muted/30">
                <th className="text-left px-4 py-3 font-medium text-muted-foreground">ISP</th>
                <th className="text-left px-4 py-3 font-medium text-muted-foreground">Direction</th>
                <th className="text-left px-4 py-3 font-medium text-muted-foreground">Old Rate</th>
                <th className="text-left px-4 py-3 font-medium text-muted-foreground">New Rate</th>
                <th className="text-right px-4 py-3 font-medium text-muted-foreground">Old Conns</th>
                <th className="text-right px-4 py-3 font-medium text-muted-foreground">New Conns</th>
                <th className="text-right px-4 py-3 font-medium text-muted-foreground">Deferral%</th>
                <th className="text-right px-4 py-3 font-medium text-muted-foreground">Accept%</th>
                <th className="text-left px-4 py-3 font-medium text-muted-foreground">Reason</th>
                <th className="text-left px-4 py-3 font-medium text-muted-foreground">Time</th>
              </tr>
            </thead>
            <tbody>
              {(throttleLogs || []).length === 0 && (
                <tr><td colSpan={10} className="text-center py-10 text-muted-foreground">No throttle adjustments in last 7 days</td></tr>
              )}
              {(throttleLogs || []).map((log, i) => (
                <tr key={i} className="border-b border-border last:border-0 hover:bg-muted/20">
                  <td className="px-4 py-3 font-medium">{log.isp}</td>
                  <td className="px-4 py-3">
                    {log.direction === 'down' ? (
                      <span className="flex items-center gap-1 text-red-600 dark:text-red-400 text-xs font-semibold">
                        <TrendingDown className="w-3 h-3" /> Throttle Down
                      </span>
                    ) : (
                      <span className="flex items-center gap-1 text-green-600 dark:text-green-400 text-xs font-semibold">
                        <TrendingUp className="w-3 h-3" /> Scale Up
                      </span>
                    )}
                  </td>
                  <td className="px-4 py-3 font-mono text-xs">{log.old_rate}</td>
                  <td className="px-4 py-3 font-mono text-xs">{log.new_rate}</td>
                  <td className="px-4 py-3 text-right">{log.old_connections}</td>
                  <td className="px-4 py-3 text-right">{log.new_connections}</td>
                  <td className="px-4 py-3 text-right">{pct(log.deferral_rate)}</td>
                  <td className="px-4 py-3 text-right">{pct(log.accept_rate)}</td>
                  <td className="px-4 py-3 text-xs text-muted-foreground truncate max-w-xs">{log.reason}</td>
                  <td className="px-4 py-3 text-xs text-muted-foreground">{fmt(log.created_at)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
