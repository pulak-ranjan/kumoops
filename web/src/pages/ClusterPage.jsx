import React, { useState, useEffect, useCallback } from 'react';
import {
  Network, RefreshCw, CheckCircle2, XCircle, AlertTriangle,
  Upload, BarChart2, Server, Clock, Zap, Mail
} from 'lucide-react';
import { cn } from '../lib/utils';

const API = '/api';

function authHeaders() {
  const token = localStorage.getItem('kumoui_token');
  return { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` };
}

async function apiFetch(path, opts = {}) {
  const r = await fetch(API + path, { headers: authHeaders(), ...opts });
  if (!r.ok) throw new Error(await r.text());
  return r.json();
}

function NodeStatusBadge({ status }) {
  const map = {
    online:  { cls: 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-400', icon: CheckCircle2 },
    offline: { cls: 'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-400', icon: XCircle },
    unknown: { cls: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/40 dark:text-yellow-400', icon: AlertTriangle },
  };
  const m = map[status] || map.unknown;
  const Icon = m.icon;
  return (
    <span className={cn('inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-semibold capitalize', m.cls)}>
      <Icon className="w-3 h-3" /> {status || 'Unknown'}
    </span>
  );
}

function MetricCard({ icon: Icon, label, value, sub, color }) {
  return (
    <div className="border rounded-lg p-3 bg-card">
      <div className="flex items-start justify-between">
        <div>
          <p className="text-xs text-muted-foreground">{label}</p>
          <p className={cn('text-lg font-bold mt-0.5', color || 'text-foreground')}>{value ?? '—'}</p>
          {sub && <p className="text-xs text-muted-foreground">{sub}</p>}
        </div>
        <Icon className={cn('w-4 h-4 mt-0.5', color || 'text-muted-foreground')} />
      </div>
    </div>
  );
}

function ago(ts) {
  if (!ts) return 'Never';
  const d = Math.round((Date.now() - new Date(ts).getTime()) / 1000);
  if (d < 60) return `${d}s ago`;
  if (d < 3600) return `${Math.round(d / 60)}m ago`;
  if (d < 86400) return `${Math.round(d / 3600)}h ago`;
  return `${Math.round(d / 86400)}d ago`;
}

export default function ClusterPage() {
  const [nodes, setNodes] = useState([]);
  const [metrics, setMetrics] = useState([]);
  const [loading, setLoading] = useState(true);
  const [metricsLoading, setMetricsLoading] = useState(false);
  const [pushing, setPushing] = useState(false);
  const [pushResult, setPushResult] = useState(null);
  const [error, setError] = useState('');
  const [tab, setTab] = useState('nodes');

  const loadNodes = useCallback(async () => {
    setLoading(true); setError('');
    try {
      const n = await apiFetch('/cluster/nodes');
      setNodes(n || []);
    } catch (e) { setError(e.message); }
    setLoading(false);
  }, []);

  const loadMetrics = useCallback(async () => {
    setMetricsLoading(true);
    try {
      const m = await apiFetch('/cluster/metrics');
      setMetrics(m || []);
    } catch { }
    setMetricsLoading(false);
  }, []);

  useEffect(() => { loadNodes(); }, [loadNodes]);

  const handlePushConfig = async () => {
    if (!confirm('Push current configuration to all online nodes?')) return;
    setPushing(true); setPushResult(null); setError('');
    try {
      const r = await apiFetch('/cluster/push-config', { method: 'POST' });
      setPushResult(r);
    } catch (e) { setError(e.message); }
    setPushing(false);
  };

  const handleTabChange = (t) => {
    setTab(t);
    if (t === 'metrics' && metrics.length === 0) loadMetrics();
  };

  const onlineCount = nodes.filter(n => n.status === 'online').length;

  const TABS = [
    { id: 'nodes', label: 'Nodes', icon: Server },
    { id: 'metrics', label: 'Cluster Metrics', icon: BarChart2 },
  ];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between flex-wrap gap-3">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <Network className="w-6 h-6 text-primary" /> Cluster Management
          </h1>
          <p className="text-muted-foreground text-sm mt-1">
            Manage multi-node KumoMTA deployments, push config, and view aggregated metrics.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button onClick={loadNodes} className="flex items-center gap-1.5 px-3 py-2 rounded-md border hover:bg-muted text-sm">
            <RefreshCw className="w-4 h-4" /> Refresh
          </button>
          <button onClick={handlePushConfig} disabled={pushing || nodes.length === 0}
            className="flex items-center gap-2 px-3 py-2 rounded-md bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 disabled:opacity-60">
            <Upload className="w-4 h-4" /> {pushing ? 'Pushing…' : 'Push Config'}
          </button>
        </div>
      </div>

      {error && <div className="text-sm text-destructive bg-destructive/10 px-4 py-2.5 rounded-md">{error}</div>}

      {/* Push result */}
      {pushResult && (
        <div className="border rounded-lg overflow-hidden">
          <div className="px-4 py-3 border-b bg-muted/30 text-sm font-semibold">Push Config Results</div>
          <div className="divide-y">
            {pushResult.map((r, i) => (
              <div key={i} className="flex items-center gap-3 px-4 py-2.5 text-sm">
                {r.success ? <CheckCircle2 className="w-4 h-4 text-green-500 shrink-0" /> : <XCircle className="w-4 h-4 text-red-500 shrink-0" />}
                <span className="font-medium">{r.node}</span>
                <span className="text-muted-foreground">{r.message}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Summary stats */}
      {!loading && nodes.length > 0 && (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
          <MetricCard icon={Server} label="Total Nodes" value={nodes.length} color="text-foreground" />
          <MetricCard icon={CheckCircle2} label="Online" value={onlineCount} color="text-green-600 dark:text-green-400" />
          <MetricCard icon={XCircle} label="Offline" value={nodes.length - onlineCount} color={nodes.length - onlineCount > 0 ? 'text-red-500' : 'text-muted-foreground'} />
          <MetricCard icon={Zap} label="Health" value={nodes.length > 0 ? `${Math.round((onlineCount / nodes.length) * 100)}%` : '—'} color="text-primary" />
        </div>
      )}

      {/* Tabs */}
      <div className="flex gap-1 border-b">
        {TABS.map(t => {
          const Icon = t.icon;
          return (
            <button key={t.id} onClick={() => handleTabChange(t.id)}
              className={cn('flex items-center gap-2 px-4 py-2.5 text-sm font-medium border-b-2 -mb-px transition-colors',
                tab === t.id ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground')}>
              <Icon className="w-4 h-4" /> {t.label}
            </button>
          );
        })}
      </div>

      {/* Nodes Tab */}
      {tab === 'nodes' && (
        loading ? (
          <div className="text-center py-16 text-muted-foreground text-sm">Loading nodes…</div>
        ) : nodes.length === 0 ? (
          <div className="text-center py-16 text-muted-foreground">
            <Server className="w-12 h-12 mx-auto mb-3 opacity-30" />
            <p className="text-sm font-medium">No remote nodes configured.</p>
            <p className="text-xs mt-1">Add nodes in <strong>Remote Servers</strong> to start managing them here.</p>
          </div>
        ) : (
          <div className="border rounded-lg overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-muted/50">
                <tr>
                  {['Node', 'URL', 'Status', 'Last Seen', 'Latency', ''].map(h => (
                    <th key={h} className="px-4 py-2.5 text-left text-xs font-semibold text-muted-foreground uppercase">{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y">
                {nodes.map(n => (
                  <tr key={n.id} className="hover:bg-muted/30">
                    <td className="px-4 py-3 font-medium">{n.name}</td>
                    <td className="px-4 py-3 text-muted-foreground font-mono text-xs">{n.url}</td>
                    <td className="px-4 py-3"><NodeStatusBadge status={n.status} /></td>
                    <td className="px-4 py-3 text-xs text-muted-foreground">{ago(n.last_seen)}</td>
                    <td className="px-4 py-3 text-xs">{n.latency_ms != null ? `${n.latency_ms}ms` : '—'}</td>
                    <td className="px-4 py-3">
                      {n.error && (
                        <span className="text-xs text-red-500 truncate max-w-[200px] block" title={n.error}>{n.error}</span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )
      )}

      {/* Metrics Tab */}
      {tab === 'metrics' && (
        metricsLoading ? (
          <div className="text-center py-16 text-muted-foreground text-sm">Fetching metrics from nodes…</div>
        ) : metrics.length === 0 ? (
          <div className="text-center py-16 text-muted-foreground">
            <BarChart2 className="w-12 h-12 mx-auto mb-3 opacity-30" />
            <p className="text-sm">No metrics available. Nodes may be offline.</p>
          </div>
        ) : (
          <div className="space-y-4">
            {metrics.map((m, i) => (
              <div key={i} className="border rounded-lg overflow-hidden">
                <div className="px-4 py-3 border-b bg-muted/30 flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <Server className="w-4 h-4 text-primary" />
                    <span className="font-semibold text-sm">{m.node}</span>
                    <NodeStatusBadge status={m.status || 'online'} />
                  </div>
                  <span className="text-xs text-muted-foreground">{m.url}</span>
                </div>
                {m.error ? (
                  <div className="px-4 py-3 text-sm text-red-500">{m.error}</div>
                ) : (
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-3 p-4">
                    <MetricCard icon={Mail} label="Total Messages" value={m.total?.toLocaleString()} />
                    <MetricCard icon={CheckCircle2} label="Delivered" value={m.delivered?.toLocaleString()}
                      color="text-green-600 dark:text-green-400" />
                    <MetricCard icon={XCircle} label="Bounced" value={m.bounced?.toLocaleString()}
                      color={m.bounced > 0 ? 'text-red-500' : 'text-muted-foreground'} />
                    <MetricCard icon={Clock} label="Queue Size" value={m.queue_size?.toLocaleString()}
                      color={m.queue_size > 1000 ? 'text-yellow-600 dark:text-yellow-400' : 'text-muted-foreground'} />
                  </div>
                )}
              </div>
            ))}
          </div>
        )
      )}
    </div>
  );
}
