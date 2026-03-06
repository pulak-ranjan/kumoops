import React, { useState, useEffect, useRef, useCallback } from 'react';
import {
  Shield,
  Plus,
  Upload,
  Download,
  Trash2,
  Search,
  ChevronLeft,
  ChevronRight,
  X,
  CheckCircle2,
  AlertTriangle,
  Loader2,
  FileText,
  Users
} from 'lucide-react';
import { cn } from '../lib/utils';

const REASON_STYLES = {
  hard_bounce:     'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400',
  spam_complaint:  'bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400',
  manual:          'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400',
  unsubscribe:     'bg-gray-100 text-gray-700 dark:bg-gray-800/60 dark:text-gray-400',
  import:          'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400',
};

const REASON_LABEL = {
  hard_bounce:    'Hard Bounce',
  spam_complaint: 'Spam Complaint',
  manual:         'Manual',
  unsubscribe:    'Unsubscribe',
  import:         'Import',
};

function ReasonBadge({ reason }) {
  return (
    <span className={cn(
      'inline-flex items-center px-2 py-0.5 rounded text-xs font-medium capitalize',
      REASON_STYLES[reason] || 'bg-gray-100 text-gray-700 dark:bg-gray-800/60 dark:text-gray-400'
    )}>
      {REASON_LABEL[reason] || reason || 'Unknown'}
    </span>
  );
}

