import React, { useState } from 'react';
import { Server, Plus, Trash2, Wifi, WifiOff, RefreshCw, X, Eye, EyeOff, Circle } from 'lucide-react';
import { useServer } from '../ServerContext';
import { cn } from '../lib/utils';

function StatusDot({ status }) {
  return (
    <Circle className={cn(
      'w-2.5 h-2.5 fill-current shrink-0',
      status === 'online'  ? 'text-green-500' :
      status === 'offline' ? 'text-red-500'   : 'text-muted-foreground'
    )} />
  );
}

function AddServerModal({ onClose, onAdd }) {
  const [name, setName]         = useState('');
  const [url, setUrl]           = useState('');
  const [token, setToken]       = useState('');
  const [showToken, setShow]    = useState(false);
  const [loading, setLoading]   = useState(false);
  const [error, setError]       = useState('');

  const submit = async (e) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      await onAdd(name.trim(), url.trim(), token.trim());
      onClose();
    } catch (err) {
      setError(err.message);
    }
    setLoading(false);
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="bg-card border rounded-xl shadow-xl w-full max-w-md mx-4">
        <div className="flex items-center justify-between p-5 border-b">
          <h2 className="text-base font-semibold">Add Remote Server</h2>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground"><X className="w-4 h-4" /></button>
        </div>
        <form onSubmit={submit} className="p-5 space-y-4">
          <div>
            <label className="text-xs font-medium text-muted-foreground">Display Name</label>
            <input value={name} onChange={e => setName(e.target.value)}
              placeholder="VPS-2 (Frankfurt)" required
              className="mt-1 w-full h-9 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring" />
          </div>
          <div>
            <label className="text-xs font-medium text-muted-foreground">URL</label>
            <input value={url} onChange={e => setUrl(e.target.value)}
              placeholder="https://vps2.example.com:8080" required
              className="mt-1 w-full h-9 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring" />
            <p className="text-[11px] text-muted-foreground mt-1">The base URL where f-kumo is running on the remote server.</p>
          </div>
          <div>
            <label className="text-xs font-medium text-muted-foreground">API Token</label>
            <div className="relative mt-1">
              <input
                type={showToken ? 'text' : 'password'}
                value={token} onChange={e => setToken(e.target.value)}
                placeholder="Bearer token from remote server's API Keys page"
                required
                className="w-full h-9 rounded-md border bg-background px-3 pr-9 text-sm focus:ring-2 focus:ring-ring font-mono"
              />
              <button type="button" onClick={() => setShow(v => !v)}
                className="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground">
                {showToken ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
              </button>
            </div>
            <p className="text-[11px] text-muted-foreground mt-1">
              Generate one at <strong>System → API Keys</strong> on the remote server.
            </p>
          </div>

          {error && <p className="text-xs text-red-600 dark:text-red-400">{error}</p>}

          <div className="flex justify-end gap-2 pt-2">
            <button type="button" onClick={onClose}
              className="h-9 px-4 rounded-md border text-sm font-medium hover:bg-accent transition-colors">
              Cancel
            </button>
            <button type="submit" disabled={loading}
              className="h-9 px-4 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-sm font-medium transition-colors disabled:opacity-50">
              {loading ? 'Adding…' : 'Add Server'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

export default function ServersPage() {
  const { servers, activeServer, switchServer, addServer, removeServer, testServer } = useServer();
  const [showModal, setShowModal] = useState(false);
  const [testing, setTesting]     = useState({});

  const remoteServers = servers.filter(s => s.id !== 'local');

  const handleTest = async (id) => {
    setTesting(t => ({ ...t, [id]: true }));
    await testServer(id);
    setTesting(t => ({ ...t, [id]: false }));
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Remote Servers</h1>
          <p className="text-muted-foreground text-sm mt-1">
            Connect multiple VPS instances and switch between them from the sidebar.
          </p>
        </div>
        <button onClick={() => setShowModal(true)}
          className="flex items-center gap-2 h-9 px-4 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-sm font-medium transition-colors">
          <Plus className="w-4 h-4" /> Add Server
        </button>
      </div>

      {/* How it works */}
      <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-xl p-4 text-sm text-blue-800 dark:text-blue-300 space-y-1">
        <p className="font-semibold">How it works</p>
        <ul className="list-disc list-inside text-xs space-y-0.5 text-blue-700 dark:text-blue-400">
          <li>Each remote VPS must have <strong>f-kumo running</strong> and be reachable from this server</li>
          <li>Generate an API token at <strong>System → API Keys</strong> on the remote instance</li>
          <li>Once added, use the <strong>server switcher</strong> at the top of the sidebar to switch context</li>
          <li>All pages (Stats, Queue, Delivery Log, Reputation…) will show data from the selected server</li>
        </ul>
      </div>

      {/* Local server card */}
      <div className="bg-card border rounded-xl p-5 flex items-center justify-between shadow-sm">
        <div className="flex items-center gap-3">
          <div className="w-9 h-9 bg-green-500/10 rounded-lg flex items-center justify-center">
            <Server className="w-5 h-5 text-green-500" />
          </div>
          <div>
            <div className="flex items-center gap-2 font-semibold text-sm">
              <StatusDot status="online" />
              This Server (Local)
            </div>
            <div className="text-xs text-muted-foreground mt-0.5">localhost · always available</div>
          </div>
        </div>
        {String(activeServer?.id) === 'local' ? (
          <span className="text-xs font-medium px-2.5 py-1 rounded-full bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400">Active</span>
        ) : (
          <button onClick={() => switchServer({ id: 'local', name: 'This Server (Local)', status: 'online' })}
            className="text-xs h-8 px-3 rounded-md border hover:bg-accent transition-colors font-medium">
            Switch to Local
          </button>
        )}
      </div>

      {/* Remote servers */}
      {remoteServers.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16 border rounded-xl bg-card text-muted-foreground gap-3">
          <Server className="w-12 h-12 opacity-20" />
          <p className="text-sm font-medium">No remote servers added yet.</p>
          <button onClick={() => setShowModal(true)}
            className="flex items-center gap-1.5 h-8 px-4 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-xs font-medium transition-colors">
            <Plus className="w-3.5 h-3.5" /> Add your first remote server
          </button>
        </div>
      ) : (
        <div className="space-y-3">
          {remoteServers.map(srv => {
            const isActive = String(activeServer?.id) === String(srv.id);
            return (
              <div key={srv.id} className={cn(
                'bg-card border rounded-xl p-5 flex flex-col sm:flex-row sm:items-center gap-4 shadow-sm',
                isActive && 'border-blue-400 dark:border-blue-600 bg-blue-50/30 dark:bg-blue-900/10'
              )}>
                <div className="flex items-center gap-3 flex-1 min-w-0">
                  <div className={cn(
                    'w-9 h-9 rounded-lg flex items-center justify-center shrink-0',
                    srv.status === 'online'  ? 'bg-green-500/10' :
                    srv.status === 'offline' ? 'bg-red-500/10'   : 'bg-muted'
                  )}>
                    {srv.status === 'offline'
                      ? <WifiOff className="w-5 h-5 text-red-500" />
                      : <Wifi className={cn('w-5 h-5', srv.status === 'online' ? 'text-green-500' : 'text-muted-foreground')} />}
                  </div>
                  <div className="min-w-0">
                    <div className="flex items-center gap-2 font-semibold text-sm">
                      <StatusDot status={srv.status} />
                      <span className="truncate">{srv.name}</span>
                      {isActive && <span className="text-[10px] font-medium bg-blue-100 dark:bg-blue-900/40 text-blue-600 dark:text-blue-400 px-1.5 py-0.5 rounded-full">Active</span>}
                    </div>
                    <div className="text-xs text-muted-foreground mt-0.5 truncate">{srv.url}</div>
                    {srv.last_seen && (
                      <div className="text-[11px] text-muted-foreground/70 mt-0.5">
                        Last seen: {new Date(srv.last_seen).toLocaleString()}
                      </div>
                    )}
                  </div>
                </div>

                <div className="flex items-center gap-2 shrink-0">
                  <button onClick={() => handleTest(srv.id)} disabled={testing[srv.id]}
                    className="flex items-center gap-1.5 h-8 px-3 rounded-md border text-xs font-medium hover:bg-accent transition-colors disabled:opacity-50">
                    <RefreshCw className={cn('w-3.5 h-3.5', testing[srv.id] && 'animate-spin')} />
                    {testing[srv.id] ? 'Testing…' : 'Test'}
                  </button>
                  {!isActive && (
                    <button onClick={() => switchServer(srv)}
                      className="h-8 px-3 rounded-md border text-xs font-medium hover:bg-accent transition-colors">
                      Switch
                    </button>
                  )}
                  <button onClick={() => removeServer(srv.id)}
                    className="h-8 w-8 flex items-center justify-center rounded-md border text-muted-foreground hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 transition-colors">
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {showModal && <AddServerModal onClose={() => setShowModal(false)} onAdd={addServer} />}
    </div>
  );
}
