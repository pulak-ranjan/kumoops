import React, { useState, useEffect, useCallback, useRef } from 'react';
import {
  AlertTriangle, RefreshCw, Search, Filter, Download,
  Clock, XCircle, MailWarning, ChevronDown, ChevronUp,
  CheckCircle2, Inbox, Activity
} from 'lucide-react';
import { cn } from '../lib/utils';

const token = () => localStorage.getItem('kumoui_token') || '';
const headers = () => ({ Authorization: `Bearer ${token()}` });

// --- helpers ---
const EVENT_META = {
  Bounce:           { label: 'Hard Bounce',  bg: 'bg-red-100 dark:bg-red-900/30',    text: 'text-red-700 dark:text-red-400',    dot: 'bg-red-500' },
  TransientFailure: { label: 'Deferred',     bg: 'bg-orange-100 dark:bg-orange-900/30', text: 'text-orange-700 dark:text-orange-400', dot: 'bg-orange-500' },
};

function TypeBadge({ type }) {
  const m = EVENT_META[type] || { label: type, bg: 'bg-muted', text: 'text-muted-foreground', dot: 'bg-muted-foreground' };
  return (
    <span className={cn('inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium shrink-0', m.bg, m.text)}>
      <span className={cn('w-1.5 h-1.5 rounded-full', m.dot)} />
      {m.label}
    </span>
  );
}

