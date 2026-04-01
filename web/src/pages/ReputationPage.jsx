import React, { useState, useEffect, useCallback, useRef } from 'react';
import {
  Shield, ShieldAlert, ShieldCheck, RefreshCw, Server,
  Globe, Clock, AlertTriangle, CheckCircle2, Loader2, ExternalLink,
} from 'lucide-react';
import { cn } from '../lib/utils';

const token = () => localStorage.getItem('kumoui_token') || '';
const hdrs = () => ({ Authorization: `Bearer ${token()}` });

// Hardcoded delist URLs as fallback (also fetched from API)
const DELIST_FALLBACK = {
  'zen.spamhaus.org':        'https://check.spamhaus.org/',
  'dbl.spamhaus.org':        'https://check.spamhaus.org/',
  'b.barracudacentral.org':  'https://www.barracudacentral.org/rbl/removal-request',
  'bl.spamcop.net':          'https://www.spamcop.net/bl.shtml',
  'truncate.gbudb.net':      'http://www.gbudb.com/truncate/',
  'dnsbl-1.uceprotect.net':  'https://www.uceprotect.net/en/index.php?m=7&s=0',
  'psbl.surriel.com':        'https://psbl.org/remove',
  'multi.surbl.org':         'https://www.surbl.org/surbl-analysis',
  'black.uribl.com':         'https://admin.uribl.com/',
  'dbl.nordspam.com':        'https://www.nordspam.com/delist/',
};

const DELIST_NOTES = {
  'zen.spamhaus.org':        'Free. CSS/XBL auto-remove in minutes. SBL may need ISP contact.',
  'dbl.spamhaus.org':        'Free. Auto-expires when activity stops.',
  'b.barracudacentral.org':  'Free. Processed within ~12 hours.',
  'bl.spamcop.net':          'Free. Auto-delists after 24h if no new reports.',
  'truncate.gbudb.net':      'Automated only. Auto-delists in 1-2 days when spam stops.',
  'dnsbl-1.uceprotect.net':  'Free auto-delist in 7 days. Paid express ~50 EUR.',
  'psbl.surriel.com':        'Free. Simple web form, instant removal.',
  'multi.surbl.org':         'Free. 24-48h processing.',
  'black.uribl.com':         'Free. Requires account registration.',
  'dbl.nordspam.com':        'Free. Email delist@nordspam.com from your domain.',
};

function StatusBadge({ status }) {
  if (status === 'clean') return (
    <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400">
      <ShieldCheck className="w-3.5 h-3.5" /> Clean
    </span>
  );
  if (status === 'listed') return (
    <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400">
      <ShieldAlert className="w-3.5 h-3.5" /> Listed
    </span>
  );
  return (
    <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-muted text-muted-foreground">
      <Shield className="w-3.5 h-3.5" /> Unknown
    </span>
  );
}

function TypeIcon({ type }) {
  if (type === 'ip') return <Server className="w-4 h-4 text-blue-500" />;
  return <Globe className="w-4 h-4 text-purple-500" />;
}

function KpiCard({ label, value, icon: Icon, color, sub }) {
  return (
    <div className="bg-card border rounded-xl p-5 flex flex-col gap-1 shadow-sm">
      <div className="flex items-center justify-between">
        <span className="text-xs font-medium text-muted-foreground uppercase tracking-wide">{label}</span>
        <Icon className={cn('w-4 h-4', color)} />
      </div>
      <div className="text-3xl font-bold tabular-nums">{value ?? 0}</div>
      {sub && <div className="text-xs text-muted-foreground">{sub}</div>}
    </div>
  );
}

function fmt(ts) {
  if (!ts) return '—';
  const d = new Date(ts);
  if (isNaN(d.getTime()) || d.getFullYear() < 2000) return '—';
  return d.toLocaleDateString('en-GB', { month: 'short', day: '2-digit' }) + ' ' +
    d.toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit' });
}

