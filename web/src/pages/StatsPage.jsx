import React, { useState, useEffect, useCallback, useRef } from 'react';
import { Line, Bar } from 'react-chartjs-2';
import {
  Chart as ChartJS, CategoryScale, LinearScale, PointElement,
  LineElement, BarElement, Title, Tooltip, Legend, Filler
} from 'chart.js';
import {
  BarChart3, Send, CheckCircle2, XCircle, Clock, RefreshCw,
  Filter, Globe, AlertTriangle, TrendingDown, Activity, Zap,
  TrendingUp, Radio,
} from 'lucide-react';
import { cn } from '../lib/utils';

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, BarElement, Title, Tooltip, Legend, Filler);

const token = () => localStorage.getItem('kumoui_token') || '';
const hdrs  = () => ({ Authorization: `Bearer ${token()}` });

// ─── Provider metadata ────────────────────────────────────────────────────────
const PROVIDER_META = {
  Gmail:   { color: '#EA4335', bg: 'bg-red-100 dark:bg-red-900/30',      text: 'text-red-700 dark:text-red-400',      dot: 'bg-red-500' },
  Outlook: { color: '#0078D4', bg: 'bg-blue-100 dark:bg-blue-900/30',    text: 'text-blue-700 dark:text-blue-400',    dot: 'bg-blue-500' },
  Yahoo:   { color: '#6001D2', bg: 'bg-purple-100 dark:bg-purple-900/30', text: 'text-purple-700 dark:text-purple-400', dot: 'bg-purple-500' },
  Apple:   { color: '#555555', bg: 'bg-gray-100 dark:bg-gray-800',        text: 'text-gray-700 dark:text-gray-300',    dot: 'bg-gray-500' },
  AOL:     { color: '#FF0B00', bg: 'bg-orange-100 dark:bg-orange-900/30', text: 'text-orange-700 dark:text-orange-400', dot: 'bg-orange-500' },
  Other:   { color: '#9CA3AF', bg: 'bg-slate-100 dark:bg-slate-800',      text: 'text-slate-600 dark:text-slate-400',  dot: 'bg-slate-400' },
};
const getProviderMeta = p => PROVIDER_META[p] || PROVIDER_META.Other;

function isDark() { return document.documentElement.classList.contains('dark'); }
function chartTheme() {
  const dark = isDark();
  return {
    grid: dark ? 'rgba(255,255,255,0.07)' : 'rgba(0,0,0,0.06)',
    text: dark ? '#9ca3af' : '#6b7280',
  };
}
function baseOpts() {
  const { grid, text } = chartTheme();
  return {
    responsive: true,
    maintainAspectRatio: false,
    interaction: { mode: 'index', intersect: false },
    plugins: {
      legend: { position: 'top', labels: { color: text, padding: 16, usePointStyle: true, pointStyleWidth: 8 } },
      tooltip: {
        backgroundColor: isDark() ? '#1f2937' : '#fff',
        titleColor: text, bodyColor: text,
        borderColor: isDark() ? '#374151' : '#e5e7eb', borderWidth: 1, padding: 10,
      },
    },
    scales: {
      x: { ticks: { color: text, maxRotation: 45 }, grid: { color: grid } },
      y: { ticks: { color: text }, grid: { color: grid }, beginAtZero: true },
    },
  };
}

function DeliveryRateBar({ rate }) {
  const color = rate >= 95 ? 'bg-green-500' : rate >= 85 ? 'bg-yellow-500' : 'bg-red-500';
  return (
    <div className="flex items-center gap-2">
      <div className="flex-1 bg-muted rounded-full h-2 overflow-hidden min-w-[60px]">
        <div className={cn('h-2 rounded-full transition-all', color)} style={{ width: `${Math.min(rate, 100)}%` }} />
      </div>
      <span className={cn('text-xs font-semibold tabular-nums w-12 text-right',
        rate >= 95 ? 'text-green-600 dark:text-green-400' :
        rate >= 85 ? 'text-yellow-600 dark:text-yellow-400' :
        'text-red-600 dark:text-red-400')}>
        {rate.toFixed(1)}%
      </span>
    </div>
  );
}

