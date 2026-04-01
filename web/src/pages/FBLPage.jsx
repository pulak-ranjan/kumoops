import React, { useState, useEffect, useCallback } from 'react';
import {
  AlertTriangle, ShieldAlert, RefreshCw, Trash2, Upload,
  TrendingUp, Mail, Ban, CheckCircle2, Clock, Filter, ChevronDown
} from 'lucide-react';
import { cn } from '../lib/utils';

// ─── ISP complaint rate thresholds (%) ───────────────────────────────────────
const ISP_THRESHOLDS = {
  'gmail.com': 0.08,
  'yahoo.com': 0.10,
  'hotmail.com': 0.30,
  'outlook.com': 0.30,
};

const FEEDBACK_COLORS = {
  abuse:       { bg: 'bg-red-500',    text: 'text-red-600 dark:text-red-400',     label: 'Abuse' },
  fraud:       { bg: 'bg-orange-500', text: 'text-orange-600 dark:text-orange-400', label: 'Fraud' },
  virus:       { bg: 'bg-purple-500', text: 'text-purple-600 dark:text-purple-400', label: 'Virus' },
  unsubscribe: { bg: 'bg-yellow-500', text: 'text-yellow-600 dark:text-yellow-400', label: 'Unsub' },
  other:       { bg: 'bg-gray-500',   text: 'text-gray-500 dark:text-gray-400',    label: 'Other' },
};

const BOUNCE_CATEGORY_META = {
  hard:    { label: 'Hard Bounce',   color: 'bg-red-500',    textColor: 'text-red-600 dark:text-red-400' },
  soft:    { label: 'Soft Bounce',   color: 'bg-amber-500',  textColor: 'text-amber-600 dark:text-amber-400' },
  block:   { label: 'Policy Block',  color: 'bg-orange-500', textColor: 'text-orange-600 dark:text-orange-400' },
  quota:   { label: 'Mailbox Full',  color: 'bg-purple-500', textColor: 'text-purple-600 dark:text-purple-400' },
  dns:     { label: 'DNS Failure',   color: 'bg-gray-500',   textColor: 'text-gray-500' },
  tls:     { label: 'TLS Failure',   color: 'bg-blue-500',   textColor: 'text-blue-600 dark:text-blue-400' },
  auth:    { label: 'Auth Failure',  color: 'bg-rose-500',   textColor: 'text-rose-600 dark:text-rose-400' },
  unknown: { label: 'Unknown',       color: 'bg-slate-400',  textColor: 'text-slate-500' },
};

function StatCard({ title, value, sub, icon: Icon, colorClass = 'text-foreground', warn }) {
  return (
    <div className={cn(
      'rounded-xl border bg-card p-4 flex flex-col gap-1',
      warn && 'border-red-500/50 bg-red-500/5'
    )}>
      <div className="flex items-center justify-between">
        <span className="text-xs font-medium text-muted-foreground">{title}</span>
        {Icon && <Icon className={cn('w-4 h-4', colorClass)} />}
      </div>
      <div className={cn('text-2xl font-bold', warn ? 'text-red-600 dark:text-red-400' : colorClass)}>{value}</div>
      {sub && <div className="text-xs text-muted-foreground">{sub}</div>}
    </div>
  );
}

function Badge({ type }) {
  const meta = FEEDBACK_COLORS[type] || FEEDBACK_COLORS.other;
  return (
    <span className={cn('inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium text-white', meta.bg)}>
      {meta.label}
    </span>
  );
}

function BounceBadge({ category }) {
  const meta = BOUNCE_CATEGORY_META[category] || BOUNCE_CATEGORY_META.unknown;
  return (
    <span className={cn('inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium text-white', meta.color)}>
      {meta.label}
    </span>
  );
}

function SuppressedDot({ suppressed }) {
  return suppressed
    ? <span title="Auto-suppressed" className="inline-block w-2 h-2 rounded-full bg-green-500 ml-1" />
    : <span title="Not suppressed" className="inline-block w-2 h-2 rounded-full bg-gray-300 ml-1" />;
}