export default function ReputationPage() {
  const [rows, setRows]           = useState([]);
  const [loading, setLoading]     = useState(false);
  const [running, setRunning]     = useState(false);
  const [filter, setFilter]       = useState('all');   // all | ip | domain
  const [statusFilter, setStatus] = useState('all');   // all | clean | listed
  const [lastUpdated, setLastUpdated] = useState(null);
  const [delistUrls, setDelistUrls] = useState(DELIST_FALLBACK);
  const pollRef = useRef(null);

  const fetchRows = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/reputation', { headers: hdrs() });
      if (res.status === 401) { window.location.href = '/login'; return; }
      const data = await res.json();
      setRows(Array.isArray(data) ? data : []);
      setLastUpdated(new Date());
    } catch (_) {}
    setLoading(false);
  }, []);

  const pollStatus = useCallback(async () => {
    try {
      const res = await fetch('/api/reputation/status', { headers: hdrs() });
      if (res.status === 401) { window.location.href = '/login'; return; }
      const { running: r } = await res.json();
      setRunning(r);
      if (!r) {
        clearInterval(pollRef.current);
        pollRef.current = null;
        fetchRows();
      }
    } catch (_) {}
  }, [fetchRows]);

  const runCheck = async () => {
    if (running) return;
    const res = await fetch('/api/reputation/check', { method: 'POST', headers: hdrs() });
    if (res.status === 401) { window.location.href = '/login'; return; }
    const data = await res.json();
    if (data.status === 'started' || data.status === 'already_running') {
      setRunning(true);
      if (!pollRef.current) {
        pollRef.current = setInterval(pollStatus, 3000);
      }
    }
  };

  useEffect(() => {
    // Fetch delist URLs from API
    fetch('/api/reputation/delist-urls', { headers: hdrs() })
      .then(r => r.ok ? r.json() : {})
      .then(d => { if (d && typeof d === 'object') setDelistUrls(prev => ({ ...prev, ...d })); })
      .catch(() => {});

    // Fetch cached data first, then auto-trigger a fresh DNS check if data is stale (>6h)
    fetch('/api/reputation', { headers: hdrs() })
      .then(r => { if (r.status === 401) { window.location.href = '/login'; return; } return r.json(); })
      .then(data => {
        const arr = Array.isArray(data) ? data : [];
        setRows(arr);
        setLastUpdated(new Date());

        // Auto-run a fresh check if no data, or oldest checked_at > 6 hours ago
        const SIX_HOURS = 6 * 60 * 60 * 1000;
        const oldest = arr.reduce((min, r) => {
          const t = r.checked_at && new Date(r.checked_at).getFullYear() >= 2000 ? new Date(r.checked_at).getTime() : 0;
          return t < min ? t : min;
        }, Date.now());
        if (arr.length === 0 || (Date.now() - oldest) > SIX_HOURS) {
          fetch('/api/reputation/check', { method: 'POST', headers: hdrs() })
            .then(r => { if (r.status === 401) { window.location.href = '/login'; return; } return r.json(); })
            .then(({ status }) => {
              if (status === 'started' || status === 'already_running') {
                setRunning(true);
                if (!pollRef.current) {
                  pollRef.current = setInterval(pollStatus, 3000);
                }
              }
            })
            .catch(() => {});
        }
      })
      .catch(() => {});

    // Also check if a scan is already running from a previous trigger
    fetch('/api/reputation/status', { headers: hdrs() })
      .then(r => { if (r.status === 401) { window.location.href = '/login'; return; } return r.json(); })
      .then(({ running: r }) => {
        if (r) {
          setRunning(true);
          if (!pollRef.current) pollRef.current = setInterval(pollStatus, 3000);
        }
      })
      .catch(() => {});
    return () => clearInterval(pollRef.current);
  }, [fetchRows, pollStatus]);

  const [ispSnaps, setIspSnaps] = useState([]);
  useEffect(() => {
    fetch('/api/isp-intel/snapshots/latest?domain=', { headers: hdrs() })
      .then(r => { if (r.status === 401) { window.location.href = '/login'; return []; } return r.ok ? r.json() : []; })
      .then(d => setIspSnaps(Array.isArray(d) ? d : []))
      .catch(() => {});
  }, []);

  const visible = rows.filter(r => {
    if (filter !== 'all' && r.target_type !== filter) return false;
    if (statusFilter !== 'all' && r.status !== statusFilter) return false;
    return true;
  });

  const listedCount = rows.filter(r => r.status === 'listed').length;
  const cleanCount  = rows.filter(r => r.status === 'clean').length;
  const ipCount     = rows.filter(r => r.target_type === 'ip').length;
  const domainCount = rows.filter(r => r.target_type === 'domain').length;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Reputation Monitor</h1>
          <div className="flex items-center gap-2 mt-1">
            <p className="text-muted-foreground text-sm">
              Blacklist checks for all sending IPs and domains.
            </p>
            {rows.length > 0 && (() => {
              const oldest = rows.reduce((min, r) => {
                const t = r.checked_at && new Date(r.checked_at).getFullYear() >= 2000 ? new Date(r.checked_at).getTime() : 0;
                return t < min ? t : min;
              }, Date.now());
              const ageH = Math.round((Date.now() - oldest) / 3600000);
              return (
                <span className={cn('text-xs ml-1', ageH >= 6 ? 'text-amber-500' : 'text-muted-foreground opacity-60')}>
                  · DNS check {ageH < 1 ? 'just now' : `${ageH}h ago`}
                  {ageH >= 6 && ' — refreshing...'}
                </span>
              );
            })()}
          </div>
        </div>
        <button
          onClick={runCheck}
          disabled={running}
          className={cn(
            'flex items-center gap-2 h-9 px-4 rounded-md text-sm font-medium transition-colors',
            running
              ? 'bg-muted text-muted-foreground cursor-not-allowed'
              : 'bg-primary text-primary-foreground hover:bg-primary/90'
          )}
        >
          {running
            ? <><Loader2 className="w-4 h-4 animate-spin" /> Scanning...</>
            : <><RefreshCw className="w-4 h-4" /> Run Check Now</>}
        </button>
      </div>

      {/* KPI strip */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <KpiCard label="Total Targets"   value={rows.length}  icon={Shield}       color="text-blue-500"   sub={`${ipCount} IPs · ${domainCount} domains`} />
        <KpiCard label="Clean"           value={cleanCount}   icon={ShieldCheck}  color="text-green-500"  sub="Not on any blacklist" />
        <KpiCard label="Blacklisted"     value={listedCount}  icon={ShieldAlert}  color="text-red-500"    sub="Require immediate action" />
        <KpiCard label="Never Checked"   value={rows.filter(r => !r.checked_at).length} icon={Clock} color="text-muted-foreground" sub="Run a scan to populate" />
      </div>

      {/* Listed alert banner */}
      {listedCount > 0 && (
        <div className="flex items-start gap-3 rounded-xl border border-red-300 dark:border-red-800 bg-red-50 dark:bg-red-900/20 p-4">
          <AlertTriangle className="w-5 h-5 text-red-500 shrink-0 mt-0.5" />
          <div>
            <p className="text-sm font-semibold text-red-700 dark:text-red-400">
              {listedCount} target{listedCount > 1 ? 's are' : ' is'} listed on a blacklist.
            </p>
            <p className="text-xs text-red-600 dark:text-red-500 mt-0.5">
              Listings will cause mail to be rejected or junked. Use the delist links below to submit removal requests.
            </p>
          </div>
        </div>
      )}

      {/* Filters */}
      <div className="flex flex-wrap gap-2">
        {[
          { id: 'all',    label: 'All' },
          { id: 'ip',     label: 'IPs only' },
          { id: 'domain', label: 'Domains only' },
        ].map(f => (
          <button key={f.id} onClick={() => setFilter(f.id)}
            className={cn('px-3 py-1.5 rounded-md text-xs font-medium border transition-colors',
              filter === f.id
                ? 'bg-primary text-primary-foreground border-primary'
                : 'bg-background border-border text-muted-foreground hover:bg-accent')}>
            {f.label}
          </button>
        ))}
        <div className="w-px bg-border mx-1" />
        {[
          { id: 'all',    label: 'Any status' },
          { id: 'clean',  label: 'Clean' },
          { id: 'listed', label: 'Listed' },
        ].map(f => (
          <button key={f.id} onClick={() => setStatus(f.id)}
            className={cn('px-3 py-1.5 rounded-md text-xs font-medium border transition-colors',
              statusFilter === f.id
                ? 'bg-primary text-primary-foreground border-primary'
                : 'bg-background border-border text-muted-foreground hover:bg-accent')}>
            {f.label}
          </button>
        ))}
      </div>

      {/* Table */}
      <div className="bg-card border rounded-xl shadow-sm overflow-hidden">
        <div className="overflow-x-auto">
          {rows.length === 0 && !loading ? (
            <div className="flex flex-col items-center justify-center py-16 text-muted-foreground gap-3">
              <Shield className="w-12 h-12 opacity-20" />
              <p className="text-sm font-medium">No reputation data yet.</p>
              <p className="text-xs">Click <strong>Run Check Now</strong> to scan your IPs and domains.</p>
            </div>
          ) : (
            <table className="w-full text-sm text-left">
              <thead className="bg-muted/50 text-muted-foreground uppercase text-xs">
                <tr>
                  <th className="px-5 py-3 font-medium">Type</th>
                  <th className="px-5 py-3 font-medium">Target</th>
                  <th className="px-5 py-3 font-medium">Status</th>
                  <th className="px-5 py-3 font-medium">Listed On</th>
                  <th className="px-5 py-3 font-medium">Delist</th>
                  <th className="px-5 py-3 font-medium">Last Checked</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {loading && rows.length === 0 ? (
                  <tr>
                    <td colSpan="6" className="p-8 text-center text-muted-foreground">
                      <Loader2 className="w-5 h-5 animate-spin mx-auto" />
                    </td>
                  </tr>
                ) : visible.length === 0 ? (
                  <tr>
                    <td colSpan="6" className="p-8 text-center text-muted-foreground">No results for selected filter.</td>
                  </tr>
                ) : visible.map(row => (
                  <tr key={row.id} className={cn(
                    'hover:bg-muted/40 transition-colors',
                    row.status === 'listed' && 'bg-red-50/40 dark:bg-red-900/10'
                  )}>
                    <td className="px-5 py-4">
                      <div className="flex items-center gap-2">
                        <TypeIcon type={row.target_type} />
                        <span className="text-xs text-muted-foreground capitalize">{row.target_type}</span>
                      </div>
                    </td>
                    <td className="px-5 py-4 font-mono font-medium">{row.target}</td>
                    <td className="px-5 py-4"><StatusBadge status={row.status} /></td>
                    <td className="px-5 py-4">
                      {row.listed_on ? (
                        <div className="flex flex-wrap gap-1">
                          {row.listed_on.split(',').map(rbl => (
                            <span key={rbl} className="px-1.5 py-0.5 rounded bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 text-xs font-mono">
                              {rbl}
                            </span>
                          ))}
                        </div>
                      ) : (
                        <span className="text-muted-foreground text-xs">—</span>
                      )}
                    </td>
                    <td className="px-5 py-4">
                      {row.listed_on ? (
                        <div className="flex flex-wrap gap-1">
                          {[...new Set(row.listed_on.split(','))].map(rbl => {
                            const url = delistUrls[rbl];
                            const note = DELIST_NOTES[rbl];
                            if (!url) return null;
                            return (
                              <a key={rbl} href={url} target="_blank" rel="noopener noreferrer"
                                title={note || `Delist from ${rbl}`}
                                className="inline-flex items-center gap-1 px-2 py-1 rounded text-xs font-medium
                                  bg-amber-100 dark:bg-amber-900/30 text-amber-800 dark:text-amber-300
                                  hover:bg-amber-200 dark:hover:bg-amber-900/50 transition-colors">
                                <ExternalLink className="w-3 h-3" />
                                {rbl.split('.')[0]}
                              </a>
                            );
                          })}
                        </div>
                      ) : (
                        <span className="text-muted-foreground text-xs">—</span>
                      )}
                    </td>
                    <td className="px-5 py-4 text-muted-foreground text-xs">
                      <div className="flex items-center gap-1.5">
                        <Clock className="w-3.5 h-3.5" />
                        {fmt(row.checked_at)}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>

      {/* Google Postmaster / ISP Sender Reputation */}
      {ispSnaps.length > 0 && (
        <div className="bg-card border rounded-xl shadow-sm overflow-hidden">
          <div className="px-5 py-3 border-b bg-muted/30">
            <h3 className="text-sm font-semibold">Google Postmaster Domain Reputation</h3>
            <p className="text-xs text-muted-foreground mt-0.5">Live sender reputation data from Google Postmaster Tools via ISP Intel.</p>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-sm text-left">
              <thead className="bg-muted/50 text-muted-foreground uppercase text-xs">
                <tr>
                  {['Domain', 'ISP', 'Domain Rep', 'IP Rep', 'Spam Rate', 'Delivery Error', 'Last Seen'].map(h => (
                    <th key={h} className="px-4 py-2.5 font-medium">{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {ispSnaps.map(snap => {
                  const repColor = (r) => {
                    if (!r || r === 'UNKNOWN') return 'text-muted-foreground';
                    if (r === 'HIGH') return 'text-green-600 dark:text-green-400 font-semibold';
                    if (r === 'MEDIUM') return 'text-yellow-600 dark:text-yellow-400 font-semibold';
                    if (r === 'LOW') return 'text-orange-600 dark:text-orange-400 font-semibold';
                    return 'text-red-600 dark:text-red-400 font-bold'; // BAD
                  };
                  return (
                    <tr key={snap.id} className="hover:bg-muted/40 transition-colors">
                      <td className="px-4 py-3 font-mono text-xs">{snap.domain}</td>
                      <td className="px-4 py-3 text-xs">{snap.isp}</td>
                      <td className={cn('px-4 py-3 text-xs', repColor(snap.gpt_domain_reputation))}>
                        {snap.gpt_domain_reputation || '—'}
                      </td>
                      <td className={cn('px-4 py-3 text-xs', repColor(snap.gpt_ip_reputation))}>
                        {snap.gpt_ip_reputation || '—'}
                      </td>
                      <td className="px-4 py-3 text-xs">
                        {snap.gpt_spam_rate != null ? `${(snap.gpt_spam_rate * 100).toFixed(3)}%` : '—'}
                      </td>
                      <td className="px-4 py-3 text-xs">
                        {snap.gpt_delivery_error_rate != null ? `${(snap.gpt_delivery_error_rate * 100).toFixed(2)}%` : '—'}
                      </td>
                      <td className="px-4 py-3 text-xs text-muted-foreground">{fmt(snap.captured_at)}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* RBL reference with delist links */}
      <div className="bg-card border rounded-xl p-5 shadow-sm">
        <h3 className="text-sm font-semibold mb-3">Blacklists Checked</h3>
        <div className="grid sm:grid-cols-2 gap-4">
          <div>
            <p className="text-xs text-muted-foreground font-medium mb-2 uppercase tracking-wide">IP Blacklists (RBL)</p>
            <ul className="space-y-1.5 text-xs font-mono text-muted-foreground">
              {[
                { zone: 'zen.spamhaus.org', note: 'SBL + XBL + PBL + CSS' },
                { zone: 'b.barracudacentral.org', note: 'Barracuda BRBL' },
                { zone: 'bl.spamcop.net', note: 'Auto-delists 24h' },
                { zone: 'truncate.gbudb.net', note: 'Auto only, no manual delist' },
                { zone: 'dnsbl-1.uceprotect.net', note: 'Auto-delists 7 days' },
                { zone: 'psbl.surriel.com', note: 'Instant free removal' },
              ].map(r => (
                <li key={r.zone} className="flex items-center gap-1.5">
                  <CheckCircle2 className="w-3 h-3 text-green-500 shrink-0" />
                  <span>{r.zone}</span>
                  {delistUrls[r.zone] && (
                    <a href={delistUrls[r.zone]} target="_blank" rel="noopener noreferrer"
                      className="ml-auto text-primary hover:underline flex items-center gap-0.5 font-sans">
                      delist <ExternalLink className="w-2.5 h-2.5" />
                    </a>
                  )}
                </li>
              ))}
            </ul>
          </div>
          <div>
            <p className="text-xs text-muted-foreground font-medium mb-2 uppercase tracking-wide">Domain Blacklists (DBL)</p>
            <ul className="space-y-1.5 text-xs font-mono text-muted-foreground">
              {[
                { zone: 'dbl.spamhaus.org', note: 'Spamhaus DBL' },
                { zone: 'multi.surbl.org', note: 'SURBL combined' },
                { zone: 'black.uribl.com', note: 'Requires account' },
                { zone: 'dbl.nordspam.com', note: 'Email-based delist' },
              ].map(r => (
                <li key={r.zone} className="flex items-center gap-1.5">
                  <CheckCircle2 className="w-3 h-3 text-purple-500 shrink-0" />
                  <span>{r.zone}</span>
                  {delistUrls[r.zone] && (
                    <a href={delistUrls[r.zone]} target="_blank" rel="noopener noreferrer"
                      className="ml-auto text-primary hover:underline flex items-center gap-0.5 font-sans">
                      delist <ExternalLink className="w-2.5 h-2.5" />
                    </a>
                  )}
                </li>
              ))}
            </ul>
          </div>
        </div>
      </div>
    </div>
  );
}
