import React, { useState, useEffect, useCallback } from 'react';
import {
  FlaskConical, Plus, Trash2, Trophy, RefreshCw, ChevronDown, ChevronRight,
  BarChart2, CheckCircle2, TrendingUp
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

function pct(a, b) {
  if (!b || b === 0) return '—';
  return ((a / b) * 100).toFixed(1) + '%';
}

function WinnerBadge() {
  return (
    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-bold bg-yellow-100 text-yellow-700 dark:bg-yellow-900/40 dark:text-yellow-400">
      <Trophy className="w-3 h-3" /> Winner
    </span>
  );
}

function StatBar({ label, value, total, color = 'blue' }) {
  const p = total > 0 ? (value / total) * 100 : 0;
  const barColor = color === 'green' ? 'bg-green-500' : color === 'blue' ? 'bg-blue-500' : 'bg-purple-500';
  return (
    <div className="space-y-1">
      <div className="flex justify-between text-xs">
        <span className="text-muted-foreground">{label}</span>
        <span className="font-semibold">{value} <span className="text-muted-foreground">({p.toFixed(1)}%)</span></span>
      </div>
      <div className="h-1.5 bg-muted rounded-full overflow-hidden">
        <div className={cn('h-full rounded-full transition-all', barColor)} style={{ width: `${Math.min(p, 100)}%` }} />
      </div>
    </div>
  );
}

// ─── Variants panel for a single campaign ────────────────────────────────────
function CampaignABPanel({ campaign }) {
  const [variants, setVariants] = useState([]);
  const [summary, setSummary] = useState(null);
  const [loading, setLoading] = useState(true);
  const [showAdd, setShowAdd] = useState(false);
  const [form, setForm] = useState({ name: '', subject: '', html_body: '', split_pct: 0.5 });
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [v, s] = await Promise.all([
        apiFetch(`/campaigns/${campaign.id}/variants`),
        apiFetch(`/campaigns/${campaign.id}/ab-summary`),
      ]);
      setVariants(v || []);
      setSummary(s);
    } catch { }
    setLoading(false);
  }, [campaign.id]);

  useEffect(() => { load(); }, [load]);

  const handleAdd = async () => {
    setSaving(true); setError('');
    try {
      await apiFetch(`/campaigns/${campaign.id}/variants`, {
        method: 'POST',
        body: JSON.stringify({ ...form, split_pct: +form.split_pct }),
      });
      setShowAdd(false);
      setForm({ name: '', subject: '', html_body: '', split_pct: 0.5 });
      load();
    } catch (e) { setError(e.message); }
    setSaving(false);
  };

  const handleDelete = async (vid) => {
    if (!confirm('Delete this variant?')) return;
    try { await apiFetch(`/campaigns/${campaign.id}/variants/${vid}`, { method: 'DELETE' }); load(); } catch { }
  };

  const handleSetWinner = async (vid) => {
    if (!confirm('Set this variant as winner and stop A/B testing?')) return;
    try { await apiFetch(`/campaigns/${campaign.id}/variants/${vid}/set-winner`, { method: 'POST' }); load(); } catch (e) { alert(e.message); }
  };

  const winnerID = summary?.winner_variant_id;
  const metric = summary?.ab_win_metric || 'open_rate';
  const afterHours = summary?.ab_win_after_hours || 24;

  return (
    <div className="space-y-4 pt-2 pb-1">
      {loading ? (
        <p className="text-sm text-muted-foreground py-4 text-center">Loading…</p>
      ) : (
        <>
          {/* Summary bar */}
          {summary && (
            <div className="grid grid-cols-4 gap-3">
              {[
                { label: 'Total Sent', val: summary.total_recipients, icon: '📨' },
                { label: 'Win Metric', val: metric === 'open_rate' ? 'Open Rate' : 'Click Rate', icon: '📊' },
                { label: 'Win After', val: `${afterHours}h`, icon: '⏱' },
                { label: 'Winner', val: winnerID ? `Variant #${winnerID}` : 'TBD', icon: '🏆' },
              ].map(s => (
                <div key={s.label} className="border rounded-lg p-3 bg-muted/20">
                  <div className="text-lg">{s.icon}</div>
                  <div className="text-sm font-bold mt-1">{s.val}</div>
                  <div className="text-xs text-muted-foreground">{s.label}</div>
                </div>
              ))}
            </div>
          )}

          {/* Variants */}
          <div className="space-y-2">
            {variants.length === 0 ? (
              <div className="text-center py-6 text-muted-foreground">
                <FlaskConical className="w-8 h-8 mx-auto mb-2 opacity-30" />
                <p className="text-sm">No variants yet. Add at least 2 to start A/B testing.</p>
              </div>
            ) : variants.map(v => (
              <div key={v.id} className={cn('border rounded-lg p-4 transition-colors', v.id === winnerID ? 'border-yellow-500/50 bg-yellow-50/30 dark:bg-yellow-900/10' : 'bg-muted/10')}>
                <div className="flex items-start justify-between gap-3">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      <span className="font-semibold text-sm">{v.name}</span>
                      {v.id === winnerID && <WinnerBadge />}
                      <span className="text-xs text-muted-foreground bg-muted px-1.5 py-0.5 rounded">{Math.round((v.split_pct || 0) * 100)}% split</span>
                    </div>
                    <p className="text-xs text-muted-foreground truncate">Subject: {v.subject || '(inherits campaign)'}</p>
                  </div>
                  <div className="flex gap-2 shrink-0">
                    {v.id !== winnerID && (
                      <button onClick={() => handleSetWinner(v.id)}
                        className="flex items-center gap-1 text-xs px-2.5 py-1.5 rounded border hover:bg-yellow-50 hover:border-yellow-400 dark:hover:bg-yellow-900/20 text-yellow-600 dark:text-yellow-400 font-medium transition-colors">
                        <Trophy className="w-3.5 h-3.5" /> Set Winner
                      </button>
                    )}
                    <button onClick={() => handleDelete(v.id)} className="text-destructive hover:bg-destructive/10 p-1.5 rounded">
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                </div>
                <div className="mt-3 grid grid-cols-3 gap-3">
                  <StatBar label="Sent" value={v.sent_count || 0} total={summary?.total_recipients || 1} color="blue" />
                  <StatBar label="Opens" value={v.open_count || 0} total={v.sent_count || 1} color="green" />
                  <StatBar label="Clicks" value={v.click_count || 0} total={v.sent_count || 1} color="purple" />
                </div>
              </div>
            ))}
          </div>

          {/* Add variant form */}
          {showAdd && (
            <div className="border rounded-lg p-4 bg-muted/30 space-y-3">
              <h4 className="text-sm font-semibold">Add Variant</h4>
              {error && <p className="text-xs text-destructive bg-destructive/10 px-3 py-2 rounded">{error}</p>}
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-xs text-muted-foreground font-medium">Variant Name</label>
                  <input value={form.name} onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
                    placeholder="e.g. Variant B"
                    className="w-full mt-1 px-3 py-2 rounded-md border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary/50" />
                </div>
                <div>
                  <label className="text-xs text-muted-foreground font-medium">Split % (0–1)</label>
                  <input type="number" step="0.05" min="0.05" max="0.95" value={form.split_pct}
                    onChange={e => setForm(f => ({ ...f, split_pct: e.target.value }))}
                    className="w-full mt-1 px-3 py-2 rounded-md border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary/50" />
                </div>
                <div className="col-span-2">
                  <label className="text-xs text-muted-foreground font-medium">Subject (leave blank to inherit campaign subject)</label>
                  <input value={form.subject} onChange={e => setForm(f => ({ ...f, subject: e.target.value }))}
                    placeholder="Override subject line"
                    className="w-full mt-1 px-3 py-2 rounded-md border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary/50" />
                </div>
                <div className="col-span-2">
                  <label className="text-xs text-muted-foreground font-medium">HTML Body (leave blank to inherit campaign body)</label>
                  <textarea value={form.html_body} onChange={e => setForm(f => ({ ...f, html_body: e.target.value }))}
                    rows={3} placeholder="<html>…</html>"
                    className="w-full mt-1 px-3 py-2 rounded-md border bg-background text-sm font-mono focus:outline-none focus:ring-2 focus:ring-primary/50" />
                </div>
              </div>
              <div className="flex gap-2 justify-end">
                <button onClick={() => setShowAdd(false)} className="px-3 py-1.5 text-sm border rounded-md hover:bg-muted">Cancel</button>
                <button onClick={handleAdd} disabled={saving}
                  className="px-4 py-1.5 text-sm bg-primary text-primary-foreground rounded-md hover:bg-primary/90 disabled:opacity-60">
                  {saving ? 'Saving…' : 'Add Variant'}
                </button>
              </div>
            </div>
          )}

          <button onClick={() => setShowAdd(v => !v)}
            className="flex items-center gap-2 text-sm text-primary hover:underline font-medium">
            <Plus className="w-4 h-4" /> Add Variant
          </button>
        </>
      )}
    </div>
  );
}