export default function FBLPage() {
  const [tab, setTab] = useState('complaints');
  const [days, setDays] = useState(30);
  const [loading, setLoading] = useState(true);

  // FBL state
  const [records, setRecords] = useState([]);
  const [stats, setStats] = useState([]);
  // DSN state
  const [bounces, setBounces] = useState([]);
  const [bounceSummary, setBounceSummary] = useState([]);
  // Upload state
  const [uploading, setUploading] = useState(false);
  const [uploadMsg, setUploadMsg] = useState('');

  const token = localStorage.getItem('kumoui_token');
  const headers = { Authorization: `Bearer ${token}` };

  const fetchAll = useCallback(async () => {
    setLoading(true);
    try {
      const [recRes, statsRes, bncRes, bncSumRes] = await Promise.all([
        fetch(`/api/fbl?days=${days}&limit=200`, { headers }),
        fetch(`/api/fbl/stats?days=${days}`, { headers }),
        fetch(`/api/fbl/bounces?days=${days}&limit=200`, { headers }),
        fetch(`/api/fbl/bounces/summary?days=${days}`, { headers }),
      ]);
      if ([recRes, statsRes, bncRes, bncSumRes].some(r => r.status === 401)) { window.location.href = '/login'; return; }
      if (recRes.ok)    { const d = await recRes.json();    setRecords(Array.isArray(d) ? d : []); }
      if (statsRes.ok)  { const d = await statsRes.json();  setStats(Array.isArray(d) ? d : []); }
      if (bncRes.ok)    { const d = await bncRes.json();    setBounces(Array.isArray(d) ? d : []); }
      if (bncSumRes.ok) { const d = await bncSumRes.json(); setBounceSummary(Array.isArray(d) ? d : []); }
    } catch (e) { console.error(e); }
    setLoading(false);
  }, [days]);

  useEffect(() => { fetchAll(); }, [fetchAll]);

  const deleteRecord = async (id) => {
    if (!confirm('Delete this FBL record?')) return;
    try {
      const res = await fetch(`/api/fbl/${id}`, { method: 'DELETE', headers });
      if (res.status === 401) { window.location.href = '/login'; return; }
      setRecords(r => r.filter(x => x.id !== id));
    } catch (e) { console.error(e); }
  };

  const uploadFile = async (e, type) => {
    const file = e.target.files[0];
    if (!file) return;
    setUploading(true);
    setUploadMsg('');
    const form = new FormData();
    form.append('email', file);
    const endpoint = type === 'fbl' ? '/api/fbl/upload' : '/api/fbl/upload-dsn';
    try {
      const res = await fetch(endpoint, { method: 'POST', headers, body: form });
      if (res.status === 401) { window.location.href = '/login'; return; }
      const data = await res.json();
      if (res.ok) {
        setUploadMsg(`✓ Processed successfully. ID: ${data.id}`);
        fetchAll();
      } else {
        setUploadMsg(`✗ ${data.error}`);
      }
    } catch (err) {
      setUploadMsg('✗ Upload failed: ' + err.message);
    }
    setUploading(false);
    e.target.value = '';
  };

  const totalComplaints = records.length;
  const autoSuppressed = records.filter(r => r.auto_suppressed).length;
  const abuseComplaints = records.filter(r => r.feedback_type === 'abuse').length;
  const hardBounces = bounces.filter(b => b.is_hard).length;
  const totalBounces = bounces.length;
  const bounceSuppressed = bounces.filter(b => b.auto_suppressed).length;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight flex items-center gap-2">
            <ShieldAlert className="w-7 h-7 text-red-500" />
            FBL &amp; Bounce Engine
          </h1>
          <p className="text-muted-foreground text-sm mt-1">
            Feedback Loop complaints (ARF/RFC 5965) and DSN bounce classification (RFC 3464).
            Hard bounces and complaints are auto-suppressed.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <select
            value={days}
            onChange={e => setDays(+e.target.value)}
            className="h-9 rounded-md border bg-background px-3 text-sm"
          >
            {[7,14,30,60,90].map(d => (
              <option key={d} value={d}>Last {d} days</option>
            ))}
          </select>
          <button
            onClick={fetchAll}
            disabled={loading}
            className="h-9 px-3 rounded-md border bg-background text-sm flex items-center gap-1.5 hover:bg-muted transition-colors"
          >
            <RefreshCw className={cn('w-3.5 h-3.5', loading && 'animate-spin')} />
            Refresh
          </button>
        </div>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <StatCard title="FBL Complaints" value={totalComplaints} icon={AlertTriangle}
          colorClass="text-red-500" warn={abuseComplaints > 0}
          sub={`${abuseComplaints} abuse reports`} />
        <StatCard title="Auto-Suppressed" value={autoSuppressed + bounceSuppressed} icon={Ban}
          colorClass="text-green-500"
          sub="via FBL + hard bounce" />
        <StatCard title="Classified Bounces" value={totalBounces} icon={Mail}
          colorClass="text-amber-500"
          sub={`${hardBounces} hard bounces`} />
        <StatCard title="Bounce Categories" value={bounceSummary.length} icon={TrendingUp}
          colorClass="text-blue-500"
          sub="distinct classifications" />
      </div>

      {/* Bounce Category Breakdown */}
      {bounceSummary.length > 0 && (
        <div className="rounded-xl border bg-card p-4">
          <h2 className="text-sm font-semibold mb-3">Bounce Classification Breakdown</h2>
          <div className="flex flex-wrap gap-3">
            {bounceSummary.map(s => {
              const meta = BOUNCE_CATEGORY_META[s.category] || BOUNCE_CATEGORY_META.unknown;
              return (
                <div key={s.category} className="flex items-center gap-2 text-sm">
                  <span className={cn('w-3 h-3 rounded-full', meta.color)} />
                  <span className={meta.textColor}>{meta.label}</span>
                  <span className="font-bold">{s.count}</span>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Tabs */}
      <div className="flex gap-1 border-b">
        {[
          { id: 'complaints', label: 'FBL Complaints' },
          { id: 'bounces',    label: 'DSN Bounces' },
          { id: 'rates',      label: 'Complaint Rates' },
          { id: 'upload',     label: 'Manual Upload' },
        ].map(t => (
          <button
            key={t.id}
            onClick={() => setTab(t.id)}
            className={cn(
              'px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors',
              tab === t.id
                ? 'border-primary text-primary'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            )}
          >
            {t.label}
          </button>
        ))}
      </div>

      {/* ── FBL Complaints Tab ── */}
      {tab === 'complaints' && (
        <div className="rounded-xl border bg-card overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-muted/50 border-b">
              <tr>
                {['Received', 'Type', 'Recipient', 'From Sender', 'ISP', 'Suppressed', ''].map(h => (
                  <th key={h} className="text-left px-4 py-2.5 text-xs font-medium text-muted-foreground">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {loading && (
                <tr><td colSpan={7} className="px-4 py-8 text-center text-muted-foreground">Loading…</td></tr>
              )}
              {!loading && records.length === 0 && (
                <tr>
                  <td colSpan={7} className="px-4 py-12 text-center">
                    <CheckCircle2 className="w-8 h-8 text-green-500 mx-auto mb-2" />
                    <p className="text-muted-foreground text-sm">No FBL complaints in the last {days} days.</p>
                    <p className="text-xs text-muted-foreground mt-1">
                      Make sure bounce accounts are configured and FBL mailboxes are set up with ISPs.
                    </p>
                  </td>
                </tr>
              )}
              {records.map(r => (
                <tr key={r.id} className="hover:bg-muted/30 transition-colors">
                  <td className="px-4 py-2.5 text-xs text-muted-foreground whitespace-nowrap">
                    {new Date(r.received_at).toLocaleString()}
                  </td>
                  <td className="px-4 py-2.5"><Badge type={r.feedback_type} /></td>
                  <td className="px-4 py-2.5 font-mono text-xs">{r.original_rcpt_to || '—'}</td>
                  <td className="px-4 py-2.5 text-xs">{r.original_sender || '—'}</td>
                  <td className="px-4 py-2.5 text-xs text-muted-foreground">{r.reporting_mta || '—'}</td>
                  <td className="px-4 py-2.5">
                    <SuppressedDot suppressed={r.auto_suppressed} />
                  </td>
                  <td className="px-4 py-2.5">
                    <button
                      onClick={() => deleteRecord(r.id)}
                      className="text-muted-foreground hover:text-red-500 transition-colors"
                      title="Delete record"
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* ── DSN Bounces Tab ── */}
      {tab === 'bounces' && (
        <div className="rounded-xl border bg-card overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-muted/50 border-b">
              <tr>
                {['Received', 'Category', 'Recipient', 'Status', 'Diagnostic', 'Provider', 'VERP', 'Suppressed'].map(h => (
                  <th key={h} className="text-left px-4 py-2.5 text-xs font-medium text-muted-foreground">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {loading && (
                <tr><td colSpan={8} className="px-4 py-8 text-center text-muted-foreground">Loading…</td></tr>
              )}
              {!loading && bounces.length === 0 && (
                <tr>
                  <td colSpan={8} className="px-4 py-12 text-center">
                    <CheckCircle2 className="w-8 h-8 text-green-500 mx-auto mb-2" />
                    <p className="text-muted-foreground text-sm">No classified bounces yet.</p>
                    <p className="text-xs text-muted-foreground mt-1">
                      Bounces appear here once DSN messages are received and processed from bounce mailboxes.
                    </p>
                  </td>
                </tr>
              )}
              {bounces.map(b => (
                <tr key={b.id} className="hover:bg-muted/30 transition-colors">
                  <td className="px-4 py-2.5 text-xs text-muted-foreground whitespace-nowrap">
                    {new Date(b.received_at).toLocaleString()}
                  </td>
                  <td className="px-4 py-2.5"><BounceBadge category={b.category} /></td>
                  <td className="px-4 py-2.5 font-mono text-xs max-w-[200px] truncate">{b.final_recipient || '—'}</td>
                  <td className="px-4 py-2.5 font-mono text-xs text-muted-foreground">{b.enhanced_status || '—'}</td>
                  <td className="px-4 py-2.5 text-xs text-muted-foreground max-w-[220px] truncate" title={b.diagnostic_code}>
                    {b.diagnostic_code || '—'}
                  </td>
                  <td className="px-4 py-2.5 text-xs">{b.provider || '—'}</td>
                  <td className="px-4 py-2.5">
                    {b.verp_decoded
                      ? <span className="text-xs text-green-600 dark:text-green-400 font-medium">VERP ✓</span>
                      : <span className="text-xs text-muted-foreground">—</span>}
                  </td>
                  <td className="px-4 py-2.5">
                    <SuppressedDot suppressed={b.auto_suppressed} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* ── Complaint Rates Tab ── */}
      {tab === 'rates' && (
        <div className="space-y-4">
          <p className="text-sm text-muted-foreground">
            ISP complaint rate thresholds: Gmail &lt;0.08%, Yahoo &lt;0.10%, Outlook &lt;0.30%.
            Exceeding these leads to filtering or blocking.
          </p>
          {stats.length === 0 && !loading && (
            <div className="rounded-xl border bg-card p-8 text-center text-muted-foreground text-sm">
              No complaint rate data available yet.
            </div>
          )}
          <div className="grid gap-4">
            {stats.map((s, i) => {
              const threshold = ISP_THRESHOLDS[s.domain] ?? 0.10;
              const rate = s.total_complaints; // raw count — show as absolute since we don't have sent count here
              const warn = s.total_complaints > 5;
              return (
                <div key={i} className={cn(
                  'rounded-xl border bg-card p-4',
                  warn && 'border-amber-500/50'
                )}>
                  <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
                    <div>
                      <div className="font-medium text-sm">{s.sender_email || s.domain}</div>
                      <div className="text-xs text-muted-foreground">{s.domain}</div>
                    </div>
                    <div className="flex items-center gap-4 text-sm">
                      <div className="text-center">
                        <div className="font-bold text-red-500">{s.total_complaints}</div>
                        <div className="text-xs text-muted-foreground">Total</div>
                      </div>
                      <div className="text-center">
                        <div className="font-bold text-orange-500">{s.abuse_complaints}</div>
                        <div className="text-xs text-muted-foreground">Abuse</div>
                      </div>
                      <div className="text-center">
                        <div className="font-bold text-yellow-500">{s.unsub_complaints}</div>
                        <div className="text-xs text-muted-foreground">Unsub</div>
                      </div>
                      <div className="text-center">
                        <div className="font-bold text-green-500">{s.auto_suppressed}</div>
                        <div className="text-xs text-muted-foreground">Suppressed</div>
                      </div>
                    </div>
                  </div>
                  {warn && (
                    <div className="mt-2 flex items-center gap-1.5 text-xs text-amber-600 dark:text-amber-400">
                      <AlertTriangle className="w-3.5 h-3.5" />
                      Complaint volume is elevated. Review sending practices for this sender.
                    </div>
                  )}
                  {s.last_seen && (
                    <div className="mt-1 text-xs text-muted-foreground flex items-center gap-1">
                      <Clock className="w-3 h-3" /> Last seen: {new Date(s.last_seen).toLocaleString()}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* ── Manual Upload Tab ── */}
      {tab === 'upload' && (
        <div className="space-y-6">
          <p className="text-sm text-muted-foreground">
            Manually upload raw .eml files to test the FBL and DSN parsers, or to import
            backlogged complaints from ISP postmaster mailboxes.
          </p>

          {uploadMsg && (
            <div className={cn(
              'rounded-lg border px-4 py-3 text-sm',
              uploadMsg.startsWith('✓') ? 'border-green-500/50 bg-green-500/5 text-green-700 dark:text-green-400'
                : 'border-red-500/50 bg-red-500/5 text-red-700 dark:text-red-400'
            )}>
              {uploadMsg}
            </div>
          )}

          <div className="grid md:grid-cols-2 gap-6">
            {/* FBL Upload */}
            <div className="rounded-xl border bg-card p-5 space-y-3">
              <div className="flex items-center gap-2">
                <ShieldAlert className="w-5 h-5 text-red-500" />
                <h3 className="font-semibold">Upload FBL Complaint</h3>
              </div>
              <p className="text-xs text-muted-foreground">
                Upload a raw ARF/RFC 5965 complaint email (.eml).
                The recipient will be auto-suppressed if found.
              </p>
              <label className={cn(
                'flex items-center gap-2 cursor-pointer px-4 py-2.5 rounded-lg border-2 border-dashed',
                'text-sm text-muted-foreground hover:border-primary hover:text-primary transition-colors',
                uploading && 'opacity-50 cursor-not-allowed'
              )}>
                <Upload className="w-4 h-4" />
                {uploading ? 'Processing…' : 'Select .eml file'}
                <input
                  type="file" accept=".eml,message/rfc822,text/plain" className="hidden"
                  disabled={uploading}
                  onChange={e => uploadFile(e, 'fbl')}
                />
              </label>
            </div>

            {/* DSN Upload */}
            <div className="rounded-xl border bg-card p-5 space-y-3">
              <div className="flex items-center gap-2">
                <Mail className="w-5 h-5 text-amber-500" />
                <h3 className="font-semibold">Upload DSN Bounce</h3>
              </div>
              <p className="text-xs text-muted-foreground">
                Upload a raw DSN/RFC 3464 bounce notification (.eml).
                Hard bounces will be auto-suppressed. VERP decoding is applied if detected.
              </p>
              <label className={cn(
                'flex items-center gap-2 cursor-pointer px-4 py-2.5 rounded-lg border-2 border-dashed',
                'text-sm text-muted-foreground hover:border-primary hover:text-primary transition-colors',
                uploading && 'opacity-50 cursor-not-allowed'
              )}>
                <Upload className="w-4 h-4" />
                {uploading ? 'Processing…' : 'Select .eml file'}
                <input
                  type="file" accept=".eml,message/rfc822,text/plain" className="hidden"
                  disabled={uploading}
                  onChange={e => uploadFile(e, 'dsn')}
                />
              </label>
            </div>
          </div>

          <div className="rounded-xl border bg-card p-4 space-y-2">
            <h3 className="text-sm font-semibold">How FBL Processing Works</h3>
            <ol className="text-xs text-muted-foreground space-y-1 list-decimal list-inside">
              <li>Register with ISP Feedback Loop programs (Gmail, Yahoo, AOL, Outlook).</li>
              <li>Configure the FBL report destination email to go to a bounce mailbox (created in Bounce Accounts).</li>
              <li>KumoOps scans the Maildir every 60 seconds for new messages.</li>
              <li>ARF/FBL reports are parsed: recipient extracted, complainant suppressed automatically.</li>
              <li>DSN bounces are classified (hard/soft/block/quota/dns/tls/auth) and stored.</li>
              <li>Hard bounce recipients are auto-suppressed from all future sends.</li>
              <li>VERP-encoded return paths are decoded to match bounces to exact recipients.</li>
            </ol>
          </div>
        </div>
      )}
    </div>
  );
}
