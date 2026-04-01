import React, { useEffect, useState, useCallback } from "react";
import {
  MailWarning, Filter, Trash2, User, Globe, AlertCircle, Key, Edit2,
  X, Save, Loader2, Info, Server, Shield, Plus, Inbox, RefreshCw,
  Mail, ChevronDown, ChevronRight, Eye, Sparkles
} from "lucide-react";
import { listBounces, listDomains, deleteBounce, saveBounce } from "../api";
import { cn } from "../lib/utils";

const token = () => localStorage.getItem("kumoui_token") || "";
const hdrs = () => ({ Authorization: `Bearer ${token()}`, "Content-Type": "application/json" });

export default function BouncePage() {
  const [list, setList] = useState([]);
  const [domains, setDomains] = useState([]);
  const [filter, setFilter] = useState("");
  const [msg, setMsg] = useState({ text: "", ok: true });
  const [loading, setLoading] = useState(true);

  // Create / Edit modals
  const [creating, setCreating] = useState(false);
  const [createForm, setCreateForm] = useState({ username: "", domain: "", password: "", notes: "" });
  const [editing, setEditing] = useState(null);
  const [passForm, setPassForm] = useState({ password: "", notes: "" });
  const [busy, setBusy] = useState(false);

  // Info modal
  const [showInfo, setShowInfo] = useState(false);

  // Required inboxes
  const [reqBusy, setReqBusy] = useState(false);
  const [reqResults, setReqResults] = useState(null);

  // Mailbox viewer
  const [viewAccount, setViewAccount] = useState(null); // the bounce account
  const [messages, setMessages] = useState([]);
  const [msgsLoading, setMsgsLoading] = useState(false);
  const [openMsg, setOpenMsg] = useState(null); // { id, raw }

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [b, d] = await Promise.all([listBounces(), listDomains()]);
      setList(Array.isArray(b) ? b : []);
      setDomains(Array.isArray(d) ? d : []);
    } catch {
      setMsg({ text: "Failed to load data", ok: false });
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  // ── Create ────────────────────────────────────────────────
  const handleCreate = async (e) => {
    e.preventDefault();
    setBusy(true);
    try {
      const res = await fetch("/api/bounces", {
        method: "POST",
        headers: hdrs(),
        body: JSON.stringify({ ...createForm, id: 0 }),
      });
      if (res.status === 401) { window.location.href = '/login'; return; }
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || "Failed to create");
      setCreating(false);
      setCreateForm({ username: "", domain: "", password: "", notes: "" });
      setMsg({ text: `Account "${data.username}" created successfully.`, ok: true });
      load();
    } catch (err) {
      setMsg({ text: err.message, ok: false });
    } finally {
      setBusy(false);
    }
  };

  // ── Edit ─────────────────────────────────────────────────
  const openEdit = (account) => {
    setEditing(account);
    setPassForm({ password: "", notes: account.notes || "" });
  };
  const handleSave = async (e) => {
    e.preventDefault();
    setBusy(true);
    try {
      const res = await fetch("/api/bounces", {
        method: "POST",
        headers: hdrs(),
        body: JSON.stringify({ id: editing.id, username: editing.username, domain: editing.domain, notes: passForm.notes, password: passForm.password || undefined }),
      });
      if (res.status === 401) { window.location.href = '/login'; return; }
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || "Failed to update");
      setEditing(null);
      setMsg({ text: "Account updated.", ok: true });
      load();
    } catch (err) {
      setMsg({ text: err.message, ok: false });
    } finally {
      setBusy(false);
    }
  };

  // ── Delete ────────────────────────────────────────────────
  const handleDelete = async (id, username) => {
    if (!confirm(`Delete bounce account "${username}"?\nThis removes the system user and all bounce emails.`)) return;
    try {
      await deleteBounce(id);
      load();
    } catch (err) {
      setMsg({ text: err.message || "Failed to delete", ok: false });
    }
  };

  // ── Required Inboxes ──────────────────────────────────────
  const handleCreateRequired = async () => {
    setReqBusy(true);
    setReqResults(null);
    try {
      const res = await fetch("/api/bounces/create-required", { method: "POST", headers: hdrs(), body: "{}" });
      if (res.status === 401) { window.location.href = '/login'; return; }
      const data = await res.json();
      setReqResults(data.results || []);
      load();
    } catch (err) {
      setMsg({ text: err.message, ok: false });
    } finally {
      setReqBusy(false);
    }
  };

  // ── Mailbox Viewer ────────────────────────────────────────
  const openMailbox = async (account) => {
    setViewAccount(account);
    setMessages([]);
    setOpenMsg(null);
    setMsgsLoading(true);
    try {
      const res = await fetch(`/api/bounces/${account.id}/messages`, { headers: hdrs() });
      if (res.status === 401) { window.location.href = '/login'; return; }
      const data = await res.json();
      setMessages(Array.isArray(data) ? data : []);
    } catch { setMessages([]); }
    setMsgsLoading(false);
  };
  const openMessage = async (msgId) => {
    if (!viewAccount) return;
    try {
      const res = await fetch(`/api/bounces/${viewAccount.id}/messages/${encodeURIComponent(msgId)}`, { headers: hdrs() });
      if (res.status === 401) { window.location.href = '/login'; return; }
      const data = await res.json();
      setOpenMsg({ id: msgId, raw: data.raw || "" });
    } catch { setOpenMsg({ id: msgId, raw: "Failed to load message." }); }
  };

  const filteredList = filter ? list.filter(b => b.domain === filter) : list;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Bounce Accounts</h1>
          <p className="text-muted-foreground text-sm">System users handling incoming bounce + FBL messages via Maildir.</p>
        </div>
        <div className="flex items-center gap-2 flex-wrap">
          <button onClick={() => setShowInfo(true)}
            className="flex items-center gap-2 h-9 px-3 rounded-md border bg-background hover:bg-muted text-sm">
            <Info className="w-4 h-4 text-blue-500" /> Connection Info
          </button>
          <button onClick={handleCreateRequired} disabled={reqBusy}
            className="flex items-center gap-2 h-9 px-3 rounded-md border bg-amber-500/10 text-amber-700 dark:text-amber-400 border-amber-500/30 hover:bg-amber-500/20 text-sm font-medium">
            {reqBusy ? <Loader2 className="w-4 h-4 animate-spin" /> : <Sparkles className="w-4 h-4" />}
            Required Inboxes
          </button>
          <button onClick={() => setCreating(true)}
            className="flex items-center gap-2 h-9 px-4 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-sm font-medium">
            <Plus className="w-4 h-4" /> New Account
          </button>
          <div className="relative">
            <Filter className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
            <select className="h-9 rounded-md border bg-background pl-9 pr-8 text-sm focus:ring-2 focus:ring-ring min-w-[160px]"
              value={filter} onChange={e => setFilter(e.target.value)}>
              <option value="">All Domains</option>
              {domains.map(d => <option key={d.id} value={d.name}>{d.name}</option>)}
            </select>
          </div>
        </div>
      </div>

      {/* Status message */}
      {msg.text && (
        <div className={cn("px-4 py-2.5 rounded-md text-sm flex justify-between items-center",
          msg.ok ? "bg-green-500/10 text-green-700 dark:text-green-400" : "bg-red-500/10 text-red-700 dark:text-red-400")}>
          {msg.text}
          <button onClick={() => setMsg({ text: "" })}><X className="w-4 h-4" /></button>
        </div>
      )}

      {/* Required inbox results */}
      {reqResults && (
        <div className="rounded-lg border bg-card p-4 text-sm space-y-2">
          <div className="font-semibold flex items-center gap-2 mb-2">
            <Sparkles className="w-4 h-4 text-amber-500" /> Required Inboxes Results
          </div>
          <div className="grid sm:grid-cols-2 gap-2">
            {reqResults.map((r, i) => (
              <div key={i} className={cn("flex items-center gap-2 px-3 py-2 rounded-md border",
                r.status === "created" ? "bg-green-500/5 border-green-500/30" :
                r.status === "skipped" ? "bg-muted/40 border-border" : "bg-red-500/5 border-red-500/30")}>
                <span className={cn("w-2 h-2 rounded-full shrink-0",
                  r.status === "created" ? "bg-green-500" : r.status === "skipped" ? "bg-gray-400" : "bg-red-500")} />
                <span className="font-mono text-xs">{r.username}</span>
                <span className="text-xs text-muted-foreground ml-auto">{r.status}</span>
              </div>
            ))}
          </div>
          <button onClick={() => setReqResults(null)} className="text-xs text-muted-foreground hover:text-foreground">Dismiss</button>
        </div>
      )}

      {/* Account grid */}
      {loading ? (
        <div className="text-center py-12 text-muted-foreground">Loading accounts…</div>
      ) : filteredList.length === 0 ? (
        <div className="flex flex-col items-center justify-center p-12 bg-card border rounded-xl border-dashed">
          <MailWarning className="w-12 h-12 text-muted-foreground/50 mb-3" />
          <h3 className="text-lg font-medium">No Bounce Accounts</h3>
          <p className="text-muted-foreground text-sm mt-1">Click <strong>New Account</strong> to create one, or use <strong>Required Inboxes</strong> to auto-create standard accounts.</p>
        </div>
      ) : (
        <div className="grid sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          {filteredList.map(b => (
            <div key={b.id} className="group bg-card border rounded-xl p-4 shadow-sm hover:shadow-md transition-all">
              <div className="flex justify-between items-start mb-2">
                <div className="p-2 bg-purple-500/10 text-purple-500 rounded-lg">
                  <User className="w-5 h-5" />
                </div>
                <div className="flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                  <button onClick={() => openMailbox(b)}
                    className="p-1.5 text-muted-foreground hover:text-blue-500 hover:bg-blue-500/10 rounded-md" title="View Mailbox">
                    <Inbox className="w-4 h-4" />
                  </button>
                  <button onClick={() => openEdit(b)}
                    className="p-1.5 text-muted-foreground hover:text-primary hover:bg-muted rounded-md" title="Edit">
                    <Edit2 className="w-4 h-4" />
                  </button>
                  <button onClick={() => handleDelete(b.id, b.username)}
                    className="p-1.5 text-muted-foreground hover:text-destructive hover:bg-destructive/10 rounded-md" title="Delete">
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>
              </div>
              <div className="space-y-1">
                <div className="font-mono font-medium text-base truncate" title={b.username}>{b.username}</div>
                <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                  <Globe className="w-3 h-3" />{b.domain}
                </div>
              </div>
              {b.notes && (
                <div className="mt-3 pt-3 border-t text-xs text-muted-foreground flex items-start gap-2">
                  <AlertCircle className="w-3 h-3 mt-0.5 shrink-0" />
                  <span className="line-clamp-2">{b.notes}</span>
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {/* ── Mailbox Viewer Modal ── */}
      {viewAccount && (
        <div className="fixed inset-0 bg-background/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-card w-full max-w-4xl border rounded-xl shadow-xl flex flex-col max-h-[90vh]">
            <div className="flex justify-between items-center px-5 py-4 border-b shrink-0">
              <h3 className="text-base font-semibold flex items-center gap-2">
                <Inbox className="w-5 h-5 text-blue-500" />
                Mailbox: <span className="font-mono">{viewAccount.username}</span>
              </h3>
              <div className="flex items-center gap-2">
                <button onClick={() => openMailbox(viewAccount)}
                  className="p-1.5 hover:bg-muted rounded-md text-muted-foreground hover:text-foreground">
                  <RefreshCw className="w-4 h-4" />
                </button>
                <button onClick={() => { setViewAccount(null); setOpenMsg(null); }}><X className="w-5 h-5" /></button>
              </div>
            </div>
            <div className="flex flex-1 overflow-hidden min-h-0">
              {/* Message list */}
              <div className="w-80 shrink-0 border-r flex flex-col overflow-y-auto">
                {msgsLoading ? (
                  <div className="flex items-center justify-center py-12 text-muted-foreground">
                    <Loader2 className="w-5 h-5 animate-spin" />
                  </div>
                ) : messages.length === 0 ? (
                  <div className="flex flex-col items-center justify-center py-12 text-muted-foreground text-sm gap-2">
                    <Mail className="w-8 h-8 opacity-30" />
                    <p>Mailbox is empty</p>
                    <p className="text-xs">(new/ and cur/ checked)</p>
                  </div>
                ) : messages.map(m => (
                  <button key={m.id} onClick={() => openMessage(m.id)}
                    className={cn("text-left px-4 py-3 border-b hover:bg-muted/50 transition-colors",
                      openMsg?.id === m.id && "bg-primary/5 border-l-2 border-l-primary")}>
                    <div className="flex items-center gap-1.5 mb-1">
                      <span className={cn("w-1.5 h-1.5 rounded-full shrink-0",
                        m.folder === "new" ? "bg-blue-500" : "bg-gray-400")} />
                      <span className="text-xs font-medium truncate">{m.from || "(no from)"}</span>
                      <span className="ml-auto text-[10px] text-muted-foreground shrink-0">
                        {m.size_bytes > 1024 ? `${(m.size_bytes/1024).toFixed(1)}KB` : `${m.size_bytes}B`}
                      </span>
                    </div>
                    <div className="text-xs text-foreground truncate">{m.subject || "(no subject)"}</div>
                    <div className="text-[10px] text-muted-foreground mt-0.5 truncate">{m.date || "—"}</div>
                  </button>
                ))}
              </div>
              {/* Message content */}
              <div className="flex-1 overflow-auto p-4">
                {openMsg ? (
                  <pre className="text-xs font-mono whitespace-pre-wrap break-all text-foreground/80 leading-5">
                    {openMsg.raw}
                  </pre>
                ) : (
                  <div className="flex flex-col items-center justify-center h-full text-muted-foreground gap-2">
                    <Eye className="w-8 h-8 opacity-20" />
                    <p className="text-sm">Select a message to view</p>
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* ── Create Account Modal ── */}
      {creating && (
        <div className="fixed inset-0 bg-background/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-card w-full max-w-md border rounded-xl shadow-xl p-6 space-y-4">
            <div className="flex justify-between items-center">
              <h3 className="text-lg font-semibold">Create Bounce Account</h3>
              <button onClick={() => setCreating(false)}><X className="w-4 h-4" /></button>
            </div>
            <form onSubmit={handleCreate} className="space-y-4">
              <div className="space-y-1.5">
                <label className="text-sm font-medium">Username <span className="text-muted-foreground">(Linux system user)</span></label>
                <input required className="w-full h-9 rounded-md border bg-background px-3 text-sm font-mono"
                  value={createForm.username} onChange={e => setCreateForm({ ...createForm, username: e.target.value })}
                  placeholder="e.g. b-news or bounce-domain-com" />
              </div>
              <div className="space-y-1.5">
                <label className="text-sm font-medium">Domain</label>
                <select required className="w-full h-9 rounded-md border bg-background px-3 text-sm"
                  value={createForm.domain} onChange={e => setCreateForm({ ...createForm, domain: e.target.value })}>
                  <option value="">Select domain…</option>
                  {domains.map(d => <option key={d.id} value={d.name}>{d.name}</option>)}
                </select>
              </div>
              <div className="space-y-1.5">
                <label className="text-sm font-medium">Password</label>
                <div className="relative">
                  <Key className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
                  <input required type="password" className="w-full pl-9 h-9 rounded-md border bg-background px-3 text-sm"
                    value={createForm.password} onChange={e => setCreateForm({ ...createForm, password: e.target.value })}
                    placeholder="IMAP/POP3 password" />
                </div>
              </div>
              <div className="space-y-1.5">
                <label className="text-sm font-medium">Notes <span className="text-muted-foreground">(optional)</span></label>
                <textarea className="w-full rounded-md border bg-background p-3 text-sm h-16 resize-none"
                  value={createForm.notes} onChange={e => setCreateForm({ ...createForm, notes: e.target.value })}
                  placeholder="e.g. bounce handler for newsletters" />
              </div>
              <div className="flex justify-end gap-2 pt-1">
                <button type="button" onClick={() => setCreating(false)} className="px-4 py-2 text-sm rounded-md hover:bg-muted">Cancel</button>
                <button type="submit" disabled={busy}
                  className="flex items-center gap-2 px-4 py-2 text-sm rounded-md bg-primary text-primary-foreground hover:bg-primary/90">
                  {busy ? <Loader2 className="w-4 h-4 animate-spin" /> : <Plus className="w-4 h-4" />}
                  Create Account
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* ── Edit Modal ── */}
      {editing && (
        <div className="fixed inset-0 bg-background/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-card w-full max-w-md border rounded-xl shadow-xl p-6 space-y-4">
            <div className="flex justify-between items-center">
              <h3 className="text-lg font-semibold">Edit Bounce Account</h3>
              <button onClick={() => setEditing(null)}><X className="w-4 h-4" /></button>
            </div>
            <div className="p-3 bg-muted rounded-md text-sm space-y-1">
              <div className="flex justify-between">
                <span className="text-muted-foreground">Username:</span>
                <span className="font-mono">{editing.username}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Domain:</span>
                <span>{editing.domain}</span>
              </div>
            </div>
            <form onSubmit={handleSave} className="space-y-4">
              <div className="space-y-1.5">
                <label className="text-sm font-medium">New Password</label>
                <div className="relative">
                  <Key className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
                  <input type="password" className="w-full pl-9 h-9 rounded-md border bg-background px-3 text-sm"
                    value={passForm.password} onChange={e => setPassForm({ ...passForm, password: e.target.value })}
                    placeholder="Leave blank to keep current" />
                </div>
              </div>
              <div className="space-y-1.5">
                <label className="text-sm font-medium">Notes</label>
                <textarea className="w-full rounded-md border bg-background p-3 text-sm h-16 resize-none"
                  value={passForm.notes} onChange={e => setPassForm({ ...passForm, notes: e.target.value })} />
              </div>
              <div className="flex justify-end gap-2 pt-1">
                <button type="button" onClick={() => setEditing(null)} className="px-4 py-2 text-sm rounded-md hover:bg-muted">Cancel</button>
                <button type="submit" disabled={busy}
                  className="flex items-center gap-2 px-4 py-2 text-sm rounded-md bg-primary text-primary-foreground hover:bg-primary/90">
                  {busy ? <Loader2 className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
                  Save Changes
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* ── Connection Info Modal ── */}
      {showInfo && (
        <div className="fixed inset-0 bg-background/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-card w-full max-w-lg border rounded-xl shadow-xl p-6 space-y-4">
            <div className="flex justify-between items-center border-b pb-4">
              <h3 className="text-lg font-semibold flex items-center gap-2">
                <Server className="w-5 h-5 text-blue-500" /> Server Connection Info
              </h3>
              <button onClick={() => setShowInfo(false)}><X className="w-4 h-4" /></button>
            </div>
            <div className="space-y-4 text-sm">
              <p className="text-muted-foreground">Use these settings in MailWizz, Mautic, or your ESP to collect bounces via IMAP/POP3.</p>
              <div className="grid gap-3">
                {[
                  ["Hostname", "Your server IP or bounce.yourdomain.com"],
                  ["Username", "e.g. b-news (the account username)"],
                  ["Password", "Set when creating the account"],
                ].map(([k, v]) => (
                  <div key={k} className="bg-muted/30 p-3 rounded-md border flex justify-between items-center">
                    <span className="font-medium">{k}</span>
                    <span className="font-mono bg-background px-2 py-1 rounded text-muted-foreground text-xs">{v}</span>
                  </div>
                ))}
              </div>
              <div className="grid grid-cols-2 gap-4">
                {[["IMAP (SSL)", "993", "Recommended for MailWizz"], ["POP3 (SSL)", "995", "Alternative"]].map(([proto, port, hint]) => (
                  <div key={proto} className="border rounded-md p-4 space-y-1">
                    <div className="flex items-center gap-2 font-semibold text-primary text-sm">
                      <Shield className="w-4 h-4" /> {proto}
                    </div>
                    <div className="text-2xl font-bold">Port {port}</div>
                    <div className="text-xs text-muted-foreground">{hint}</div>
                  </div>
                ))}
              </div>
              <div className="bg-blue-500/5 border border-blue-500/20 rounded-md p-3 text-xs text-blue-600 dark:text-blue-400">
                Dovecot must be configured to serve these system users via IMAP/POP3. Run the Domains setup to ensure Dovecot is configured.
              </div>
            </div>
            <div className="flex justify-end pt-2">
              <button onClick={() => setShowInfo(false)} className="px-4 py-2 text-sm rounded-md bg-secondary hover:bg-secondary/80">Close</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
