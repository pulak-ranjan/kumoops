import React, { useState, useEffect } from 'react';
import {
  RefreshCw,
  Trash2,
  Zap,
  Inbox,
  Clock,
  AlertCircle,
  CheckCircle2,
  Mail,
  Globe,
  AlertTriangle,
  X,
  Loader2
} from 'lucide-react';
import { cn } from '../lib/utils';
import { apiRequest } from '../api';

// --- Provider Row color coding ---
function getVolumeColor(total) {
  if (total >= 1000) return "bg-red-500/10 border-red-500/20";
  if (total >= 100) return "bg-amber-500/10 border-amber-500/20";
  if (total >= 10) return "bg-blue-500/10 border-blue-500/20";
  return "";
}

// --- Stuck Messages Modal ---
function StuckMessagesModal({ onClose }) {
  const [stuck, setStuck] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [deleting, setDeleting] = useState(null);

  const token = localStorage.getItem('kumoui_token');
  const headers = { Authorization: `Bearer ${token}` };

  const load = async () => {
    setLoading(true);
    setError('');
    try {
      const res = await fetch('/api/queue/stuck', { headers });
      if (res.status === 401) { window.location.href = '/login'; return; }
      const data = await res.json();
      setStuck(Array.isArray(data) ? data : (data.messages || []));
    } catch (e) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, []);

  const deleteMsg = async (id) => {
    if (!confirm('Delete this stuck message?')) return;
    setDeleting(id);
    try {
      await fetch(`/api/queue/${id}`, { method: 'DELETE', headers });
      setStuck(prev => prev.filter(m => m.id !== id));
    } catch (e) {
      setError(e.message);
    } finally {
      setDeleting(null);
    }
  };

  const formatAge = (seconds) => {
    if (!seconds) return '-';
    if (seconds < 60) return `${seconds}s`;
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
    if (seconds < 86400) return `${Math.floor(seconds / 3600)}h`;
    return `${Math.floor(seconds / 86400)}d`;
  };

  return (
    <div className="fixed inset-0 bg-background/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
      <div className="bg-card w-full max-w-2xl border rounded-xl shadow-xl flex flex-col max-h-[80vh]">
        {/* Header */}
        <div className="flex items-center justify-between p-5 border-b flex-shrink-0">
          <div className="flex items-center gap-2">
            <AlertTriangle className="w-5 h-5 text-amber-500" />
            <div>
              <h3 className="font-semibold">Stuck Messages</h3>
              <p className="text-xs text-muted-foreground">Messages with repeated delivery failures</p>
            </div>
          </div>
          <button onClick={onClose} className="p-1.5 hover:bg-muted rounded-md transition-colors">
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Body */}
        <div className="flex-1 overflow-y-auto p-5">
          {loading && (
            <div className="flex items-center justify-center py-10 text-muted-foreground text-sm">
              <Loader2 className="w-4 h-4 animate-spin mr-2" /> Loading stuck messages...
            </div>
          )}

          {error && (
            <div className="bg-destructive/10 text-destructive p-3 rounded-md text-sm flex items-center gap-2 mb-3">
              <AlertCircle className="w-4 h-4 flex-shrink-0" /> {error}
            </div>
          )}

          {!loading && stuck.length === 0 && !error && (
            <div className="flex flex-col items-center justify-center py-10 text-center">
              <CheckCircle2 className="w-10 h-10 text-green-500 mb-3" />
              <p className="font-medium">No stuck messages</p>
              <p className="text-sm text-muted-foreground mt-1">All messages are progressing normally.</p>
            </div>
          )}

          {!loading && stuck.length > 0 && (
            <div className="space-y-3">
              {stuck.map(msg => (
                <div key={msg.id} className="bg-amber-500/5 border border-amber-500/20 rounded-lg p-4">
                  <div className="flex items-start justify-between gap-3">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-3 flex-wrap mb-2">
                        <span className="font-mono text-xs text-muted-foreground">{msg.id?.substring(0, 12)}...</span>
                        <span className="text-xs bg-amber-500/10 text-amber-700 dark:text-amber-400 px-2 py-0.5 rounded-full font-medium">
                          Age: {formatAge(msg.age_seconds || msg.age)}
                        </span>
                        <span className="text-xs bg-red-500/10 text-red-700 dark:text-red-400 px-2 py-0.5 rounded-full font-medium">
                          {msg.attempts || 0} attempts
                        </span>
                      </div>
                      <div className="text-sm truncate mb-1">
                        <span className="text-muted-foreground">To:</span> <span className="font-medium">{msg.recipient || '-'}</span>
                      </div>
                      {msg.last_error && (
                        <p className="text-xs text-destructive font-mono bg-destructive/5 px-2 py-1 rounded mt-2 break-all">
                          {msg.last_error}
                        </p>
                      )}
                    </div>
                    <button
                      onClick={() => deleteMsg(msg.id)}
                      disabled={deleting === msg.id}
                      className="flex-shrink-0 p-1.5 rounded-md hover:bg-destructive/10 text-muted-foreground hover:text-destructive transition-colors"
                      title="Delete message"
                    >
                      {deleting === msg.id ? <Loader2 className="w-4 h-4 animate-spin" /> : <Trash2 className="w-4 h-4" />}
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="p-4 border-t flex-shrink-0 flex justify-end">
          <button onClick={onClose} className="px-4 py-2 text-sm rounded-md hover:bg-muted transition-colors">
            Close
          </button>
        </div>
      </div>
    </div>
  );
}

// --- By Provider Tab ---
function ProviderTab() {
  const [providers, setProviders] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const token = localStorage.getItem('kumoui_token');
  const headers = { Authorization: `Bearer ${token}` };

  const load = async () => {
    setLoading(true);
    setError('');
    try {
      const res = await fetch('/api/queue/providers', { headers });
      if (res.status === 401) { window.location.href = '/login'; return; }
      const data = await res.json();
      setProviders(Array.isArray(data) ? data : (data.providers || []));
    } catch (e) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, []);

  const formatDate = (d) => d ? new Date(d).toLocaleString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' }) : '-';

  return (
    <div className="bg-card border rounded-xl overflow-hidden shadow-sm">
      {loading ? (
        <div className="p-12 text-center text-muted-foreground">Loading provider data...</div>
      ) : error ? (
        <div className="p-6 text-center">
          <div className="bg-destructive/10 text-destructive p-3 rounded-md text-sm">{error}</div>
        </div>
      ) : providers.length === 0 ? (
        <div className="flex flex-col items-center justify-center p-16 text-center">
          <CheckCircle2 className="w-12 h-12 text-green-500 mb-4" />
          <h3 className="text-xl font-semibold mb-1">No Queued Providers</h3>
          <p className="text-muted-foreground">All mail has been delivered.</p>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm text-left">
            <thead className="bg-muted/50 text-muted-foreground uppercase text-xs">
              <tr>
                <th className="px-4 py-3 font-medium">Provider Domain</th>
                <th className="px-4 py-3 font-medium text-right">Total</th>
                <th className="px-4 py-3 font-medium text-right">Queued</th>
                <th className="px-4 py-3 font-medium text-right">Deferred</th>
                <th className="px-4 py-3 font-medium">Oldest Message</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {providers.map((p, i) => (
                <tr key={i} className={cn("transition-colors hover:bg-muted/30 border-l-2 border-l-transparent", getVolumeColor(p.total))}>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
                      <Globe className="w-4 h-4 text-muted-foreground flex-shrink-0" />
                      <span className="font-medium">{p.domain || p.provider_domain || '-'}</span>
                    </div>
                  </td>
                  <td className="px-4 py-3 text-right font-bold">{p.total || 0}</td>
                  <td className="px-4 py-3 text-right">
                    <span className="text-blue-600 dark:text-blue-400 font-medium">{p.queued || 0}</span>
                  </td>
                  <td className="px-4 py-3 text-right">
                    <span className={cn("font-medium", (p.deferred || 0) > 0 ? "text-amber-600 dark:text-amber-400" : "text-muted-foreground")}>
                      {p.deferred || 0}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-muted-foreground text-xs">
                    {formatDate(p.oldest_message || p.oldest_at)}
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

// --- QueueStat ---
function QueueStat({ label, value, icon: Icon, color }) {
  return (
    <div className="bg-card border rounded-xl p-4 shadow-sm flex items-center justify-between">
      <div>
        <p className="text-sm font-medium text-muted-foreground">{label}</p>
        <p className="text-2xl font-bold mt-1">{value || 0}</p>
      </div>
      <div className={cn("p-2 rounded-lg bg-secondary", color)}>
        <Icon className="w-5 h-5" />
      </div>
    </div>
  );
}

export default function QueuePage() {
  const [messages, setMessages] = useState([]);
  const [stats, setStats] = useState(null);
  const [loading, setLoading] = useState(true);
  const [limit, setLimit] = useState(100);
  const [activeTab, setActiveTab] = useState('messages');
  const [stuckCount, setStuckCount] = useState(null);
  const [showStuck, setShowStuck] = useState(false);

  const token = localStorage.getItem('kumoui_token');
  const headers = { Authorization: `Bearer ${token}` };

  useEffect(() => { fetchQueue(); fetchStats(); fetchStuckCount(); }, [limit]);

  const fetchQueue = async () => {
    setLoading(true);
    try {
      const res = await fetch(`/api/queue?limit=${limit}`, { headers });
      if (res.status === 401) { window.location.href = '/login'; return; }
      const data = await res.json();
      setMessages(Array.isArray(data) ? data : []);
    } catch (e) { console.error(e); setMessages([]); }
    setLoading(false);
  };

  const fetchStats = async () => {
    try {
      const res = await fetch('/api/queue/stats', { headers });
      if (res.ok) setStats(await res.json());
    } catch (e) { console.error(e); }
  };

  const fetchStuckCount = async () => {
    try {
      const res = await fetch('/api/queue/stuck', { headers });
      if (res.ok) {
        const data = await res.json();
        const list = Array.isArray(data) ? data : (data.messages || []);
        setStuckCount(list.length);
      }
    } catch (e) { console.error(e); }
  };

  const deleteMessage = async (id) => {
    if (!confirm('Delete this message from queue?')) return;
    try {
      await fetch(`/api/queue/${id}`, { method: 'DELETE', headers });
      fetchQueue(); fetchStats();
    } catch (e) { console.error(e); }
  };

  const flushQueue = async () => {
    if (!confirm('Retry all deferred messages?')) return;
    try {
      await fetch('/api/queue/flush', { method: 'POST', headers });
      fetchQueue(); fetchStats();
    } catch (e) { console.error(e); }
  };

  const handleRefresh = () => {
    fetchQueue();
    fetchStats();
    fetchStuckCount();
  };

  const formatDate = (d) => d ? new Date(d).toLocaleString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' }) : '-';
  const formatSize = (b) => b > 1024 ? `${(b/1024).toFixed(1)} KB` : `${b} B`;

  const tabs = [
    { id: 'messages', label: 'Messages' },
    { id: 'providers', label: 'By Provider' },
  ];

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Mail Queue</h1>
          <p className="text-muted-foreground">Monitor and manage outbound messages.</p>
        </div>
        <div className="flex gap-2 flex-wrap">
          {/* Stuck Messages Badge */}
          <button
            onClick={() => setShowStuck(true)}
            className={cn(
              "flex items-center gap-2 h-10 px-4 rounded-md text-sm font-medium transition-colors border relative",
              stuckCount && stuckCount > 0
                ? "bg-amber-500/10 border-amber-500/30 text-amber-700 dark:text-amber-400 hover:bg-amber-500/20"
                : "bg-secondary text-secondary-foreground hover:bg-secondary/80"
            )}
            title="View stuck messages"
          >
            <AlertTriangle className="w-4 h-4" />
            Stuck Messages
            {stuckCount !== null && stuckCount > 0 && (
              <span className="absolute -top-1.5 -right-1.5 bg-amber-500 text-white text-xs rounded-full min-w-[18px] h-[18px] flex items-center justify-center px-1 font-bold">
                {stuckCount}
              </span>
            )}
          </button>

          <select value={limit} onChange={e => setLimit(+e.target.value)} className="h-10 rounded-md border bg-background px-3 py-2 text-sm focus:ring-2 focus:ring-ring">
            <option value={50}>50 Items</option>
            <option value={100}>100 Items</option>
            <option value={500}>500 Items</option>
          </select>
          <button onClick={handleRefresh} className="flex items-center gap-2 h-10 px-4 rounded-md bg-secondary text-secondary-foreground hover:bg-secondary/80 text-sm font-medium transition-colors">
            <RefreshCw className="w-4 h-4" /> Refresh
          </button>
          <button onClick={flushQueue} className="flex items-center gap-2 h-10 px-4 rounded-md bg-amber-600 text-white hover:bg-amber-700 text-sm font-medium transition-colors shadow-sm">
            <Zap className="w-4 h-4" /> Flush Queue
          </button>
        </div>
      </div>

      {stats && (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <QueueStat label="Total Messages" value={stats.total} icon={Inbox} />
          <QueueStat label="Queued (Active)" value={stats.queued} icon={Mail} color="text-blue-500" />
          <QueueStat label="Deferred (Retry)" value={stats.deferred} icon={Clock} color="text-amber-500" />
          <QueueStat label="Total Size" value={formatSize(stats.total_size)} icon={AlertCircle} />
        </div>
      )}

      {/* Tabs */}
      <div className="flex items-center space-x-1 bg-muted p-1 rounded-lg w-fit">
        {tabs.map(tab => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            className={cn(
              "px-4 py-2 rounded-md text-sm font-medium transition-all",
              activeTab === tab.id
                ? "bg-background text-foreground shadow-sm"
                : "text-muted-foreground hover:text-foreground hover:bg-background/50"
            )}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* Messages Tab */}
      {activeTab === 'messages' && (
        <>
          <div className="bg-card border rounded-xl overflow-hidden shadow-sm">
            {loading ? (
              <div className="p-12 text-center text-muted-foreground">Loading queue data...</div>
            ) : messages.length === 0 ? (
              <div className="flex flex-col items-center justify-center p-16 text-center">
                <div className="p-4 bg-green-100 dark:bg-green-900/20 rounded-full mb-4">
                  <CheckCircle2 className="w-12 h-12 text-green-600 dark:text-green-400" />
                </div>
                <h3 className="text-xl font-semibold mb-1">Queue is Empty</h3>
                <p className="text-muted-foreground">All messages have been delivered or processed.</p>
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-sm text-left">
                  <thead className="bg-muted/50 text-muted-foreground uppercase text-xs">
                    <tr>
                      <th className="px-4 py-3 font-medium">ID</th>
                      <th className="px-4 py-3 font-medium">Sender</th>
                      <th className="px-4 py-3 font-medium">Recipient</th>
                      <th className="px-4 py-3 font-medium">Status</th>
                      <th className="px-4 py-3 font-medium">Created</th>
                      <th className="px-4 py-3 font-medium text-center">Tries</th>
                      <th className="px-4 py-3 font-medium text-right">Action</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-border">
                    {messages.map(msg => (
                      <tr key={msg.id} className="hover:bg-muted/50 transition-colors group">
                        <td className="px-4 py-3 font-mono text-xs text-muted-foreground" title={msg.id}>
                          {msg.id.substring(0, 8)}...
                        </td>
                        <td className="px-4 py-3 truncate max-w-[150px]" title={msg.sender}>{msg.sender || '-'}</td>
                        <td className="px-4 py-3 truncate max-w-[150px]" title={msg.recipient}>{msg.recipient || '-'}</td>
                        <td className="px-4 py-3">
                          <span className={cn(
                            "inline-flex items-center px-2 py-0.5 rounded text-xs font-medium capitalize",
                            msg.status === 'deferred'
                              ? "bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-400"
                              : "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400"
                          )}>
                            {msg.status || 'queued'}
                          </span>
                        </td>
                        <td className="px-4 py-3 text-muted-foreground whitespace-nowrap">{formatDate(msg.created_at)}</td>
                        <td className="px-4 py-3 text-center">{msg.attempts || 0}</td>
                        <td className="px-4 py-3 text-right">
                          <button
                            onClick={() => deleteMessage(msg.id)}
                            className="p-1.5 hover:bg-destructive/10 text-muted-foreground hover:text-destructive rounded-md transition-colors opacity-0 group-hover:opacity-100"
                            title="Delete Message"
                          >
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

          {/* Error Summary */}
          {messages.length > 0 && messages.some(m => m.error_msg) && (
            <div className="bg-card border rounded-xl p-6 shadow-sm">
              <h3 className="text-lg font-semibold mb-4 flex items-center gap-2 text-destructive">
                <AlertCircle className="w-5 h-5" /> Recent Deferral Reasons
              </h3>
              <div className="space-y-2 max-h-60 overflow-y-auto pr-2">
                {messages.filter(m => m.error_msg).slice(0, 10).map((m, i) => (
                  <div key={i} className="text-sm p-3 bg-destructive/5 border border-destructive/10 rounded-md">
                    <div className="font-medium text-foreground mb-1">{m.recipient}</div>
                    <div className="text-destructive font-mono text-xs break-all">{m.error_msg}</div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </>
      )}

      {/* By Provider Tab */}
      {activeTab === 'providers' && <ProviderTab />}

      {/* Stuck Messages Modal */}
      {showStuck && (
        <StuckMessagesModal onClose={() => { setShowStuck(false); fetchStuckCount(); }} />
      )}
    </div>
  );
}