// ─── Main Page ─────────────────────────────────────────────────────────────────
export default function ABTestPage() {
  const [campaigns, setCampaigns] = useState([]);
  const [loading, setLoading] = useState(true);
  const [expanded, setExpanded] = useState(null);
  const [filter, setFilter] = useState('ab');

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const c = await apiFetch('/campaigns');
      setCampaigns(c || []);
    } catch { }
    setLoading(false);
  }, []);

  useEffect(() => { load(); }, [load]);

  const shown = filter === 'ab' ? campaigns.filter(c => c.is_ab_test) : campaigns;

  const statusColor = (s) => ({
    draft:     'bg-muted text-muted-foreground',
    scheduled: 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-400',
    sending:   'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/40 dark:text-yellow-400',
    sent:      'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-400',
    paused:    'bg-orange-100 text-orange-700 dark:bg-orange-900/40 dark:text-orange-400',
  }[s] || 'bg-muted text-muted-foreground');

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between flex-wrap gap-3">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <FlaskConical className="w-6 h-6 text-primary" /> A/B Testing
          </h1>
          <p className="text-muted-foreground text-sm mt-1">
            Manage variants for campaigns and track which subject/body performs best.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <select value={filter} onChange={e => setFilter(e.target.value)}
            className="px-3 py-2 rounded-md border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary/50">
            <option value="ab">A/B Campaigns Only</option>
            <option value="all">All Campaigns</option>
          </select>
          <button onClick={load} className="flex items-center gap-1.5 px-3 py-2 rounded-md border hover:bg-muted text-sm">
            <RefreshCw className="w-4 h-4" /> Refresh
          </button>
        </div>
      </div>

      {loading ? (
        <div className="text-center py-16 text-muted-foreground text-sm">Loading campaigns…</div>
      ) : shown.length === 0 ? (
        <div className="text-center py-16 text-muted-foreground">
          <FlaskConical className="w-12 h-12 mx-auto mb-3 opacity-30" />
          <p className="text-sm font-medium">No A/B test campaigns found.</p>
          <p className="text-xs mt-1">Create a campaign with "Enable A/B Test" checked to get started.</p>
        </div>
      ) : (
        <div className="space-y-2">
          {shown.map(c => (
            <div key={c.id} className="border rounded-lg overflow-hidden">
              <div
                className="flex items-center gap-4 px-4 py-3 cursor-pointer hover:bg-muted/30"
                onClick={() => setExpanded(expanded === c.id ? null : c.id)}
              >
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="font-medium text-sm truncate">{c.name}</span>
                    <span className={cn('px-2 py-0.5 rounded text-xs font-semibold capitalize', statusColor(c.status))}>
                      {c.status}
                    </span>
                    {c.is_ab_test && (
                      <span className="px-2 py-0.5 rounded text-xs font-semibold bg-purple-100 text-purple-700 dark:bg-purple-900/40 dark:text-purple-400">
                        A/B
                      </span>
                    )}
                  </div>
                  <p className="text-xs text-muted-foreground mt-0.5 truncate">{c.subject}</p>
                </div>
                <div className="flex items-center gap-4 text-xs text-muted-foreground shrink-0">
                  <span>{c.total_sent || 0} sent</span>
                  <span>{pct(c.total_opens, c.total_sent)} opens</span>
                  <span>{pct(c.total_clicks, c.total_sent)} clicks</span>
                </div>
                {expanded === c.id
                  ? <ChevronDown className="w-4 h-4 text-muted-foreground shrink-0" />
                  : <ChevronRight className="w-4 h-4 text-muted-foreground shrink-0" />}
              </div>

              {expanded === c.id && (
                <div className="border-t px-4 py-3 bg-muted/10">
                  <CampaignABPanel campaign={c} />
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
