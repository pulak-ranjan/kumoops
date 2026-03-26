import React, { useState, useEffect, useCallback } from 'react';
import {
  Server, CheckCircle2, XCircle, AlertTriangle, RefreshCw,
  Save, Play, Info, Zap, Mail
} from 'lucide-react';
import { cn } from '../lib/utils';

const API = '/api';

function authHeaders() {
  const token = localStorage.getItem('kumoui_token');
  return { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` };
}

async function apiFetch(path, opts = {}) {
  const r = await fetch(API + path, { headers: authHeaders(), ...opts });
  if (r.status === 401) { window.location.href = '/login'; return null; }
  if (!r.ok) {
    const body = await r.text();
    let msg; try { msg = JSON.parse(body).error || body; } catch { msg = body; }
    throw new Error(msg);
  }
  return r.json();
}

function StatusIcon({ ok }) {
  if (ok === true) return <CheckCircle2 className="w-5 h-5 text-green-500" />;
  if (ok === false) return <XCircle className="w-5 h-5 text-red-500" />;
  return <AlertTriangle className="w-5 h-5 text-yellow-500" />;
}

function StatCard({ icon: Icon, label, value, sub, color }) {
  return (
    <div className="border rounded-lg p-4 bg-card">
      <div className="flex items-start justify-between">
        <div>
          <p className="text-xs text-muted-foreground font-medium">{label}</p>
          <p className={cn('text-xl font-bold mt-1', color)}>{value}</p>
          {sub && <p className="text-xs text-muted-foreground mt-0.5">{sub}</p>}
        </div>
        <Icon className={cn('w-5 h-5 mt-0.5', color || 'text-muted-foreground')} />
      </div>
    </div>
  );
}

export default function RelayPage() {
  const [status, setStatus] = useState(null);
  const [settings, setSettings] = useState({ smtp_relay_enabled: false, smtp_relay_host: '', smtp_relay_port: 587, main_hostname: '' });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [applying, setApplying] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  const load = useCallback(async () => {
    setLoading(true); setError('');
    try {
      const s = await apiFetch('/relay/status');
      setStatus(s);
      setSettings({
        smtp_relay_enabled: s.settings?.smtp_relay_enabled || false,
        smtp_relay_host: s.settings?.smtp_relay_host || '',
        smtp_relay_port: s.settings?.smtp_relay_port || 587,
        main_hostname: s.settings?.main_hostname || '',
      });
    } catch (e) { setError(e.message); }
    setLoading(false);
  }, []);

  useEffect(() => { load(); }, [load]);

  const handleSave = async () => {
    setSaving(true); setError(''); setSuccess('');
    try {
      await apiFetch('/relay/settings', { method: 'PUT', body: JSON.stringify(settings) });
      setSuccess('Settings saved.');
      load();
    } catch (e) { setError(e.message); }
    setSaving(false);
  };

  const handleApply = async () => {
    setApplying(true); setError(''); setSuccess('');
    try {
      await apiFetch('/relay/apply', { method: 'POST' });
      setSuccess('KumoMTA configuration applied successfully.');
      load();
    } catch (e) { setError(e.message); }
    setApplying(false);
  };

  const kumoBound = status?.kumo_smtp_bound || [];
  const enabled = settings.smtp_relay_enabled;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <Server className="w-6 h-6 text-primary" /> SMTP Relay
          </h1>
          <p className="text-muted-foreground text-sm mt-1">
            Manage KumoMTA's SMTP relay interface for accepting inbound messages from your applications.
          </p>
        </div>
        <button onClick={load} className="flex items-center gap-1.5 px-3 py-2 rounded-md border hover:bg-muted text-sm">
          <RefreshCw className="w-4 h-4" /> Refresh
        </button>
      </div>

      {error && <div className="text-sm text-destructive bg-destructive/10 px-4 py-2.5 rounded-md">{error}</div>}
      {success && <div className="text-sm text-green-600 dark:text-green-400 bg-green-50 dark:bg-green-900/20 px-4 py-2.5 rounded-md">{success}</div>}

      {loading ? (
        <div className="text-center py-16 text-muted-foreground text-sm">Loading relay status…</div>
      ) : (
        <>
          {/* Status Cards */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
            <StatCard icon={status?.relay_running ? CheckCircle2 : XCircle} label="Relay Status"
              value={status?.relay_running ? 'Running' : 'Stopped'}
              color={status?.relay_running ? 'text-green-600 dark:text-green-400' : 'text-red-500'} />
            <StatCard icon={Zap} label="SMTP Connections"
              value={status?.active_connections ?? '—'}
              sub="active now" color="text-primary" />
            <StatCard icon={Mail} label="Queue Size"
              value={status?.queue_size ?? '—'}
              sub="messages pending" color="text-yellow-600 dark:text-yellow-400" />
            <StatCard icon={Server} label="Hostname"
              value={settings.main_hostname || 'Not set'}
              sub="EHLO name" color="text-foreground" />
          </div>

          {/* Bound SMTP Listeners */}
          <div className="border rounded-lg overflow-hidden">
            <div className="px-4 py-3 border-b bg-muted/30 flex items-center gap-2">
              <Server className="w-4 h-4 text-primary" />
              <span className="font-semibold text-sm">KumoMTA SMTP Listeners</span>
            </div>
            {kumoBound.length === 0 ? (
              <div className="px-4 py-6 text-sm text-muted-foreground text-center">
                No bound SMTP ports detected. KumoMTA may not be running or accessible.
              </div>
            ) : (
              <div className="divide-y">
                {kumoBound.map((b, i) => (
                  <div key={i} className="flex items-center gap-4 px-4 py-3 text-sm">
                    <StatusIcon ok={b.active} />
                    <div className="flex-1">
                      <span className="font-medium">{b.address}</span>
                      {b.tls && <span className="ml-2 text-xs px-1.5 py-0.5 bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-400 rounded font-medium">TLS</span>}
                      {b.auth && <span className="ml-1 text-xs px-1.5 py-0.5 bg-purple-100 text-purple-700 dark:bg-purple-900/40 dark:text-purple-400 rounded font-medium">AUTH</span>}
                    </div>
                    <span className="text-xs text-muted-foreground">{b.description || ''}</span>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Settings */}
          <div className="border rounded-lg overflow-hidden">
            <div className="px-4 py-3 border-b bg-muted/30">
              <h3 className="font-semibold text-sm">Relay Settings</h3>
            </div>
            <div className="p-4 space-y-4">
              <div className="flex items-center gap-3">
                <label className="relative inline-flex items-center cursor-pointer">
                  <input type="checkbox" className="sr-only peer"
                    checked={settings.smtp_relay_enabled}
                    onChange={e => setSettings(s => ({ ...s, smtp_relay_enabled: e.target.checked }))} />
                  <div className="w-10 h-6 bg-muted rounded-full peer peer-checked:bg-primary transition-colors after:content-[''] after:absolute after:top-0.5 after:left-0.5 after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:after:translate-x-4" />
                </label>
                <div>
                  <span className="text-sm font-medium">Enable SMTP Relay Acceptance</span>
                  <p className="text-xs text-muted-foreground">Allow KumoMTA to accept inbound SMTP relay connections.</p>
                </div>
              </div>

              <div className={cn('grid grid-cols-2 gap-3 transition-opacity', !enabled && 'opacity-50 pointer-events-none')}>
                <div>
                  <label className="text-xs text-muted-foreground font-medium">Relay Hostname / IP</label>
                  <input value={settings.smtp_relay_host} onChange={e => setSettings(s => ({ ...s, smtp_relay_host: e.target.value }))}
                    placeholder="127.0.0.1 or mail.example.com"
                    className="w-full mt-1 px-3 py-2 rounded-md border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary/50" />
                </div>
                <div>
                  <label className="text-xs text-muted-foreground font-medium">Port</label>
                  <input type="number" value={settings.smtp_relay_port} onChange={e => setSettings(s => ({ ...s, smtp_relay_port: +e.target.value }))}
                    className="w-full mt-1 px-3 py-2 rounded-md border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary/50" />
                </div>
                <div className="col-span-2">
                  <label className="text-xs text-muted-foreground font-medium">EHLO Hostname</label>
                  <input value={settings.main_hostname} onChange={e => setSettings(s => ({ ...s, main_hostname: e.target.value }))}
                    placeholder="mail.yourdomain.com"
                    className="w-full mt-1 px-3 py-2 rounded-md border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary/50" />
                  <p className="text-xs text-muted-foreground mt-1">Used in EHLO greeting and List-Unsubscribe headers.</p>
                </div>
              </div>

              {/* Info box */}
              <div className="flex gap-2 bg-blue-50 dark:bg-blue-950/30 border border-blue-200 dark:border-blue-800/50 rounded-md px-3 py-2.5">
                <Info className="w-4 h-4 text-blue-500 shrink-0 mt-0.5" />
                <p className="text-xs text-blue-700 dark:text-blue-400">
                  KumoMTA is your SMTP relay. Senders in your domain list use KumoMTA's SMTP (port 25 or 587)
                  with DKIM signing applied automatically. Use the HTTP Sending API or point your app's SMTP
                  settings to this server.
                </p>
              </div>

              <div className="flex gap-2 pt-1">
                <button onClick={handleSave} disabled={saving}
                  className="flex items-center gap-2 px-4 py-2 rounded-md bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 disabled:opacity-60">
                  <Save className="w-4 h-4" /> {saving ? 'Saving…' : 'Save Settings'}
                </button>
                <button onClick={handleApply} disabled={applying}
                  className="flex items-center gap-2 px-4 py-2 rounded-md border hover:bg-muted text-sm font-medium disabled:opacity-60">
                  <Play className="w-4 h-4 text-green-600" /> {applying ? 'Applying…' : 'Apply to KumoMTA'}
                </button>
              </div>
            </div>
          </div>

          {/* How-to */}
          <div className="border rounded-lg p-4 space-y-2 bg-muted/20">
            <h3 className="text-sm font-semibold">How to use the SMTP Relay</h3>
            <div className="text-xs text-muted-foreground space-y-1">
              <p><strong>Option 1 – HTTP API (recommended):</strong> Use <code className="bg-muted px-1 rounded">POST /api/v1/messages</code> with your API key. Mailgun-compatible.</p>
              <p><strong>Option 2 – SMTP:</strong> Configure your app to use <code className="bg-muted px-1 rounded">{settings.smtp_relay_host || 'your-server'}:{settings.smtp_relay_port}</code> with a sender credential from your domain settings.</p>
              <p><strong>Auth:</strong> SMTP credentials are managed per-sender in Domains → Senders. DKIM signing is applied automatically.</p>
            </div>
          </div>
        </>
      )}
    </div>
  );
}
