import React, { useEffect, useState } from "react";
import {
  Save, Server, Globe, Network, Bot, Key, Loader2, Lock, FileKey,
  Send, MessageCircle, CheckCircle2
} from "lucide-react";
import { getSettings, saveSettings } from "../api";
import { cn } from "../lib/utils";

const hdrs = () => ({
  'Content-Type': 'application/json',
  Authorization: `Bearer ${localStorage.getItem('kumoui_token') || ''}`,
});

export default function Settings() {
  const [form, setForm] = useState({
    main_hostname: "", main_server_ip: "", relay_ips: "",
    tls_cert_path: "", tls_key_path: "",
    ai_provider: "", ai_api_key: "",
    ollama_base_url: "", ollama_model: "",
    telegram_bot_token: "", telegram_chat_id: "",
    telegram_enabled: false, telegram_digest_hour: 8,
    discord_webhook_url: "", discord_enabled: false,
    discord_bot_token: "", discord_application_id: "", discord_public_key: "", discord_bot_enabled: false,
    server_label: "",
  });
  const [saving, setSaving] = useState(false);
  const [msg, setMsg] = useState("");
  const [testMsg, setTestMsg] = useState({ tg: '', dc: '', discord_bot: '' });

  useEffect(() => {
    (async () => {
      try {
        const s = await getSettings();
        setForm((f) => ({ ...f, ...s }));
      } catch (err) {
        setMsg("Failed to load settings");
      }
    })();
  }, []);

  const onChange = (e) => {
    const { name, value, type, checked } = e.target;
    setForm((f) => ({ ...f, [name]: type === 'checkbox' ? checked : value }));
  };

  const testTelegram = async () => {
    setTestMsg(m => ({ ...m, tg: 'Sending…' }));
    try {
      const res = await fetch('/api/notify/test-telegram', {
        method: 'POST', headers: hdrs(),
        body: JSON.stringify({ token: form.telegram_bot_token, chat_id: form.telegram_chat_id }),
      });
      const d = await res.json();
      setTestMsg(m => ({ ...m, tg: res.ok ? '✅ Message sent!' : '❌ ' + d.error }));
    } catch { setTestMsg(m => ({ ...m, tg: '❌ Request failed' })); }
  };

  const testDiscord = async () => {
    setTestMsg(m => ({ ...m, dc: 'Sending…' }));
    try {
      const res = await fetch('/api/notify/test-discord', {
        method: 'POST', headers: hdrs(),
        body: JSON.stringify({ url: form.discord_webhook_url }),
      });
      const d = await res.json();
      setTestMsg(m => ({ ...m, dc: res.ok ? '✅ Message sent!' : '❌ ' + d.error }));
    } catch { setTestMsg(m => ({ ...m, dc: '❌ Request failed' })); }
  };

  const registerDiscordCommands = async () => {
    setTestMsg(m => ({ ...m, discord_bot: 'Registering…' }));
    try {
      const res = await fetch('/api/discord/register-commands', { method: 'POST', headers: hdrs() });
      const d = await res.json();
      setTestMsg(m => ({ ...m, discord_bot: res.ok ? '✅ ' + d.status : '❌ ' + d.error }));
    } catch { setTestMsg(m => ({ ...m, discord_bot: '❌ Request failed' })); }
  };

  const onSubmit = async (e) => {
    e.preventDefault();
    setSaving(true);
    setMsg("");
    try {
      await saveSettings(form);
      setMsg("Settings saved successfully.");
      setForm((f) => ({ ...f, ai_api_key: "" })); // clear sensitive field
    } catch (err) {
      setMsg(err.message || "Failed to save settings");
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="max-w-2xl mx-auto space-y-8 py-4">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">System Settings</h1>
        <p className="text-muted-foreground">Configure global server parameters and integrations.</p>
      </div>

      <form onSubmit={onSubmit} className="space-y-8">
        
        {/* Server Config Section */}
        <div className="space-y-4">
          <h3 className="text-lg font-semibold flex items-center gap-2 border-b pb-2">
            <Server className="w-5 h-5" /> Server Configuration
          </h3>
          
          <div className="grid gap-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">Main Hostname</label>
              <div className="relative">
                <Globe className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
                <input
                  name="main_hostname"
                  value={form.main_hostname}
                  onChange={onChange}
                  className="w-full pl-9 h-10 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring"
                  placeholder="mta.example.com"
                />
              </div>
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">Main Server IP</label>
              <div className="relative">
                <Network className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
                <input
                  name="main_server_ip"
                  value={form.main_server_ip}
                  onChange={onChange}
                  className="w-full pl-9 h-10 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring"
                  placeholder="1.2.3.4"
                />
              </div>
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">Server Label</label>
              <input
                name="server_label"
                value={form.server_label}
                onChange={onChange}
                className="w-full h-10 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring"
                placeholder="NYC-01"
              />
              <p className="text-[10px] text-muted-foreground">Short name for this VPS. Prefixed on all Telegram/Discord alerts so you know which server sent it when managing multiple VPS.</p>
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">Relay IPs (CSV)</label>
              <input
                name="relay_ips"
                value={form.relay_ips}
                onChange={onChange}
                className="w-full h-10 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring"
                placeholder="127.0.0.1, 10.0.0.5"
              />
              <p className="text-[10px] text-muted-foreground">IPs allowed to relay through this MTA without authentication.</p>
            </div>
          </div>
        </div>

        {/* TLS/SSL Section */}
        <div className="space-y-4">
          <h3 className="text-lg font-semibold flex items-center gap-2 border-b pb-2">
            <Lock className="w-5 h-5" /> TLS/SSL Configuration
          </h3>
          <p className="text-sm text-muted-foreground">Required for SMTP authentication on ports 587/465. Without TLS, AUTH will not work.</p>

          <div className="grid sm:grid-cols-2 gap-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">TLS Certificate Path</label>
              <div className="relative">
                <FileKey className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
                <input
                  name="tls_cert_path"
                  value={form.tls_cert_path}
                  onChange={onChange}
                  className="w-full pl-9 h-10 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring"
                  placeholder="/etc/ssl/certs/mail.crt"
                />
              </div>
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">TLS Private Key Path</label>
              <div className="relative">
                <Key className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
                <input
                  name="tls_key_path"
                  value={form.tls_key_path}
                  onChange={onChange}
                  className="w-full pl-9 h-10 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring"
                  placeholder="/etc/ssl/private/mail.key"
                />
              </div>
            </div>
          </div>
          <p className="text-[10px] text-muted-foreground">Use Let's Encrypt or your certificate provider. Common paths: /etc/letsencrypt/live/yourdomain/fullchain.pem and privkey.pem</p>
        </div>

        {/* AI Integration Section */}
        <div className="space-y-4">
          <h3 className="text-lg font-semibold flex items-center gap-2 border-b pb-2">
            <Bot className="w-5 h-5" /> AI Integration
          </h3>

          {/* Provider cards */}
          {(() => {
            const providers = [
              { id: 'openai',    label: 'OpenAI',      model: 'GPT-4o-mini',              badge: 'Cloud',  badgeCls: 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-400',  note: 'Reliable. Best overall quality.' },
              { id: 'anthropic', label: 'Anthropic',   model: 'Claude 3.5 Haiku',         badge: 'Cloud',  badgeCls: 'bg-orange-100 text-orange-700 dark:bg-orange-900/40 dark:text-orange-400', note: 'Excellent reasoning. Great for analysis.' },
              { id: 'gemini',    label: 'Google',      model: 'Gemini 2.0 Flash',         badge: 'Cloud',  badgeCls: 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-400',   note: 'Fast & cheap. Free tier available.' },
              { id: 'groq',      label: 'Groq',        model: 'Llama 3.3 70B',            badge: 'Cloud',  badgeCls: 'bg-purple-100 text-purple-700 dark:bg-purple-900/40 dark:text-purple-400', note: 'Ultra-fast inference. Generous free tier.' },
              { id: 'mistral',   label: 'Mistral',     model: 'Mistral Small',            badge: 'Cloud',  badgeCls: 'bg-sky-100 text-sky-700 dark:bg-sky-900/40 dark:text-sky-400',     note: 'Strong European model. Privacy-focused.' },
              { id: 'together',  label: 'Together AI', model: 'Llama 3.2 11B',            badge: 'Cloud',  badgeCls: 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/40 dark:text-indigo-400', note: '100+ open-source models on demand.' },
              { id: 'deepseek',  label: 'DeepSeek',    model: 'DeepSeek Chat',            badge: 'Cloud',  badgeCls: 'bg-cyan-100 text-cyan-700 dark:bg-cyan-900/40 dark:text-cyan-400',  note: 'Affordable. Strong code & reasoning.' },
              { id: 'ollama',    label: 'Ollama',      model: 'Your local model',         badge: '🖥 Local', badgeCls: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-400', note: 'FREE. Runs on your VPS — zero cost, full privacy.' },
            ];
            return (
              <div className="grid grid-cols-2 md:grid-cols-4 gap-2">
                {providers.map(p => (
                  <button key={p.id} type="button"
                    onClick={() => setForm(f => ({ ...f, ai_provider: p.id }))}
                    className={cn(
                      'flex flex-col items-start gap-1 p-3 rounded-lg border text-left transition-all',
                      form.ai_provider === p.id
                        ? 'border-primary bg-primary/5 ring-2 ring-primary/30'
                        : 'border-border hover:border-primary/50 hover:bg-muted/30'
                    )}>
                    <div className="flex items-center justify-between w-full">
                      <span className="text-sm font-bold">{p.label}</span>
                      <span className={cn('text-[10px] font-semibold px-1.5 py-0.5 rounded', p.badgeCls)}>{p.badge}</span>
                    </div>
                    <span className="text-[11px] text-muted-foreground font-medium">{p.model}</span>
                    <span className="text-[10px] text-muted-foreground leading-tight">{p.note}</span>
                  </button>
                ))}
              </div>
            );
          })()}

          {/* API Key — hidden for Ollama */}
          {form.ai_provider && form.ai_provider !== 'ollama' && (
            <div className="grid sm:grid-cols-2 gap-4">
              <div className="space-y-2">
                <label className="text-sm font-medium">API Key</label>
                <div className="relative">
                  <Key className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
                  <input
                    name="ai_api_key"
                    type="password"
                    value={form.ai_api_key}
                    onChange={onChange}
                    className="w-full pl-9 h-10 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring"
                    placeholder={
                      form.ai_provider === 'openai'    ? 'sk-...' :
                      form.ai_provider === 'anthropic' ? 'sk-ant-...' :
                      form.ai_provider === 'gemini'    ? 'AIza...' :
                      form.ai_provider === 'groq'      ? 'gsk_...' :
                      form.ai_provider === 'mistral'   ? 'your Mistral API key' :
                      form.ai_provider === 'together'  ? 'your Together AI key' :
                      'your API key'
                    }
                  />
                </div>
                <p className="text-[10px] text-muted-foreground">Key is write-only and encrypted at rest.</p>
              </div>
              <div className="space-y-2 self-start pt-1">
                {form.ai_provider === 'openai'    && <a href="https://platform.openai.com/api-keys" target="_blank" rel="noreferrer" className="text-xs text-primary hover:underline">→ Get OpenAI key</a>}
                {form.ai_provider === 'anthropic' && <a href="https://console.anthropic.com/settings/keys" target="_blank" rel="noreferrer" className="text-xs text-primary hover:underline">→ Get Anthropic key</a>}
                {form.ai_provider === 'gemini'    && <a href="https://aistudio.google.com/app/apikey" target="_blank" rel="noreferrer" className="text-xs text-primary hover:underline">→ Get Gemini key (free)</a>}
                {form.ai_provider === 'groq'      && <a href="https://console.groq.com/keys" target="_blank" rel="noreferrer" className="text-xs text-primary hover:underline">→ Get Groq key (free tier)</a>}
                {form.ai_provider === 'mistral'   && <a href="https://console.mistral.ai/api-keys" target="_blank" rel="noreferrer" className="text-xs text-primary hover:underline">→ Get Mistral key</a>}
                {form.ai_provider === 'together'  && <a href="https://api.together.xyz/settings/api-keys" target="_blank" rel="noreferrer" className="text-xs text-primary hover:underline">→ Get Together AI key</a>}
                {form.ai_provider === 'deepseek'  && <a href="https://platform.deepseek.com/api_keys" target="_blank" rel="noreferrer" className="text-xs text-primary hover:underline">→ Get DeepSeek key</a>}
              </div>
            </div>
          )}

          {/* Ollama config — shown only when Ollama is selected */}
          {form.ai_provider === 'ollama' && (
            <div className="border rounded-lg p-4 space-y-3 bg-emerald-50/50 dark:bg-emerald-950/20 border-emerald-200 dark:border-emerald-800/50">
              <div className="flex items-start gap-2">
                <span className="text-lg">🖥</span>
                <div>
                  <p className="text-sm font-semibold text-emerald-700 dark:text-emerald-400">Ollama — Free local AI on your VPS</p>
                  <p className="text-xs text-muted-foreground mt-0.5">No API key needed. Install Ollama, pull a model, done.</p>
                </div>
              </div>
              <div className="grid sm:grid-cols-2 gap-3">
                <div className="space-y-1">
                  <label className="text-xs font-medium text-muted-foreground">Ollama Base URL</label>
                  <input name="ollama_base_url" value={form.ollama_base_url} onChange={onChange}
                    placeholder="http://localhost:11434"
                    className="w-full h-9 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring" />
                  <p className="text-[10px] text-muted-foreground">Default: http://localhost:11434</p>
                </div>
                <div className="space-y-1">
                  <label className="text-xs font-medium text-muted-foreground">Model Name</label>
                  <input name="ollama_model" value={form.ollama_model} onChange={onChange}
                    placeholder="llama3.2"
                    className="w-full h-9 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring" />
                  <p className="text-[10px] text-muted-foreground">e.g. llama3.2, mistral, phi4, gemma3, qwen2.5</p>
                </div>
              </div>
              <div className="bg-muted/60 rounded-md p-3 space-y-1">
                <p className="text-xs font-semibold">Quick setup on Rocky Linux 9:</p>
                <pre className="text-[11px] text-muted-foreground font-mono whitespace-pre-wrap leading-relaxed">{`curl -fsSL https://ollama.com/install.sh | sh
ollama pull llama3.2        # 2GB — fast & capable
# or: ollama pull mistral   # 4GB — great for analysis
# or: ollama pull phi4      # 9GB — Microsoft's best small model
systemctl enable --now ollama`}</pre>
              </div>
            </div>
          )}
        </div>

        {/* Telegram Bot Section */}
        <div className="space-y-4">
          <h3 className="text-lg font-semibold flex items-center gap-2 border-b pb-2">
            <Send className="w-5 h-5 text-blue-500" /> Telegram Bot
          </h3>
          <p className="text-xs text-muted-foreground">
            Create a bot via <strong>@BotFather</strong>, get your Chat ID from <strong>@userinfobot</strong>.
            Supports daily digest + full chat-ops: /stats /queue /flush /restart /campaigns /warmup /tail /disk /mem and more.
            Enter multiple Chat IDs comma-separated to allow a whole team — only listed chats can run commands.
          </p>
          <div className="grid sm:grid-cols-2 gap-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">Bot Token</label>
              <input name="telegram_bot_token" value={form.telegram_bot_token} onChange={onChange}
                placeholder="123456789:AAxxxxxx…" type="password"
                className="w-full h-10 rounded-md border bg-background px-3 text-sm font-mono focus:ring-2 focus:ring-ring" />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">Allowed Chat ID(s)</label>
              <input name="telegram_chat_id" value={form.telegram_chat_id} onChange={onChange}
                placeholder="-100123456789, 987654321"
                className="w-full h-10 rounded-md border bg-background px-3 text-sm font-mono focus:ring-2 focus:ring-ring" />
              <p className="text-[10px] text-muted-foreground">Comma-separated. Digest + alerts go to the first ID. All listed IDs can run commands.</p>
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">Daily Digest Hour (0–23)</label>
              <input name="telegram_digest_hour" type="number" min="0" max="23"
                value={form.telegram_digest_hour} onChange={onChange}
                className="w-full h-10 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring" />
            </div>
            <div className="flex items-end gap-4 pb-1">
              <label className="flex items-center gap-2 text-sm font-medium cursor-pointer">
                <input type="checkbox" name="telegram_enabled" checked={!!form.telegram_enabled} onChange={onChange}
                  className="w-4 h-4 rounded" />
                Enable Telegram notifications
              </label>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <button type="button" onClick={testTelegram}
              className="flex items-center gap-2 h-9 px-4 rounded-md border text-sm font-medium hover:bg-accent transition-colors">
              <MessageCircle className="w-4 h-4" /> Send Test Message
            </button>
            {testMsg.tg && <span className={cn('text-sm', testMsg.tg.startsWith('✅') ? 'text-green-600' : 'text-red-500')}>{testMsg.tg}</span>}
          </div>
        </div>

        {/* Discord Webhook Section */}
        <div className="space-y-4">
          <h3 className="text-lg font-semibold flex items-center gap-2 border-b pb-2">
            <MessageCircle className="w-5 h-5 text-indigo-500" /> Discord Webhook
          </h3>
          <p className="text-xs text-muted-foreground">
            In Discord: Server Settings → Integrations → Webhooks → Create Webhook → Copy URL.
            Sends daily digest with rich embeds.
          </p>
          <div className="grid sm:grid-cols-2 gap-4">
            <div className="space-y-2 sm:col-span-2">
              <label className="text-sm font-medium">Webhook URL</label>
              <input name="discord_webhook_url" value={form.discord_webhook_url} onChange={onChange}
                placeholder="https://discord.com/api/webhooks/…"
                className="w-full h-10 rounded-md border bg-background px-3 text-sm font-mono focus:ring-2 focus:ring-ring" />
            </div>
            <div className="flex items-center gap-2">
              <label className="flex items-center gap-2 text-sm font-medium cursor-pointer">
                <input type="checkbox" name="discord_enabled" checked={!!form.discord_enabled} onChange={onChange}
                  className="w-4 h-4 rounded" />
                Enable Discord notifications
              </label>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <button type="button" onClick={testDiscord}
              className="flex items-center gap-2 h-9 px-4 rounded-md border text-sm font-medium hover:bg-accent transition-colors">
              <CheckCircle2 className="w-4 h-4" /> Send Test Embed
            </button>
            {testMsg.dc && <span className={cn('text-sm', testMsg.dc.startsWith('✅') ? 'text-green-600' : 'text-red-500')}>{testMsg.dc}</span>}
          </div>
        </div>

        {/* Discord Bot (Interactive Slash Commands) */}
        <div className="space-y-4">
          <h3 className="text-lg font-semibold flex items-center gap-2 border-b pb-2">
            <MessageCircle className="w-5 h-5 text-violet-500" /> Discord Bot — Slash Commands
          </h3>
          <p className="text-xs text-muted-foreground">
            Full two-way interactive bot. Go to{' '}
            <strong>discord.com/developers/applications</strong> → New Application → Bot → copy the token.
            Set the <strong>Interactions Endpoint URL</strong> in your app to{' '}
            <code className="bg-muted px-1 rounded text-[11px]">https://your-domain/api/discord/interactions</code>.
            Then click <strong>Register Commands</strong> once.
          </p>
          <div className="grid sm:grid-cols-2 gap-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">Application / Client ID</label>
              <input name="discord_application_id" value={form.discord_application_id} onChange={onChange}
                placeholder="1234567890123456789"
                className="w-full h-10 rounded-md border bg-background px-3 text-sm font-mono focus:ring-2 focus:ring-ring" />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">Bot Token</label>
              <input name="discord_bot_token" value={form.discord_bot_token} onChange={onChange}
                type="password" placeholder="MTI3…"
                className="w-full h-10 rounded-md border bg-background px-3 text-sm font-mono focus:ring-2 focus:ring-ring" />
            </div>
            <div className="space-y-2 sm:col-span-2">
              <label className="text-sm font-medium">Public Key <span className="text-muted-foreground font-normal">(for signature verification)</span></label>
              <input name="discord_public_key" value={form.discord_public_key} onChange={onChange}
                placeholder="a1b2c3… (from Developer Portal → General Information)"
                className="w-full h-10 rounded-md border bg-background px-3 text-sm font-mono focus:ring-2 focus:ring-ring" />
            </div>
            <div className="flex items-center gap-2">
              <label className="flex items-center gap-2 text-sm font-medium cursor-pointer">
                <input type="checkbox" name="discord_bot_enabled" checked={!!form.discord_bot_enabled} onChange={onChange}
                  className="w-4 h-4 rounded" />
                Enable Discord Bot
              </label>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <button type="button" onClick={registerDiscordCommands}
              className="flex items-center gap-2 h-9 px-4 rounded-md bg-violet-600 text-white hover:bg-violet-700 text-sm font-medium transition-colors">
              <CheckCircle2 className="w-4 h-4" /> Register Slash Commands
            </button>
            {testMsg.discord_bot && (
              <span className={cn('text-sm', testMsg.discord_bot.startsWith('✅') ? 'text-green-600' : 'text-red-500')}>
                {testMsg.discord_bot}
              </span>
            )}
          </div>
        </div>

        {/* Footer Actions */}
        <div className="flex items-center justify-between pt-4">
          <div className={cn("text-sm font-medium", msg.includes("Failed") ? "text-destructive" : "text-green-600")}>
            {msg}
          </div>
          <button
            type="submit"
            disabled={saving}
            className="flex items-center gap-2 bg-primary text-primary-foreground hover:bg-primary/90 px-6 py-2 rounded-md text-sm font-medium transition-colors disabled:opacity-50"
          >
            {saving ? <Loader2 className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
            {saving ? "Saving..." : "Save Changes"}
          </button>
        </div>
      </form>
    </div>
  );
}
