import React, { useState, useEffect, useCallback } from 'react';
import {
  ShieldCheck,
  ShieldX,
  Copy,
  Check,
  ChevronDown,
  ChevronUp,
  Globe,
  Image,
  Lock,
  Mail,
  AlertCircle,
  RefreshCw,
  Save,
  Loader2,
  CheckCircle2,
  XCircle
} from 'lucide-react';
import { apiRequest, listDomains } from '../api';
import { cn } from '../lib/utils';

// --- Utility ---
function useCopy() {
  const [copied, setCopied] = useState(null);
  const copy = (text, key) => {
    navigator.clipboard.writeText(text).then(() => {
      setCopied(key);
      setTimeout(() => setCopied(null), 2000);
    });
  };
  return { copied, copy };
}

function CodeBlock({ text, copyKey, copied, onCopy }) {
  return (
    <div className="relative group">
      <pre className="bg-zinc-950 border border-zinc-800 rounded-lg p-4 text-xs font-mono text-zinc-300 whitespace-pre-wrap break-all leading-relaxed">
        {text}
      </pre>
      <button
        onClick={() => onCopy(text, copyKey)}
        className="absolute top-2 right-2 p-1.5 rounded-md bg-zinc-800 hover:bg-zinc-700 text-zinc-400 hover:text-white opacity-0 group-hover:opacity-100 transition-all"
        title="Copy to clipboard"
      >
        {copied === copyKey ? <Check className="w-3.5 h-3.5 text-green-400" /> : <Copy className="w-3.5 h-3.5" />}
      </button>
    </div>
  );
}

function Toggle({ checked, onChange, label, description }) {
  return (
    <div className="flex items-center justify-between p-4 bg-muted/30 rounded-lg border">
      <div>
        <p className="text-sm font-medium">{label}</p>
        {description && <p className="text-xs text-muted-foreground mt-0.5">{description}</p>}
      </div>
      <button
        type="button"
        onClick={() => onChange(!checked)}
        className={cn(
          "relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors focus:outline-none",
          checked ? "bg-primary" : "bg-muted-foreground/30"
        )}
        role="switch"
        aria-checked={checked}
      >
        <span
          className={cn(
            "pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out",
            checked ? "translate-x-5" : "translate-x-0"
          )}
        />
      </button>
    </div>
  );
}

function SectionAccordion({ title, icon: Icon, children, defaultOpen = false }) {
  const [open, setOpen] = useState(defaultOpen);
  return (
    <div className="bg-card border rounded-xl shadow-sm overflow-hidden">
      <button
        onClick={() => setOpen(!open)}
        className="w-full flex items-center justify-between p-5 hover:bg-muted/30 transition-colors"
      >
        <div className="flex items-center gap-3">
          <div className="p-2 bg-primary/10 rounded-lg">
            <Icon className="w-4 h-4 text-primary" />
          </div>
          <span className="font-semibold text-base">{title}</span>
        </div>
        {open ? <ChevronUp className="w-4 h-4 text-muted-foreground" /> : <ChevronDown className="w-4 h-4 text-muted-foreground" />}
      </button>
      {open && <div className="border-t p-5 space-y-4">{children}</div>}
    </div>
  );
}