export default function SuppressionPage() {
  const [items, setItems]         = useState([]);
  const [total, setTotal]         = useState(0);
  const [page, setPage]           = useState(1);
  const [pageSize]                = useState(50);
  const [search, setSearch]       = useState('');
  const [loading, setLoading]     = useState(true);
  const [toast, setToast]         = useState(null); // {type:'success'|'error', msg}

  // Check email
  const [checkEmail, setCheckEmail]   = useState('');
  const [checkResult, setCheckResult] = useState(null); // null | {suppressed, reason}
  const [checking, setChecking]       = useState(false);

  // Modals
  const [showAdd, setShowAdd]         = useState(false);
  const [showBulk, setShowBulk]       = useState(false);
  const [showImport, setShowImport]   = useState(false);

  const searchTimeout = useRef(null);

  const token   = localStorage.getItem('kumoui_token');
  const headers = { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' };

  const showToast = (type, msg) => {
    setToast({ type, msg });
    setTimeout(() => setToast(null), 4000);
  };

  const fetchItems = useCallback(async (pg = page, q = search) => {
    setLoading(true);
    try {
      const params = new URLSearchParams({ page: pg, page_size: pageSize, search: q });
      const res = await fetch(`/api/suppression?${params}`, { headers });
      if (res.status === 401) { window.location.href = '/login'; return; }
      if (!res.ok) throw new Error('Failed to fetch');
      const data = await res.json();
      setItems(Array.isArray(data.items) ? data.items : []);
      setTotal(data.total || 0);
    } catch (e) {
      console.error(e);
      setItems([]);
      showToast('error', 'Failed to load suppression list.');
    }
    setLoading(false);
  }, [page, pageSize, search]); // eslint-disable-line

  useEffect(() => { fetchItems(page, search); }, [page]); // eslint-disable-line

  // Debounced search
  const handleSearch = (val) => {
    setSearch(val);
    clearTimeout(searchTimeout.current);
    searchTimeout.current = setTimeout(() => {
      setPage(1);
      fetchItems(1, val);
    }, 300);
  };

  const deleteItem = async (id) => {
    if (!confirm('Remove this email from the suppression list?')) return;
    try {
      const res = await fetch(`/api/suppression/${id}`, { method: 'DELETE', headers });
      if (!res.ok) throw new Error('Delete failed');
      showToast('success', 'Email removed from suppression list.');
      fetchItems(page, search);
    } catch (e) {
      showToast('error', e.message || 'Failed to delete.');
    }
  };

  const checkSuppressed = async () => {
    if (!checkEmail.trim()) return;
    setChecking(true);
    setCheckResult(null);
    try {
      const res = await fetch(`/api/suppression/check?email=${encodeURIComponent(checkEmail.trim())}`, { headers });
      if (!res.ok) throw new Error('Check failed');
      const data = await res.json();
      setCheckResult(data);
    } catch (e) {
      showToast('error', 'Failed to check email.');
    }
    setChecking(false);
  };

  const exportCSV = async () => {
    try {
      const res = await fetch('/api/suppression/export', { headers });
      if (!res.ok) throw new Error('Export failed');
      const blob = await res.blob();
      const url  = URL.createObjectURL(blob);
      const a    = document.createElement('a');
      a.href     = url;
      a.download = `suppression_${new Date().toISOString().slice(0,10)}.csv`;
      a.click();
      URL.revokeObjectURL(url);
    } catch (e) {
      showToast('error', 'Export failed.');
    }
  };

  const totalPages = Math.max(1, Math.ceil(total / pageSize));
  const formatDate = (d) => d ? new Date(d).toLocaleString(undefined, { month: 'short', day: 'numeric', year: 'numeric', hour: '2-digit', minute: '2-digit' }) : '-';

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Suppression List</h1>
          <p className="text-muted-foreground">Global list of emails that will never be sent to. Hard bounces and spam complaints are auto-added.</p>
        </div>
        <div className="flex gap-2 flex-wrap">
          <button
            onClick={() => setShowAdd(true)}
            className="flex items-center gap-2 h-10 px-4 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-sm font-medium transition-colors shadow-sm"
          >
            <Plus className="w-4 h-4" /> Add Email
          </button>
          <button
            onClick={() => setShowImport(true)}
            className="flex items-center gap-2 h-10 px-4 rounded-md bg-secondary text-secondary-foreground hover:bg-secondary/80 text-sm font-medium transition-colors"
          >
            <Upload className="w-4 h-4" /> Import CSV
          </button>
          <button
            onClick={exportCSV}
            className="flex items-center gap-2 h-10 px-4 rounded-md border bg-background hover:bg-muted text-sm font-medium transition-colors"
          >
            <Download className="w-4 h-4" /> Export CSV
          </button>
        </div>
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

      {/* Stats bar */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div className="bg-card border rounded-xl p-4 shadow-sm flex items-center justify-between">
          <div>
            <p className="text-sm font-medium text-muted-foreground">Total Suppressed</p>
            <p className="text-2xl font-bold mt-1">{total.toLocaleString()}</p>
          </div>
          <div className="p-2 rounded-lg bg-red-500/10 text-red-500">
            <Shield className="w-5 h-5" />
          </div>
        </div>

        {/* Check email */}
        <div className="bg-card border rounded-xl p-4 shadow-sm">
          <p className="text-sm font-medium text-muted-foreground mb-2">Check Email</p>
          <div className="flex gap-2">
            <input
              type="email"
              value={checkEmail}
              onChange={e => { setCheckEmail(e.target.value); setCheckResult(null); }}
              onKeyDown={e => e.key === 'Enter' && checkSuppressed()}
              placeholder="user@example.com"
              className="flex-1 h-9 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring"
            />
            <button
              onClick={checkSuppressed}
              disabled={checking || !checkEmail.trim()}
              className="h-9 px-3 rounded-md bg-secondary text-secondary-foreground hover:bg-secondary/80 text-sm font-medium transition-colors disabled:opacity-50"
            >
              {checking ? <Loader2 className="w-4 h-4 animate-spin" /> : 'Check'}
            </button>
          </div>
          {checkResult !== null && (
            <div className={cn(
              'mt-2 text-xs px-3 py-1.5 rounded-md font-medium flex items-center gap-1.5',
              checkResult.suppressed
                ? 'bg-red-500/10 text-red-600 dark:text-red-400'
                : 'bg-green-500/10 text-green-600 dark:text-green-400'
            )}>
              {checkResult.suppressed
                ? <><AlertTriangle className="w-3 h-3" /> Suppressed ({checkResult.reason || 'unknown reason'})</>
                : <><CheckCircle2 className="w-3 h-3" /> Not suppressed</>}
            </div>
          )}
        </div>
      </div>

      {/* Search */}
      <div className="relative">
        <Search className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
        <input
          type="text"
          value={search}
          onChange={e => handleSearch(e.target.value)}
          placeholder="Search by email or domain..."
          className="w-full h-10 pl-9 pr-4 rounded-md border bg-background text-sm focus:ring-2 focus:ring-ring"
        />
      </div>

      {/* Table */}
      <div className="bg-card border rounded-xl overflow-hidden shadow-sm">
        {loading ? (
          <div className="p-12 text-center text-muted-foreground flex items-center justify-center gap-2">
            <Loader2 className="w-5 h-5 animate-spin" /> Loading suppression list...
          </div>
        ) : items.length === 0 ? (
          <div className="flex flex-col items-center justify-center p-16 text-center">
            <div className="p-4 bg-blue-100 dark:bg-blue-900/20 rounded-full mb-4">
              <Shield className="w-12 h-12 text-blue-500 dark:text-blue-400" />
            </div>
            <h3 className="text-xl font-semibold mb-1">
              {search ? 'No Results Found' : 'Suppression List is Empty'}
            </h3>
            <p className="text-muted-foreground text-sm max-w-sm">
              {search
                ? 'No emails match your search query.'
                : 'Hard bounces and spam complaints will be automatically added here.'}
            </p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm text-left">
              <thead className="bg-muted/50 text-muted-foreground uppercase text-xs">
                <tr>
                  <th className="px-4 py-3 font-medium">Email</th>
                  <th className="px-4 py-3 font-medium">Reason</th>
                  <th className="px-4 py-3 font-medium">Domain</th>
                  <th className="px-4 py-3 font-medium">Source</th>
                  <th className="px-4 py-3 font-medium whitespace-nowrap">Date Added</th>
                  <th className="px-4 py-3 font-medium text-right">Action</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {items.map(item => (
                  <tr key={item.id} className="hover:bg-muted/50 transition-colors group">
                    <td className="px-4 py-3 font-mono text-xs truncate max-w-[200px]" title={item.email}>
                      {item.email}
                    </td>
                    <td className="px-4 py-3">
                      <ReasonBadge reason={item.reason} />
                    </td>
                    <td className="px-4 py-3 text-muted-foreground text-xs truncate max-w-[150px]">
                      {item.domain || '-'}
                    </td>
                    <td className="px-4 py-3 text-muted-foreground text-xs truncate max-w-[150px]" title={item.source_info}>
                      {item.source_info || '-'}
                    </td>
                    <td className="px-4 py-3 text-muted-foreground whitespace-nowrap text-xs">
                      {formatDate(item.created_at)}
                    </td>
                    <td className="px-4 py-3 text-right">
                      <button
                        onClick={() => deleteItem(item.id)}
                        className="p-1.5 hover:bg-destructive/10 text-muted-foreground hover:text-destructive rounded-md transition-colors opacity-0 group-hover:opacity-100"
                        title="Remove from suppression list"
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

      {/* Pagination */}
      {!loading && total > pageSize && (
        <div className="flex items-center justify-between">
          <p className="text-sm text-muted-foreground">
            Showing {((page - 1) * pageSize) + 1}–{Math.min(page * pageSize, total)} of {total.toLocaleString()} emails
          </p>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setPage(p => Math.max(1, p - 1))}
              disabled={page === 1}
              className="flex items-center gap-1 h-9 px-3 rounded-md border bg-background text-sm hover:bg-muted transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <ChevronLeft className="w-4 h-4" /> Previous
            </button>
            <span className="text-sm font-medium px-2">Page {page} of {totalPages}</span>
            <button
              onClick={() => setPage(p => Math.min(totalPages, p + 1))}
              disabled={page === totalPages}
              className="flex items-center gap-1 h-9 px-3 rounded-md border bg-background text-sm hover:bg-muted transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Next <ChevronRight className="w-4 h-4" />
            </button>
          </div>
        </div>
      )}

      {/* Add Email Modal */}
      {showAdd && (
        <AddEmailModal
          headers={headers}
          onClose={() => setShowAdd(false)}
          onSuccess={(msg) => { showToast('success', msg); fetchItems(page, search); }}
          onError={(msg) => showToast('error', msg)}
        />
      )}

      {/* Bulk Add Modal */}
      {showBulk && (
        <BulkAddModal
          headers={headers}
          onClose={() => setShowBulk(false)}
          onSuccess={(msg) => { showToast('success', msg); fetchItems(1, search); setPage(1); }}
          onError={(msg) => showToast('error', msg)}
        />
      )}

      {/* Import CSV Modal */}
      {showImport && (
        <ImportCSVModal
          token={token}
          onClose={() => setShowImport(false)}
          onSuccess={(msg) => { showToast('success', msg); fetchItems(1, ''); setSearch(''); setPage(1); }}
          onError={(msg) => showToast('error', msg)}
        />
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Add Email Modal
// ---------------------------------------------------------------------------
function AddEmailModal({ headers, onClose, onSuccess, onError }) {
  const [form, setForm] = useState({ email: '', reason: 'manual', source_info: '' });
  const [busy, setBusy] = useState(false);
  const [showBulk, setShowBulk] = useState(false);

  const submit = async (e) => {
    e.preventDefault();
    if (!form.email.trim()) return;
    setBusy(true);
    try {
      const res = await fetch('/api/suppression', {
        method: 'POST',
        headers,
        body: JSON.stringify({ email: form.email.trim(), reason: form.reason, source_info: form.source_info }),
      });
      if (!res.ok) {
        const err = await res.json().catch(() => ({}));
        throw new Error(err.detail || 'Failed to add email');
      }
      onSuccess('Email added to suppression list.');
      onClose();
    } catch (e) {
      onError(e.message || 'Failed to add email.');
    }
    setBusy(false);
  };

  if (showBulk) {
    return (
      <BulkAddModal
        headers={headers}
        onClose={onClose}
        onSuccess={onSuccess}
        onError={onError}
      />
    );
  }

  return (
    <Modal title="Add Email to Suppression List" onClose={onClose} icon={<Plus className="w-5 h-5 text-primary" />}>
      <form onSubmit={submit} className="space-y-4">
        <div className="space-y-2">
          <label className="text-sm font-medium">Email Address <span className="text-destructive">*</span></label>
          <input
            type="email"
            required
            autoFocus
            value={form.email}
            onChange={e => setForm({ ...form, email: e.target.value })}
            placeholder="user@example.com"
            className="w-full h-10 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring"
          />
        </div>

        <div className="space-y-2">
          <label className="text-sm font-medium">Reason</label>
          <select
            value={form.reason}
            onChange={e => setForm({ ...form, reason: e.target.value })}
            className="w-full h-10 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring"
          >
            <option value="manual">Manual</option>
            <option value="unsubscribe">Unsubscribe</option>
            <option value="hard_bounce">Hard Bounce</option>
            <option value="spam_complaint">Spam Complaint</option>
          </select>
        </div>

        <div className="space-y-2">
          <label className="text-sm font-medium">Source Info <span className="text-muted-foreground text-xs">(optional)</span></label>
          <input
            type="text"
            value={form.source_info}
            onChange={e => setForm({ ...form, source_info: e.target.value })}
            placeholder="e.g. Campaign name, list source..."
            className="w-full h-10 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring"
          />
        </div>

        <div className="flex justify-between items-center pt-2 border-t">
          <button
            type="button"
            onClick={() => setShowBulk(true)}
            className="text-sm text-muted-foreground hover:text-foreground flex items-center gap-1 transition-colors"
          >
            <Users className="w-3.5 h-3.5" /> Switch to Bulk Add
          </button>
          <div className="flex gap-2">
            <button type="button" onClick={onClose} className="px-4 py-2 text-sm rounded-md hover:bg-muted transition-colors">Cancel</button>
            <button
              type="submit"
              disabled={busy}
              className="flex items-center gap-2 px-4 py-2 text-sm rounded-md bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
            >
              {busy ? <Loader2 className="w-4 h-4 animate-spin" /> : <Plus className="w-4 h-4" />}
              Add Email
            </button>
          </div>
        </div>
      </form>
    </Modal>
  );
}

// ---------------------------------------------------------------------------
// Bulk Add Modal
// ---------------------------------------------------------------------------
function BulkAddModal({ headers, onClose, onSuccess, onError }) {
  const [emailsText, setEmailsText] = useState('');
  const [reason, setReason]         = useState('manual');
  const [busy, setBusy]             = useState(false);

  const submit = async (e) => {
    e.preventDefault();
    const emails = emailsText
      .split('\n')
      .map(l => l.trim())
      .filter(l => l.length > 0);
    if (emails.length === 0) return;
    setBusy(true);
    try {
      const res = await fetch('/api/suppression/bulk', {
        method: 'POST',
        headers,
        body: JSON.stringify({ emails, reason }),
      });
      if (!res.ok) {
        const err = await res.json().catch(() => ({}));
        throw new Error(err.detail || 'Bulk add failed');
      }
      onSuccess(`${emails.length} email(s) added to suppression list.`);
      onClose();
    } catch (e) {
      onError(e.message || 'Bulk add failed.');
    }
    setBusy(false);
  };

  const emailCount = emailsText.split('\n').filter(l => l.trim()).length;

  return (
    <Modal title="Bulk Add Emails" onClose={onClose} icon={<Users className="w-5 h-5 text-primary" />}>
      <form onSubmit={submit} className="space-y-4">
        <div className="space-y-2">
          <div className="flex justify-between">
            <label className="text-sm font-medium">Email Addresses</label>
            {emailCount > 0 && (
              <span className="text-xs text-muted-foreground">{emailCount} email{emailCount !== 1 ? 's' : ''}</span>
            )}
          </div>
          <textarea
            autoFocus
            value={emailsText}
            onChange={e => setEmailsText(e.target.value)}
            placeholder={"user1@example.com\nuser2@example.com\nuser3@example.com"}
            className="w-full rounded-md border bg-background p-3 text-sm h-40 resize-none focus:ring-2 focus:ring-ring font-mono"
          />
          <p className="text-xs text-muted-foreground">One email address per line.</p>
        </div>

        <div className="space-y-2">
          <label className="text-sm font-medium">Reason</label>
          <select
            value={reason}
            onChange={e => setReason(e.target.value)}
            className="w-full h-10 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring"
          >
            <option value="manual">Manual</option>
            <option value="unsubscribe">Unsubscribe</option>
            <option value="hard_bounce">Hard Bounce</option>
            <option value="spam_complaint">Spam Complaint</option>
          </select>
        </div>

        <div className="flex justify-end gap-2 pt-2 border-t">
          <button type="button" onClick={onClose} className="px-4 py-2 text-sm rounded-md hover:bg-muted transition-colors">Cancel</button>
          <button
            type="submit"
            disabled={busy || emailCount === 0}
            className="flex items-center gap-2 px-4 py-2 text-sm rounded-md bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
          >
            {busy ? <Loader2 className="w-4 h-4 animate-spin" /> : <Plus className="w-4 h-4" />}
            Add {emailCount > 0 ? emailCount : ''} Email{emailCount !== 1 ? 's' : ''}
          </button>
        </div>
      </form>
    </Modal>
  );
}

// ---------------------------------------------------------------------------
// Import CSV Modal
// ---------------------------------------------------------------------------
function ImportCSVModal({ token, onClose, onSuccess, onError }) {
  const [file, setFile]   = useState(null);
  const [busy, setBusy]   = useState(false);
  const fileRef           = useRef(null);

  const submit = async (e) => {
    e.preventDefault();
    if (!file) return;
    setBusy(true);
    try {
      const formData = new FormData();
      formData.append('file', file);
      const res = await fetch('/api/suppression/import', {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
        body: formData,
      });
      if (!res.ok) {
        const err = await res.json().catch(() => ({}));
        throw new Error(err.detail || 'Import failed');
      }
      const data = await res.json().catch(() => ({}));
      onSuccess(data.message || 'CSV imported successfully.');
      onClose();
    } catch (e) {
      onError(e.message || 'CSV import failed.');
    }
    setBusy(false);
  };

  return (
    <Modal title="Import CSV" onClose={onClose} icon={<Upload className="w-5 h-5 text-primary" />}>
      <form onSubmit={submit} className="space-y-4">
        <div
          className="border-2 border-dashed rounded-lg p-8 text-center cursor-pointer hover:border-primary/50 hover:bg-muted/30 transition-colors"
          onClick={() => fileRef.current?.click()}
        >
          <FileText className="w-10 h-10 text-muted-foreground mx-auto mb-3" />
          {file ? (
            <div>
              <p className="font-medium text-sm">{file.name}</p>
              <p className="text-xs text-muted-foreground mt-1">{(file.size / 1024).toFixed(1)} KB</p>
            </div>
          ) : (
            <div>
              <p className="font-medium text-sm">Click to select a CSV file</p>
              <p className="text-xs text-muted-foreground mt-1">or drag and drop</p>
            </div>
          )}
          <input
            ref={fileRef}
            type="file"
            accept=".csv,text/csv"
            className="hidden"
            onChange={e => setFile(e.target.files[0] || null)}
          />
        </div>

        <div className="bg-muted/30 rounded-md p-3 text-xs text-muted-foreground space-y-1">
          <p className="font-medium text-foreground">CSV Format</p>
          <p>Required column: <code className="bg-background px-1 rounded">email</code></p>
          <p>Optional column: <code className="bg-background px-1 rounded">reason</code> (hard_bounce, spam_complaint, manual, unsubscribe)</p>
          <p className="font-mono mt-2 bg-background p-2 rounded border">email,reason<br/>user@example.com,hard_bounce</p>
        </div>

        <div className="flex justify-end gap-2 pt-2 border-t">
          <button type="button" onClick={onClose} className="px-4 py-2 text-sm rounded-md hover:bg-muted transition-colors">Cancel</button>
          <button
            type="submit"
            disabled={busy || !file}
            className="flex items-center gap-2 px-4 py-2 text-sm rounded-md bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
          >
            {busy ? <Loader2 className="w-4 h-4 animate-spin" /> : <Upload className="w-4 h-4" />}
            Import
          </button>
        </div>
      </form>
    </Modal>
  );
}

// ---------------------------------------------------------------------------
// Shared Modal wrapper
// ---------------------------------------------------------------------------
function Modal({ title, onClose, icon, children }) {
  useEffect(() => {
    const handler = (e) => { if (e.key === 'Escape') onClose(); };
    document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }, [onClose]);

  return (
    <div className="fixed inset-0 bg-background/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
      <div className="bg-card w-full max-w-md border rounded-xl shadow-lg">
        <div className="flex items-center justify-between p-6 border-b">
          <h3 className="text-lg font-semibold flex items-center gap-2">
            {icon} {title}
          </h3>
          <button onClick={onClose} className="p-1 rounded-md hover:bg-muted transition-colors">
            <X className="w-4 h-4" />
          </button>
        </div>
        <div className="p-6">{children}</div>
      </div>
    </div>
  );
}
