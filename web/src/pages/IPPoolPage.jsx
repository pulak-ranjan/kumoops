import React, { useState, useEffect } from 'react';
import {
  Plus,
  Pencil,
  Trash2,
  X,
  Save,
  Loader2,
  AlertCircle,
  RefreshCw,
  Server,
  Network,
} from 'lucide-react';
import { cn } from '../lib/utils';

// Distinct pastel-ish color palettes for pools
const POOL_PALETTES = [
  {
    card: 'border-blue-200 dark:border-blue-800',
    header: 'bg-blue-50 dark:bg-blue-900/20',
    pill: 'bg-blue-100 text-blue-800 dark:bg-blue-900/40 dark:text-blue-300',
    badge: 'bg-blue-500 text-white',
    addBtn: 'bg-blue-600 hover:bg-blue-700 text-white',
    ipBg: 'bg-blue-50 dark:bg-blue-900/10 border-blue-100 dark:border-blue-900',
  },
  {
    card: 'border-violet-200 dark:border-violet-800',
    header: 'bg-violet-50 dark:bg-violet-900/20',
    pill: 'bg-violet-100 text-violet-800 dark:bg-violet-900/40 dark:text-violet-300',
    badge: 'bg-violet-500 text-white',
    addBtn: 'bg-violet-600 hover:bg-violet-700 text-white',
    ipBg: 'bg-violet-50 dark:bg-violet-900/10 border-violet-100 dark:border-violet-900',
  },
  {
    card: 'border-emerald-200 dark:border-emerald-800',
    header: 'bg-emerald-50 dark:bg-emerald-900/20',
    pill: 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/40 dark:text-emerald-300',
    badge: 'bg-emerald-500 text-white',
    addBtn: 'bg-emerald-600 hover:bg-emerald-700 text-white',
    ipBg: 'bg-emerald-50 dark:bg-emerald-900/10 border-emerald-100 dark:border-emerald-900',
  },
  {
    card: 'border-orange-200 dark:border-orange-800',
    header: 'bg-orange-50 dark:bg-orange-900/20',
    pill: 'bg-orange-100 text-orange-800 dark:bg-orange-900/40 dark:text-orange-300',
    badge: 'bg-orange-500 text-white',
    addBtn: 'bg-orange-600 hover:bg-orange-700 text-white',
    ipBg: 'bg-orange-50 dark:bg-orange-900/10 border-orange-100 dark:border-orange-900',
  },
  {
    card: 'border-rose-200 dark:border-rose-800',
    header: 'bg-rose-50 dark:bg-rose-900/20',
    pill: 'bg-rose-100 text-rose-800 dark:bg-rose-900/40 dark:text-rose-300',
    badge: 'bg-rose-500 text-white',
    addBtn: 'bg-rose-600 hover:bg-rose-700 text-white',
    ipBg: 'bg-rose-50 dark:bg-rose-900/10 border-rose-100 dark:border-rose-900',
  },
];

function paletteFor(index) {
  return POOL_PALETTES[index % POOL_PALETTES.length];
}

function isValidIP(value) {
  // Basic IPv4 or IPv6 check
  const ipv4 = /^(\d{1,3}\.){3}\d{1,3}(\/\d{1,2})?$/;
  const ipv6 = /^[0-9a-fA-F:]+$/;
  return ipv4.test(value) || ipv6.test(value);
}

