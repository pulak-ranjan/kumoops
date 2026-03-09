import React, { useState, useCallback } from 'react';
import {
  Brain, RefreshCw, Zap, CheckCircle2, XCircle, AlertTriangle,
  TrendingUp, TrendingDown, Minus, Copy, ChevronDown, ChevronRight,
  FlaskConical, FileText, Sparkles, Send, BarChart2, Info
} from 'lucide-react';
import { cn } from '../lib/utils';

const API = '/api';

function authHeaders() {
  const token = localStorage.getItem('token');
  return { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` };
}

async function apiFetch(path, opts = {}) {
  const r = await fetch(API + path, { headers: authHeaders(), ...opts });
  if (!r.ok) throw new Error(await r.text());
  return r.json();
}

// ─── Shared Components ─────────────────────────────────────────────────────────
function SeverityBadge({ severity }) {
  const map = {
    critical: 'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-400',
    warning:  'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/40 dark:text-yellow-400',
    info:     'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-400',
  };
  return (
    <span className={cn('px-2 py-0.5 rounded text-xs font-semibold capitalize', map[severity] || 'bg-muted text-muted-foreground')}>
      {severity}
    </span>
  );
}

function ScoreRing({ score, size = 80 }) {
  const radius = (size - 8) / 2;
  const circumference = 2 * Math.PI * radius;
  const progress = (score / 100) * circumference;
  const color = score >= 80 ? '#22c55e' : score >= 60 ? '#eab308' : score >= 40 ? '#f97316' : '#ef4444';

  return (
    <div className="relative" style={{ width: size, height: size }}>
      <svg width={size} height={size} className="-rotate-90">
        <circle cx={size / 2} cy={size / 2} r={radius} fill="none" stroke="currentColor"
          strokeWidth={6} className="text-muted/30" />
        <circle cx={size / 2} cy={size / 2} r={radius} fill="none"
          stroke={color} strokeWidth={6} strokeLinecap="round"
          strokeDasharray={circumference} strokeDashoffset={circumference - progress}
          style={{ transition: 'stroke-dashoffset 0.5s ease' }} />
      </svg>
      <div className="absolute inset-0 flex items-center justify-center">
        <span className="text-base font-black" style={{ color }}>{score}</span>
      </div>
    </div>
  );
}

function TrendIcon({ trend }) {
  if (trend === 'improving') return <TrendingUp className="w-4 h-4 text-green-500" />;
  if (trend === 'declining') return <TrendingDown className="w-4 h-4 text-red-500" />;
  return <Minus className="w-4 h-4 text-muted-foreground" />;
}

function MarkdownView({ content }) {
  if (!content) return null;
  // Simple markdown: ## headers, **bold**, bullet lists, code
  const lines = content.split('\n');
  return (
    <div className="prose prose-sm dark:prose-invert max-w-none text-sm leading-relaxed">
      {lines.map((line, i) => {
        if (line.startsWith('## ')) return <h2 key={i} className="text-base font-bold mt-4 mb-1 text-foreground">{line.slice(3)}</h2>;
        if (line.startsWith('### ')) return <h3 key={i} className="text-sm font-semibold mt-3 mb-1 text-foreground">{line.slice(4)}</h3>;
        if (line.startsWith('- ') || line.startsWith('• ')) {
          const text = line.slice(2).replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>');
          return <li key={i} className="ml-4 text-sm text-foreground/90 list-disc" dangerouslySetInnerHTML={{ __html: text }} />;
        }
        if (line.startsWith('**') && line.endsWith('**')) return <p key={i} className="font-bold text-sm mt-2">{line.slice(2, -2)}</p>;
        if (line.trim() === '') return <div key={i} className="h-2" />;
        const formatted = line.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>').replace(/`(.+?)`/g, '<code class="bg-muted px-1 py-0.5 rounded text-xs font-mono">$1</code>');
        return <p key={i} className="text-sm text-foreground/90" dangerouslySetInnerHTML={{ __html: formatted }} />;
      })}
    </div>
  );
}

// ─── Tab: Deliverability Advisor ──────────────────────────────────────────────
function DeliverabilityAdvisorTab() {
  const [report, setReport] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [expanded, setExpanded] = useState(true);

  const runAdvisor = useCallback(async () => {
    setLoading(true); setError('');
    try {
      const r = await apiFetch('/ai/deliverability-advisor');
      setReport(r);
    } catch (e) { setError(e.message); }
    setLoading(false);
  }, []);

  const trendLabel = { improving: 'Improving ↑', stable: 'Stable →', declining: 'Declining ↓' };
  const trendColor = {
    improving: 'text-green-600 dark:text-green-400',
    stable: 'text-muted-foreground',
    declining: 'text-red-500',
  };

  return (
    <div className="space-y-4">
      <div className="flex items-start justify-between flex-wrap gap-3">
        <div>
          <p className="text-sm text-muted-foreground max-w-xl">
            AI analyzes your ISP health, anomalies, FBL complaints, bounce rates and throttle history to generate an actionable deliverability report.
          </p>
        </div>
        <button onClick={runAdvisor} disabled={loading}
          className="flex items-center gap-2 px-4 py-2 rounded-md bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 disabled:opacity-60 shrink-0">
          {loading ? <RefreshCw className="w-4 h-4 animate-spin" /> : <Brain className="w-4 h-4" />}
          {loading ? 'Analyzing…' : 'Run Advisor'}
        </button>
      </div>

      {error && <div className="text-sm text-destructive bg-destructive/10 px-4 py-2.5 rounded-md">{error}</div>}

      {!report && !loading && (
        <div className="text-center py-16 text-muted-foreground border rounded-lg">
          <Brain className="w-12 h-12 mx-auto mb-3 opacity-20" />
          <p className="text-sm font-medium">Click "Run Advisor" to get your AI deliverability report.</p>
          <p className="text-xs mt-1">Requires AI API key in Settings.</p>
        </div>
      )}

      {report && (
        <>
          {/* Score summary */}
          <div className="border rounded-lg p-5 flex items-center gap-6 bg-card flex-wrap">
            <ScoreRing score={report.score} size={88} />
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2 mb-1">
                <span className="text-xl font-black">
                  {report.score >= 80 ? 'Healthy' : report.score >= 60 ? 'Needs Attention' : report.score >= 40 ? 'At Risk' : 'Critical'}
                </span>
                <TrendIcon trend={report.trend} />
                <span className={cn('text-sm font-semibold', trendColor[report.trend])}>
                  {trendLabel[report.trend] || report.trend}
                </span>
              </div>
              <p className="text-xs text-muted-foreground">
                Generated {report.generated_at ? new Date(report.generated_at).toLocaleString() : ''}
              </p>
              {report.data_summary && (
                <div className="flex flex-wrap gap-3 mt-2">
                  {Object.entries(report.data_summary).map(([k, v]) => (
                    <span key={k} className="text-xs px-2 py-0.5 bg-muted rounded font-medium">
                      {k.replace(/_/g, ' ')}: {v}
                    </span>
                  ))}
                </div>
              )}
            </div>
          </div>

          {/* Issues */}
          {report.issues?.length > 0 && (
            <div className="border rounded-lg overflow-hidden">
              <div className="px-4 py-3 border-b bg-muted/30 flex items-center justify-between cursor-pointer"
                onClick={() => setExpanded(v => !v)}>
                <div className="flex items-center gap-2">
                  <AlertTriangle className="w-4 h-4 text-yellow-500" />
                  <span className="font-semibold text-sm">Issues Found ({report.issues.length})</span>
                </div>
                {expanded ? <ChevronDown className="w-4 h-4" /> : <ChevronRight className="w-4 h-4" />}
              </div>
              {expanded && (
                <div className="divide-y">
                  {report.issues.map((issue, i) => (
                    <div key={i} className="px-4 py-3 space-y-1">
                      <div className="flex items-center gap-2">
                        <SeverityBadge severity={issue.severity} />
                        {issue.domain && issue.domain !== 'all' && (
                          <span className="text-xs font-mono text-muted-foreground">{issue.domain}</span>
                        )}
                        {issue.isp && issue.isp !== 'all' && (
                          <span className="text-xs text-muted-foreground">— {issue.isp}</span>
                        )}
                      </div>
                      <p className="text-sm font-medium">{issue.issue}</p>
                      <p className="text-xs text-muted-foreground flex items-start gap-1">
                        <Zap className="w-3 h-3 shrink-0 mt-0.5 text-primary" />
                        {issue.action}
                      </p>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}

          {/* Full analysis */}
          {report.analysis && (
            <div className="border rounded-lg overflow-hidden">
              <div className="px-4 py-3 border-b bg-muted/30 flex items-center gap-2">
                <FileText className="w-4 h-4 text-primary" />
                <span className="font-semibold text-sm">Full Analysis</span>
              </div>
              <div className="p-4">
                <MarkdownView content={report.analysis} />
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}

// ─── Tab: Content Analyzer ────────────────────────────────────────────────────
function ContentAnalyzerTab() {
  const [form, setForm] = useState({ subject: '', html_body: '', sender_domain: '' });
  const [result, setResult] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const analyze = async () => {
    setLoading(true); setError('');
    try {
      const r = await apiFetch('/ai/analyze-content', { method: 'POST', body: JSON.stringify(form) });
      setResult(r);
    } catch (e) { setError(e.message); }
    setLoading(false);
  };

  const spamGrade = (score) => {
    if (score <= 2) return { label: 'Excellent', color: 'text-green-600 dark:text-green-400' };
    if (score <= 4) return { label: 'Good', color: 'text-green-500' };
    if (score <= 6) return { label: 'Risky', color: 'text-yellow-600 dark:text-yellow-400' };
    if (score <= 8) return { label: 'Dangerous', color: 'text-orange-500' };
    return { label: 'Spam', color: 'text-red-500' };
  };

  return (
    <div className="space-y-4">
      <p className="text-sm text-muted-foreground">Analyze your email content for spam triggers, inbox placement issues, and HTML best practices before sending.</p>

      <div className="border rounded-lg p-4 space-y-3 bg-muted/20">
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="text-xs text-muted-foreground font-medium">Subject Line</label>
            <input value={form.subject} onChange={e => setForm(f => ({ ...f, subject: e.target.value }))}
              placeholder="Your email subject line"
              className="w-full mt-1 px-3 py-2 rounded-md border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary/50" />
          </div>
          <div>
            <label className="text-xs text-muted-foreground font-medium">Sender Domain</label>
            <input value={form.sender_domain} onChange={e => setForm(f => ({ ...f, sender_domain: e.target.value }))}
              placeholder="yourdomain.com"
              className="w-full mt-1 px-3 py-2 rounded-md border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary/50" />
          </div>
          <div className="col-span-2">
            <label className="text-xs text-muted-foreground font-medium">HTML Body</label>
            <textarea value={form.html_body} onChange={e => setForm(f => ({ ...f, html_body: e.target.value }))}
              rows={8} placeholder="<html>Paste your full email HTML here…</html>"
              className="w-full mt-1 px-3 py-2 rounded-md border bg-background text-sm font-mono focus:outline-none focus:ring-2 focus:ring-primary/50" />
          </div>
        </div>
        <button onClick={analyze} disabled={loading || !form.html_body}
          className="flex items-center gap-2 px-4 py-2 rounded-md bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 disabled:opacity-60">
          {loading ? <RefreshCw className="w-4 h-4 animate-spin" /> : <Zap className="w-4 h-4" />}
          {loading ? 'Analyzing…' : 'Analyze Content'}
        </button>
      </div>

      {error && <div className="text-sm text-destructive bg-destructive/10 px-4 py-2.5 rounded-md">{error}</div>}

      {result && (
        <div className="space-y-4">
          {/* Score summary */}
          <div className="grid grid-cols-2 gap-3">
            <div className="border rounded-lg p-4 text-center bg-card">
              <div className={cn('text-3xl font-black', spamGrade(result.spam_score).color)}>
                {result.spam_score?.toFixed(1)}/10
              </div>
              <div className={cn('text-sm font-semibold mt-0.5', spamGrade(result.spam_score).color)}>
                {spamGrade(result.spam_score).label}
              </div>
              <div className="text-xs text-muted-foreground mt-1">Spam Score (lower = better)</div>
            </div>
            <div className="border rounded-lg p-4 text-center bg-card">
              <ScoreRing score={result.deliverability_score || 0} size={72} />
              <div className="text-xs text-muted-foreground mt-1">Deliverability Score</div>
            </div>
          </div>

          {/* Issues */}
          {result.issues?.length > 0 && (
            <div className="border rounded-lg overflow-hidden">
              <div className="px-4 py-2.5 border-b bg-muted/30 flex items-center gap-2">
                <XCircle className="w-4 h-4 text-red-500" />
                <span className="text-sm font-semibold">Issues Detected</span>
              </div>
              <ul className="divide-y">
                {result.issues.map((issue, i) => (
                  <li key={i} className="px-4 py-2.5 text-sm flex items-start gap-2">
                    <AlertTriangle className="w-4 h-4 text-yellow-500 shrink-0 mt-0.5" />
                    {issue}
                  </li>
                ))}
              </ul>
            </div>
          )}

          {/* Suggestions */}
          {result.suggestions?.length > 0 && (
            <div className="border rounded-lg overflow-hidden">
              <div className="px-4 py-2.5 border-b bg-muted/30 flex items-center gap-2">
                <CheckCircle2 className="w-4 h-4 text-green-500" />
                <span className="text-sm font-semibold">Suggestions</span>
              </div>
              <ul className="divide-y">
                {result.suggestions.map((s, i) => (
                  <li key={i} className="px-4 py-2.5 text-sm flex items-start gap-2">
                    <Zap className="w-4 h-4 text-primary shrink-0 mt-0.5" />
                    {s}
                  </li>
                ))}
              </ul>
            </div>
          )}

          {/* Full analysis */}
          {result.analysis && (
            <div className="border rounded-lg p-4">
              <MarkdownView content={result.analysis} />
            </div>
          )}
        </div>
      )}
    </div>
  );
}

// ─── Tab: Subject Line Generator ──────────────────────────────────────────────
function SubjectLineGeneratorTab() {
  const [form, setForm] = useState({ topic: '', audience: '', tone: 'professional', goal: 'open_rate', count: 5 });
  const [variants, setVariants] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [copied, setCopied] = useState(null);

  const generate = async () => {
    setLoading(true); setError('');
    try {
      const r = await apiFetch('/ai/subject-lines', { method: 'POST', body: JSON.stringify(form) });
      setVariants(r.variants || []);
    } catch (e) { setError(e.message); }
    setLoading(false);
  };

  const copy = (text, i) => {
    navigator.clipboard.writeText(text);
    setCopied(i);
    setTimeout(() => setCopied(null), 1500);
  };

  const styleColor = {
    curiosity: 'bg-purple-100 text-purple-700 dark:bg-purple-900/40 dark:text-purple-400',
    urgency: 'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-400',
    benefit: 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-400',
    'social-proof': 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-400',
    personal: 'bg-orange-100 text-orange-700 dark:bg-orange-900/40 dark:text-orange-400',
    question: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/40 dark:text-yellow-400',
  };

  return (
    <div className="space-y-4">
      <p className="text-sm text-muted-foreground">Generate high-converting subject line variants for A/B testing, optimized by tone and goal.</p>

      <div className="border rounded-lg p-4 space-y-3 bg-muted/20">
        <div className="grid grid-cols-2 gap-3">
          <div className="col-span-2">
            <label className="text-xs text-muted-foreground font-medium">Email Topic / Product</label>
            <input value={form.topic} onChange={e => setForm(f => ({ ...f, topic: e.target.value }))}
              placeholder="e.g. Summer sale, New feature launch, Monthly newsletter"
              className="w-full mt-1 px-3 py-2 rounded-md border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary/50" />
          </div>
          <div>
            <label className="text-xs text-muted-foreground font-medium">Target Audience</label>
            <input value={form.audience} onChange={e => setForm(f => ({ ...f, audience: e.target.value }))}
              placeholder="e.g. SaaS founders, E-commerce shoppers"
              className="w-full mt-1 px-3 py-2 rounded-md border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary/50" />
          </div>
          <div>
            <label className="text-xs text-muted-foreground font-medium">Tone</label>
            <select value={form.tone} onChange={e => setForm(f => ({ ...f, tone: e.target.value }))}
              className="w-full mt-1 px-3 py-2 rounded-md border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary/50">
              {['professional', 'friendly', 'urgent', 'playful', 'direct'].map(t =>
                <option key={t} value={t}>{t.charAt(0).toUpperCase() + t.slice(1)}</option>
              )}
            </select>
          </div>
          <div>
            <label className="text-xs text-muted-foreground font-medium">Optimize For</label>
            <select value={form.goal} onChange={e => setForm(f => ({ ...f, goal: e.target.value }))}
              className="w-full mt-1 px-3 py-2 rounded-md border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary/50">
              <option value="open_rate">Open Rate</option>
              <option value="click_rate">Click Rate</option>
              <option value="awareness">Brand Awareness</option>
            </select>
          </div>
          <div>
            <label className="text-xs text-muted-foreground font-medium">Number of Variants</label>
            <select value={form.count} onChange={e => setForm(f => ({ ...f, count: +e.target.value }))}
              className="w-full mt-1 px-3 py-2 rounded-md border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary/50">
              {[3, 5, 7, 10].map(n => <option key={n} value={n}>{n}</option>)}
            </select>
          </div>
        </div>
        <button onClick={generate} disabled={loading || !form.topic}
          className="flex items-center gap-2 px-4 py-2 rounded-md bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 disabled:opacity-60">
          {loading ? <RefreshCw className="w-4 h-4 animate-spin" /> : <Sparkles className="w-4 h-4" />}
          {loading ? 'Generating…' : 'Generate Subject Lines'}
        </button>
      </div>

      {error && <div className="text-sm text-destructive bg-destructive/10 px-4 py-2.5 rounded-md">{error}</div>}

      {variants.length > 0 && (
        <div className="space-y-2">
          {variants.map((v, i) => (
            <div key={i} className="border rounded-lg p-4 bg-card hover:bg-muted/20 transition-colors">
              <div className="flex items-start justify-between gap-3">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-1">
                    <span className="font-bold text-sm truncate">{v.text}</span>
                    <span className={cn('px-2 py-0.5 rounded text-xs font-semibold capitalize shrink-0', styleColor[v.style] || 'bg-muted text-muted-foreground')}>
                      {v.style}
                    </span>
                  </div>
                  {v.emoji_ver && v.emoji_ver !== v.text && (
                    <p className="text-xs text-muted-foreground mb-1">✨ {v.emoji_ver}</p>
                  )}
                  {v.notes && <p className="text-xs text-muted-foreground">{v.notes}</p>}
                </div>
                <button onClick={() => copy(v.text, i)}
                  className={cn('px-2.5 py-1.5 text-xs border rounded-md shrink-0 flex items-center gap-1 transition-colors',
                    copied === i ? 'bg-green-100 text-green-700 border-green-300' : 'hover:bg-muted')}>
                  <Copy className="w-3 h-3" />
                  {copied === i ? 'Copied!' : 'Copy'}
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

// ─── Tab: Campaign Send Score ─────────────────────────────────────────────────
function CampaignSendScoreTab() {
  const [campaigns, setCampaigns] = useState([]);
  const [selectedID, setSelectedID] = useState('');
  const [result, setResult] = useState(null);
  const [loading, setLoading] = useState(false);
  const [loadingCampaigns, setLoadingCampaigns] = useState(true);
  const [error, setError] = useState('');

  React.useEffect(() => {
    apiFetch('/campaigns')
      .then(c => setCampaigns(c || []))
      .catch(() => {})
      .finally(() => setLoadingCampaigns(false));
  }, []);

  const checkScore = async () => {
    if (!selectedID) return;
    setLoading(true); setError('');
    try {
      const r = await apiFetch(`/campaigns/${selectedID}/send-score`);
      setResult(r);
    } catch (e) { setError(e.message); }
    setLoading(false);
  };

  const gradeColor = { A: 'text-green-600 dark:text-green-400', B: 'text-green-500', C: 'text-yellow-600 dark:text-yellow-400', D: 'text-orange-500', F: 'text-red-500' };

  return (
    <div className="space-y-4">
      <p className="text-sm text-muted-foreground">
        Run a pre-send health check on your campaign. Checks sender, recipients, complaint rates, anomalies, and unsubscribe configuration.
      </p>

      <div className="flex gap-3">
        <select value={selectedID} onChange={e => { setSelectedID(e.target.value); setResult(null); }}
          disabled={loadingCampaigns}
          className="flex-1 px-3 py-2 rounded-md border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary/50">
          <option value="">Select a campaign…</option>
          {campaigns.map(c => <option key={c.id} value={c.id}>{c.name} ({c.status})</option>)}
        </select>
        <button onClick={checkScore} disabled={loading || !selectedID}
          className="flex items-center gap-2 px-4 py-2 rounded-md bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 disabled:opacity-60">
          {loading ? <RefreshCw className="w-4 h-4 animate-spin" /> : <BarChart2 className="w-4 h-4" />}
          {loading ? 'Checking…' : 'Run Check'}
        </button>
      </div>

      {error && <div className="text-sm text-destructive bg-destructive/10 px-4 py-2.5 rounded-md">{error}</div>}

      {result && (
        <div className="space-y-4">
          {/* Score header */}
          <div className="border rounded-lg p-5 flex items-center gap-5 bg-card">
            <div className="text-center">
              <div className={cn('text-4xl font-black', gradeColor[result.grade] || 'text-foreground')}>{result.grade}</div>
              <div className="text-xs text-muted-foreground mt-0.5">Grade</div>
            </div>
            <div className="w-px h-12 bg-border" />
            <ScoreRing score={result.score} size={72} />
            <div className="flex-1">
              <div className="flex items-center gap-2">
                {result.ready_to_send
                  ? <><CheckCircle2 className="w-5 h-5 text-green-500" /> <span className="font-bold text-green-600 dark:text-green-400">Ready to Send</span></>
                  : <><XCircle className="w-5 h-5 text-red-500" /> <span className="font-bold text-red-500">Not Ready</span></>
                }
              </div>
              {result.blockers?.length > 0 && (
                <ul className="mt-2 space-y-1">
                  {result.blockers.map((b, i) => (
                    <li key={i} className="text-xs text-red-500 flex items-center gap-1">
                      <AlertTriangle className="w-3 h-3 shrink-0" /> {b}
                    </li>
                  ))}
                </ul>
              )}
            </div>
          </div>

          {/* Per-check breakdown */}
          <div className="border rounded-lg overflow-hidden">
            <div className="px-4 py-2.5 border-b bg-muted/30 text-sm font-semibold">Check Breakdown</div>
            <div className="divide-y">
              {result.checks.map((c, i) => (
                <div key={i} className="flex items-center gap-3 px-4 py-3">
                  {c.passed
                    ? <CheckCircle2 className="w-4 h-4 text-green-500 shrink-0" />
                    : <XCircle className="w-4 h-4 text-red-500 shrink-0" />}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium">{c.name}</span>
                      <span className="text-xs text-muted-foreground">{c.points}/{c.max_pts} pts</span>
                    </div>
                    <p className="text-xs text-muted-foreground truncate">{c.details}</p>
                  </div>
                  <div className="w-20 bg-muted rounded-full h-1.5 overflow-hidden shrink-0">
                    <div className={cn('h-full rounded-full', c.passed ? 'bg-green-500' : c.points > 0 ? 'bg-yellow-500' : 'bg-red-500')}
                      style={{ width: `${(c.points / c.max_pts) * 100}%` }} />
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

// ─── Main Page ─────────────────────────────────────────────────────────────────
export default function AIAdvisorPage() {
  const [tab, setTab] = useState('advisor');

  const TABS = [
    { id: 'advisor', label: 'Deliverability Advisor', icon: Brain },
    { id: 'content', label: 'Content Analyzer', icon: FileText },
    { id: 'subjects', label: 'Subject Generator', icon: Sparkles },
    { id: 'score', label: 'Pre-Send Score', icon: Send },
  ];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <Brain className="w-6 h-6 text-primary" /> AI Intelligence Layer
          </h1>
          <p className="text-muted-foreground text-sm mt-1">
            AI-powered deliverability analysis, content scoring, and campaign optimization.
          </p>
        </div>
        <div className="flex items-center gap-2 bg-muted/40 border rounded-md px-3 py-1.5">
          <Info className="w-3.5 h-3.5 text-muted-foreground" />
          <span className="text-xs text-muted-foreground">Requires AI API key in <strong>Settings → AI Provider</strong></span>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 border-b overflow-x-auto">
        {TABS.map(t => {
          const Icon = t.icon;
          return (
            <button key={t.id} onClick={() => setTab(t.id)}
              className={cn('flex items-center gap-2 px-4 py-2.5 text-sm font-medium border-b-2 -mb-px transition-colors whitespace-nowrap shrink-0',
                tab === t.id ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground')}>
              <Icon className="w-4 h-4" /> {t.label}
            </button>
          );
        })}
      </div>

      {tab === 'advisor'  && <DeliverabilityAdvisorTab />}
      {tab === 'content'  && <ContentAnalyzerTab />}
      {tab === 'subjects' && <SubjectLineGeneratorTab />}
      {tab === 'score'    && <CampaignSendScoreTab />}
    </div>
  );
}