// --- BIMI Section ---
function BimiSection({ domain }) {
  const [config, setConfig] = useState({ is_enabled: false, logo_url: '', vmc_url: '', dns_record: '' });
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const { copied, copy } = useCopy();

  const load = useCallback(async () => {
    if (!domain) return;
    setLoading(true);
    setError('');
    try {
      const data = await apiRequest(`/authtools/bimi/${domain}`);
      setConfig(data);
    } catch (e) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }, [domain]);

  useEffect(() => { load(); }, [load]);

  const save = async () => {
    setSaving(true);
    setError('');
    setSuccess('');
    try {
      const data = await apiRequest(`/authtools/bimi/${domain}`, {
        method: 'POST',
        body: { logo_url: config.logo_url, vmc_url: config.vmc_url, is_enabled: config.is_enabled }
      });
      setConfig(data);
      setSuccess('BIMI configuration saved.');
      setTimeout(() => setSuccess(''), 3000);
    } catch (e) {
      setError(e.message);
    } finally {
      setSaving(false);
    }
  };

  if (loading) return <div className="py-6 text-center text-muted-foreground text-sm">Loading BIMI config...</div>;

  return (
    <div className="space-y-4">
      {error && <div className="bg-destructive/10 text-destructive p-3 rounded-md text-sm flex items-center gap-2"><AlertCircle className="w-4 h-4 flex-shrink-0" />{error}</div>}
      {success && <div className="bg-green-500/10 text-green-600 dark:text-green-400 p-3 rounded-md text-sm flex items-center gap-2"><CheckCircle2 className="w-4 h-4 flex-shrink-0" />{success}</div>}

      <Toggle
        checked={config.is_enabled}
        onChange={(v) => setConfig({ ...config, is_enabled: v })}
        label="Enable BIMI"
        description="Brand Indicators for Message Identification — display your logo in supporting email clients."
      />

      <div className="space-y-3">
        <div>
          <label className="text-sm font-medium block mb-1.5">Logo URL <span className="text-muted-foreground font-normal">(HTTPS, SVG format)</span></label>
          <input
            type="url"
            value={config.logo_url || ''}
            onChange={e => setConfig({ ...config, logo_url: e.target.value })}
            placeholder="https://yourdomain.com/logo.svg"
            className="w-full h-10 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring focus:outline-none"
          />
          {config.logo_url && (
            <div className="mt-2 flex items-center gap-3">
              <img
                src={config.logo_url}
                alt="Logo preview"
                className="h-12 w-12 object-contain border rounded-md bg-white p-1"
                onError={e => { e.target.style.display = 'none'; }}
              />
              <span className="text-xs text-muted-foreground">Logo preview</span>
            </div>
          )}
        </div>

        <div>
          <label className="text-sm font-medium block mb-1.5">VMC URL <span className="text-muted-foreground font-normal">(optional — Verified Mark Certificate)</span></label>
          <input
            type="url"
            value={config.vmc_url || ''}
            onChange={e => setConfig({ ...config, vmc_url: e.target.value })}
            placeholder="https://yourdomain.com/logo.pem"
            className="w-full h-10 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring focus:outline-none"
          />
        </div>
      </div>

      {config.dns_record && (
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <p className="text-sm font-medium">Generated DNS Record</p>
            <span className="text-xs text-muted-foreground">Publish at: <code className="bg-muted px-1 rounded">default._bimi.{domain}</code></span>
          </div>
          <CodeBlock text={config.dns_record} copyKey="bimi-dns" copied={copied} onCopy={copy} />
          <p className="text-xs text-muted-foreground">Add this TXT record to your DNS provider for the domain <strong>default._bimi.{domain}</strong>.</p>
        </div>
      )}

      <div className="flex justify-end">
        <button
          onClick={save}
          disabled={saving}
          className="flex items-center gap-2 h-9 px-4 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-sm font-medium transition-colors"
        >
          {saving ? <Loader2 className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
          Save BIMI Config
        </button>
      </div>
    </div>
  );
}

// --- MTA-STS Section ---
function MtaStsSection({ domain }) {
  const [config, setConfig] = useState({ is_enabled: false, mode: 'testing', max_age: 86400, mx_hosts: [], policy_file: '', dns_record: '' });
  const [mxText, setMxText] = useState('');
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const { copied, copy } = useCopy();

  const modeDescriptions = {
    none: 'No MTA-STS policy is enforced. Safe to use while testing.',
    testing: 'Policy is advertised but not enforced. Failures are reported only.',
    enforce: 'Policy is enforced. Senders that cannot verify TLS will fail delivery.'
  };

  const load = useCallback(async () => {
    if (!domain) return;
    setLoading(true);
    setError('');
    try {
      const data = await apiRequest(`/authtools/mtasts/${domain}`);
      setConfig(data);
      setMxText((data.mx_hosts || []).join('\n'));
    } catch (e) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }, [domain]);

  useEffect(() => { load(); }, [load]);

  const save = async () => {
    setSaving(true);
    setError('');
    setSuccess('');
    try {
      const mx_hosts = mxText.split('\n').map(s => s.trim()).filter(Boolean);
      const data = await apiRequest(`/authtools/mtasts/${domain}`, {
        method: 'POST',
        body: { mode: config.mode, max_age: Number(config.max_age), mx_hosts, is_enabled: config.is_enabled }
      });
      setConfig(data);
      setMxText((data.mx_hosts || []).join('\n'));
      setSuccess('MTA-STS configuration saved.');
      setTimeout(() => setSuccess(''), 3000);
    } catch (e) {
      setError(e.message);
    } finally {
      setSaving(false);
    }
  };

  if (loading) return <div className="py-6 text-center text-muted-foreground text-sm">Loading MTA-STS config...</div>;

  return (
    <div className="space-y-4">
      {error && <div className="bg-destructive/10 text-destructive p-3 rounded-md text-sm flex items-center gap-2"><AlertCircle className="w-4 h-4 flex-shrink-0" />{error}</div>}
      {success && <div className="bg-green-500/10 text-green-600 dark:text-green-400 p-3 rounded-md text-sm flex items-center gap-2"><CheckCircle2 className="w-4 h-4 flex-shrink-0" />{success}</div>}

      <Toggle
        checked={config.is_enabled}
        onChange={(v) => setConfig({ ...config, is_enabled: v })}
        label="Enable MTA-STS"
        description="Prevents downgrade attacks and requires TLS for inbound email."
      />

      <div>
        <label className="text-sm font-medium block mb-2">Mode</label>
        <div className="grid grid-cols-1 gap-2">
          {['none', 'testing', 'enforce'].map(mode => (
            <label
              key={mode}
              className={cn(
                "flex items-start gap-3 p-3 rounded-md border cursor-pointer transition-all",
                config.mode === mode ? "border-primary bg-primary/5 ring-1 ring-primary" : "hover:bg-muted"
              )}
            >
              <input
                type="radio"
                name="mtasts-mode"
                value={mode}
                checked={config.mode === mode}
                onChange={() => setConfig({ ...config, mode })}
                className="mt-0.5 h-4 w-4 text-primary"
              />
              <div>
                <span className="font-medium capitalize text-sm">{mode}</span>
                <p className="text-xs text-muted-foreground mt-0.5">{modeDescriptions[mode]}</p>
              </div>
            </label>
          ))}
        </div>
      </div>

      <div>
        <label className="text-sm font-medium block mb-1.5">Max Age <span className="text-muted-foreground font-normal">(seconds)</span></label>
        <input
          type="number"
          value={config.max_age || 86400}
          onChange={e => setConfig({ ...config, max_age: e.target.value })}
          className="w-full h-10 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring focus:outline-none"
          min={60}
          max={31557600}
        />
        <p className="text-xs text-muted-foreground mt-1">Default: 86400 (1 day). Max recommended: 604800 (7 days).</p>
      </div>

      <div>
        <label className="text-sm font-medium block mb-1.5">MX Hosts <span className="text-muted-foreground font-normal">(one per line)</span></label>
        <textarea
          value={mxText}
          onChange={e => setMxText(e.target.value)}
          rows={4}
          placeholder={'mail.yourdomain.com\nmx2.yourdomain.com'}
          className="w-full rounded-md border bg-background px-3 py-2 text-sm font-mono focus:ring-2 focus:ring-ring focus:outline-none resize-y"
        />
      </div>

      {config.policy_file && (
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <p className="text-sm font-medium">Generated Policy File</p>
            <span className="text-xs text-muted-foreground">Serve at: <code className="bg-muted px-1 rounded">https://mta-sts.{domain}/.well-known/mta-sts.txt</code></span>
          </div>
          <CodeBlock text={config.policy_file} copyKey="sts-policy" copied={copied} onCopy={copy} />
        </div>
      )}

      {config.dns_record && (
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <p className="text-sm font-medium">Generated DNS Record</p>
            <span className="text-xs text-muted-foreground">Publish at: <code className="bg-muted px-1 rounded">_mta-sts.{domain}</code></span>
          </div>
          <CodeBlock text={config.dns_record} copyKey="sts-dns" copied={copied} onCopy={copy} />
        </div>
      )}

      <div className="flex justify-end">
        <button
          onClick={save}
          disabled={saving}
          className="flex items-center gap-2 h-9 px-4 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-sm font-medium transition-colors"
        >
          {saving ? <Loader2 className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
          Save MTA-STS Config
        </button>
      </div>
    </div>
  );
}

// --- TLS-RPT Section ---
function TlsRptSection({ domain }) {
  const [reportEmail, setReportEmail] = useState('');
  const { copied, copy } = useCopy();

  const dnsRecord = reportEmail ? `v=TLSRPTv1; rua=mailto:${reportEmail}` : '';

  return (
    <div className="space-y-4">
      <div className="bg-blue-500/10 border border-blue-500/20 rounded-lg p-4 text-sm text-blue-700 dark:text-blue-300 space-y-2">
        <p className="font-semibold flex items-center gap-2"><Mail className="w-4 h-4" /> About TLS-RPT</p>
        <p>TLS Reporting (RFC 8460) allows receiving servers to send diagnostic reports about TLS failures when delivering to your domain. Publish a <code className="bg-blue-500/10 px-1 rounded">_smtp._tls.{domain}</code> TXT record to receive these reports.</p>
      </div>

      <div>
        <label className="text-sm font-medium block mb-1.5">Report Email Address</label>
        <input
          type="email"
          value={reportEmail}
          onChange={e => setReportEmail(e.target.value)}
          placeholder={`tls-reports@${domain || 'yourdomain.com'}`}
          className="w-full h-10 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring focus:outline-none"
        />
        <p className="text-xs text-muted-foreground mt-1">TLS failure reports will be sent to this address in JSON format.</p>
      </div>

      {dnsRecord && (
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <p className="text-sm font-medium">Generated DNS Record</p>
            <span className="text-xs text-muted-foreground">Publish at: <code className="bg-muted px-1 rounded">_smtp._tls.{domain}</code></span>
          </div>
          <CodeBlock text={dnsRecord} copyKey="tlsrpt-dns" copied={copied} onCopy={copy} />
          <p className="text-xs text-muted-foreground">Add this TXT record to your DNS provider for <strong>_smtp._tls.{domain}</strong>.</p>
        </div>
      )}
    </div>
  );
}

// --- Auth Checklist ---
function AuthChecklist({ domain }) {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    if (!domain) return;
    setLoading(true);
    setError('');
    try {
      const res = await apiRequest(`/authtools/check/${domain}`);
      setData(res);
    } catch (e) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }, [domain]);

  useEffect(() => { load(); }, [load]);

  if (!domain) return null;

  return (
    <div className="bg-card border rounded-xl shadow-sm p-5">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-base font-semibold flex items-center gap-2">
          <ShieldCheck className="w-5 h-5 text-primary" /> Auth Checklist
        </h2>
        <button onClick={load} disabled={loading} className="p-1.5 hover:bg-muted rounded-md transition-colors" title="Refresh">
          <RefreshCw className={cn("w-4 h-4 text-muted-foreground", loading && "animate-spin")} />
        </button>
      </div>

      {error && <div className="bg-destructive/10 text-destructive p-3 rounded-md text-sm mb-3">{error}</div>}

      {loading && !data && (
        <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
          {['SPF', 'DKIM', 'DMARC', 'BIMI', 'MTA-STS', 'TLS-RPT'].map(name => (
            <div key={name} className="flex items-center gap-2 p-3 rounded-lg border bg-muted/30 animate-pulse">
              <div className="w-5 h-5 rounded-full bg-muted" />
              <span className="text-sm font-medium text-muted-foreground">{name}</span>
            </div>
          ))}
        </div>
      )}

      {data && (
        <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
          {(data.checklist || []).map(item => (
            <div
              key={item.name}
              className={cn(
                "flex items-start gap-2 p-3 rounded-lg border transition-colors",
                item.configured
                  ? "bg-green-500/10 border-green-500/20"
                  : "bg-muted/30 border-border"
              )}
              title={item.description}
            >
              {item.configured
                ? <CheckCircle2 className="w-5 h-5 text-green-600 dark:text-green-400 flex-shrink-0 mt-0.5" />
                : <XCircle className="w-5 h-5 text-muted-foreground flex-shrink-0 mt-0.5" />}
              <div>
                <span className={cn("text-sm font-semibold", item.configured ? "text-green-700 dark:text-green-300" : "text-muted-foreground")}>
                  {item.name}
                </span>
                {item.description && (
                  <p className="text-xs text-muted-foreground mt-0.5 leading-snug">{item.description}</p>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

// --- Main Page ---
export default function EmailAuthPage() {
  const [domains, setDomains] = useState([]);
  const [selectedDomain, setSelectedDomain] = useState('');
  const [loadingDomains, setLoadingDomains] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    const load = async () => {
      setLoadingDomains(true);
      try {
        const data = await listDomains();
        const list = Array.isArray(data) ? data : [];
        setDomains(list);
        if (list.length > 0) {
          setSelectedDomain(list[0].domain || list[0].name || '');
        }
      } catch (e) {
        setError(e.message);
      } finally {
        setLoadingDomains(false);
      }
    };
    load();
  }, []);

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Email Authentication</h1>
          <p className="text-muted-foreground">Configure advanced email auth standards to improve deliverability and brand recognition.</p>
        </div>
      </div>

      {/* Domain Selector */}
      <div className="bg-card border rounded-xl p-4 shadow-sm flex flex-col sm:flex-row items-start sm:items-center gap-3">
        <Globe className="w-5 h-5 text-muted-foreground flex-shrink-0" />
        <div className="flex-1">
          <label className="text-sm font-medium block mb-1">Select Domain</label>
          {loadingDomains ? (
            <div className="h-10 w-64 bg-muted rounded-md animate-pulse" />
          ) : (
            <select
              value={selectedDomain}
              onChange={e => setSelectedDomain(e.target.value)}
              className="h-10 w-full max-w-sm rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring focus:outline-none"
            >
              <option value="">-- Select a domain --</option>
              {domains.map(d => {
                const name = d.domain || d.name || '';
                return <option key={name} value={name}>{name}</option>;
              })}
            </select>
          )}
        </div>
        {selectedDomain && (
          <span className="text-xs bg-primary/10 text-primary px-2.5 py-1 rounded-full font-medium">{selectedDomain}</span>
        )}
      </div>

      {error && (
        <div className="bg-destructive/10 text-destructive p-4 rounded-xl flex items-center gap-2 text-sm">
          <AlertCircle className="w-4 h-4 flex-shrink-0" /> {error}
        </div>
      )}

      {selectedDomain ? (
        <>
          {/* Auth Checklist */}
          <AuthChecklist domain={selectedDomain} />

          {/* BIMI Section */}
          <SectionAccordion title="BIMI — Brand Logo in Email" icon={Image} defaultOpen>
            <BimiSection domain={selectedDomain} />
          </SectionAccordion>

          {/* MTA-STS Section */}
          <SectionAccordion title="MTA-STS — Enforce TLS for Inbound Mail" icon={Lock}>
            <MtaStsSection domain={selectedDomain} />
          </SectionAccordion>

          {/* TLS-RPT Section */}
          <SectionAccordion title="TLS-RPT — TLS Failure Reporting" icon={Mail}>
            <TlsRptSection domain={selectedDomain} />
          </SectionAccordion>
        </>
      ) : !loadingDomains ? (
        <div className="flex flex-col items-center justify-center py-16 text-center bg-card border rounded-xl">
          <ShieldX className="w-12 h-12 text-muted-foreground mb-4" />
          <h3 className="text-lg font-semibold mb-1">No Domain Selected</h3>
          <p className="text-muted-foreground text-sm">Select a domain above to configure email authentication settings.</p>
        </div>
      ) : null}
    </div>
  );
}