export default function IPPoolPage() {
  const [pools, setPools] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  // New / edit pool modal
  const [poolModal, setPoolModal] = useState(false);
  const [editingPool, setEditingPool] = useState(null);
  const [poolForm, setPoolForm] = useState({ name: '', description: '' });
  const [savingPool, setSavingPool] = useState(false);

  // Per-pool inline "add IP" state: { [poolId]: string }
  const [addIPValues, setAddIPValues] = useState({});
  const [addingIP, setAddingIP] = useState({}); // { [poolId]: bool }
  const [ipError, setIPError] = useState({}); // { [poolId]: string }

  const token = localStorage.getItem('kumoui_token');
  const headers = {
    Authorization: `Bearer ${token}`,
    'Content-Type': 'application/json',
  };

  useEffect(() => {
    fetchPools();
  }, []);

  const fetchPools = async () => {
    setLoading(true);
    setError('');
    try {
      const res = await fetch('/api/ippools', { headers });
      if (res.status === 401) { window.location.href = '/login'; return; }
      if (!res.ok) throw new Error(`Server returned ${res.status}`);
      const data = await res.json();
      setPools(Array.isArray(data) ? data : []);
    } catch (e) {
      setError(e.message || 'Failed to load pools');
    } finally {
      setLoading(false);
    }
  };

  // --- Pool CRUD ---
  const openNewPool = () => {
    setEditingPool(null);
    setPoolForm({ name: '', description: '' });
    setPoolModal(true);
  };

  const openEditPool = (pool) => {
    setEditingPool(pool);
    setPoolForm({ name: pool.name, description: pool.description || '' });
    setPoolModal(true);
  };

  const closePoolModal = () => {
    setPoolModal(false);
    setEditingPool(null);
    setPoolForm({ name: '', description: '' });
  };

  const handleSavePool = async (e) => {
    e.preventDefault();
    setSavingPool(true);
    try {
      if (editingPool) {
        const res = await fetch(`/api/ippools/${editingPool.id}`, {
          method: 'PUT',
          headers,
          body: JSON.stringify(poolForm),
        });
        if (res.status === 401) { window.location.href = '/login'; return; }
        if (!res.ok) throw new Error('Failed to update pool');
      } else {
        const res = await fetch('/api/ippools', {
          method: 'POST',
          headers,
          body: JSON.stringify(poolForm),
        });
        if (res.status === 401) { window.location.href = '/login'; return; }
        if (!res.ok) throw new Error('Failed to create pool');
      }
      closePoolModal();
      fetchPools();
    } catch (e) {
      setError(e.message);
    } finally {
      setSavingPool(false);
    }
  };

  const deletePool = async (pool) => {
    if (!confirm(`Delete pool "${pool.name}" and all its members? This cannot be undone.`)) return;
    try {
      const res = await fetch(`/api/ippools/${pool.id}`, { method: 'DELETE', headers });
      if (res.status === 401) { window.location.href = '/login'; return; }
      if (!res.ok) throw new Error('Failed to delete pool');
      fetchPools();
    } catch (e) {
      setError(e.message);
    }
  };

  // --- Member CRUD ---
  const handleAddIP = async (pool) => {
    const value = (addIPValues[pool.id] || '').trim();
    if (!value) return;
    if (!isValidIP(value)) {
      setIPError({ ...ipError, [pool.id]: 'Enter a valid IP address (e.g. 1.2.3.4)' });
      return;
    }
    setIPError({ ...ipError, [pool.id]: '' });
    setAddingIP({ ...addingIP, [pool.id]: true });
    try {
      const res = await fetch(`/api/ippools/${pool.id}/members`, {
        method: 'POST',
        headers,
        body: JSON.stringify({ ip_value: value }),
      });
      if (res.status === 401) { window.location.href = '/login'; return; }
      if (!res.ok) throw new Error('Failed to add IP');
      setAddIPValues({ ...addIPValues, [pool.id]: '' });
      fetchPools();
    } catch (e) {
      setIPError({ ...ipError, [pool.id]: e.message });
    } finally {
      setAddingIP({ ...addingIP, [pool.id]: false });
    }
  };

  const handleRemoveMember = async (pool, member) => {
    try {
      const res = await fetch(`/api/ippools/${pool.id}/members/${member.id}`, {
        method: 'DELETE',
        headers,
      });
      if (res.status === 401) { window.location.href = '/login'; return; }
      if (!res.ok) throw new Error('Failed to remove member');
      fetchPools();
    } catch (e) {
      setError(e.message);
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">IP Pools</h1>
          <p className="text-muted-foreground">
            Group server IPs into named pools for different sending purposes.
          </p>
        </div>
        <div className="flex gap-2">
          <button
            onClick={fetchPools}
            className="flex items-center gap-2 h-10 px-3 rounded-md bg-secondary text-secondary-foreground hover:bg-secondary/80 text-sm font-medium transition-colors border"
            title="Refresh"
          >
            <RefreshCw className={cn('w-4 h-4', loading && 'animate-spin')} />
          </button>
          <button
            onClick={openNewPool}
            className="flex items-center gap-2 h-10 px-4 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-sm font-medium transition-colors shadow-sm"
          >
            <Plus className="w-4 h-4" /> New Pool
          </button>
        </div>
      </div>

      {/* Error */}
      {error && (
        <div className="bg-destructive/10 text-destructive p-4 rounded-md flex items-center gap-2 text-sm">
          <AlertCircle className="w-4 h-4 shrink-0" /> {error}
          <button
            onClick={() => setError('')}
            className="ml-auto p-1 hover:bg-destructive/10 rounded"
          >
            <X className="w-3 h-3" />
          </button>
        </div>
      )}

      {/* Loading */}
      {loading && (
        <div className="bg-card border rounded-xl p-12 text-center text-muted-foreground shadow-sm">
          Loading IP pools...
        </div>
      )}

      {/* Empty state */}
      {!loading && pools.length === 0 && (
        <div className="bg-card border rounded-xl p-16 text-center shadow-sm">
          <div className="mx-auto mb-4 p-4 bg-muted rounded-full w-fit">
            <Network className="w-10 h-10 text-muted-foreground" />
          </div>
          <h3 className="text-lg font-semibold mb-1">No Pools Configured</h3>
          <p className="text-muted-foreground mb-4">
            Create your first pool to organize IPs by purpose.
          </p>
          <button
            onClick={openNewPool}
            className="inline-flex items-center gap-2 px-4 py-2 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-sm font-medium"
          >
            <Plus className="w-4 h-4" /> Create Pool
          </button>
        </div>
      )}

      {/* Pool grid */}
      {!loading && pools.length > 0 && (
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-5">
          {pools.map((pool, idx) => {
            const palette = paletteFor(idx);
            const ipValue = addIPValues[pool.id] || '';
            const isAdding = addingIP[pool.id] || false;
            const memberCount = (pool.members || []).length;

            return (
              <div
                key={pool.id}
                className={cn(
                  'bg-card border-2 rounded-xl shadow-sm flex flex-col overflow-hidden',
                  palette.card
                )}
              >
                {/* Card header */}
                <div className={cn('px-5 py-4', palette.header)}>
                  <div className="flex items-start justify-between gap-2">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 flex-wrap">
                        <span className={cn('px-2.5 py-0.5 rounded-full text-xs font-bold tracking-wide truncate max-w-[180px]', palette.pill)}>
                          {pool.name}
                        </span>
                        <span className={cn('text-xs font-bold px-2 py-0.5 rounded-full', palette.badge)}>
                          {memberCount} {memberCount === 1 ? 'IP' : 'IPs'}
                        </span>
                      </div>
                      {pool.description && (
                        <p className="text-xs text-muted-foreground mt-1.5 leading-snug line-clamp-2">
                          {pool.description}
                        </p>
                      )}
                    </div>
                    <div className="flex gap-1 shrink-0">
                      <button
                        onClick={() => openEditPool(pool)}
                        className="p-1.5 rounded-md hover:bg-background/60 text-muted-foreground hover:text-foreground transition-colors"
                        title="Edit pool"
                      >
                        <Pencil className="w-3.5 h-3.5" />
                      </button>
                      <button
                        onClick={() => deletePool(pool)}
                        className="p-1.5 rounded-md hover:bg-destructive/10 text-muted-foreground hover:text-destructive transition-colors"
                        title="Delete pool"
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    </div>
                  </div>
                </div>

                {/* IP list */}
                <div className="flex-1 px-5 py-3 space-y-1.5">
                  {(pool.members || []).length === 0 ? (
                    <div className="flex flex-col items-center py-4 text-center">
                      <Server className="w-6 h-6 text-muted-foreground/40 mb-1" />
                      <p className="text-xs text-muted-foreground">No IPs in this pool</p>
                    </div>
                  ) : (
                    (pool.members || []).map((member) => (
                      <div
                        key={member.id}
                        className={cn(
                          'flex items-center justify-between px-3 py-1.5 rounded-md border text-sm font-mono group',
                          palette.ipBg
                        )}
                      >
                        <span className="truncate">{member.ip_value}</span>
                        <button
                          onClick={() => handleRemoveMember(pool, member)}
                          className="ml-2 p-0.5 rounded hover:bg-destructive/10 text-muted-foreground hover:text-destructive transition-colors opacity-0 group-hover:opacity-100 shrink-0"
                          title="Remove IP"
                        >
                          <X className="w-3.5 h-3.5" />
                        </button>
                      </div>
                    ))
                  )}
                </div>

                {/* Add IP form */}
                <div className="px-5 py-3 border-t">
                  <div className="flex gap-2">
                    <input
                      type="text"
                      placeholder="Add IP (e.g. 1.2.3.4)"
                      value={ipValue}
                      onChange={(e) =>
                        setAddIPValues({ ...addIPValues, [pool.id]: e.target.value })
                      }
                      onKeyDown={(e) => e.key === 'Enter' && (e.preventDefault(), handleAddIP(pool))}
                      className="flex-1 h-8 rounded-md border bg-background px-2.5 text-xs focus:outline-none focus:ring-2 focus:ring-ring font-mono"
                    />
                    <button
                      onClick={() => handleAddIP(pool)}
                      disabled={isAdding || !ipValue}
                      className={cn(
                        'h-8 px-3 rounded-md text-xs font-medium flex items-center gap-1 transition-colors disabled:opacity-50',
                        palette.addBtn
                      )}
                    >
                      {isAdding ? (
                        <Loader2 className="w-3 h-3 animate-spin" />
                      ) : (
                        <Plus className="w-3 h-3" />
                      )}
                      Add
                    </button>
                  </div>
                  {ipError[pool.id] && (
                    <p className="text-xs text-destructive mt-1">{ipError[pool.id]}</p>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* New / Edit Pool Modal */}
      {poolModal && (
        <div className="fixed inset-0 bg-background/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-card w-full max-w-md border rounded-xl shadow-xl overflow-hidden">
            {/* Modal header */}
            <div className="flex justify-between items-center px-6 py-4 border-b">
              <h3 className="text-lg font-semibold">
                {editingPool ? 'Edit Pool' : 'New IP Pool'}
              </h3>
              <button
                onClick={closePoolModal}
                className="p-2 hover:bg-muted rounded-md text-muted-foreground transition-colors"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            {/* Modal body */}
            <form onSubmit={handleSavePool}>
              <div className="px-6 py-5 space-y-4">
                <div className="space-y-1.5">
                  <label htmlFor="pool-name" className="block text-sm font-medium">
                    Pool Name <span className="text-destructive">*</span>
                  </label>
                  <input
                    id="pool-name"
                    required
                    placeholder="e.g. transactional, marketing, dedicated"
                    className="w-full h-9 rounded-md border bg-background px-3 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-ring"
                    value={poolForm.name}
                    onChange={(e) => setPoolForm({ ...poolForm, name: e.target.value })}
                  />
                </div>
                <div className="space-y-1.5">
                  <label htmlFor="pool-desc" className="block text-sm font-medium">
                    Description
                  </label>
                  <textarea
                    id="pool-desc"
                    rows={3}
                    placeholder="What is this pool used for?"
                    className="w-full rounded-md border bg-background px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-ring resize-none"
                    value={poolForm.description}
                    onChange={(e) => setPoolForm({ ...poolForm, description: e.target.value })}
                  />
                </div>
              </div>

              {/* Modal footer */}
              <div className="flex justify-end gap-2 px-6 py-4 border-t bg-muted/20">
                <button
                  type="button"
                  onClick={closePoolModal}
                  className="px-4 py-2 text-sm rounded-md hover:bg-muted border transition-colors"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={savingPool}
                  className="flex items-center gap-2 px-4 py-2 text-sm rounded-md bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
                >
                  {savingPool ? (
                    <Loader2 className="w-4 h-4 animate-spin" />
                  ) : (
                    <Save className="w-4 h-4" />
                  )}
                  {editingPool ? 'Save Changes' : 'Create Pool'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
