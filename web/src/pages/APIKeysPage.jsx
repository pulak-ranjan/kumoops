import React, { useEffect, useState } from "react";
import {
  Key, Trash2, Plus, Copy, Check, Info, Zap, Server,
  Mail, Globe, Clock, ShieldCheck, RefreshCw
} from "lucide-react";
import { listKeys, createKey, deleteKey } from "../api";
import { cn } from "../lib/utils";

const SCOPES = [
  { id: "send",    label: "Send",    desc: "POST /api/v1/messages — HTTP Sending API (Mailgun-compat)" },
  { id: "relay",   label: "Relay",   desc: "SMTP relay authentication" },
  { id: "verify",  label: "Verify",  desc: "Email verification endpoints" },
  { id: "cluster", label: "Cluster", desc: "Multi-node cluster authentication (remote server token)" },
  { id: "read",    label: "Read",    desc: "Read-only stats & queue access" },
];

function CopyButton({ text }) {
  const [copied, setCopied] = useState(false);
  const copy = () => {
    navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };
  return (
    <button onClick={copy}
      className="p-2.5 bg-background border rounded-md hover:bg-muted transition-colors">
      {copied
        ? <Check className="w-4 h-4 text-green-600" />
        : <Copy className="w-4 h-4 text-muted-foreground" />}
    </button>
  );
}

function ScopeBadge({ scope }) {
  const colors = {
    send:    "bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-400",
    relay:   "bg-purple-100 text-purple-700 dark:bg-purple-900/40 dark:text-purple-400",
    verify:  "bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-400",
    cluster: "bg-orange-100 text-orange-700 dark:bg-orange-900/40 dark:text-orange-400",
    read:    "bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-400",
  };
  return (
    <span className={cn("px-1.5 py-0.5 rounded text-[10px] font-semibold uppercase tracking-wide",
      colors[scope] || "bg-muted text-muted-foreground")}>
      {scope}
    </span>
  );
}

function ago(ts) {
  if (!ts || ts === "0001-01-01T00:00:00Z") return "Never";
  const d = Math.round((Date.now() - new Date(ts).getTime()) / 1000);
  if (d < 60) return `${d}s ago`;
  if (d < 3600) return `${Math.round(d / 60)}m ago`;
  if (d < 86400) return `${Math.round(d / 3600)}h ago`;
  return `${Math.round(d / 86400)}d ago`;
}

