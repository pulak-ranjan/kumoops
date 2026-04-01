import React, { useState, useEffect, useRef, useCallback } from 'react';
import {
  Bell,
  BellOff,
  Plus,
  Trash2,
  Edit2,
  Play,
  RefreshCw,
  X,
  CheckCircle2,
  AlertTriangle,
  Loader2,
  Globe,
  Mail,
  Hash,
  History,
  ToggleLeft,
  ToggleRight,
  Clock
} from 'lucide-react';
import { cn } from '../lib/utils';

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------
const TYPE_STYLES = {
  bounce_rate:    'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400',
  delivery_rate:  'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400',
  queue_depth:    'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400',
  blacklist:      'bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400',
};

const TYPE_LABELS = {
  bounce_rate:   'Bounce Rate',
  delivery_rate: 'Delivery Rate',
  queue_depth:   'Queue Depth',
  blacklist:     'Blacklist',
};

const CHANNEL_ICON = {
  slack:   Hash,
  webhook: Globe,
  email:   Mail,
};

const CHANNEL_LABELS = {
  slack:   'Slack',
  webhook: 'Webhook',
  email:   'Email',
};

const STATUS_STYLES = {
  sent:   'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400',
  failed: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400',
};

const EMPTY_RULE = {
  name: '',
  type: 'bounce_rate',
  operator: 'gt',
  threshold: '',
  channel: 'slack',
  destination: '',
  cooldown_min: 60,
  is_enabled: true,
};

function TypeBadge({ type }) {
  return (
    <span className={cn(
      'inline-flex items-center px-2 py-0.5 rounded text-xs font-medium',
      TYPE_STYLES[type] || 'bg-gray-100 text-gray-700 dark:bg-gray-800/60 dark:text-gray-400'
    )}>
      {TYPE_LABELS[type] || type}
    </span>
  );
}

function StatusBadge({ status }) {
  return (
    <span className={cn(
      'inline-flex items-center px-2 py-0.5 rounded text-xs font-medium capitalize',
      STATUS_STYLES[status] || 'bg-gray-100 text-gray-700 dark:bg-gray-800/60 dark:text-gray-400'
    )}>
      {status}
    </span>
  );
}

function ChannelIcon({ channel, className }) {
  const Icon = CHANNEL_ICON[channel] || Globe;
  return <Icon className={className} />;
}

