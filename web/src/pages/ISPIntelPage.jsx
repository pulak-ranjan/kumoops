import React, { useState, useEffect, useCallback } from 'react';
import {
  Activity, RefreshCw, TrendingUp, TrendingDown, AlertTriangle,
  CheckCircle2, Shield, Globe, Zap, ChevronDown, BarChart2
} from 'lucide-react';
import { cn } from '../lib/utils';

const API = '/api';
const hdrs = () => ({ Authorization: `Bearer ${localStorage.getItem('kumoui_token') || ''}`, 'Content-Type': 'application/json' });
function check401(res) { if (res.status === 401) { window.location.href = '/login'; return true; } return false; }

const ISP_COLORS = {
  Gmail:   { bg: 'bg-red-100 dark:bg-red-900/30',    text: 'text-red-700 dark:text-red-400',    dot: 'bg-red-500' },
  Yahoo:   { bg: 'bg-purple-100 dark:bg-purple-900/30', text: 'text-purple-700 dark:text-purple-400', dot: 'bg-purple-500' },
  Outlook: { bg: 'bg-blue-100 dark:bg-blue-900/30',  text: 'text-blue-700 dark:text-blue-400',  dot: 'bg-blue-500' },
  AOL:     { bg: 'bg-yellow-100 dark:bg-yellow-900/30', text: 'text-yellow-700 dark:text-yellow-400', dot: 'bg-yellow-500' },
  Apple:   { bg: 'bg-gray-100 dark:bg-gray-800',     text: 'text-gray-700 dark:text-gray-300',  dot: 'bg-gray-400' },
  Other:   { bg: 'bg-slate-100 dark:bg-slate-800',   text: 'text-slate-600 dark:text-slate-400', dot: 'bg-slate-400' },
};

function HealthBar({ score }) {
  const color = score >= 80 ? 'bg-green-500' : score >= 60 ? 'bg-yellow-500' : score >= 40 ? 'bg-orange-500' : 'bg-red-500';
  const label = score >= 80 ? 'Excellent' : score >= 60 ? 'Good' : score >= 40 ? 'Fair' : 'Poor';
  return (
    <div className="flex items-center gap-2">
      <div className="flex-1 bg-muted rounded-full h-2 overflow-hidden">
        <div className={cn('h-full rounded-full transition-all', color)} style={{ width: `${score}%` }} />
      </div>
      <span className={cn('text-xs font-semibold w-16 text-right',
        score >= 80 ? 'text-green-600 dark:text-green-400' :
        score >= 60 ? 'text-yellow-600 dark:text-yellow-400' :
        score >= 40 ? 'text-orange-600 dark:text-orange-400' : 'text-red-600 dark:text-red-400'
      )}>{score} — {label}</span>
    </div>
  );
}