export default function APIKeysPage() {
  const [keys, setKeys] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [newName, setNewName] = useState("");
  const [selectedScopes, setSelectedScopes] = useState(["send", "relay"]);
  const [newKeyData, setNewKeyData] = useState(null);
  const [creating, setCreating] = useState(false);

  const load = async () => {
    setLoading(true);
    try { setKeys((await listKeys()) || []); } catch { }
    setLoading(false);
  };

  useEffect(() => { load(); }, []);

  const toggleScope = (s) => setSelectedScopes(prev =>
    prev.includes(s) ? prev.filter(x => x !== s) : [...prev, s]
  );

  const handleCreate = async (e) => {
    e.preventDefault();
    if (!newName.trim()) return;
    setCreating(true);
    try {
      const res = await createKey(newName.trim(), selectedScopes.join(","));
      setNewKeyData(res);
      setShowForm(false);
      setNewName("");
      setSelectedScopes(["send", "relay"]);
      load();
    } catch (err) { alert(err.message); }
    setCreating(false);
  };

  const handleDelete = async (id) => {
    if (!confirm("Revoke this key? All apps using it will immediately lose access.")) return;
    await deleteKey(id);
    load();
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between gap-4 flex-wrap">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <Key className="w-6 h-6 text-primary" /> API Keys
          </h1>
          <p className="text-muted-foreground text-sm mt-1">
            Generate keys to authenticate external apps, the HTTP Sending API, and multi-VPS cluster nodes.
          </p>
        </div>
        <button onClick={() => { setShowForm(v => !v); setNewKeyData(null); }}
          className="flex items-center gap-2 px-4 py-2 rounded-md bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90">
          <Plus className="w-4 h-4" /> Create Key
        </button>
      </div>

      {/* How-to info boxes */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
        {[
          {
            icon: Mail,
            color: "text-blue-500",
            title: "HTTP Sending API",
            body: "Use any key with the `send` scope to call POST /api/v1/messages — Mailgun-compatible. Set Authorization header to your key.",
            code: `curl -X POST ${window.location.origin}/api/v1/messages \\\n  -H "Authorization: kumo_xxxxx" \\\n  -d '{"to":"user@example.com","subject":"Hi","html":"<p>Hello</p>","from_email":"sender@yourdomain.com"}'`,
          },
          {
            icon: Server,
            color: "text-orange-500",
            title: "Multi-VPS Cluster",
            body: "To add a remote KumoOps node: generate a key with the `cluster` scope on the remote server, then paste it as the API Token in Remote Servers.",
            code: `1. On VPS-2 → API Keys → Create Key (cluster scope)\n2. Copy the kumo_xxx key\n3. On VPS-1 → Remote Servers → Add Server\n   URL: ${window.location.origin}\n   API Token: kumo_xxx (paste here)`,
          },
          {
            icon: Globe,
            color: "text-green-500",
            title: "External Integrations",
            body: "Give external tools like MailWizz, Mautic, or your own app a scoped key. Use read-only keys for monitoring dashboards.",
            code: `# Example: Python app sending via KumoOps\nimport requests\nrequests.post('${window.location.origin}/api/v1/messages',\n  headers={'Authorization': 'kumo_xxx'},\n  json={'to': 'user@example.com', ...})`,
          },
        ].map((item) => {
          const Icon = item.icon;
          const [showCode, setShowCode] = useState(false);
          return (
            <div key={item.title} className="border rounded-lg overflow-hidden">
              <div className="px-4 py-3 flex items-start gap-2">
                <Icon className={cn("w-4 h-4 mt-0.5 shrink-0", item.color)} />
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-semibold">{item.title}</p>
                  <p className="text-xs text-muted-foreground mt-0.5 leading-relaxed">{item.body}</p>
                </div>
              </div>
              <div className="px-4 pb-3">
                <button onClick={() => setShowCode(v => !v)}
                  className="text-xs text-primary hover:underline font-medium">
                  {showCode ? "Hide example ↑" : "Show example ↓"}
                </button>
                {showCode && (
                  <pre className="mt-2 text-[10px] font-mono bg-muted/60 rounded p-2 overflow-x-auto whitespace-pre-wrap leading-relaxed text-muted-foreground">
                    {item.code}
                  </pre>
                )}
              </div>
            </div>
          );
        })}
      </div>

      {/* New Key — one-time reveal */}
      {newKeyData && (
        <div className="border border-green-500/30 bg-green-50 dark:bg-green-950/30 rounded-lg p-5 space-y-3">
          <div className="flex items-center gap-2">
            <ShieldCheck className="w-5 h-5 text-green-600" />
            <h3 className="font-bold text-green-700 dark:text-green-400">Key Created — Copy It Now</h3>
          </div>
          <p className="text-xs text-green-700 dark:text-green-400">
            This is the only time the full key will be shown. Store it securely.
          </p>
          <div className="flex items-center gap-2">
            <code className="flex-1 bg-white dark:bg-black/30 border border-green-200 dark:border-green-800/50 p-3 rounded-md font-mono text-sm break-all text-foreground">
              {newKeyData.key}
            </code>
            <CopyButton text={newKeyData.key} />
          </div>
          <div className="flex items-center gap-4 text-xs text-muted-foreground">
            <span>Name: <strong>{newKeyData.name}</strong></span>
            <span>Scopes: {(newKeyData.scopes || "").split(",").map(s => <ScopeBadge key={s} scope={s} />)}</span>
          </div>
          <button onClick={() => setNewKeyData(null)}
            className="text-xs text-green-700 dark:text-green-400 underline mt-1">
            I've saved it — dismiss
          </button>
        </div>
      )}

      {/* Create form */}
      {showForm && !newKeyData && (
        <form onSubmit={handleCreate} className="border rounded-lg p-5 space-y-4 bg-muted/20">
          <h3 className="text-sm font-semibold">Create New API Key</h3>
          <div>
            <label className="text-xs font-medium text-muted-foreground">Application / Key Name</label>
            <input value={newName} onChange={e => setNewName(e.target.value)}
              placeholder="e.g. Production Sending, VPS-2 Cluster Node, MailWizz"
              required
              className="w-full mt-1 h-10 px-3 rounded-md border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary/50" />
          </div>
          <div>
            <label className="text-xs font-medium text-muted-foreground">Scopes (select all that apply)</label>
            <div className="mt-2 space-y-2">
              {SCOPES.map(scope => (
                <label key={scope.id} className="flex items-start gap-3 cursor-pointer group">
                  <input type="checkbox"
                    checked={selectedScopes.includes(scope.id)}
                    onChange={() => toggleScope(scope.id)}
                    className="mt-0.5 h-4 w-4 rounded border accent-primary cursor-pointer" />
                  <div>
                    <span className="text-sm font-medium group-hover:text-primary transition-colors">{scope.label}</span>
                    <p className="text-xs text-muted-foreground">{scope.desc}</p>
                  </div>
                </label>
              ))}
            </div>
          </div>
          <div className="flex gap-2 pt-1">
            <button type="submit" disabled={creating || !newName.trim() || selectedScopes.length === 0}
              className="flex items-center gap-2 px-4 py-2 rounded-md bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 disabled:opacity-60">
              {creating ? <RefreshCw className="w-4 h-4 animate-spin" /> : <Key className="w-4 h-4" />}
              {creating ? "Generating…" : "Generate Key"}
            </button>
            <button type="button" onClick={() => setShowForm(false)}
              className="px-4 py-2 rounded-md border text-sm hover:bg-muted">
              Cancel
            </button>
          </div>
        </form>
      )}

      {/* Key list */}
      {loading ? (
        <div className="text-center py-10 text-muted-foreground text-sm">Loading…</div>
      ) : keys.length === 0 ? (
        <div className="text-center py-16 text-muted-foreground border rounded-lg">
          <Key className="w-10 h-10 mx-auto mb-2 opacity-20" />
          <p className="text-sm font-medium">No API keys yet.</p>
          <p className="text-xs mt-1">Create your first key to start using the sending API or connect cluster nodes.</p>
        </div>
      ) : (
        <div className="border rounded-lg overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-muted/50">
              <tr>
                {["Name", "Key Prefix", "Scopes", "Created", "Last Used", ""].map(h => (
                  <th key={h} className="px-4 py-2.5 text-left text-xs font-semibold text-muted-foreground uppercase">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y">
              {keys.map(k => (
                <tr key={k.id} className="hover:bg-muted/30 transition-colors">
                  <td className="px-4 py-3 font-medium">{k.name}</td>
                  <td className="px-4 py-3">
                    <code className="text-xs font-mono bg-muted px-2 py-1 rounded text-muted-foreground">
                      {k.key ? k.key.slice(0, 12) + "…" : "kumo_•••••••"}
                    </code>
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex flex-wrap gap-1">
                      {(k.scopes || "").split(",").filter(Boolean).map(s => (
                        <ScopeBadge key={s} scope={s.trim()} />
                      ))}
                    </div>
                  </td>
                  <td className="px-4 py-3 text-xs text-muted-foreground">
                    {k.created_at ? new Date(k.created_at).toLocaleDateString() : "—"}
                  </td>
                  <td className="px-4 py-3 text-xs">
                    <div className="flex items-center gap-1">
                      <Clock className="w-3 h-3 text-muted-foreground" />
                      <span className={cn(
                        ago(k.last_used) === "Never" ? "text-muted-foreground" : "text-foreground font-medium"
                      )}>
                        {ago(k.last_used)}
                      </span>
                    </div>
                  </td>
                  <td className="px-4 py-3 text-right">
                    <button onClick={() => handleDelete(k.id)}
                      className="text-muted-foreground hover:text-destructive p-1.5 rounded hover:bg-destructive/10 transition-colors">
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