// ---------------------------------------------------------------------------
// Main Page
// ---------------------------------------------------------------------------
export default function AlertsPage() {
  const [tab, setTab]           = useState('rules'); // 'rules' | 'history'
  const [rules, setRules]       = useState([]);
  const [events, setEvents]     = useState([]);
  const [loadingRules, setLoadingRules]   = useState(true);
  const [loadingEvents, setLoadingEvents] = useState(false);
  const [toast, setToast]       = useState(null);
  const [autoRefresh, setAutoRefresh]     = useState(false);
  const autoRefreshRef          = useRef(null);

  // Modal state
  const [showModal, setShowModal]   = useState(false);
  const [editingRule, setEditingRule] = useState(null); // null = new

  const token   = localStorage.getItem('kumoui_token');
  const headers = { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' };

  const showToast = (type, msg) => {
    setToast({ type, msg });
    setTimeout(() => setToast(null), 4000);
  };

  // ------ Rules ------
  const fetchRules = useCallback(async () => {
    setLoadingRules(true);
    try {
      const res = await fetch('/api/alerts/rules', { headers });
      if (res.status === 401) { window.location.href = '/login'; return; }
      if (!res.ok) throw new Error('Failed to fetch rules');
      const data = await res.json();
      setRules(Array.isArray(data) ? data : []);
    } catch (e) {
      console.error(e);
      setRules([]);
      showToast('error', 'Failed to load alert rules.');
    }
    setLoadingRules(false);
  }, []); // eslint-disable-line

  // ------ Events ------
  const fetchEvents = useCallback(async () => {
    setLoadingEvents(true);
    try {
      const res = await fetch('/api/alerts/events?limit=50', { headers });
      if (res.status === 401) { window.location.href = '/login'; return; }
      if (!res.ok) throw new Error('Failed to fetch events');
      const data = await res.json();
      setEvents(Array.isArray(data) ? data : []);
    } catch (e) {
      console.error(e);
      setEvents([]);
    }
    setLoadingEvents(false);
  }, []); // eslint-disable-line

  useEffect(() => { fetchRules(); }, []); // eslint-disable-line

  useEffect(() => {
    if (tab === 'history') fetchEvents();
  }, [tab]); // eslint-disable-line

  // Auto-refresh for history tab
  useEffect(() => {
    if (autoRefresh && tab === 'history') {
      autoRefreshRef.current = setInterval(fetchEvents, 30000);
    } else {
      clearInterval(autoRefreshRef.current);
    }
    return () => clearInterval(autoRefreshRef.current);
  }, [autoRefresh, tab]); // eslint-disable-line

  const deleteRule = async (id) => {
    if (!confirm('Delete this alert rule?')) return;
    try {
      const res = await fetch(`/api/alerts/rules/${id}`, { method: 'DELETE', headers });
      if (res.status === 401) { window.location.href = '/login'; return; }
      if (!res.ok) throw new Error('Delete failed');
      showToast('success', 'Alert rule deleted.');
      fetchRules();
    } catch (e) {
      showToast('error', e.message || 'Failed to delete rule.');
    }
  };

  const toggleRule = async (rule) => {
    try {
      const res = await fetch(`/api/alerts/rules/${rule.id}`, {
        method: 'PUT',
        headers,
        body: JSON.stringify({ ...rule, is_enabled: !rule.is_enabled }),
      });
      if (res.status === 401) { window.location.href = '/login'; return; }
      if (!res.ok) throw new Error('Toggle failed');
      fetchRules();
    } catch (e) {
      showToast('error', 'Failed to update rule status.');
    }
  };

  const testRule = async (id) => {
    try {
      const res = await fetch(`/api/alerts/test/${id}`, { method: 'POST', headers });
      if (res.status === 401) { window.location.href = '/login'; return; }
      if (!res.ok) throw new Error('Test failed');
      showToast('success', 'Test alert fired. Check your channel.');
    } catch (e) {
      showToast('error', 'Failed to fire test alert.');
    }
  };

  const openNew = () => { setEditingRule(null); setShowModal(true); };
  const openEdit = (rule) => { setEditingRule(rule); setShowModal(true); };

  const handleSaved = () => {
    setShowModal(false);
    fetchRules();
  };

  const formatDate = (d) => d ? new Date(d).toLocaleString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' }) : 'Never';

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Real-Time Alerts</h1>
          <p className="text-muted-foreground">Monitor delivery metrics and get notified when thresholds are crossed.</p>
        </div>
        {tab === 'rules' && (
          <button
            onClick={openNew}
            className="flex items-center gap-2 h-10 px-4 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-sm font-medium transition-colors shadow-sm"
          >
            <Plus className="w-4 h-4" /> New Alert
          </button>
        )}
        {tab === 'history' && (
          <div className="flex items-center gap-2">
            <button
              onClick={fetchEvents}
              disabled={loadingEvents}
              className="flex items-center gap-2 h-10 px-4 rounded-md bg-secondary text-secondary-foreground hover:bg-secondary/80 text-sm font-medium transition-colors"
            >
              <RefreshCw className={cn('w-4 h-4', loadingEvents && 'animate-spin')} /> Refresh
            </button>
            <button
              onClick={() => setAutoRefresh(v => !v)}
              className={cn(
                'flex items-center gap-2 h-10 px-4 rounded-md text-sm font-medium transition-colors border',
                autoRefresh
                  ? 'bg-green-500/10 text-green-600 border-green-500/30 hover:bg-green-500/20'
                  : 'bg-background hover:bg-muted'
              )}
            >
              {autoRefresh
                ? <><ToggleRight className="w-4 h-4" /> Auto-refresh on</>
                : <><ToggleLeft className="w-4 h-4" /> Auto-refresh</>}
            </button>
          </div>
        )}
      </div>

      {/* Toast */}
      {toast && (
        <div className={cn(
          'p-4 rounded-md text-sm font-medium flex items-center gap-2',
          toast.type === 'error'
            ? 'bg-destructive/10 text-destructive'
            : 'bg-green-500/10 text-green-600 dark:text-green-400'
        )}>
          {toast.type === 'error'
            ? <AlertTriangle className="w-4 h-4 shrink-0" />
            : <CheckCircle2 className="w-4 h-4 shrink-0" />}
          {toast.msg}
        </div>
      )}

      {/* Tabs */}
      <div className="flex gap-1 p-1 bg-muted rounded-lg w-fit">
        <TabButton active={tab === 'rules'} onClick={() => setTab('rules')} icon={<Bell className="w-4 h-4" />}>
          Alert Rules
        </TabButton>
        <TabButton active={tab === 'history'} onClick={() => setTab('history')} icon={<History className="w-4 h-4" />}>
          Alert History
        </TabButton>
      </div>

      {/* Rules Tab */}
      {tab === 'rules' && (
        <RulesTab
          rules={rules}
          loading={loadingRules}
          onDelete={deleteRule}
          onToggle={toggleRule}
          onTest={testRule}
          onEdit={openEdit}
          onNew={openNew}
          formatDate={formatDate}
        />
      )}

      {/* History Tab */}
      {tab === 'history' && (
        <HistoryTab
          events={events}
          loading={loadingEvents}
          autoRefresh={autoRefresh}
          formatDate={formatDate}
        />
      )}

      {/* Rule Modal */}
      {showModal && (
        <RuleModal
          headers={headers}
          editingRule={editingRule}
          onClose={() => setShowModal(false)}
          onSaved={handleSaved}
          showToast={showToast}
        />
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Tab button
// ---------------------------------------------------------------------------
function TabButton({ active, onClick, icon, children }) {
  return (
    <button
      onClick={onClick}
      className={cn(
        'flex items-center gap-2 px-4 py-2 rounded-md text-sm font-medium transition-colors',
        active
          ? 'bg-background text-foreground shadow-sm'
          : 'text-muted-foreground hover:text-foreground'
      )}
    >
      {icon} {children}
    </button>
  );
}

// ---------------------------------------------------------------------------
// Rules Tab
// ---------------------------------------------------------------------------
function RulesTab({ rules, loading, onDelete, onToggle, onTest, onEdit, onNew, formatDate }) {
  if (loading) {
    return (
      <div className="p-12 text-center text-muted-foreground flex items-center justify-center gap-2">
        <Loader2 className="w-5 h-5 animate-spin" /> Loading alert rules...
      </div>
    );
  }

  if (rules.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center p-16 text-center bg-card border rounded-xl shadow-sm">
        <div className="p-4 bg-amber-100 dark:bg-amber-900/20 rounded-full mb-4">
          <BellOff className="w-12 h-12 text-amber-600 dark:text-amber-400" />
        </div>
        <h3 className="text-xl font-semibold mb-1">No Alert Rules</h3>
        <p className="text-muted-foreground text-sm max-w-sm mb-6">
          Create alert rules to get notified when delivery metrics cross your defined thresholds.
        </p>
        <button
          onClick={onNew}
          className="flex items-center gap-2 h-10 px-4 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-sm font-medium transition-colors"
        >
          <Plus className="w-4 h-4" /> Create First Alert
        </button>
      </div>
    );
  }

  return (
    <div className="grid gap-4">
      {rules.map(rule => (
        <RuleCard
          key={rule.id}
          rule={rule}
          onDelete={onDelete}
          onToggle={onToggle}
          onTest={onTest}
          onEdit={onEdit}
          formatDate={formatDate}
        />
      ))}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Rule Card
// ---------------------------------------------------------------------------
function RuleCard({ rule, onDelete, onToggle, onTest, onEdit, formatDate }) {
  const [testing, setTesting] = useState(false);

  const handleTest = async () => {
    setTesting(true);
    await onTest(rule.id);
    setTesting(false);
  };

  const isRate = rule.type === 'bounce_rate' || rule.type === 'delivery_rate';
  const unit   = isRate ? '%' : rule.type === 'queue_depth' ? ' msgs' : '';
  const opLabel = rule.operator === 'gt' ? '>' : '<';

  return (
    <div className={cn(
      'bg-card border rounded-xl p-5 shadow-sm transition-all hover:shadow-md',
      !rule.is_enabled && 'opacity-60'
    )}>
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        {/* Left: info */}
        <div className="flex items-start gap-4 min-w-0">
          <div className={cn(
            'p-2.5 rounded-lg shrink-0',
            rule.is_enabled ? 'bg-primary/10 text-primary' : 'bg-muted text-muted-foreground'
          )}>
            <Bell className="w-5 h-5" />
          </div>
          <div className="min-w-0">
            <div className="flex items-center gap-2 flex-wrap">
              <span className="font-semibold text-base">{rule.name}</span>
              <TypeBadge type={rule.type} />
              {!rule.is_enabled && (
                <span className="text-xs px-2 py-0.5 rounded bg-muted text-muted-foreground">Disabled</span>
              )}
            </div>
            <div className="mt-1.5 flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-muted-foreground">
              <span>Threshold: <strong className="text-foreground">{opLabel} {rule.threshold}{unit}</strong></span>
              <span className="flex items-center gap-1">
                <ChannelIcon channel={rule.channel} className="w-3.5 h-3.5" />
                {CHANNEL_LABELS[rule.channel] || rule.channel}
              </span>
              <span className="flex items-center gap-1">
                <Clock className="w-3.5 h-3.5" />
                Cooldown: {rule.cooldown_min}m
              </span>
              {rule.last_fired && (
                <span className="text-xs">Last fired: {formatDate(rule.last_fired)}</span>
              )}
            </div>
            {rule.destination && (
              <div className="mt-1 text-xs text-muted-foreground truncate max-w-xs" title={rule.destination}>
                {rule.destination}
              </div>
            )}
          </div>
        </div>

        {/* Right: actions */}
        <div className="flex items-center gap-2 shrink-0 flex-wrap sm:flex-nowrap">
          <button
            onClick={() => onToggle(rule)}
            className={cn(
              'flex items-center gap-1.5 h-9 px-3 rounded-md text-sm font-medium transition-colors border',
              rule.is_enabled
                ? 'text-green-600 border-green-500/30 bg-green-500/10 hover:bg-green-500/20'
                : 'text-muted-foreground border-border bg-background hover:bg-muted'
            )}
            title={rule.is_enabled ? 'Disable rule' : 'Enable rule'}
          >
            {rule.is_enabled
              ? <ToggleRight className="w-4 h-4" />
              : <ToggleLeft className="w-4 h-4" />}
            {rule.is_enabled ? 'Enabled' : 'Disabled'}
          </button>

          <button
            onClick={handleTest}
            disabled={testing}
            className="flex items-center gap-1.5 h-9 px-3 rounded-md border bg-background hover:bg-muted text-sm font-medium transition-colors disabled:opacity-50"
            title="Send a test notification"
          >
            {testing ? <Loader2 className="w-4 h-4 animate-spin" /> : <Play className="w-4 h-4" />}
            Test
          </button>

          <button
            onClick={() => onEdit(rule)}
            className="p-2 rounded-md border bg-background hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
            title="Edit rule"
          >
            <Edit2 className="w-4 h-4" />
          </button>

          <button
            onClick={() => onDelete(rule.id)}
            className="p-2 rounded-md border bg-background hover:bg-destructive/10 text-muted-foreground hover:text-destructive transition-colors"
            title="Delete rule"
          >
            <Trash2 className="w-4 h-4" />
          </button>
        </div>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// History Tab
// ---------------------------------------------------------------------------
function HistoryTab({ events, loading, autoRefresh, formatDate }) {
  if (loading && events.length === 0) {
    return (
      <div className="p-12 text-center text-muted-foreground flex items-center justify-center gap-2">
        <Loader2 className="w-5 h-5 animate-spin" /> Loading alert history...
      </div>
    );
  }

  return (
    <div className="bg-card border rounded-xl overflow-hidden shadow-sm">
      {autoRefresh && (
        <div className="px-4 py-2 bg-green-500/10 border-b border-green-500/20 text-xs text-green-600 dark:text-green-400 flex items-center gap-1.5">
          <RefreshCw className="w-3 h-3" /> Auto-refreshing every 30 seconds
        </div>
      )}
      {events.length === 0 ? (
        <div className="flex flex-col items-center justify-center p-16 text-center">
          <div className="p-4 bg-muted rounded-full mb-4">
            <History className="w-10 h-10 text-muted-foreground" />
          </div>
          <h3 className="text-lg font-semibold mb-1">No Alert Events</h3>
          <p className="text-muted-foreground text-sm">Alert events will appear here when rules are triggered.</p>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm text-left">
            <thead className="bg-muted/50 text-muted-foreground uppercase text-xs">
              <tr>
                <th className="px-4 py-3 font-medium">Rule Name</th>
                <th className="px-4 py-3 font-medium">Type</th>
                <th className="px-4 py-3 font-medium">Value</th>
                <th className="px-4 py-3 font-medium">Threshold</th>
                <th className="px-4 py-3 font-medium">Channel</th>
                <th className="px-4 py-3 font-medium">Status</th>
                <th className="px-4 py-3 font-medium whitespace-nowrap">Time</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {events.map(ev => (
                <tr key={ev.id} className="hover:bg-muted/50 transition-colors group">
                  <td className="px-4 py-3">
                    <div className="font-medium">{ev.rule_name || '-'}</div>
                    {ev.message && (
                      <div className="text-xs text-muted-foreground truncate max-w-[200px]" title={ev.message}>
                        {ev.message}
                      </div>
                    )}
                  </td>
                  <td className="px-4 py-3">
                    <TypeBadge type={ev.type} />
                  </td>
                  <td className="px-4 py-3 font-mono text-xs">
                    {ev.value !== undefined && ev.value !== null ? ev.value : '-'}
                  </td>
                  <td className="px-4 py-3 font-mono text-xs text-muted-foreground">
                    {ev.threshold !== undefined && ev.threshold !== null ? ev.threshold : '-'}
                  </td>
                  <td className="px-4 py-3">
                    <span className="flex items-center gap-1.5 text-xs text-muted-foreground">
                      <ChannelIcon channel={ev.channel} className="w-3.5 h-3.5" />
                      {CHANNEL_LABELS[ev.channel] || ev.channel || '-'}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <div>
                      <StatusBadge status={ev.status} />
                      {ev.error && (
                        <div className="text-xs text-destructive mt-1 max-w-[150px] truncate" title={ev.error}>
                          {ev.error}
                        </div>
                      )}
                    </div>
                  </td>
                  <td className="px-4 py-3 text-muted-foreground whitespace-nowrap text-xs">
                    {formatDate(ev.fired_at)}
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

// ---------------------------------------------------------------------------
// Rule Modal (create / edit)
// ---------------------------------------------------------------------------
function RuleModal({ headers, editingRule, onClose, onSaved, showToast }) {
  const [form, setForm] = useState(editingRule ? { ...editingRule } : { ...EMPTY_RULE });
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    const handler = (e) => { if (e.key === 'Escape') onClose(); };
    document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }, [onClose]);

  const isEdit = !!editingRule;

  const isRate = form.type === 'bounce_rate' || form.type === 'delivery_rate';
  const thresholdUnit = isRate ? '%' : form.type === 'queue_depth' ? 'messages' : '';
  const thresholdHint = isRate
    ? 'Percentage value (0–100)'
    : form.type === 'queue_depth'
    ? 'Number of messages in queue'
    : 'Threshold value';

  const destinationLabel = form.channel === 'email'
    ? 'Email Address'
    : form.channel === 'slack'
    ? 'Slack Webhook URL'
    : 'Webhook URL';
  const destinationPlaceholder = form.channel === 'email'
    ? 'alerts@example.com'
    : form.channel === 'slack'
    ? 'https://hooks.slack.com/services/...'
    : 'https://your-webhook-url.com/hook';

  const submit = async (e) => {
    e.preventDefault();
    if (!form.name.trim() || form.threshold === '' || !form.destination.trim()) return;
    setBusy(true);
    try {
      const payload = {
        ...form,
        threshold: Number(form.threshold),
        cooldown_min: Number(form.cooldown_min),
      };
      const url    = isEdit ? `/api/alerts/rules/${editingRule.id}` : '/api/alerts/rules';
      const method = isEdit ? 'PUT' : 'POST';
      const res    = await fetch(url, { method, headers, body: JSON.stringify(payload) });
      if (!res.ok) {
        const err = await res.json().catch(() => ({}));
        throw new Error(err.detail || (isEdit ? 'Update failed' : 'Create failed'));
      }
      showToast('success', isEdit ? 'Alert rule updated.' : 'Alert rule created.');
      onSaved();
    } catch (e) {
      showToast('error', e.message || 'Failed to save rule.');
    }
    setBusy(false);
  };

  const set = (field, val) => setForm(f => ({ ...f, [field]: val }));

  return (
    <div className="fixed inset-0 bg-background/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
      <div className="bg-card w-full max-w-lg border rounded-xl shadow-lg">
        <div className="flex items-center justify-between p-6 border-b">
          <h3 className="text-lg font-semibold flex items-center gap-2">
            <Bell className="w-5 h-5 text-primary" />
            {isEdit ? 'Edit Alert Rule' : 'New Alert Rule'}
          </h3>
          <button onClick={onClose} className="p-1 rounded-md hover:bg-muted transition-colors">
            <X className="w-4 h-4" />
          </button>
        </div>

        <form onSubmit={submit} className="p-6 space-y-5">
          {/* Name */}
          <div className="space-y-2">
            <label className="text-sm font-medium">Name <span className="text-destructive">*</span></label>
            <input
              type="text"
              required
              autoFocus
              value={form.name}
              onChange={e => set('name', e.target.value)}
              placeholder="e.g. High Bounce Rate Alert"
              className="w-full h-10 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring"
            />
          </div>

          {/* Type + Operator */}
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">Type <span className="text-destructive">*</span></label>
              <select
                value={form.type}
                onChange={e => set('type', e.target.value)}
                className="w-full h-10 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring"
              >
                <option value="bounce_rate">Bounce Rate</option>
                <option value="delivery_rate">Delivery Rate</option>
                <option value="queue_depth">Queue Depth</option>
                <option value="blacklist">Blacklist</option>
              </select>
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">Operator</label>
              <select
                value={form.operator}
                onChange={e => set('operator', e.target.value)}
                className="w-full h-10 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring"
              >
                <option value="gt">Greater than (&gt;)</option>
                <option value="lt">Less than (&lt;)</option>
              </select>
            </div>
          </div>

          {/* Threshold */}
          <div className="space-y-2">
            <label className="text-sm font-medium">
              Threshold {thresholdUnit && <span className="text-muted-foreground font-normal">({thresholdUnit})</span>}
              <span className="text-destructive"> *</span>
            </label>
            <input
              type="number"
              required
              min="0"
              step="any"
              value={form.threshold}
              onChange={e => set('threshold', e.target.value)}
              placeholder={isRate ? '5' : '1000'}
              className="w-full h-10 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring"
            />
            <p className="text-xs text-muted-foreground">{thresholdHint}</p>
          </div>

          {/* Channel + Destination */}
          <div className="space-y-2">
            <label className="text-sm font-medium">Notification Channel</label>
            <div className="grid grid-cols-3 gap-2">
              {['slack', 'webhook', 'email'].map(ch => {
                const Icon = CHANNEL_ICON[ch];
                return (
                  <button
                    key={ch}
                    type="button"
                    onClick={() => set('channel', ch)}
                    className={cn(
                      'flex flex-col items-center gap-1.5 py-3 rounded-md border text-sm font-medium transition-colors',
                      form.channel === ch
                        ? 'border-primary bg-primary/10 text-primary'
                        : 'border-border bg-background hover:bg-muted text-muted-foreground hover:text-foreground'
                    )}
                  >
                    <Icon className="w-4 h-4" />
                    {CHANNEL_LABELS[ch]}
                  </button>
                );
              })}
            </div>
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium">{destinationLabel} <span className="text-destructive">*</span></label>
            <input
              type={form.channel === 'email' ? 'email' : 'url'}
              required
              value={form.destination}
              onChange={e => set('destination', e.target.value)}
              placeholder={destinationPlaceholder}
              className="w-full h-10 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring"
            />
          </div>

          {/* Cooldown + Enabled */}
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">Cooldown (minutes)</label>
              <input
                type="number"
                min="1"
                value={form.cooldown_min}
                onChange={e => set('cooldown_min', e.target.value)}
                className="w-full h-10 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring"
              />
              <p className="text-xs text-muted-foreground">Min time between repeat alerts.</p>
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">Status</label>
              <div
                className="flex items-center gap-3 h-10 p-3 rounded-md border bg-background cursor-pointer select-none"
                onClick={() => set('is_enabled', !form.is_enabled)}
              >
                <input
                  type="checkbox"
                  checked={form.is_enabled}
                  onChange={() => {}}
                  className="h-4 w-4 rounded border-input"
                />
                <span className="text-sm">{form.is_enabled ? 'Enabled' : 'Disabled'}</span>
              </div>
            </div>
          </div>

          {/* Actions */}
          <div className="flex justify-end gap-2 pt-2 border-t">
            <button type="button" onClick={onClose} className="px-4 py-2 text-sm rounded-md hover:bg-muted transition-colors">
              Cancel
            </button>
            <button
              type="submit"
              disabled={busy}
              className="flex items-center gap-2 px-4 py-2 text-sm rounded-md bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
            >
              {busy ? <Loader2 className="w-4 h-4 animate-spin" /> : <CheckCircle2 className="w-4 h-4" />}
              {isEdit ? 'Save Changes' : 'Create Rule'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
