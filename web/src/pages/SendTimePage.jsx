import React, { useState, useEffect, useCallback } from 'react';
import { Clock, TrendingUp, RefreshCw, Zap, Info } from 'lucide-react';
import { cn } from '../lib/utils';

const API = '/api';
const DAYS_OF_WEEK = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
const HOURS = Array.from({ length: 24 }, (_, i) => i);

function authHeaders() {
  const token = localStorage.getItem('kumoui_token');
  return { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` };
}

async function apiFetch(path, opts = {}) {
  const r = await fetch(API + path, { headers: authHeaders(), ...opts });
  if (r.status === 401) { window.location.href = '/login'; return null; }
  if (!r.ok) {
    const body = await r.text();
    let msg;
    try { msg = JSON.parse(body).error || body; } catch { msg = body; }
    throw new Error(msg);
  }
  return r.json();
}

function fmt12h(h) {
  if (h === 0) return '12am';
  if (h === 12) return '12pm';
  return h < 12 ? `${h}am` : `${h - 12}pm`;
}

function heatColor(rate, max) {
  if (!rate || max === 0) return 'bg-muted/40 text-muted-foreground/30';
  const pct = rate / max;
  if (pct >= 0.85) return 'bg-green-600 text-white';
  if (pct >= 0.65) return 'bg-green-500 text-white';
  if (pct >= 0.45) return 'bg-green-400 text-white';
  if (pct >= 0.30) return 'bg-yellow-400 text-black';
  if (pct >= 0.15) return 'bg-yellow-300 text-black';
  if (pct >= 0.05) return 'bg-orange-200 text-black';
  return 'bg-muted/40 text-muted-foreground/30';
}

function ScoreBadge({ rank }) {
  if (rank === 0) return <span className="px-2 py-0.5 rounded text-xs font-bold bg-green-600 text-white">#1 Best</span>;
  if (rank === 1) return <span className="px-2 py-0.5 rounded text-xs font-bold bg-green-500 text-white">#2</span>;
  if (rank === 2) return <span className="px-2 py-0.5 rounded text-xs font-bold bg-yellow-500 text-black">#3</span>;
  return <span className="px-2 py-0.5 rounded text-xs font-semibold bg-muted text-muted-foreground">#{rank + 1}</span>;
}

export default function SendTimePage() {
  const [heatmap, setHeatmap] = useState(null);
  const [loading, setLoading] = useState(true);
  const [domain, setDomain] = useState('');
  const [days, setDays] = useState(90);
  const [domains, setDomains] = useState([]);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    setLoading(true); setError('');
    try {
      const params = new URLSearchParams();
      if (domain) params.set('domain', domain);
      params.set('days', days);
      const data = await apiFetch(`/analytics/send-time?${params}`);
      setHeatmap(data);
    } catch (e) { setError(e.message); }
    setLoading(false);
  }, [domain, days]);

  useEffect(() => {
    apiFetch('/domains').then(d => setDomains(d || [])).catch(() => {});
    load();
  }, [load]);

  // Build cell lookup: day→hour→engageRate
  const cellMap = {};
  let maxRate = 0;
  (heatmap?.cells || []).forEach(c => {
    if (!cellMap[c.day_of_week]) cellMap[c.day_of_week] = {};
    cellMap[c.day_of_week][c.hour] = c.engage_rate;
    if (c.engage_rate > maxRate) maxRate = c.engage_rate;
  });

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between flex-wrap gap-3">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <Clock className="w-6 h-6 text-primary" /> Send-Time Optimization
          </h1>
          <p className="text-muted-foreground text-sm mt-1">
            Analyze when your recipients engage most to pick the best send windows.
          </p>
        </div>
        <div className="flex items-center gap-2 flex-wrap">
          <select value={domain} onChange={e => setDomain(e.target.value)}
            className="px-3 py-2 rounded-md border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary/50">
            <option value="">All Domains</option>
            {domains.map(d => <option key={d.id} value={d.name}>{d.name}</option>)}
          </select>
          <select value={days} onChange={e => setDays(+e.target.value)}
            className="px-3 py-2 rounded-md border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary/50">
            {[30, 60, 90, 180].map(d => <option key={d} value={d}>Last {d} days</option>)}
          </select>
          <button onClick={load} className="flex items-center gap-1.5 px-3 py-2 rounded-md border hover:bg-muted text-sm">
            <RefreshCw className="w-4 h-4" /> Refresh
          </button>
        </div>
      </div>

      {error && <div className="text-sm text-destructive bg-destructive/10 px-4 py-2.5 rounded-md">{error}</div>}

      {loading ? (
        <div className="text-center py-16 text-muted-foreground text-sm">Analyzing engagement data…</div>
      ) : !heatmap ? null : (
        <>
          {/* Data info */}
          {heatmap.data_days > 0 && (
            <div className="flex items-center gap-2 text-xs text-muted-foreground bg-muted/40 px-4 py-2.5 rounded-md">
              <Info className="w-4 h-4 shrink-0" />
              Heatmap based on <strong>{heatmap.data_days} days</strong> of engagement data.
              {heatmap.data_days < 14 && ' More data will improve accuracy.'}
            </div>
          )}

          {/* Top Recommendations */}
          {heatmap.recommendations?.length > 0 && (
            <div className="border rounded-lg overflow-hidden">
              <div className="px-4 py-3 border-b bg-muted/30 flex items-center gap-2">
                <Zap className="w-4 h-4 text-yellow-500" />
                <span className="font-semibold text-sm">Top Send Windows</span>
              </div>
              <div className="divide-y">
                {heatmap.recommendations.map((rec, i) => (
                  <div key={i} className="flex items-center gap-4 px-4 py-3">
                    <ScoreBadge rank={i} />
                    <div className="flex-1">
                      <div className="font-medium text-sm">{rec.label}</div>
                      <div className="text-xs text-muted-foreground">{DAYS_OF_WEEK[rec.day_of_week]} at {fmt12h(rec.hour)}</div>
                    </div>
                    <div className="text-right">
                      <div className="text-sm font-semibold text-green-600 dark:text-green-400">{(rec.score * 100).toFixed(1)}%</div>
                      <div className="text-xs text-muted-foreground">engage rate</div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Heatmap */}
          <div className="border rounded-lg overflow-hidden">
            <div className="px-4 py-3 border-b bg-muted/30 flex items-center gap-2">
              <TrendingUp className="w-4 h-4 text-primary" />
              <span className="font-semibold text-sm">Engagement Heatmap (Opens + Clicks)</span>
            </div>
            <div className="p-4 overflow-x-auto">
              {heatmap.data_days === 0 ? (
                <div className="text-center py-12 text-muted-foreground">
                  <Clock className="w-10 h-10 mx-auto mb-2 opacity-30" />
                  <p className="text-sm">No engagement data yet. Send some campaigns to see results.</p>
                </div>
              ) : (
                <div className="min-w-max">
                  {/* Hour axis header */}
                  <div className="flex items-center mb-1">
                    <div className="w-10 shrink-0" />
                    {HOURS.map(h => (
                      <div key={h} className="w-9 text-center text-[10px] text-muted-foreground font-medium leading-none shrink-0">
                        {h % 3 === 0 ? fmt12h(h) : ''}
                      </div>
                    ))}
                  </div>
                  {/* Rows: day of week */}
                  {DAYS_OF_WEEK.map((day, dayIdx) => (
                    <div key={day} className="flex items-center mb-0.5">
                      <div className="w-10 shrink-0 text-xs text-muted-foreground font-medium text-right pr-2">{day}</div>
                      {HOURS.map(h => {
                        const rate = cellMap[dayIdx]?.[h] || 0;
                        const pct = maxRate > 0 ? (rate / maxRate * 100).toFixed(1) : 0;
                        return (
                          <div key={h} title={`${day} ${fmt12h(h)}: ${(rate * 100).toFixed(2)}% engage rate`}
                            className={cn('w-9 h-7 rounded-sm mr-0.5 flex items-center justify-center text-[10px] font-medium shrink-0 cursor-default transition-all hover:ring-2 hover:ring-primary/50', heatColor(rate, maxRate))}>
                            {rate > 0 ? `${pct}` : ''}
                          </div>
                        );
                      })}
                    </div>
                  ))}
                  {/* Legend */}
                  <div className="flex items-center gap-2 mt-3 justify-end">
                    <span className="text-xs text-muted-foreground">Low</span>
                    {['bg-muted/40', 'bg-yellow-300', 'bg-yellow-400', 'bg-green-400', 'bg-green-500', 'bg-green-600'].map(c => (
                      <div key={c} className={cn('w-6 h-4 rounded-sm', c)} />
                    ))}
                    <span className="text-xs text-muted-foreground">High</span>
                  </div>
                </div>
              )}
            </div>
          </div>
        </>
      )}
    </div>
  );
}
