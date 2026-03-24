import React, { useState, useEffect, useCallback } from 'react';
import {
  Inbox, Plus, Trash2, RefreshCw, Play, CheckCircle2,
  XCircle, AlertTriangle, Clock, Mail, TestTube2, ChevronDown, ChevronRight
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

function PlacementBadge({ placement }) {
  if (!placement) return <span className="text-muted-foreground text-xs">—</span>;
  const map = {
    inbox:   'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-400',
    spam:    'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-400',
    missing: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/40 dark:text-yellow-400',
  };
  return (
    <span className={cn('px-2 py-0.5 rounded text-xs font-semibold capitalize', map[placement] || 'bg-muted text-muted-foreground')}>
      {placement}
    </span>
  );
}

function StatusBadge({ status }) {
  const map = {
    pending:   { cls: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/40 dark:text-yellow-400', icon: Clock },
    running:   { cls: 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-400', icon: RefreshCw },
    completed: { cls: 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-400', icon: CheckCircle2 },
    failed:    { cls: 'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-400', icon: XCircle },
  };
  const m = map[status] || { cls: 'bg-muted text-muted-foreground', icon: AlertTriangle };
  const Icon = m.icon;
  return (
    <span className={cn('inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-semibold capitalize', m.cls)}>
      <Icon className="w-3 h-3" /> {status}
    </span>
  );
}

function RateBar({ label, value, color }) {
  const pct = Math.round((value || 0) * 100);
  const barColor = color === 'green' ? 'bg-green-500' : color === 'red' ? 'bg-red-500' : 'bg-yellow-500';
  return (
    <div className="flex items-center gap-2 min-w-0">
      <span className="text-xs text-muted-foreground w-14 shrink-0">{label}</span>
      <div className="flex-1 bg-muted rounded-full h-1.5 overflow-hidden">
        <div className={cn('h-full rounded-full', barColor)} style={{ width: `${pct}%` }} />
      </div>
      <span className="text-xs font-semibold w-10 text-right">{pct}%</span>
    </div>
  );
}

// ─── Seed Mailboxes Tab ────────────────────────────────────────────────────────
function SeedMailboxesTab() {
  const [mailboxes, setMailboxes] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showAdd, setShowAdd] = useState(false);
  const [form, setForm] = useState({ isp: '', email: '', imap_host: '', imap_port: 993, username: '', password: '' });
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    try { setMailboxes((await apiFetch('/placement/seed-mailboxes')) ?? []); } catch { }
    setLoading(false);
  }, []);

  useEffect(() => { load(); }, [load]);

  const handleAdd = async () => {
    setSaving(true); setError('');
    try {
      await apiFetch('/placement/seed-mailboxes', { method: 'POST', body: JSON.stringify(form) });
      setShowAdd(false);
      setForm({ isp: '', email: '', imap_host: '', imap_port: 993, username: '', password: '' });
      load();
    } catch (e) { setError(e.message); }
    setSaving(false);
  };

  const handleDelete = async (id) => {
    if (!confirm('Delete this seed mailbox?')) return;
    try { await apiFetch(`/placement/seed-mailboxes/${id}`, { method: 'DELETE' }); load(); } catch { }
  };

  const Input = ({ label, field, type = 'text', placeholder }) => (
    <div>
      <label className="text-xs text-muted-foreground font-medium">{label}</label>
      <input type={type} value={form[field]} onChange={e => setForm(f => ({ ...f, [field]: type === 'number' ? +e.target.value : e.target.value }))}
        placeholder={placeholder}
        className="w-full mt-1 px-3 py-2 rounded-md border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary/50" />
    </div>
  );

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">Seed mailboxes are used to test where your emails land — inbox or spam.</p>
        <button onClick={() => setShowAdd(v => !v)}
          className="flex items-center gap-2 px-3 py-2 rounded-md bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90">
          <Plus className="w-4 h-4" /> Add Mailbox
        </button>
      </div>

      {showAdd && (
        <div className="border rounded-lg p-4 bg-muted/30 space-y-3">
          <h3 className="text-sm font-semibold">Add Seed Mailbox</h3>
          {error && <p className="text-xs text-destructive bg-destructive/10 px-3 py-2 rounded">{error}</p>}
          <div className="grid grid-cols-2 gap-3">
            <Input label="ISP Name" field="isp" placeholder="Gmail" />
            <Input label="Email Address" field="email" placeholder="seed@gmail.com" />
            <Input label="IMAP Host" field="imap_host" placeholder="imap.gmail.com" />
            <Input label="IMAP Port" field="imap_port" type="number" />
            <Input label="Username" field="username" placeholder="seed@gmail.com" />
            <Input label="Password / App Password" field="password" type="password" />
          </div>
          <div className="flex gap-2 justify-end">
            <button onClick={() => setShowAdd(false)} className="px-3 py-1.5 text-sm border rounded-md hover:bg-muted">Cancel</button>
            <button onClick={handleAdd} disabled={saving}
              className="px-4 py-1.5 text-sm bg-primary text-primary-foreground rounded-md hover:bg-primary/90 disabled:opacity-60">
              {saving ? 'Saving…' : 'Save'}
            </button>
          </div>
        </div>
      )}

      {loading ? (
        <div className="text-center py-8 text-muted-foreground text-sm">Loading…</div>
      ) : mailboxes.length === 0 ? (
        <div className="text-center py-12 text-muted-foreground">
          <Mail className="w-10 h-10 mx-auto mb-2 opacity-30" />
          <p className="text-sm">No seed mailboxes yet. Add one to start testing.</p>
        </div>
      ) : (
        <div className="border rounded-lg overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-muted/50">
              <tr>
                {['ISP', 'Email', 'IMAP Host', 'Port', 'Status', ''].map(h => (
                  <th key={h} className="px-4 py-2.5 text-left text-xs font-semibold text-muted-foreground uppercase">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y">
              {mailboxes.map(mb => (
                <tr key={mb.id} className="hover:bg-muted/30">
                  <td className="px-4 py-2.5 font-medium">{mb.isp}</td>
                  <td className="px-4 py-2.5 text-muted-foreground">{mb.email}</td>
                  <td className="px-4 py-2.5 text-muted-foreground">{mb.imap_host}</td>
                  <td className="px-4 py-2.5">{mb.imap_port}</td>
                  <td className="px-4 py-2.5">
                    <span className={cn('px-2 py-0.5 rounded text-xs font-semibold', mb.is_active
                      ? 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-400'
                      : 'bg-muted text-muted-foreground')}>
                      {mb.is_active ? 'Active' : 'Inactive'}
                    </span>
                  </td>
                  <td className="px-4 py-2.5 text-right">
                    <button onClick={() => handleDelete(mb.id)} className="text-destructive hover:text-destructive/80 p-1 rounded hover:bg-destructive/10">
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

// ─── Placement Tests Tab ────────────────────────────────────────────────────────
function PlacementTestsTab() {
  const [tests, setTests] = useState([]);
  const [loading, setLoading] = useState(true);
  const [expanded, setExpanded] = useState(null);
  const [showCreate, setShowCreate] = useState(false);
  const [senders, setSenders] = useState([]);
  const [form, setForm] = useState({ name: '', subject: '', sender_id: 0, html_body: '' });
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [t, s] = await Promise.all([
        apiFetch('/placement/tests'),
        apiFetch('/senders'),
      ]);
      setTests(t || []);
      setSenders(s || []);
    } catch { }
    setLoading(false);
  }, []);

  useEffect(() => { load(); }, [load]);

  const handleCreate = async () => {
    setCreating(true); setError('');
    try {
      await apiFetch('/placement/tests', { method: 'POST', body: JSON.stringify({ ...form, sender_id: +form.sender_id }) });
      setShowCreate(false);
      setForm({ name: '', subject: '', sender_id: 0, html_body: '' });
      setTimeout(load, 500);
    } catch (e) { setError(e.message); }
    setCreating(false);
  };

  const loadResults = async (id) => {
    if (expanded === id) { setExpanded(null); return; }
    setExpanded(id);
    try {
      const t = await apiFetch(`/placement/tests/${id}`);
      setTests(prev => prev.map(x => x.id === id ? { ...x, results: t.results } : x));
    } catch { }
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">Run a placement test to see inbox vs. spam rates across your seed mailboxes.</p>
        <button onClick={() => setShowCreate(v => !v)}
          className="flex items-center gap-2 px-3 py-2 rounded-md bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90">
          <Play className="w-4 h-4" /> Run Test
        </button>
      </div>

      {showCreate && (
        <div className="border rounded-lg p-4 bg-muted/30 space-y-3">
          <h3 className="text-sm font-semibold">New Placement Test</h3>
          {error && <p className="text-xs text-destructive bg-destructive/10 px-3 py-2 rounded">{error}</p>}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="text-xs text-muted-foreground font-medium">Test Name</label>
              <input value={form.name} onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
                placeholder="e.g. June Newsletter Test"
                className="w-full mt-1 px-3 py-2 rounded-md border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary/50" />
            </div>
            <div>
              <label className="text-xs text-muted-foreground font-medium">Sender</label>
              <select value={form.sender_id} onChange={e => setForm(f => ({ ...f, sender_id: e.target.value }))}
                className="w-full mt-1 px-3 py-2 rounded-md border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary/50">
                <option value={0}>Select sender…</option>
                {senders.map(s => <option key={s.id} value={s.id}>{s.email}</option>)}
              </select>
            </div>
            <div className="col-span-2">
              <label className="text-xs text-muted-foreground font-medium">Subject</label>
              <input value={form.subject} onChange={e => setForm(f => ({ ...f, subject: e.target.value }))}
                placeholder="Test email subject"
                className="w-full mt-1 px-3 py-2 rounded-md border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary/50" />
            </div>
            <div className="col-span-2">
              <label className="text-xs text-muted-foreground font-medium">HTML Body</label>
              <textarea value={form.html_body} onChange={e => setForm(f => ({ ...f, html_body: e.target.value }))}
                rows={4} placeholder="<html>…</html>"
                className="w-full mt-1 px-3 py-2 rounded-md border bg-background text-sm font-mono focus:outline-none focus:ring-2 focus:ring-primary/50" />
            </div>
          </div>
          <div className="flex gap-2 justify-end">
            <button onClick={() => setShowCreate(false)} className="px-3 py-1.5 text-sm border rounded-md hover:bg-muted">Cancel</button>
            <button onClick={handleCreate} disabled={creating}
              className="px-4 py-1.5 text-sm bg-primary text-primary-foreground rounded-md hover:bg-primary/90 disabled:opacity-60">
              {creating ? 'Starting…' : 'Start Test'}
            </button>
          </div>
        </div>
      )}

      {loading ? (
        <div className="text-center py-8 text-muted-foreground text-sm">Loading…</div>
      ) : tests.length === 0 ? (
        <div className="text-center py-12 text-muted-foreground">
          <TestTube2 className="w-10 h-10 mx-auto mb-2 opacity-30" />
          <p className="text-sm">No placement tests yet. Run your first test above.</p>
        </div>
      ) : (
        <div className="space-y-2">
          {tests.map(t => (
            <div key={t.id} className="border rounded-lg overflow-hidden">
              <div className="flex items-center gap-4 px-4 py-3 cursor-pointer hover:bg-muted/30" onClick={() => loadResults(t.id)}>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="font-medium text-sm truncate">{t.name}</span>
                    <StatusBadge status={t.status} />
                  </div>
                  <p className="text-xs text-muted-foreground mt-0.5">{t.subject}</p>
                </div>
                {t.status === 'completed' && (
                  <div className="flex gap-4 text-xs">
                    <span className="text-green-600 dark:text-green-400 font-semibold">{Math.round((t.inbox_rate || 0) * 100)}% Inbox</span>
                    <span className="text-red-500 font-semibold">{Math.round((t.spam_rate || 0) * 100)}% Spam</span>
                    <span className="text-yellow-500 font-semibold">{Math.round((t.missing_rate || 0) * 100)}% Missing</span>
                  </div>
                )}
                {expanded === t.id ? <ChevronDown className="w-4 h-4 text-muted-foreground shrink-0" /> : <ChevronRight className="w-4 h-4 text-muted-foreground shrink-0" />}
              </div>

              {expanded === t.id && t.results && (
                <div className="border-t bg-muted/20 p-3">
                  {t.status === 'completed' && (
                    <div className="grid grid-cols-3 gap-3 mb-4">
                      <RateBar label="Inbox" value={t.inbox_rate} color="green" />
                      <RateBar label="Spam" value={t.spam_rate} color="red" />
                      <RateBar label="Missing" value={t.missing_rate} color="yellow" />
                    </div>
                  )}
                  <div className="text-xs font-semibold text-muted-foreground uppercase mb-2">Per-Mailbox Results</div>
                  <div className="space-y-1">
                    {t.results.length === 0 ? (
                      <p className="text-xs text-muted-foreground">No results yet — test may still be running.</p>
                    ) : t.results.map(r => (
                      <div key={r.id} className="flex items-center gap-3 text-sm py-1 border-b border-border/50 last:border-0">
                        <span className="font-medium w-20 shrink-0">{r.isp}</span>
                        <span className="text-muted-foreground flex-1 truncate">{r.email}</span>
                        <PlacementBadge placement={r.placement} />
                        {r.inbox_folder && <span className="text-xs text-muted-foreground">({r.inbox_folder})</span>}
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

// ─── Main Page ─────────────────────────────────────────────────────────────────
export default function InboxPlacementPage() {
  const [tab, setTab] = useState('tests');
  const TABS = [
    { id: 'tests', label: 'Placement Tests', icon: TestTube2 },
    { id: 'mailboxes', label: 'Seed Mailboxes', icon: Mail },
  ];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <Inbox className="w-6 h-6 text-primary" /> Inbox Placement Testing
          </h1>
          <p className="text-muted-foreground text-sm mt-1">Send test emails to seed mailboxes and verify inbox vs. spam placement.</p>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 border-b">
        {TABS.map(t => {
          const Icon = t.icon;
          return (
            <button key={t.id} onClick={() => setTab(t.id)}
              className={cn('flex items-center gap-2 px-4 py-2.5 text-sm font-medium border-b-2 -mb-px transition-colors',
                tab === t.id ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground')}>
              <Icon className="w-4 h-4" /> {t.label}
            </button>
          );
        })}
      </div>

      {tab === 'tests' && <PlacementTestsTab />}
      {tab === 'mailboxes' && <SeedMailboxesTab />}
    </div>
  );
}
