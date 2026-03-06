import React, { useState, useEffect, useRef, useCallback } from 'react';
import { Terminal, Play, Square, Trash2, Download, ChevronDown } from 'lucide-react';
import { cn } from '../lib/utils';

const token = () => localStorage.getItem('kumoui_token') || '';

const MAX_LINES = 1000;

const SERVICES = [
  { id: 'kumomta', label: 'KumoMTA' },
  { id: 'dovecot',  label: 'Dovecot' },
  { id: 'fail2ban', label: 'Fail2ban' },
  { id: 'postfix',  label: 'Postfix' },
];

// Colour-code log lines
function colorLine(line) {
  const l = line.toLowerCase();
  if (l.includes('error') || l.includes('fatal') || l.includes('crit') || l.includes('bounce'))
    return 'text-red-400';
  if (l.includes('warn') || l.includes('deferred') || l.includes('timeout'))
    return 'text-yellow-400';
  if (l.includes('info') || l.includes('delivery') || l.includes('accept'))
    return 'text-green-400';
  if (l.includes('debug') || l.includes('trace'))
    return 'text-blue-400';
  return 'text-gray-300';
}

export default function LiveLogsPage() {
  const [lines, setLines]       = useState([]);
  const [running, setRunning]   = useState(false);
  const [service, setService]   = useState('kumomta');
  const [autoScroll, setAuto]   = useState(true);
  const [filter, setFilter]     = useState('');
  const esRef   = useRef(null);
  const bottomRef = useRef(null);
  const idRef   = useRef(0);

  const stop = useCallback(() => {
    if (esRef.current) {
      esRef.current.close();
      esRef.current = null;
    }
    setRunning(false);
  }, []);

  const start = useCallback(() => {
    stop();
    const url = `/api/logs/stream?service=${service}&token=${encodeURIComponent(token())}`;
    // SSE doesn't support custom headers, so pass token as query param
    // The backend will accept it via query param check (see below)
    const es = new EventSource(url);
    esRef.current = es;
    setRunning(true);

    es.onmessage = (e) => {
      const line = e.data;
      setLines(prev => {
        const next = [...prev, { id: idRef.current++, text: line }];
        return next.length > MAX_LINES ? next.slice(next.length - MAX_LINES) : next;
      });
    };

    es.onerror = () => {
      stop();
    };
  }, [service, stop]);

  // Auto-scroll
  useEffect(() => {
    if (autoScroll && bottomRef.current) {
      bottomRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [lines, autoScroll]);

  // Stop stream on unmount
  useEffect(() => () => stop(), [stop]);

  const clear = () => setLines([]);

  const download = () => {
    const text = lines.map(l => l.text).join('\n');
    const url  = URL.createObjectURL(new Blob([text], { type: 'text/plain' }));
    const a    = document.createElement('a');
    a.href = url;
    a.download = `${service}-logs-${Date.now()}.txt`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const visible = filter
    ? lines.filter(l => l.text.toLowerCase().includes(filter.toLowerCase()))
    : lines;

  return (
    <div className="flex flex-col h-[calc(100vh-8rem)] space-y-4">
      {/* Header */}
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-3 shrink-0">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Live Logs</h1>
          <p className="text-muted-foreground text-sm mt-1">Real-time KumoMTA log stream in your browser.</p>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {/* Service selector */}
          <select value={service} onChange={e => { setService(e.target.value); if (running) stop(); }}
            className="h-9 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring">
            {SERVICES.map(s => <option key={s.id} value={s.id}>{s.label}</option>)}
          </select>

          {/* Filter */}
          <input value={filter} onChange={e => setFilter(e.target.value)}
            placeholder="Filter…"
            className="h-9 rounded-md border bg-background px-3 text-sm w-32 focus:ring-2 focus:ring-ring" />

          {/* Auto-scroll toggle */}
          <button onClick={() => setAuto(v => !v)}
            className={cn('flex items-center gap-1.5 h-9 px-3 rounded-md border text-xs font-medium transition-colors',
              autoScroll
                ? 'bg-blue-500/10 border-blue-500/40 text-blue-600 dark:text-blue-400'
                : 'bg-background border-border text-muted-foreground hover:bg-accent')}>
            <ChevronDown className="w-3.5 h-3.5" />
            Auto-scroll
          </button>

          <button onClick={clear}
            className="flex items-center gap-1.5 h-9 px-3 rounded-md border text-xs font-medium hover:bg-accent transition-colors">
            <Trash2 className="w-3.5 h-3.5" /> Clear
          </button>

          <button onClick={download} disabled={lines.length === 0}
            className="flex items-center gap-1.5 h-9 px-3 rounded-md border text-xs font-medium hover:bg-accent transition-colors disabled:opacity-40">
            <Download className="w-3.5 h-3.5" /> Save
          </button>

          {running ? (
            <button onClick={stop}
              className="flex items-center gap-2 h-9 px-4 rounded-md bg-red-500 text-white hover:bg-red-600 text-sm font-medium transition-colors">
              <Square className="w-4 h-4" /> Stop
            </button>
          ) : (
            <button onClick={start}
              className="flex items-center gap-2 h-9 px-4 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-sm font-medium transition-colors">
              <Play className="w-4 h-4" /> Start Stream
            </button>
          )}
        </div>
      </div>

      {/* Terminal */}
      <div className="flex-1 bg-gray-950 rounded-xl border border-gray-800 overflow-hidden flex flex-col shadow-inner">
        {/* Title bar */}
        <div className="flex items-center gap-2 px-4 py-2 bg-gray-900 border-b border-gray-800 shrink-0">
          <div className="flex gap-1.5">
            <span className="w-3 h-3 rounded-full bg-red-500" />
            <span className="w-3 h-3 rounded-full bg-yellow-500" />
            <span className="w-3 h-3 rounded-full bg-green-500" />
          </div>
          <Terminal className="w-3.5 h-3.5 text-gray-500 ml-2" />
          <span className="text-gray-400 text-xs font-mono">
            {service} {running ? (
              <span className="text-green-400 animate-pulse">● LIVE</span>
            ) : (
              <span className="text-gray-600">● stopped</span>
            )}
          </span>
          <span className="ml-auto text-gray-600 text-xs font-mono">{visible.length} lines</span>
        </div>

        {/* Log output */}
        <div className="flex-1 overflow-y-auto p-4 font-mono text-xs leading-5 select-text">
          {lines.length === 0 ? (
            <div className="flex flex-col items-center justify-center h-full text-gray-600 gap-2">
              <Terminal className="w-10 h-10 opacity-20" />
              <p>Press <strong className="text-gray-500">Start Stream</strong> to begin tailing logs.</p>
            </div>
          ) : visible.map(l => (
            <div key={l.id} className={cn('whitespace-pre-wrap break-all', colorLine(l.text))}>
              {l.text}
            </div>
          ))}
          <div ref={bottomRef} />
        </div>
      </div>

      {/* Legend */}
      <div className="flex flex-wrap gap-4 text-xs text-muted-foreground shrink-0">
        <span className="flex items-center gap-1"><span className="w-2 h-2 rounded-full bg-red-400" /> Error / Bounce</span>
        <span className="flex items-center gap-1"><span className="w-2 h-2 rounded-full bg-yellow-400" /> Warning / Deferred</span>
        <span className="flex items-center gap-1"><span className="w-2 h-2 rounded-full bg-green-400" /> Delivery / Info</span>
        <span className="flex items-center gap-1"><span className="w-2 h-2 rounded-full bg-blue-400" /> Debug</span>
        <span className="ml-auto">Max {MAX_LINES.toLocaleString()} lines retained in memory</span>
      </div>
    </div>
  );
}