function ReputationBadge({ rep }) {
  if (!rep) return <span className="text-muted-foreground text-xs">N/A</span>;
  const map = {
    HIGH:   { label: 'High',   cls: 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-400' },
    MEDIUM: { label: 'Medium', cls: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/40 dark:text-yellow-400' },
    LOW:    { label: 'Low',    cls: 'bg-orange-100 text-orange-700 dark:bg-orange-900/40 dark:text-orange-400' },
    BAD:    { label: 'Bad',    cls: 'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-400' },
  };
  const m = map[rep] || { label: rep, cls: 'bg-muted text-muted-foreground' };
  return <span className={cn('px-2 py-0.5 rounded text-xs font-semibold', m.cls)}>{m.label}</span>;
}

function SNDSBadge({ result }) {
  if (!result) return <span className="text-muted-foreground text-xs">N/A</span>;
  const map = {
    GREEN:  { cls: 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-400' },
    YELLOW: { cls: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/40 dark:text-yellow-400' },
    RED:    { cls: 'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-400' },
  };
  const m = map[result] || { cls: 'bg-muted text-muted-foreground' };
  return <span className={cn('px-2 py-0.5 rounded text-xs font-semibold', m.cls)}>{result}</span>;
}

function pct(v) {
  if (v == null || isNaN(v)) return '—';
  return `${parseFloat(v).toFixed(2)}%`;
}

function num(v) {
  if (v == null || isNaN(v)) return '—';
  return parseInt(v).toLocaleString();
}

function fmt(ts) {
  if (!ts) return '—';
  return new Date(ts).toLocaleString();
}

export default function ISPIntelPage() {
  const [snapshots, setSnapshots] = useState([]);
  const [domain, setDomain] = useState('');
  const [days, setDays] = useState(7);
  const [loading, setLoading] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const [tab, setTab] = useState('overview');
  const [expandedISP, setExpandedISP] = useState(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams({ days, limit: 500 });
      if (domain) params.set('domain', domain);
      const res = await fetch(`${API}/isp-intel/snapshots/latest?${domain ? `domain=${domain}` : ''}`, { headers: hdrs() });
      if (check401(res)) return;
      if (res.ok) setSnapshots((await res.json()) ?? []);
    } catch {}
    setLoading(false);
  }, [domain, days]);

  useEffect(() => { load(); }, [load]);

  const triggerRefresh = async () => {
    setRefreshing(true);
    try {
      await fetch(`${API}/isp-intel/refresh`, { method: 'POST', headers: hdrs() });
      setTimeout(() => { load(); setRefreshing(false); }, 3000);
    } catch { setRefreshing(false); }
  };

  // Group by domain
  const byDomain = {};
  (snapshots || []).forEach(s => {
    if (!byDomain[s.domain]) byDomain[s.domain] = {};
    byDomain[s.domain][s.isp] = s;
  });

  const domainList = Object.keys(byDomain);
  const ispNames = ['Gmail', 'Yahoo', 'Outlook', 'AOL'];

  return (
    <div className="space-y-6 p-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <Globe className="w-6 h-6 text-primary" />
            ISP Intelligence
          </h1>
          <p className="text-muted-foreground text-sm mt-1">
            Per-ISP delivery health, reputation, and complaint rates. Updated hourly.
          </p>
        </div>
        <button
          onClick={triggerRefresh}
          disabled={refreshing}
          className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground text-sm font-medium disabled:opacity-50"
        >
          <RefreshCw className={cn('w-4 h-4', refreshing && 'animate-spin')} />
          {refreshing ? 'Refreshing...' : 'Refresh Now'}
        </button>
      </div>

      {/* Filters */}
      <div className="flex gap-3 flex-wrap">
        <input
          type="text"
          placeholder="Filter by domain..."
          value={domain}
          onChange={e => setDomain(e.target.value)}
          className="border border-border rounded-lg px-3 py-2 text-sm bg-background w-56"
        />
        <select
          value={days}
          onChange={e => setDays(Number(e.target.value))}
          className="border border-border rounded-lg px-3 py-2 text-sm bg-background"
        >
          <option value={1}>Last 24h</option>
          <option value={7}>Last 7 days</option>
          <option value={30}>Last 30 days</option>
        </select>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 border-b border-border">
        {['overview', 'detailed'].map(t => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={cn('px-4 py-2 text-sm font-medium capitalize border-b-2 -mb-px transition',
              tab === t ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground')}
          >
            {t === 'overview' ? 'Overview Dashboard' : 'Detailed View'}
          </button>
        ))}
      </div>

      {loading && (
        <div className="flex items-center justify-center py-12 text-muted-foreground">
          <RefreshCw className="w-5 h-5 animate-spin mr-2" /> Loading...
        </div>
      )}

      {!loading && tab === 'overview' && (
        <div className="space-y-6">
          {domainList.length === 0 && (
            <div className="text-center py-12 text-muted-foreground">
              No ISP snapshots yet. Click "Refresh Now" to collect intelligence data.
            </div>
          )}
          {domainList.map(dom => (
            <div key={dom} className="bg-card border border-border rounded-xl overflow-hidden">
              <div className="px-5 py-4 border-b border-border">
                <h2 className="font-semibold text-lg">{dom}</h2>
              </div>
              <div className="grid grid-cols-2 gap-4 p-5 md:grid-cols-4">
                {ispNames.map(isp => {
                  const s = byDomain[dom]?.[isp];
                  const colors = ISP_COLORS[isp] || ISP_COLORS.Other;
                  return (
                    <div key={isp} className={cn('rounded-xl p-4 border', colors.bg, 'border-transparent')}>
                      <div className="flex items-center gap-2 mb-3">
                        <div className={cn('w-2 h-2 rounded-full', colors.dot)} />
                        <span className={cn('font-semibold text-sm', colors.text)}>{isp}</span>
                      </div>
                      {!s ? (
                        <p className="text-xs text-muted-foreground">No data</p>
                      ) : (
                        <div className="space-y-2">
                          <HealthBar score={s.health_score || 0} />
                          <div className="grid grid-cols-2 gap-x-2 gap-y-1 text-xs mt-2">
                            <span className="text-muted-foreground">Acceptance</span>
                            <span className="font-medium text-right">{pct(s.acceptance_rate)}</span>
                            <span className="text-muted-foreground">Bounce</span>
                            <span className="font-medium text-right">{pct(s.bounce_rate)}</span>
                            <span className="text-muted-foreground">Deferral</span>
                            <span className="font-medium text-right">{pct(s.deferral_rate)}</span>
                            <span className="text-muted-foreground">Complaint</span>
                            <span className="font-medium text-right">{pct(s.complaint_rate)}</span>
                          </div>
                          {s.gpt_enabled && (
                            <div className="pt-1 border-t border-border/40 mt-1">
                              <p className="text-xs text-muted-foreground mb-1">Google Postmaster</p>
                              <ReputationBadge rep={s.gpt_domain_reputation} />
                            </div>
                          )}
                          {s.snds_enabled && (
                            <div className="pt-1">
                              <p className="text-xs text-muted-foreground mb-1">SNDS Filter</p>
                              <SNDSBadge result={s.snds_filter_result} />
                            </div>
                          )}
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            </div>
          ))}
        </div>
      )}

      {!loading && tab === 'detailed' && (
        <div className="overflow-x-auto rounded-xl border border-border">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border bg-muted/30">
                <th className="text-left px-4 py-3 font-medium text-muted-foreground">Domain</th>
                <th className="text-left px-4 py-3 font-medium text-muted-foreground">ISP</th>
                <th className="text-right px-4 py-3 font-medium text-muted-foreground">Health</th>
                <th className="text-right px-4 py-3 font-medium text-muted-foreground">Sent</th>
                <th className="text-right px-4 py-3 font-medium text-muted-foreground">Acceptance</th>
                <th className="text-right px-4 py-3 font-medium text-muted-foreground">Bounce</th>
                <th className="text-right px-4 py-3 font-medium text-muted-foreground">Deferral</th>
                <th className="text-right px-4 py-3 font-medium text-muted-foreground">Complaint</th>
                <th className="text-left px-4 py-3 font-medium text-muted-foreground">GPT Rep</th>
                <th className="text-left px-4 py-3 font-medium text-muted-foreground">SNDS</th>
                <th className="text-right px-4 py-3 font-medium text-muted-foreground">Captured</th>
              </tr>
            </thead>
            <tbody>
              {(snapshots || []).length === 0 && (
                <tr><td colSpan={11} className="text-center py-8 text-muted-foreground">No data</td></tr>
              )}
              {(snapshots || []).map((s, i) => {
                const colors = ISP_COLORS[s.isp] || ISP_COLORS.Other;
                return (
                  <tr key={i} className="border-b border-border last:border-0 hover:bg-muted/20">
                    <td className="px-4 py-3 font-medium">{s.domain}</td>
                    <td className="px-4 py-3">
                      <span className={cn('px-2 py-0.5 rounded text-xs font-semibold', colors.bg, colors.text)}>{s.isp}</span>
                    </td>
                    <td className="px-4 py-3 text-right">
                      <span className={cn('font-bold',
                        s.health_score >= 80 ? 'text-green-600 dark:text-green-400' :
                        s.health_score >= 60 ? 'text-yellow-600 dark:text-yellow-400' :
                        s.health_score >= 40 ? 'text-orange-600 dark:text-orange-400' : 'text-red-600 dark:text-red-400'
                      )}>{s.health_score ?? '—'}</span>
                    </td>
                    <td className="px-4 py-3 text-right">{num(s.total_sent)}</td>
                    <td className="px-4 py-3 text-right">{pct(s.acceptance_rate)}</td>
                    <td className="px-4 py-3 text-right">{pct(s.bounce_rate)}</td>
                    <td className="px-4 py-3 text-right">{pct(s.deferral_rate)}</td>
                    <td className="px-4 py-3 text-right">{pct(s.complaint_rate)}</td>
                    <td className="px-4 py-3"><ReputationBadge rep={s.gpt_domain_reputation} /></td>
                    <td className="px-4 py-3"><SNDSBadge result={s.snds_filter_result} /></td>
                    <td className="px-4 py-3 text-right text-xs text-muted-foreground">{fmt(s.captured_at)}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
