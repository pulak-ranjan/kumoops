import React, { useState, useEffect } from 'react';
import {
  Shield,
  Plus,
  Pencil,
  Trash2,
  Power,
  X,
  Save,
  Loader2,
  AlertCircle,
  RefreshCw,
  Info,
} from 'lucide-react';
import { cn } from '../lib/utils';

const PROVIDER_COLORS = {
  gmail: {
    border: 'border-l-blue-500',
    badge: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400',
  },
  yahoo: {
    border: 'border-l-orange-500',
    badge: 'bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400',
  },
  microsoft: {
    border: 'border-l-green-500',
    badge: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400',
  },
  outlook: {
    border: 'border-l-green-500',
    badge: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400',
  },
  hotmail: {
    border: 'border-l-green-500',
    badge: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400',
  },
};

function getProviderStyle(provider) {
  const key = (provider || '').toLowerCase();
  for (const [match, style] of Object.entries(PROVIDER_COLORS)) {
    if (key.includes(match)) return style;
  }
  return {
    border: 'border-l-gray-400',
    badge: 'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300',
  };
}

const EMPTY_FORM = {
  provider: '',
  pattern: '',
  max_message_rate: '',
  max_connection_rate: '',
  max_deliveries_per_conn: '',
  connection_limit: '',
  retry_schedule: '',
  notes: '',
  is_enabled: true,
};