function fmt(ts) {
  if (!ts) return '—';
  const d = new Date(ts);
  return d.toLocaleDateString('en-GB', { month: 'short', day: '2-digit' }) + ' ' +
    d.toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

function KpiCard({ label, value, icon: Icon, color, sub }) {
  return (
    <div className="bg-card border rounded-xl p-5 flex flex-col gap-1 shadow-sm">
      <div className="flex items-center justify-between">
        <span className="text-xs font-medium text-muted-foreground uppercase tracking-wide">{label}</span>
        <Icon className={cn('w-4 h-4', color)} />
      </div>
      <div className="text-3xl font-bold tabular-nums">{(value ?? 0).toLocaleString()}</div>
      {sub && <div className="text-xs text-muted-foreground">{sub}</div>}
    </div>
  );
}

export default function DeliveryLogPage() {
  const [events, setEvents]     = useState([]);
  const [total, setTotal]       = useState(0);
  const [page, setPage]         = useState(1);
  const [summary, setSummary]   = useState({ Bounce: 0, TransientFailure: 0 });
  const [loading, setLoading]   = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const [lastUpdated, setLastUpdated] = useState(null);
  const [expanded, setExpanded] = useState(null);

  // filters
  const [recipient, setRecipient] = useState('');
  const [domain, setDomain]       = useState('');
  const [typeFilter, setTypeFilter] = useState('');
  const [hours, setHours]         = useState(72);

  const limit = 50;
  const autoRefreshRef = useRef(null);

  const fetchSummary = useCallback(async () => {
    try {
      const res = await fetch(`/api/delivery-log/summary?hours=${hours}`, { headers: headers() });
      if (res.ok) setSummary(await res.json());
    } catch (_) {}
  }, [hours]);

  const fetchEvents = useCallback(async (pg = page) => {
    setLoading(true);
    try {
      const params = new URLSearchParams({
        page: pg, limit,
        hours,
        ...(recipient && { recipient }),
        ...(domain    && { domain }),
        ...(typeFilter && { type: typeFilter }),
      });
      const res = await fetch(`/api/delivery-log?${params}`, { headers: headers() });
      if (res.status === 401) { window.location.href = '/login'; return; }
      const data = await res.json();
      setEvents(data.events || []);
      setTotal(data.total || 0);
      setLastUpdated(new Date());
    } catch (_) {}
    setLoading(false);
  }, [page, hours, recipient, domain, typeFilter]);

  const doRefresh = async () => {
    setRefreshing(true);
    try {
      await fetch(`/api/delivery-log/refresh?hours=${hours}`, { method: 'POST', headers: headers() });
    } catch (_) {}
    await Promise.all([fetchEvents(1), fetchSummary()]);
    setPage(1);
    setRefreshing(false);
  };

  // auto-parse on initial page load
  useEffect(() => {
    doRefresh();
  }, []); // eslint-disable-line

  useEffect(() => {
    fetchSummary();
    fetchEvents(1);
    setPage(1);
  }, [hours, recipient, domain, typeFilter]); // eslint-disable-line

  useEffect(() => {
    fetchEvents(page);
  }, [page]); // eslint-disable-line

  // auto-parse + refresh every 60s
  useEffect(() => {
    autoRefreshRef.current = setInterval(() => {
      doRefresh();
    }, 60000);
    return () => clearInterval(autoRefreshRef.current);
  }, [hours]); // eslint-disable-line

  const totalPages = Math.ceil(total / limit);
  const totalFailures = (summary.Bounce || 0) + (summary.TransientFailure || 0);

  const exportCSV = () => {
    const header = ['Time', 'Recipient', 'Sender', 'Domain', 'Provider', 'Type', 'Error Code', 'Error Message'];
    const rows = events.map(e => [
      fmt(e.timestamp), e.recipient, e.sender, e.domain, e.provider,
      e.event_type, e.error_code, `"${(e.error_msg || '').replace(/"/g, "'")}"`,
    ]);
    const csv = [header, ...rows].map(r => r.join(',')).join('\n');
    const url = URL.createObjectURL(new Blob([csv], { type: 'text/csv' }));
    const a = document.createElement('a'); a.href = url;
    a.download = `delivery-failures-${Date.now()}.csv`; a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Delivery Log</h1>
          <p className="text-muted-foreground text-sm mt-1">
            Per-recipient failure tracking — bounces and deferrals from KumoMTA logs.
            {lastUpdated && (
              <span className="ml-2 text-xs opacity-60">
                Updated {lastUpdated.toLocaleTimeString()}
              </span>
            )}
          </p>
        </div>
        <div className="flex gap-2 flex-wrap">
          <select
            value={hours}
            onChange={e => setHours(+e.target.value)}
            className="h-9 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring"
          >
            <option value={24}>Last 24h</option>
            <option value={48}>Last 48h</option>
            <option value={72}>Last 72h</option>
            <option value={168}>Last 7 days</option>
          </select>
          <button
            onClick={exportCSV}
            className="flex items-center gap-2 h-9 px-3 rounded-md border bg-background hover:bg-accent text-sm transition-colors"
          >
            <Download className="w-4 h-4" /> Export
          </button>
          <button
            onClick={doRefresh}
            disabled={refreshing}
            className="flex items-center gap-2 h-9 px-4 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-sm font-medium transition-colors disabled:opacity-60"
          >
            <RefreshCw className={cn('w-4 h-4', refreshing && 'animate-spin')} />
            {refreshing ? 'Parsing…' : 'Parse Logs'}
          </button>
        </div>
      </div>

      {/* KPI strip */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <KpiCard label="Total Failures"  value={totalFailures}              icon={AlertTriangle} color="text-yellow-500" sub={`Last ${hours}h`} />
        <KpiCard label="Hard Bounces"    value={summary.Bounce}             icon={XCircle}       color="text-red-500"    sub="Permanent (5xx)" />
        <KpiCard label="Deferred"        value={summary.TransientFailure}   icon={MailWarning}   color="text-orange-500" sub="Soft / 4xx" />
        <KpiCard label="Showing"         value={total}                      icon={Inbox}         color="text-blue-500"   sub="matching filter" />
      </div>

      {/* Filter bar */}
      <div className="bg-card border rounded-xl p-4 flex flex-col sm:flex-row gap-3">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-2.5 w-4 h-4 text-muted-foreground" />
          <input
            type="text"
            placeholder="Search recipient email…"
            value={recipient}
            onChange={e => { setRecipient(e.target.value); setPage(1); }}
            className="w-full h-9 pl-9 pr-3 rounded-md border bg-background text-sm focus:ring-2 focus:ring-ring"
          />
        </div>
        <div className="relative">
          <Filter className="absolute left-3 top-2.5 w-4 h-4 text-muted-foreground" />
          <input
            type="text"
            placeholder="Domain filter…"
            value={domain}
            onChange={e => { setDomain(e.target.value); setPage(1); }}
            className="h-9 pl-9 pr-3 rounded-md border bg-background text-sm focus:ring-2 focus:ring-ring w-44"
          />
        </div>
        <select
          value={typeFilter}
          onChange={e => { setTypeFilter(e.target.value); setPage(1); }}
          className="h-9 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring"
        >
          <option value="">All Types</option>
          <option value="Bounce">Hard Bounce</option>
          <option value="TransientFailure">Deferred</option>
        </select>
        {(recipient || domain || typeFilter) && (
          <button
            onClick={() => { setRecipient(''); setDomain(''); setTypeFilter(''); setPage(1); }}
            className="h-9 px-3 rounded-md border border-destructive/50 text-destructive text-sm hover:bg-destructive/10 transition-colors"
          >
            Clear
          </button>
        )}
      </div>

      {/* Table */}
      <div className="bg-card border rounded-xl shadow-sm overflow-hidden">
        <div className="p-4 border-b flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Activity className="w-4 h-4 text-muted-foreground" />
            <span className="text-sm font-medium">
              {total.toLocaleString()} event{total !== 1 ? 's' : ''}
            </span>
          </div>
          {loading && <RefreshCw className="w-4 h-4 animate-spin text-muted-foreground" />}
        </div>

        <div className="overflow-x-auto">
          <table className="w-full text-sm text-left">
            <thead className="bg-muted/50 text-muted-foreground text-xs uppercase">
              <tr>
                <th className="px-4 py-3 font-medium">Time</th>
                <th className="px-4 py-3 font-medium">Recipient</th>
                <th className="px-4 py-3 font-medium">Provider</th>
                <th className="px-4 py-3 font-medium">Type</th>
                <th className="px-4 py-3 font-medium text-center">Code</th>
                <th className="px-4 py-3 font-medium">Error Reason</th>
                <th className="px-4 py-3 font-medium w-8"></th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {events.length === 0 && !loading ? (
                <tr>
                  <td colSpan={7} className="py-16 text-center text-muted-foreground">
                    <CheckCircle2 className="w-8 h-8 mx-auto mb-2 opacity-20" />
                    <p>No failure events found.</p>
                    <p className="text-xs mt-1 opacity-60">Click "Parse Logs" to scan KumoMTA log files.</p>
                  </td>
                </tr>
              ) : (
                events.map(ev => {
                  const isOpen = expanded === ev.id;
                  return (
                    <React.Fragment key={ev.id}>
                      <tr
                        className={cn(
                          'hover:bg-muted/40 transition-colors cursor-pointer',
                          isOpen && 'bg-muted/30'
                        )}
                        onClick={() => setExpanded(isOpen ? null : ev.id)}
                      >
                        <td className="px-4 py-3 whitespace-nowrap text-xs text-muted-foreground font-mono">
                          {fmt(ev.timestamp)}
                        </td>
                        <td className="px-4 py-3 font-medium max-w-[200px] truncate" title={ev.recipient}>
                          {ev.recipient}
                        </td>
                        <td className="px-4 py-3 text-xs text-muted-foreground">
                          {ev.provider || '—'}
                        </td>
                        <td className="px-4 py-3">
                          <TypeBadge type={ev.event_type} />
                        </td>
                        <td className="px-4 py-3 text-center">
                          {ev.error_code ? (
                            <span className={cn(
                              'font-mono text-xs font-bold px-1.5 py-0.5 rounded',
                              ev.error_code >= 500
                                ? 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
                                : 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400'
                            )}>
                              {ev.error_code}
                            </span>
                          ) : '—'}
                        </td>
                        <td className="px-4 py-3 text-xs text-muted-foreground max-w-[280px] truncate" title={ev.error_msg}>
                          {ev.error_msg || '—'}
                        </td>
                        <td className="px-4 py-3 text-muted-foreground">
                          {isOpen
                            ? <ChevronUp className="w-4 h-4" />
                            : <ChevronDown className="w-4 h-4" />}
                        </td>
                      </tr>
                      {isOpen && (
                        <tr className="bg-muted/20">
                          <td colSpan={7} className="px-6 py-4">
                            <div className="grid sm:grid-cols-2 gap-4 text-xs">
                              <div>
                                <p className="text-muted-foreground font-medium uppercase tracking-wide mb-1">Sender</p>
                                <p className="font-mono break-all">{ev.sender || '—'}</p>
                              </div>
                              <div>
                                <p className="text-muted-foreground font-medium uppercase tracking-wide mb-1">Recipient Domain</p>
                                <p className="font-mono">{ev.domain || '—'}</p>
                              </div>
                              <div className="sm:col-span-2">
                                <p className="text-muted-foreground font-medium uppercase tracking-wide mb-1">Full SMTP Response</p>
                                <pre className="bg-zinc-950 text-zinc-300 rounded-lg p-3 text-xs font-mono whitespace-pre-wrap break-all leading-relaxed">
                                  {ev.error_msg || 'No response message recorded.'}
                                </pre>
                              </div>
                            </div>
                          </td>
                        </tr>
                      )}
                    </React.Fragment>
                  );
                })
              )}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="p-4 border-t flex items-center justify-between text-sm">
            <span className="text-muted-foreground text-xs">
              Page {page} of {totalPages} &mdash; {total.toLocaleString()} total
            </span>
            <div className="flex gap-2">
              <button
                onClick={() => setPage(p => Math.max(1, p - 1))}
                disabled={page === 1}
                className="h-8 px-3 rounded-md border text-xs hover:bg-accent disabled:opacity-40 transition-colors"
              >
                Previous
              </button>
              <button
                onClick={() => setPage(p => Math.min(totalPages, p + 1))}
                disabled={page === totalPages}
                className="h-8 px-3 rounded-md border text-xs hover:bg-accent disabled:opacity-40 transition-colors"
              >
                Next
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Auto-refresh note */}
      <p className="text-xs text-muted-foreground text-center">
        <Clock className="w-3 h-3 inline mr-1" />
        Data refreshes automatically every 60 seconds. Click "Parse Logs" to force a fresh scan.
      </p>
    </div>
  );
}
