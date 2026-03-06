import React, { useEffect, useState } from "react";
import { FileText, RefreshCw, Terminal, Shield, Mail, Search, BarChart2, X, AlertCircle } from "lucide-react";
import { getLogs, apiRequest } from "../api";
import { cn } from "../lib/utils";

// --- Highlighted search match line ---
function HighlightedLine({ line, query }) {
  if (!query) return <span>{line}</span>;
  const escapedQuery = query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  const parts = line.split(new RegExp(`(${escapedQuery})`, 'gi'));
  return (
    <span>
      {parts.map((part, i) =>
        part.toLowerCase() === query.toLowerCase()
          ? <strong key={i} className="text-yellow-300 bg-yellow-900/40 rounded px-0.5">{part}</strong>
          : part
      )}
    </span>
  );
}

// --- Patterns Panel (modal/side panel) ---
function PatternsPanel({ service, onClose }) {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    const load = async () => {
      setLoading(true);
      setError('');
      try {
        const res = await apiRequest(`/logs/patterns?service=${service}`);
        setData(res);
      } catch (e) {
        setError(e.message);
      } finally {
        setLoading(false);
      }
    };
    load();
  }, [service]);

  const patterns = data?.patterns || [];
  const recentErrors = data?.recent_errors || [];
  const maxCount = patterns.length > 0 ? Math.max(...patterns.map(p => p.count)) : 1;

  return (
    <div className="fixed inset-0 bg-background/80 backdrop-blur-sm z-50 flex items-start justify-end p-4">
      <div className="bg-card border rounded-xl shadow-xl w-full max-w-lg h-[calc(100vh-2rem)] flex flex-col overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b flex-shrink-0">
          <div className="flex items-center gap-2">
            <BarChart2 className="w-5 h-5 text-primary" />
            <div>
              <h3 className="font-semibold">Log Patterns</h3>
              <p className="text-xs text-muted-foreground capitalize">{service} service</p>
            </div>
          </div>
          <button onClick={onClose} className="p-1.5 hover:bg-muted rounded-md transition-colors">
            <X className="w-4 h-4" />
          </button>
        </div>

        <div className="flex-1 overflow-y-auto p-4 space-y-6">
          {loading && (
            <div className="flex items-center justify-center py-12 text-muted-foreground text-sm">
              <RefreshCw className="w-4 h-4 animate-spin mr-2" /> Analyzing log patterns...
            </div>
          )}

          {error && (
            <div className="bg-destructive/10 text-destructive p-3 rounded-md text-sm flex items-center gap-2">
              <AlertCircle className="w-4 h-4 flex-shrink-0" /> {error}
            </div>
          )}

          {data && !loading && (
            <>
              {/* Pattern Frequency Table */}
              <div>
                <h4 className="text-sm font-semibold mb-3">Pattern Frequency</h4>
                {patterns.length === 0 ? (
                  <p className="text-sm text-muted-foreground">No patterns found.</p>
                ) : (
                  <div className="space-y-2">
                    {patterns.map((p, i) => (
                      <div key={i} className="group">
                        <div className="flex items-center justify-between mb-1">
                          <span className="text-xs font-mono text-foreground truncate max-w-[80%]" title={p.pattern}>
                            {p.pattern}
                          </span>
                          <span className="text-xs font-bold text-muted-foreground ml-2 flex-shrink-0">{p.count}</span>
                        </div>
                        <div className="h-2 bg-muted rounded-full overflow-hidden">
                          <div
                            className="h-full bg-primary/70 rounded-full transition-all"
                            style={{ width: `${Math.max(2, (p.count / maxCount) * 100)}%` }}
                          />
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              {/* Recent Errors */}
              {recentErrors.length > 0 && (
                <div>
                  <h4 className="text-sm font-semibold mb-3 text-destructive flex items-center gap-2">
                    <AlertCircle className="w-4 h-4" /> Recent Errors
                  </h4>
                  <div className="space-y-2">
                    {recentErrors.map((err, i) => (
                      <div key={i} className="bg-destructive/5 border border-destructive/10 rounded-md p-3">
                        <p className="text-xs font-mono text-destructive break-all leading-relaxed">{err}</p>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  );
}

// --- Search Tab Content ---
function SearchTab() {
  const [query, setQuery] = useState('');
  const [searchService, setSearchService] = useState('kumomta');
  const [results, setResults] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const services = ['kumomta', 'dovecot', 'fail2ban'];

  const doSearch = async (e) => {
    if (e) e.preventDefault();
    if (!query.trim()) return;
    setLoading(true);
    setError('');
    setResults(null);
    try {
      const res = await apiRequest(`/logs/search?service=${searchService}&q=${encodeURIComponent(query)}&lines=500`);
      setResults(res);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const lines = results?.lines || [];
  const matchCount = results?.count ?? lines.length;

  return (
    <div className="space-y-4 flex-1 flex flex-col min-h-0">
      <form onSubmit={doSearch} className="flex items-center gap-2 flex-shrink-0">
        <select
          value={searchService}
          onChange={e => setSearchService(e.target.value)}
          className="h-10 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring focus:outline-none"
        >
          {services.map(s => <option key={s} value={s}>{s}</option>)}
        </select>
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground pointer-events-none" />
          <input
            type="text"
            value={query}
            onChange={e => setQuery(e.target.value)}
            placeholder="Search log lines..."
            className="w-full h-10 rounded-md border bg-background pl-9 pr-3 text-sm focus:ring-2 focus:ring-ring focus:outline-none"
          />
        </div>
        <button
          type="submit"
          disabled={loading || !query.trim()}
          className="flex items-center gap-2 h-10 px-4 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-sm font-medium transition-colors disabled:opacity-50"
        >
          {loading ? <RefreshCw className="w-4 h-4 animate-spin" /> : <Search className="w-4 h-4" />}
          Search
        </button>
      </form>

      {error && (
        <div className="bg-destructive/10 text-destructive p-3 rounded-md text-sm flex items-center gap-2 flex-shrink-0">
          <AlertCircle className="w-4 h-4 flex-shrink-0" /> {error}
        </div>
      )}

      {results && (
        <div className="flex-1 flex flex-col min-h-0">
          <div className="flex items-center justify-between mb-2 flex-shrink-0">
            <span className="text-sm text-muted-foreground">
              {matchCount > 0
                ? <><strong className="text-foreground">{matchCount}</strong> match{matchCount !== 1 ? 'es' : ''} found</>
                : 'No matches found'
              }
            </span>
          </div>
          <div className="flex-1 bg-zinc-950 border border-zinc-800 rounded-xl overflow-hidden shadow-sm flex flex-col min-h-0">
            <div className="bg-zinc-900/50 border-b border-zinc-800 p-3 flex items-center gap-2">
              <Terminal className="w-4 h-4 text-zinc-400" />
              <span className="text-xs font-mono text-zinc-400">
                search: "{query}" in {searchService} (500 lines)
              </span>
            </div>
            <div className="flex-1 p-4 text-xs font-mono text-zinc-300 overflow-auto leading-relaxed custom-scrollbar">
              {lines.length === 0 ? (
                <span className="text-zinc-600 italic">No matching log lines found.</span>
              ) : (
                lines.map((line, i) => (
                  <div key={i} className="py-0.5 hover:bg-zinc-900 rounded px-1 -mx-1">
                    <HighlightedLine line={line} query={query} />
                  </div>
                ))
              )}
            </div>
          </div>
        </div>
      )}

      {!results && !loading && (
        <div className="flex-1 flex items-center justify-center text-muted-foreground text-sm">
          Enter a search term and click Search to find matching log lines.
        </div>
      )}
    </div>
  );
}

export default function LogsPage() {
  const [service, setService] = useState("kumomta");
  const [logs, setLogs] = useState("");
  const [msg, setMsg] = useState("");
  const [busy, setBusy] = useState(false);
  const [activeTab, setActiveTab] = useState("kumomta");
  const [showPatterns, setShowPatterns] = useState(false);

  const load = async (svc) => {
    setBusy(true);
    setMsg("");
    try {
      const res = await getLogs(svc, 100);
      setLogs(res.logs || "");
    } catch (err) {
      setMsg(err.message || "Failed to load logs");
      setLogs("");
    } finally {
      setBusy(false);
    }
  };

  useEffect(() => {
    if (activeTab !== 'search') {
      setService(activeTab);
      load(activeTab);
    }
  }, [activeTab]);

  const services = [
    { id: 'kumomta', label: 'KumoMTA', icon: Mail },
    { id: 'dovecot', label: 'Dovecot', icon: FileText },
    { id: 'fail2ban', label: 'Fail2Ban', icon: Shield },
    { id: 'search', label: 'Search Logs', icon: Search },
  ];

  const isLogTab = activeTab !== 'search';

  return (
    <div className="space-y-6 h-[calc(100vh-140px)] flex flex-col">
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 flex-shrink-0">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">System Logs</h1>
          <p className="text-muted-foreground">View real-time journalctl output.</p>
        </div>
        <div className="flex items-center gap-2">
          {isLogTab && (
            <button
              onClick={() => setShowPatterns(true)}
              className="flex items-center gap-2 h-10 px-4 rounded-md bg-secondary text-secondary-foreground hover:bg-secondary/80 text-sm font-medium transition-colors border"
              title="View log patterns and error summary"
            >
              <BarChart2 className="w-4 h-4" />
              Patterns
            </button>
          )}
          {isLogTab && (
            <button
              onClick={() => load(service)}
              disabled={busy}
              className="flex items-center gap-2 h-10 px-4 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-sm font-medium transition-colors"
            >
              <RefreshCw className={cn("w-4 h-4", busy && "animate-spin")} />
              Refresh
            </button>
          )}
        </div>
      </div>

      {/* Tab Bar */}
      <div className="flex items-center space-x-1 bg-muted p-1 rounded-lg w-fit flex-shrink-0">
        {services.map((svc) => {
          const Icon = svc.icon;
          const isActive = activeTab === svc.id;
          return (
            <button
              key={svc.id}
              onClick={() => setActiveTab(svc.id)}
              className={cn(
                "flex items-center gap-2 px-4 py-2 rounded-md text-sm font-medium transition-all",
                isActive
                  ? "bg-background text-foreground shadow-sm"
                  : "text-muted-foreground hover:text-foreground hover:bg-background/50"
              )}
            >
              <Icon className="w-4 h-4" />
              {svc.label}
            </button>
          );
        })}
      </div>

      {/* Error banner for log tabs */}
      {msg && isLogTab && (
        <div className="bg-destructive/10 text-destructive p-3 rounded-md text-sm flex-shrink-0">
          {msg}
        </div>
      )}

      {/* Log viewer */}
      {isLogTab ? (
        <div className="flex-1 bg-zinc-950 border border-zinc-800 rounded-xl overflow-hidden shadow-sm flex flex-col min-h-0">
          <div className="bg-zinc-900/50 border-b border-zinc-800 p-3 flex items-center gap-2">
            <Terminal className="w-4 h-4 text-zinc-400" />
            <span className="text-xs font-mono text-zinc-400">journalctl -u {service} -n 100</span>
          </div>
          <pre className="flex-1 p-4 text-xs font-mono text-zinc-300 overflow-auto whitespace-pre-wrap leading-relaxed custom-scrollbar">
            {logs || <span className="text-zinc-600 italic">No logs available or service not running...</span>}
          </pre>
        </div>
      ) : (
        <SearchTab />
      )}

      {/* Patterns Panel */}
      {showPatterns && (
        <PatternsPanel service={service} onClose={() => setShowPatterns(false)} />
      )}
    </div>
  );
}