export default function TrafficShapingPage() {
  const [rules, setRules] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [saving, setSaving] = useState(false);
  const [seeding, setSeeding] = useState(false);

  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState(null); // null = add, object = edit
  const [form, setForm] = useState(EMPTY_FORM);

  const token = localStorage.getItem('kumoui_token');
  const headers = {
    Authorization: `Bearer ${token}`,
    'Content-Type': 'application/json',
  };

  useEffect(() => {
    fetchRules();
  }, []);

  const fetchRules = async () => {
    setLoading(true);
    setError('');
    try {
      const res = await fetch('/api/shaping', { headers });
      if (res.status === 401) { window.location.href = '/login'; return; }
      if (!res.ok) throw new Error(`Server returned ${res.status}`);
      const data = await res.json();
      setRules(Array.isArray(data) ? data : []);
    } catch (e) {
      setError(e.message || 'Failed to load rules');
    } finally {
      setLoading(false);
    }
  };

  const openAdd = () => {
    setEditing(null);
    setForm(EMPTY_FORM);
    setModalOpen(true);
  };

  const openEdit = (rule) => {
    setEditing(rule);
    setForm({
      provider: rule.provider || '',
      pattern: rule.pattern || '',
      max_message_rate: rule.max_message_rate || '',
      max_connection_rate: rule.max_connection_rate || '',
      max_deliveries_per_conn: rule.max_deliveries_per_conn ?? '',
      connection_limit: rule.connection_limit ?? '',
      retry_schedule: rule.retry_schedule || '',
      notes: rule.notes || '',
      is_enabled: rule.is_enabled !== false,
    });
    setModalOpen(true);
  };

  const closeModal = () => {
    setModalOpen(false);
    setEditing(null);
    setForm(EMPTY_FORM);
  };

  const handleSave = async (e) => {
    e.preventDefault();
    setSaving(true);
    try {
      const payload = {
        ...form,
        max_deliveries_per_conn: form.max_deliveries_per_conn !== '' ? Number(form.max_deliveries_per_conn) : null,
        connection_limit: form.connection_limit !== '' ? Number(form.connection_limit) : null,
      };

      if (editing) {
        const res = await fetch(`/api/shaping/${editing.id}`, {
          method: 'PUT',
          headers,
          body: JSON.stringify(payload),
        });
        if (!res.ok) throw new Error('Failed to update rule');
      } else {
        const res = await fetch('/api/shaping', {
          method: 'POST',
          headers,
          body: JSON.stringify(payload),
        });
        if (!res.ok) throw new Error('Failed to create rule');
      }

      closeModal();
      fetchRules();
    } catch (e) {
      setError(e.message);
    } finally {
      setSaving(false);
    }
  };

  const toggleEnabled = async (rule) => {
    try {
      const res = await fetch(`/api/shaping/${rule.id}`, {
        method: 'PUT',
        headers,
        body: JSON.stringify({ ...rule, is_enabled: !rule.is_enabled }),
      });
      if (!res.ok) throw new Error('Failed to toggle rule');
      fetchRules();
    } catch (e) {
      setError(e.message);
    }
  };

  const deleteRule = async (rule) => {
    if (!confirm(`Delete rule for "${rule.provider}"? This cannot be undone.`)) return;
    try {
      const res = await fetch(`/api/shaping/${rule.id}`, { method: 'DELETE', headers });
      if (!res.ok) throw new Error('Failed to delete rule');
      fetchRules();
    } catch (e) {
      setError(e.message);
    }
  };

  const seedDefaults = async () => {
    if (!confirm('Reset all rules to defaults? This will overwrite your current configuration.')) return;
    setSeeding(true);
    try {
      const res = await fetch('/api/shaping/seed', { method: 'POST', headers });
      if (!res.ok) throw new Error('Failed to seed defaults');
      fetchRules();
    } catch (e) {
      setError(e.message);
    } finally {
      setSeeding(false);
    }
  };

  // Group rules by provider
  const grouped = rules.reduce((acc, rule) => {
    const key = rule.provider || 'Custom';
    if (!acc[key]) acc[key] = [];
    acc[key].push(rule);
    return acc;
  }, {});

  const field = (label, id, props = {}) => (
    <div className="space-y-1.5">
      <label htmlFor={id} className="block text-sm font-medium text-foreground">
        {label}
      </label>
      <input
        id={id}
        className="w-full h-9 rounded-md border bg-background px-3 py-1.5 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-ring"
        {...props}
        value={form[id] ?? ''}
        onChange={(e) => setForm({ ...form, [id]: e.target.value })}
      />
    </div>
  );

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Traffic Shaping Rules</h1>
          <p className="text-muted-foreground">
            Configure per-ISP delivery rate limits. Controls how fast KumoMTA delivers to each mail provider.
          </p>
        </div>
        <div className="flex gap-2 flex-wrap">
          <button
            onClick={fetchRules}
            className="flex items-center gap-2 h-10 px-3 rounded-md bg-secondary text-secondary-foreground hover:bg-secondary/80 text-sm font-medium transition-colors border"
            title="Refresh"
          >
            <RefreshCw className={cn('w-4 h-4', loading && 'animate-spin')} />
          </button>
          <button
            onClick={seedDefaults}
            disabled={seeding}
            className="flex items-center gap-2 h-10 px-4 rounded-md bg-secondary text-secondary-foreground hover:bg-secondary/80 text-sm font-medium transition-colors border"
          >
            {seeding ? (
              <Loader2 className="w-4 h-4 animate-spin" />
            ) : (
              <Shield className="w-4 h-4" />
            )}
            Seed Defaults
          </button>
          <button
            onClick={openAdd}
            className="flex items-center gap-2 h-10 px-4 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-sm font-medium transition-colors shadow-sm"
          >
            <Plus className="w-4 h-4" /> Add Rule
          </button>
        </div>
      </div>

      {/* Info banner */}
      <div className="flex items-start gap-3 p-4 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg text-sm text-blue-800 dark:text-blue-300">
        <Info className="w-4 h-4 mt-0.5 shrink-0" />
        <span>
          These rules are applied to KumoMTA&apos;s <code className="font-mono bg-blue-100 dark:bg-blue-900/40 px-1 rounded text-xs">init.lua</code> configuration. Changes take effect on next config apply.
        </span>
      </div>

      {/* Error */}
      {error && (
        <div className="bg-destructive/10 text-destructive p-4 rounded-md flex items-center gap-2 text-sm">
          <AlertCircle className="w-4 h-4 shrink-0" /> {error}
          <button onClick={() => setError('')} className="ml-auto p-1 hover:bg-destructive/10 rounded">
            <X className="w-3 h-3" />
          </button>
        </div>
      )}

      {/* Loading */}
      {loading && (
        <div className="bg-card border rounded-xl p-12 text-center text-muted-foreground shadow-sm">
          Loading rules...
        </div>
      )}

      {/* Empty state */}
      {!loading && rules.length === 0 && (
        <div className="bg-card border rounded-xl p-16 text-center shadow-sm">
          <div className="mx-auto mb-4 p-4 bg-muted rounded-full w-fit">
            <Shield className="w-10 h-10 text-muted-foreground" />
          </div>
          <h3 className="text-lg font-semibold mb-1">No Rules Configured</h3>
          <p className="text-muted-foreground mb-4">
            Add rules manually or seed the defaults to get started.
          </p>
          <button
            onClick={seedDefaults}
            className="inline-flex items-center gap-2 px-4 py-2 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-sm font-medium"
          >
            <Shield className="w-4 h-4" /> Seed Defaults
          </button>
        </div>
      )}

      {/* Grouped rule tables */}
      {!loading && Object.entries(grouped).map(([provider, providerRules]) => {
        const style = getProviderStyle(provider);
        return (
          <div key={provider} className="bg-card border rounded-xl overflow-hidden shadow-sm">
            {/* Provider header */}
            <div className={cn('border-l-4 px-6 py-3 bg-muted/40 flex items-center gap-3', style.border)}>
              <span className={cn('px-2.5 py-0.5 rounded-full text-xs font-semibold', style.badge)}>
                {provider}
              </span>
              <span className="text-sm text-muted-foreground">
                {providerRules.length} {providerRules.length === 1 ? 'rule' : 'rules'}
              </span>
            </div>

            <div className="overflow-x-auto">
              <table className="w-full text-sm text-left">
                <thead className="bg-muted/30 text-muted-foreground uppercase text-xs">
                  <tr>
                    <th className="px-4 py-3 font-medium">MX Pattern</th>
                    <th className="px-4 py-3 font-medium">Msg Rate</th>
                    <th className="px-4 py-3 font-medium">Conn Rate</th>
                    <th className="px-4 py-3 font-medium">Deliveries/Conn</th>
                    <th className="px-4 py-3 font-medium">Conn Limit</th>
                    <th className="px-4 py-3 font-medium">Status</th>
                    <th className="px-4 py-3 font-medium text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {providerRules.map((rule) => (
                    <tr key={rule.id} className="hover:bg-muted/40 transition-colors group">
                      <td className="px-4 py-3 font-mono text-xs">{rule.pattern || '-'}</td>
                      <td className="px-4 py-3 font-mono text-xs">{rule.max_message_rate || '-'}</td>
                      <td className="px-4 py-3 font-mono text-xs">{rule.max_connection_rate || '-'}</td>
                      <td className="px-4 py-3 text-center">{rule.max_deliveries_per_conn ?? '-'}</td>
                      <td className="px-4 py-3 text-center">{rule.connection_limit ?? '-'}</td>
                      <td className="px-4 py-3">
                        {rule.is_enabled !== false ? (
                          <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400">
                            Active
                          </span>
                        ) : (
                          <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400">
                            Disabled
                          </span>
                        )}
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex items-center justify-end gap-1">
                          <button
                            onClick={() => openEdit(rule)}
                            className="p-1.5 rounded-md hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
                            title="Edit rule"
                          >
                            <Pencil className="w-4 h-4" />
                          </button>
                          <button
                            onClick={() => toggleEnabled(rule)}
                            className={cn(
                              'p-1.5 rounded-md transition-colors',
                              rule.is_enabled !== false
                                ? 'hover:bg-amber-100 dark:hover:bg-amber-900/20 text-amber-600'
                                : 'hover:bg-green-100 dark:hover:bg-green-900/20 text-green-600'
                            )}
                            title={rule.is_enabled !== false ? 'Disable rule' : 'Enable rule'}
                          >
                            <Power className="w-4 h-4" />
                          </button>
                          <button
                            onClick={() => deleteRule(rule)}
                            className="p-1.5 rounded-md hover:bg-destructive/10 text-muted-foreground hover:text-destructive transition-colors opacity-0 group-hover:opacity-100"
                            title="Delete rule"
                          >
                            <Trash2 className="w-4 h-4" />
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        );
      })}

      {/* Add / Edit Modal */}
      {modalOpen && (
        <div className="fixed inset-0 bg-background/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-card w-full max-w-2xl border rounded-xl shadow-xl overflow-hidden">
            {/* Modal header */}
            <div className="flex justify-between items-center px-6 py-4 border-b">
              <div>
                <h3 className="text-lg font-semibold">
                  {editing ? 'Edit Traffic Shaping Rule' : 'Add Traffic Shaping Rule'}
                </h3>
                {editing && (
                  <p className="text-xs text-muted-foreground mt-0.5">{editing.provider}</p>
                )}
              </div>
              <button
                onClick={closeModal}
                className="p-2 hover:bg-muted rounded-md text-muted-foreground transition-colors"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            {/* Modal body */}
            <form onSubmit={handleSave}>
              <div className="px-6 py-5 space-y-5 max-h-[70vh] overflow-y-auto">
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  {field('Provider', 'provider', { placeholder: 'e.g. Gmail', required: true })}
                  {field('MX Pattern', 'pattern', { placeholder: 'e.g. google.com', required: true })}
                  {field('Max Message Rate', 'max_message_rate', { placeholder: 'e.g. 50/h' })}
                  {field('Max Connection Rate', 'max_connection_rate', { placeholder: 'e.g. 5/min' })}
                  {field('Max Deliveries per Connection', 'max_deliveries_per_conn', {
                    type: 'number',
                    min: 1,
                    placeholder: 'e.g. 100',
                  })}
                  {field('Connection Limit', 'connection_limit', {
                    type: 'number',
                    min: 1,
                    placeholder: 'e.g. 10',
                  })}
                  <div className="sm:col-span-2">
                    {field('Retry Schedule', 'retry_schedule', { placeholder: 'e.g. 5m,15m,30m,1h' })}
                  </div>
                </div>

                <div className="space-y-1.5">
                  <label htmlFor="notes" className="block text-sm font-medium text-foreground">
                    Notes
                  </label>
                  <textarea
                    id="notes"
                    rows={3}
                    placeholder="Optional notes about this rule..."
                    className="w-full rounded-md border bg-background px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-ring resize-none"
                    value={form.notes}
                    onChange={(e) => setForm({ ...form, notes: e.target.value })}
                  />
                </div>

                <div className="flex items-center gap-3 p-3 bg-muted/40 rounded-md border">
                  <input
                    type="checkbox"
                    id="is_enabled"
                    checked={form.is_enabled}
                    onChange={(e) => setForm({ ...form, is_enabled: e.target.checked })}
                    className="h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary cursor-pointer"
                  />
                  <label htmlFor="is_enabled" className="text-sm font-medium cursor-pointer select-none">
                    Enable this rule
                  </label>
                  <span className="text-xs text-muted-foreground ml-auto">
                    Disabled rules are stored but not applied
                  </span>
                </div>
              </div>

              {/* Modal footer */}
              <div className="flex justify-end gap-2 px-6 py-4 border-t bg-muted/20">
                <button
                  type="button"
                  onClick={closeModal}
                  className="px-4 py-2 text-sm rounded-md hover:bg-muted border transition-colors"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={saving}
                  className="flex items-center gap-2 px-4 py-2 text-sm rounded-md bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
                >
                  {saving ? (
                    <Loader2 className="w-4 h-4 animate-spin" />
                  ) : (
                    <Save className="w-4 h-4" />
                  )}
                  {editing ? 'Save Changes' : 'Create Rule'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