function ProviderBadge({ provider }) {
  const m = getProviderMeta(provider);
  return (
    <span className={cn('inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium', m.bg, m.text)}>
      <span className={cn('w-1.5 h-1.5 rounded-full', m.dot)} />
      {provider}
    </span>
  );
}

function StatBox({ label, value, icon: Icon, color }) {
  return (
    <div className="bg-card border rounded-xl p-4 shadow-sm flex flex-col justify-between h-full">
      <div className="flex items-center justify-between mb-2">
        <span className="text-xs font-medium text-muted-foreground uppercase tracking-wide">{label}</span>
        <Icon className={cn('w-4 h-4', color)} />
      </div>
      <div className="text-2xl font-bold tabular-nums">
        {typeof value === 'string' ? value : (value ?? 0).toLocaleString()}
      </div>
    </div>
  );
}

// ─── Live (Hourly) Tab ────────────────────────────────────────────────────────
function LiveTab({ autoRefresh }) {
  const [data, setData]       = useState([]);
  const [hours, setHours]     = useState(24);
  const [loading, setLoading] = useState(true);
  const [countdown, setCountdown] = useState(30);
  const timerRef = useRef(null);
  const countRef = useRef(null);

  const fetchHourly = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch(`/api/stats/hourly?hours=${hours}`, { headers: hdrs() });
      if (res.ok) setData(await res.json());
    } catch (_) {}
    setLoading(false);
    setCountdown(30);
  }, [hours]);

  useEffect(() => { fetchHourly(); }, [fetchHourly]);

  useEffect(() => {
    clearInterval(timerRef.current);
    clearInterval(countRef.current);
    if (!autoRefresh) return;
    timerRef.current = setInterval(fetchHourly, 30000);
    countRef.current = setInterval(() => setCountdown(c => (c > 0 ? c - 1 : 30)), 1000);
    return () => { clearInterval(timerRef.current); clearInterval(countRef.current); };
  }, [autoRefresh, fetchHourly]);

  const labels = data.map(d => d.hour);
  const opts   = baseOpts();

  const areaData = {
    labels,
    datasets: [
      { label: 'Sent',      data: data.map(d => d.sent),      borderColor: '#3b82f6', backgroundColor: 'rgba(59,130,246,0.15)',  fill: true, tension: 0.4, pointRadius: 2 },
      { label: 'Delivered', data: data.map(d => d.delivered), borderColor: '#22c55e', backgroundColor: 'rgba(34,197,94,0.15)',   fill: true, tension: 0.4, pointRadius: 2 },
      { label: 'Bounced',   data: data.map(d => d.bounced),   borderColor: '#ef4444', backgroundColor: 'rgba(239,68,68,0.15)',   fill: true, tension: 0.4, pointRadius: 2 },
      { label: 'Deferred',  data: data.map(d => d.deferred),  borderColor: '#f97316', backgroundColor: 'rgba(249,115,22,0.15)',  fill: true, tension: 0.4, pointRadius: 2 },
    ],
  };

  const totalSent      = data.reduce((s, d) => s + d.sent, 0);
  const totalDelivered = data.reduce((s, d) => s + d.delivered, 0);
  const totalBounced   = data.reduce((s, d) => s + d.bounced, 0);
  const totalDeferred  = data.reduce((s, d) => s + d.deferred, 0);
  const peakHour       = data.reduce((best, d) => d.sent > (best?.sent ?? -1) ? d : best, null);

  return (
    <div className="space-y-6">
      {/* Controls */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex gap-2">
          {[6, 12, 24, 48, 72].map(h => (
            <button key={h} onClick={() => setHours(h)}
              className={cn('h-8 px-3 rounded-md text-xs font-medium border transition-colors',
                hours === h ? 'bg-primary text-primary-foreground border-primary' : 'bg-background hover:bg-accent border-border')}>
              {h}h
            </button>
          ))}
        </div>
        <div className="flex items-center gap-3">
          {autoRefresh && (
            <span className="text-xs text-muted-foreground flex items-center gap-1.5">
              <span className="w-2 h-2 rounded-full bg-green-500 animate-pulse" />
              Refreshes in {countdown}s
            </span>
          )}
          <button onClick={fetchHourly}
            className="flex items-center gap-1.5 h-8 px-3 rounded-md border bg-background hover:bg-accent text-xs transition-colors">
            <RefreshCw className={cn('w-3.5 h-3.5', loading && 'animate-spin')} /> Now
          </button>
        </div>
      </div>

      {/* Hourly KPIs */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <StatBox label={`Sent (${hours}h)`}      value={totalSent}      icon={Send}         color="text-blue-500" />
        <StatBox label={`Delivered (${hours}h)`} value={totalDelivered} icon={CheckCircle2} color="text-green-500" />
        <StatBox label={`Bounced (${hours}h)`}   value={totalBounced}   icon={XCircle}      color="text-red-500" />
        <StatBox label={`Deferred (${hours}h)`}  value={totalDeferred}  icon={Clock}        color="text-orange-500" />
      </div>

      {/* Main area chart */}
      <div className="bg-card border rounded-xl p-6 shadow-sm">
        <div className="flex items-center justify-between mb-5">
          <div>
            <h3 className="text-base font-semibold">Hourly Traffic</h3>
            <p className="text-xs text-muted-foreground mt-0.5">Last {hours} hours — area chart updates every 30s</p>
          </div>
          {peakHour && peakHour.sent > 0 && (
            <div className="text-right text-xs">
              <p className="text-muted-foreground">Peak hour</p>
              <p className="font-semibold font-mono">{peakHour.hour}</p>
              <p className="text-blue-500">{peakHour.sent.toLocaleString()} sent</p>
            </div>
          )}
        </div>
        <div className="h-[320px]">
          {loading ? (
            <div className="h-full flex items-center justify-center text-muted-foreground">
              <RefreshCw className="w-5 h-5 animate-spin mr-2" /> Loading hourly data…
            </div>
          ) : <Line data={areaData} options={opts} />}
        </div>
      </div>

      {/* Hourly breakdown table */}
      <div className="bg-card border rounded-xl shadow-sm overflow-hidden">
        <div className="p-4 border-b"><h3 className="text-base font-semibold">Hour-by-Hour Breakdown</h3></div>
        <div className="overflow-x-auto">
          <table className="w-full text-sm text-left">
            <thead className="bg-muted/50 text-muted-foreground text-xs uppercase">
              <tr>
                <th className="px-4 py-3">Hour</th>
                <th className="px-4 py-3 text-right">Sent</th>
                <th className="px-4 py-3 text-right">Delivered</th>
                <th className="px-4 py-3 text-right">Bounced</th>
                <th className="px-4 py-3 text-right">Deferred</th>
                <th className="px-4 py-3">Delivery Rate</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {[...data].reverse().map(h => {
                const rate = h.sent > 0 ? (h.delivered / h.sent * 100) : 0;
                return (
                  <tr key={h.hour} className="hover:bg-muted/40 transition-colors">
                    <td className="px-4 py-2.5 font-mono text-xs text-muted-foreground">{h.hour}</td>
                    <td className="px-4 py-2.5 text-right tabular-nums">{h.sent.toLocaleString()}</td>
                    <td className="px-4 py-2.5 text-right tabular-nums text-green-600 dark:text-green-400">{h.delivered.toLocaleString()}</td>
                    <td className="px-4 py-2.5 text-right tabular-nums text-red-600 dark:text-red-400">{h.bounced.toLocaleString()}</td>
                    <td className="px-4 py-2.5 text-right tabular-nums text-orange-600 dark:text-orange-400">{h.deferred.toLocaleString()}</td>
                    <td className="px-4 py-2.5 min-w-[140px]">
                      {h.sent > 0 ? <DeliveryRateBar rate={rate} /> : <span className="text-muted-foreground text-xs">—</span>}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}

// ─── Overview Tab ─────────────────────────────────────────────────────────────
function OverviewTab({ stats, domainList, days, loading }) {
  const [selectedDomain, setSelectedDomain] = useState('');
  const availableDomains = domainList.length > 0 ? domainList.map(d => d.name) : Object.keys(stats);

  const getChartData = () => {
    if (selectedDomain && stats[selectedDomain]) return stats[selectedDomain];
    const agg = {};
    Object.keys(stats).forEach(d => (stats[d] || []).forEach(day => {
      if (!agg[day.date]) agg[day.date] = { date: day.date, sent: 0, delivered: 0, bounced: 0, deferred: 0 };
      agg[day.date].sent      += day.sent      || 0;
      agg[day.date].delivered += day.delivered || 0;
      agg[day.date].bounced   += day.bounced   || 0;
      agg[day.date].deferred  += day.deferred  || 0;
    }));
    return Object.values(agg).sort((a, b) => a.date.localeCompare(b.date));
  };

  const chartData = getChartData();
  const labels    = chartData.map(d => d.date);
  const opts      = baseOpts();

  const lineData = {
    labels,
    datasets: [
      { label: 'Sent',      data: chartData.map(d => d.sent),      borderColor: '#3b82f6', backgroundColor: 'rgba(59,130,246,0.12)', fill: true, tension: 0.4 },
      { label: 'Delivered', data: chartData.map(d => d.delivered), borderColor: '#22c55e', backgroundColor: 'rgba(34,197,94,0.12)',  fill: true, tension: 0.4 },
      { label: 'Bounced',   data: chartData.map(d => d.bounced),   borderColor: '#ef4444', backgroundColor: 'rgba(239,68,68,0.12)',  fill: true, tension: 0.4 },
    ],
  };

  const barData = {
    labels: availableDomains.slice(0, 10),
    datasets: [
      { label: 'Sent',    data: availableDomains.slice(0, 10).map(d => (stats[d] || []).reduce((s, x) => s + (x.sent || 0), 0)),    backgroundColor: 'rgba(59,130,246,0.75)' },
      { label: 'Bounced', data: availableDomains.slice(0, 10).map(d => (stats[d] || []).reduce((s, x) => s + (x.bounced || 0), 0)), backgroundColor: 'rgba(239,68,68,0.75)' },
    ],
  };

  return (
    <div className="space-y-6">
      <div className="flex justify-end">
        <div className="relative">
          <Filter className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
          <select value={selectedDomain} onChange={e => setSelectedDomain(e.target.value)}
            className="h-9 rounded-md border bg-background pl-9 pr-3 text-sm focus:ring-2 focus:ring-ring min-w-[150px]">
            <option value="">All Domains</option>
            {availableDomains.map(d => <option key={d} value={d}>{d}</option>)}
          </select>
        </div>
      </div>

      <div className="grid lg:grid-cols-2 gap-6">
        <div className="bg-card border rounded-xl p-6 shadow-sm">
          <h3 className="text-base font-semibold mb-1">Traffic Trend</h3>
          <p className="text-xs text-muted-foreground mb-4">Daily sent / delivered / bounced</p>
          <div className="h-[300px]"><Line data={lineData} options={opts} /></div>
        </div>
        <div className="bg-card border rounded-xl p-6 shadow-sm">
          <h3 className="text-base font-semibold mb-1">Volume by Domain</h3>
          <p className="text-xs text-muted-foreground mb-4">Top 10 domains</p>
          <div className="h-[300px]"><Bar data={barData} options={opts} /></div>
        </div>
      </div>

      <div className="bg-card border rounded-xl shadow-sm overflow-hidden">
        <div className="p-5 border-b"><h3 className="text-base font-semibold">Domain Breakdown</h3></div>
        <div className="overflow-x-auto">
          {loading ? (
            <div className="p-8 text-center text-muted-foreground">Loading…</div>
          ) : (
            <table className="w-full text-sm text-left">
              <thead className="bg-muted/50 text-muted-foreground uppercase text-xs">
                <tr>
                  <th className="px-6 py-3 font-medium">Domain</th>
                  <th className="px-6 py-3 font-medium text-right">Sent</th>
                  <th className="px-6 py-3 font-medium text-right">Delivered</th>
                  <th className="px-6 py-3 font-medium text-right">Bounced</th>
                  <th className="px-6 py-3 font-medium text-right">Success Rate</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {availableDomains.length === 0 ? (
                  <tr><td colSpan="5" className="p-8 text-center text-muted-foreground">No data available</td></tr>
                ) : availableDomains.map(domain => {
                  const s = (stats[domain] || []).reduce((a, d) => ({
                    sent: a.sent + (d.sent || 0), delivered: a.delivered + (d.delivered || 0), bounced: a.bounced + (d.bounced || 0),
                  }), { sent: 0, delivered: 0, bounced: 0 });
                  const rate = s.sent > 0 ? (s.delivered / s.sent * 100).toFixed(1) : 0;
                  return (
                    <tr key={domain} className="hover:bg-muted/40 transition-colors">
                      <td className="px-6 py-4 font-medium">{domain}</td>
                      <td className="px-6 py-4 text-right tabular-nums">{s.sent.toLocaleString()}</td>
                      <td className="px-6 py-4 text-right tabular-nums text-green-600 dark:text-green-400">{s.delivered.toLocaleString()}</td>
                      <td className="px-6 py-4 text-right tabular-nums text-red-600 dark:text-red-400">{s.bounced.toLocaleString()}</td>
                      <td className="px-6 py-4 text-right">
                        <span className={cn('px-2 py-1 rounded-full text-xs font-medium',
                          rate > 95 ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400' :
                          rate > 80 ? 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400' :
                          'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400')}>
                          {rate}%
                        </span>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          )}
        </div>
      </div>
    </div>
  );
}

// ─── Providers Tab ────────────────────────────────────────────────────────────
function ProvidersTab({ days }) {
  const [providerStats, setProviderStats] = useState([]);
  const [loading, setLoading]             = useState(true);
  const [expandedProvider, setExpandedProvider] = useState(null);

  useEffect(() => {
    (async () => {
      setLoading(true);
      try {
        const res = await fetch(`/api/stats/providers?days=${days}`, { headers: hdrs() });
        if (res.status === 401) { window.location.href = '/login'; return; }
        const data = await res.json();
        setProviderStats(Array.isArray(data) ? data : []);
      } catch (_) {}
      setLoading(false);
    })();
  }, [days]);

  const totalSent = providerStats.reduce((s, p) => s + p.sent, 0);
  const opts      = baseOpts();
  const { grid, text } = chartTheme();

  const comparisonData = {
    labels: providerStats.map(p => p.provider),
    datasets: [{
      label: 'Delivery Rate %',
      data:  providerStats.map(p => +p.delivery_rate.toFixed(1)),
      backgroundColor: providerStats.map(p => getProviderMeta(p.provider).color + 'CC'),
      borderColor:     providerStats.map(p => getProviderMeta(p.provider).color),
      borderWidth: 1,
    }],
  };
  const compOpts = {
    ...opts,
    indexAxis: 'y',
    plugins: { ...opts.plugins, legend: { display: false }, tooltip: { callbacks: { label: ctx => ` ${ctx.raw}%` } } },
    scales: {
      x: { min: 0, max: 100, ticks: { color: text, callback: v => `${v}%` }, grid: { color: grid } },
      y: { ticks: { color: text }, grid: { color: grid } },
    },
  };

  if (loading) return (
    <div className="flex items-center justify-center py-20 text-muted-foreground">
      <RefreshCw className="w-5 h-5 animate-spin mr-2" /> Loading provider data…
    </div>
  );

  const withDeferrals = providerStats.filter(p => p.deferral_reasons && Object.keys(p.deferral_reasons).length > 0 && p.deferred > 0);

  return (
    <div className="space-y-6">
      <div className="bg-card border rounded-xl p-6 shadow-sm">
        <h3 className="text-base font-semibold mb-1">Delivery Rate by Provider</h3>
        <p className="text-xs text-muted-foreground mb-4">% of sent messages delivered per ISP</p>
        <div className="h-[220px]"><Bar data={comparisonData} options={compOpts} /></div>
      </div>

      <div className="bg-card border rounded-xl shadow-sm overflow-hidden">
        <div className="p-5 border-b">
          <h3 className="text-base font-semibold">Per-Provider Delivery Intelligence</h3>
          <p className="text-xs text-muted-foreground mt-0.5">Delivery, bounce and deferral rates by destination email provider</p>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-sm text-left">
            <thead className="bg-muted/50 text-muted-foreground uppercase text-xs">
              <tr>
                <th className="px-6 py-3 font-medium">Provider</th>
                <th className="px-6 py-3 font-medium text-right">Sent</th>
                <th className="px-6 py-3 font-medium text-right">Delivered</th>
                <th className="px-6 py-3 font-medium text-right">Bounced</th>
                <th className="px-6 py-3 font-medium text-right">Deferred</th>
                <th className="px-6 py-3 font-medium">Delivery Rate</th>
                <th className="px-6 py-3 font-medium text-right">Bounce%</th>
                <th className="px-6 py-3 font-medium text-right">Share</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {providerStats.length === 0 ? (
                <tr><td colSpan="8" className="p-8 text-center text-muted-foreground">
                  <Globe className="w-8 h-8 mx-auto mb-2 opacity-30" />
                  No provider data. Logs may not contain recipient addresses yet.
                </td></tr>
              ) : providerStats.map(p => {
                const share = totalSent > 0 ? (p.sent / totalSent * 100).toFixed(1) : 0;
                const bounceColor = p.bounce_rate < 2 ? 'text-green-600 dark:text-green-400' :
                  p.bounce_rate < 5 ? 'text-yellow-600 dark:text-yellow-400' : 'text-red-600 dark:text-red-400';
                return (
                  <tr key={p.provider} className="hover:bg-muted/40 transition-colors">
                    <td className="px-6 py-4"><ProviderBadge provider={p.provider} /></td>
                    <td className="px-6 py-4 text-right tabular-nums font-medium">{p.sent.toLocaleString()}</td>
                    <td className="px-6 py-4 text-right tabular-nums text-green-600 dark:text-green-400">{p.delivered.toLocaleString()}</td>
                    <td className="px-6 py-4 text-right tabular-nums text-red-600 dark:text-red-400">{p.bounced.toLocaleString()}</td>
                    <td className="px-6 py-4 text-right tabular-nums text-orange-600 dark:text-orange-400">{p.deferred.toLocaleString()}</td>
                    <td className="px-6 py-4 min-w-[160px]">
                      {p.sent > 0 ? <DeliveryRateBar rate={p.delivery_rate} /> : <span className="text-muted-foreground text-xs">No data</span>}
                    </td>
                    <td className={cn('px-6 py-4 text-right tabular-nums text-xs font-medium', bounceColor)}>
                      {p.bounce_rate.toFixed(2)}%
                    </td>
                    <td className="px-6 py-4 text-right tabular-nums text-muted-foreground text-xs">{share}%</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </div>

      {withDeferrals.length > 0 && (
        <div className="bg-card border rounded-xl shadow-sm overflow-hidden">
          <div className="p-5 border-b flex items-center gap-2">
            <AlertTriangle className="w-5 h-5 text-orange-500" />
            <div>
              <h3 className="text-base font-semibold">Deferral Reason Analysis</h3>
              <p className="text-xs text-muted-foreground mt-0.5">Why messages are being temporarily rejected per ISP</p>
            </div>
          </div>
          <div className="divide-y divide-border">
            {withDeferrals.map(p => {
              const reasons = Object.entries(p.deferral_reasons).sort((a, b) => b[1] - a[1]);
              const totalDeferrals = reasons.reduce((s, [, c]) => s + c, 0);
              const isExpanded = expandedProvider === p.provider;
              return (
                <div key={p.provider}>
                  <button onClick={() => setExpandedProvider(isExpanded ? null : p.provider)}
                    className="w-full flex items-center justify-between px-6 py-4 hover:bg-muted/30 transition-colors text-left">
                    <div className="flex items-center gap-3">
                      <ProviderBadge provider={p.provider} />
                      <span className="text-sm text-muted-foreground">
                        {totalDeferrals.toLocaleString()} deferrals · {reasons.length} reason{reasons.length !== 1 ? 's' : ''}
                      </span>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="text-xs text-orange-600 dark:text-orange-400 font-medium">{p.deferral_rate.toFixed(1)}% rate</span>
                      <TrendingDown className={cn('w-4 h-4 text-muted-foreground transition-transform', isExpanded && 'rotate-180')} />
                    </div>
                  </button>
                  {isExpanded && (
                    <div className="px-6 pb-5">
                      <div className="space-y-2">
                        {reasons.map(([reason, count]) => {
                          const pct = totalDeferrals > 0 ? count / totalDeferrals * 100 : 0;
                          return (
                            <div key={reason} className="flex items-center gap-3">
                              <span className="text-xs text-muted-foreground w-36 shrink-0">{reason}</span>
                              <div className="flex-1 bg-muted rounded-full h-2 overflow-hidden">
                                <div className="h-2 rounded-full bg-orange-400 dark:bg-orange-500 transition-all" style={{ width: `${pct}%` }} />
                              </div>
                              <span className="text-xs tabular-nums text-muted-foreground w-20 text-right">
                                {count.toLocaleString()} ({pct.toFixed(0)}%)
                              </span>
                            </div>
                          );
                        })}
                      </div>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}

// ─── Main StatsPage ───────────────────────────────────────────────────────────
export default function StatsPage() {
  const [stats, setStats]       = useState({});
  const [summary, setSummary]   = useState(null);
  const [domainList, setDomainList] = useState([]);
  const [days, setDays]         = useState(7);
  const [loading, setLoading]   = useState(true);
  const [activeTab, setActiveTab] = useState('live');
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [lastUpdated, setLastUpdated] = useState(null);
  const autoRef = useRef(null);

  const fetchAll = useCallback(async () => {
    setLoading(true);
    try {
      const [domainsRes, statsRes, summaryRes] = await Promise.all([
        fetch('/api/domains', { headers: hdrs() }),
        fetch(`/api/stats/domains?days=${days}`, { headers: hdrs() }),
        fetch('/api/stats/summary', { headers: hdrs() }),
      ]);
      if (statsRes.status === 401) { window.location.href = '/login'; return; }
      setDomainList(Array.isArray(await domainsRes.json()) ? await domainsRes.clone().json() : []);
      setStats(await statsRes.json() || {});
      setSummary(await summaryRes.json());
      setLastUpdated(new Date());
    } catch (_) {}
    setLoading(false);
  }, [days]);

  useEffect(() => { fetchAll(); }, [fetchAll]);

  useEffect(() => {
    clearInterval(autoRef.current);
    if (autoRefresh) autoRef.current = setInterval(fetchAll, 30000);
    return () => clearInterval(autoRef.current);
  }, [autoRefresh, fetchAll]);

  const tabs = [
    { id: 'live',      label: 'Live (Hourly)', icon: Radio },
    { id: 'overview',  label: 'Overview',      icon: BarChart3 },
    { id: 'providers', label: 'By Provider',   icon: Globe },
  ];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Statistics</h1>
          <div className="flex items-center gap-2 mt-1">
            <p className="text-muted-foreground text-sm">Email traffic analysis and delivery reports.</p>
            {lastUpdated && (
              <span className="text-xs text-muted-foreground opacity-60 ml-1">
                · {lastUpdated.toLocaleTimeString()}
              </span>
            )}
          </div>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {/* Live toggle */}
          <button onClick={() => setAutoRefresh(v => !v)}
            className={cn('flex items-center gap-1.5 h-9 px-3 rounded-md border text-xs font-medium transition-colors',
              autoRefresh
                ? 'bg-green-500/10 border-green-500/50 text-green-600 dark:text-green-400'
                : 'bg-background border-border text-muted-foreground hover:bg-accent')}>
            <span className={cn('w-2 h-2 rounded-full', autoRefresh ? 'bg-green-500 animate-pulse' : 'bg-muted-foreground')} />
            {autoRefresh ? 'Live' : 'Paused'}
          </button>

          {activeTab !== 'live' && (
            <select value={days} onChange={e => setDays(+e.target.value)}
              className="h-9 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring">
              <option value={1}>Today</option>
              <option value={7}>Last 7 days</option>
              <option value={14}>Last 14 days</option>
              <option value={30}>Last 30 days</option>
              <option value={90}>Last 90 days</option>
            </select>
          )}

          <button onClick={async () => {
            await fetch('/api/stats/refresh', { method: 'POST', headers: hdrs() });
            fetchAll();
          }} className="flex items-center gap-2 h-9 px-4 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-sm font-medium transition-colors">
            <RefreshCw className={cn('w-4 h-4', loading && 'animate-spin')} /> Refresh
          </button>
        </div>
      </div>

      {/* Global KPI strip */}
      {summary && (
        <div className="grid grid-cols-2 lg:grid-cols-5 gap-4">
          <StatBox label="Total Sent"    value={summary.total_sent}                       icon={Send}         color="text-blue-500" />
          <StatBox label="Delivered"     value={summary.total_delivered}                  icon={CheckCircle2} color="text-green-500" />
          <StatBox label="Bounced"       value={summary.total_bounced}                    icon={XCircle}      color="text-red-500" />
          <StatBox label="Delivery Rate" value={`${summary.delivery_rate?.toFixed(1)}%`} icon={Activity}     color="text-emerald-500" />
          <StatBox label="Bounce Rate"   value={`${summary.bounce_rate?.toFixed(1)}%`}   icon={Zap}          color="text-orange-500" />
        </div>
      )}

      {/* Tab bar */}
      <div className="border-b flex gap-1">
        {tabs.map(({ id, label, icon: Icon }) => (
          <button key={id} onClick={() => setActiveTab(id)}
            className={cn(
              'flex items-center gap-2 px-4 py-2.5 text-sm font-medium border-b-2 -mb-px transition-colors',
              activeTab === id
                ? 'border-primary text-primary'
                : 'border-transparent text-muted-foreground hover:text-foreground hover:border-muted-foreground')}>
            <Icon className="w-4 h-4" />
            {label}
            {id === 'live' && autoRefresh && <span className="w-1.5 h-1.5 rounded-full bg-green-500 animate-pulse" />}
          </button>
        ))}
      </div>

      {activeTab === 'live'      && <LiveTab autoRefresh={autoRefresh} />}
      {activeTab === 'overview'  && <OverviewTab stats={stats} domainList={domainList} days={days} loading={loading} />}
      {activeTab === 'providers' && <ProvidersTab days={days} />}
    </div>
  );
}
