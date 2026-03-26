import React, { useState, useEffect, useRef, useCallback } from 'react';
import { Terminal, Play, Square, Trash2, Download, ChevronDown, Cpu, RotateCcw, Loader2 } from 'lucide-react';
import { cn } from '../lib/utils';

const token = () => localStorage.getItem('kumoui_token') || '';
const hdrs = () => ({ Authorization: `Bearer ${token()}`, 'Content-Type': 'application/json' });

const MAX_LINES = 1000;

const SERVICES = [
  { id: 'kumomta', label: 'KumoMTA' },
  { id: 'dovecot',  label: 'Dovecot' },
  { id: 'fail2ban', label: 'Fail2ban' },
  { id: 'postfix',  label: 'Postfix' },
];

const CMD_GROUPS = [
  {
    label: 'Service Status',
    cmds: [
      { id: 'svc-kumomta',  label: 'KumoMTA status' },
      { id: 'svc-dovecot',  label: 'Dovecot status' },
      { id: 'svc-postfix',  label: 'Postfix status' },
      { id: 'svc-fail2ban', label: 'Fail2ban status' },
    ],
  },
  {
    label: 'Recent Logs',
    cmds: [
      { id: 'journal-kumo', label: 'KumoMTA last 100 lines' },
      { id: 'journal-dove', label: 'Dovecot last 100 lines' },
      { id: 'journal-f2b',  label: 'Fail2ban last 100 lines' },
    ],
  },
  {
    label: 'System Info',
    cmds: [
      { id: 'ps',      label: 'Process list (ps aux)' },
      { id: 'top1',    label: 'Top snapshot' },
      { id: 'free',    label: 'Memory (free -h)' },
      { id: 'df',      label: 'Disk usage (df -h)' },
      { id: 'uptime',  label: 'Uptime' },
      { id: 'vmstat',  label: 'VM stats' },
      { id: 'iostat',  label: 'I/O stats' },
    ],
  },
  {
    label: 'Network',
    cmds: [
      { id: 'ss',         label: 'Listening ports (ss)' },
      { id: 'lsof-ports', label: 'Open ports (lsof)' },
      { id: 'who',        label: 'Logged-in users' },
      { id: 'last',       label: 'Last logins' },
    ],
  },
  {
    label: 'Kernel',
    cmds: [
      { id: 'dmesg', label: 'dmesg errors/warnings' },
    ],
  },
];

function colorLine(line) {
  const l = line.toLowerCase();
  if (l.includes('error') || l.includes('fatal') || l.includes('crit') || l.includes('bounce'))
    return 'text-red-400';
  if (l.includes('warn') || l.includes('deferred') || l.includes('timeout'))
    return 'text-yellow-400';
  if (l.includes('info') || l.includes('delivery') || l.includes('accept') || l.includes('active'))
    return 'text-green-400';
  if (l.includes('debug') || l.includes('trace'))
    return 'text-blue-400';
  return 'text-gray-300';
}

export default function LiveLogsPage() {
  const [tab, setTab] = useState('stream'); // 'stream' | 'cmd'

  // ── Stream tab ──────────────────────────────────────────
  const [lines, setLines]       = useState([]);
  const [running, setRunning]   = useState(false);
  const [service, setService]   = useState('kumomta');
  const [autoScroll, setAuto]   = useState(true);
  const [filter, setFilter]     = useState('');
  const esRef     = useRef(null);
  const bottomRef = useRef(null);
  const idRef     = useRef(0);

  const stop = useCallback(() => {
    if (esRef.current) { esRef.current.close(); esRef.current = null; }
    setRunning(false);
  }, []);

  const start = useCallback(() => {
    stop();
    const url = `/api/logs/stream?service=${service}&token=${encodeURIComponent(token())}`;
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
    es.onerror = () => stop();
  }, [service, stop]);

  useEffect(() => {
    if (autoScroll && bottomRef.current) {
      bottomRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [lines, autoScroll]);

  useEffect(() => () => stop(), [stop]);

  const clear = () => setLines([]);
  const download = () => {
    const text = lines.map(l => l.text).join('\n');
    const url  = URL.createObjectURL(new Blob([text], { type: 'text/plain' }));
    const a    = document.createElement('a');
    a.href = url; a.download = `${service}-logs-${Date.now()}.txt`; a.click();
    URL.revokeObjectURL(url);
  };

  const visible = filter ? lines.filter(l => l.text.toLowerCase().includes(filter.toLowerCase())) : lines;

  // ── Command runner tab ──────────────────────────────────
  const [cmdOutput, setCmdOutput]   = useState('');
  const [cmdRunning, setCmdRunning] = useState(false);
  const [lastCmd, setLastCmd]       = useState('');
  const [ranAt, setRanAt]           = useState('');

  const runCmd = async (cmdId) => {
    setCmdRunning(true);
    setLastCmd(cmdId);
    setCmdOutput('');
    try {
      const res = await fetch('/api/system/run-command', {
        method: 'POST', headers: hdrs(),
        body: JSON.stringify({ cmd: cmdId }),
      });
      const data = await res.json();
      if (!res.ok) { setCmdOutput(`Error: ${data.error}`); return; }
      setCmdOutput(data.output || '(no output)');
      setRanAt(data.ran_at ? new Date(data.ran_at).toLocaleTimeString() : '');
    } catch (err) {
      setCmdOutput(`Request failed: ${err.message}`);
    } finally {
      setCmdRunning(false);
    }
  };

  return (
    <div className="flex flex-col h-[calc(100vh-8rem)] space-y-4">
      {/* Header + tab selector */}
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-3 shrink-0">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Live Logs</h1>
          <p className="text-muted-foreground text-sm mt-1">Real-time log stream and system command runner.</p>
        </div>
        <div className="flex gap-1 rounded-lg border p-1 bg-muted/40">
          {[['stream', 'Log Stream', Terminal], ['cmd', 'Commands', Cpu]].map(([id, label, Icon]) => (
            <button key={id} onClick={() => setTab(id)}
              className={cn('flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm font-medium transition-colors',
                tab === id ? 'bg-background shadow-sm text-foreground' : 'text-muted-foreground hover:text-foreground')}>
              <Icon className="w-4 h-4" />{label}
            </button>
          ))}
        </div>
      </div>

      {/* ── LOG STREAM TAB ── */}
      {tab === 'stream' && (
        <>
          <div className="flex flex-wrap items-center gap-2 shrink-0">
            <select value={service} onChange={e => { setService(e.target.value); if (running) stop(); }}
              className="h-9 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring">
              {SERVICES.map(s => <option key={s.id} value={s.id}>{s.label}</option>)}
            </select>
            <input value={filter} onChange={e => setFilter(e.target.value)} placeholder="Filter…"
              className="h-9 rounded-md border bg-background px-3 text-sm w-32 focus:ring-2 focus:ring-ring" />
            <button onClick={() => setAuto(v => !v)}
              className={cn('flex items-center gap-1.5 h-9 px-3 rounded-md border text-xs font-medium transition-colors',
                autoScroll
                  ? 'bg-blue-500/10 border-blue-500/40 text-blue-600 dark:text-blue-400'
                  : 'bg-background border-border text-muted-foreground hover:bg-accent')}>
              <ChevronDown className="w-3.5 h-3.5" /> Auto-scroll
            </button>
            <button onClick={clear}
              className="flex items-center gap-1.5 h-9 px-3 rounded-md border text-xs font-medium hover:bg-accent">
              <Trash2 className="w-3.5 h-3.5" /> Clear
            </button>
            <button onClick={download} disabled={lines.length === 0}
              className="flex items-center gap-1.5 h-9 px-3 rounded-md border text-xs font-medium hover:bg-accent disabled:opacity-40">
              <Download className="w-3.5 h-3.5" /> Save
            </button>
            {running ? (
              <button onClick={stop}
                className="flex items-center gap-2 h-9 px-4 rounded-md bg-red-500 text-white hover:bg-red-600 text-sm font-medium">
                <Square className="w-4 h-4" /> Stop
              </button>
            ) : (
              <button onClick={start}
                className="flex items-center gap-2 h-9 px-4 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-sm font-medium">
                <Play className="w-4 h-4" /> Start Stream
              </button>
            )}
          </div>

          <div className="flex-1 bg-gray-950 rounded-xl border border-gray-800 overflow-hidden flex flex-col shadow-inner">
            <div className="flex items-center gap-2 px-4 py-2 bg-gray-900 border-b border-gray-800 shrink-0">
              <div className="flex gap-1.5">
                <span className="w-3 h-3 rounded-full bg-red-500" />
                <span className="w-3 h-3 rounded-full bg-yellow-500" />
                <span className="w-3 h-3 rounded-full bg-green-500" />
              </div>
              <Terminal className="w-3.5 h-3.5 text-gray-500 ml-2" />
              <span className="text-gray-400 text-xs font-mono">
                {service} {running
                  ? <span className="text-green-400 animate-pulse">● LIVE</span>
                  : <span className="text-gray-600">● stopped</span>}
              </span>
              <span className="ml-auto text-gray-600 text-xs font-mono">{visible.length} lines</span>
            </div>
            <div className="flex-1 overflow-y-auto p-4 font-mono text-xs leading-5 select-text">
              {lines.length === 0 ? (
                <div className="flex flex-col items-center justify-center h-full text-gray-600 gap-2">
                  <Terminal className="w-10 h-10 opacity-20" />
                  <p>Press <strong className="text-gray-500">Start Stream</strong> to begin tailing logs.</p>
                </div>
              ) : visible.map(l => (
                <div key={l.id} className={cn('whitespace-pre-wrap break-all', colorLine(l.text))}>{l.text}</div>
              ))}
              <div ref={bottomRef} />
            </div>
          </div>

          <div className="flex flex-wrap gap-4 text-xs text-muted-foreground shrink-0">
            {[['bg-red-400','Error / Bounce'],['bg-yellow-400','Warning / Deferred'],['bg-green-400','Delivery / Info'],['bg-blue-400','Debug']].map(([c,l]) => (
              <span key={l} className="flex items-center gap-1"><span className={cn('w-2 h-2 rounded-full', c)} />{l}</span>
            ))}
            <span className="ml-auto">Max {MAX_LINES.toLocaleString()} lines in memory</span>
          </div>
        </>
      )}

      {/* ── COMMAND RUNNER TAB ── */}
      {tab === 'cmd' && (
        <div className="flex-1 flex gap-4 overflow-hidden min-h-0">
          {/* Sidebar - command list */}
          <div className="w-64 shrink-0 overflow-y-auto space-y-4 pr-1">
            <p className="text-xs text-muted-foreground">Read-only system commands. No sudo, no writes.</p>
            {CMD_GROUPS.map(group => (
              <div key={group.label}>
                <div className="text-xs font-semibold text-muted-foreground uppercase tracking-wide mb-1.5">{group.label}</div>
                <div className="space-y-1">
                  {group.cmds.map(cmd => (
                    <button key={cmd.id} onClick={() => runCmd(cmd.id)}
                      disabled={cmdRunning}
                      className={cn(
                        'w-full text-left px-3 py-2 rounded-md text-sm transition-colors',
                        lastCmd === cmd.id
                          ? 'bg-primary/10 text-primary font-medium'
                          : 'hover:bg-muted text-foreground',
                        cmdRunning && lastCmd === cmd.id && 'opacity-60 cursor-not-allowed'
                      )}>
                      <span className="flex items-center gap-2">
                        {cmdRunning && lastCmd === cmd.id
                          ? <Loader2 className="w-3 h-3 animate-spin shrink-0" />
                          : <span className="w-3 h-3 shrink-0" />}
                        {cmd.label}
                      </span>
                    </button>
                  ))}
                </div>
              </div>
            ))}
          </div>

          {/* Output pane */}
          <div className="flex-1 bg-gray-950 rounded-xl border border-gray-800 overflow-hidden flex flex-col">
            <div className="flex items-center gap-2 px-4 py-2 bg-gray-900 border-b border-gray-800 shrink-0">
              <div className="flex gap-1.5">
                <span className="w-3 h-3 rounded-full bg-red-500" />
                <span className="w-3 h-3 rounded-full bg-yellow-500" />
                <span className="w-3 h-3 rounded-full bg-green-500" />
              </div>
              <Cpu className="w-3.5 h-3.5 text-gray-500 ml-2" />
              <span className="text-gray-400 text-xs font-mono">
                {lastCmd ? `$ ${lastCmd}` : 'select a command'}
              </span>
              {ranAt && <span className="ml-auto text-gray-600 text-xs font-mono">{ranAt}</span>}
              {cmdOutput && (
                <button onClick={() => setCmdOutput('')}
                  className="ml-auto text-gray-600 hover:text-gray-400 text-xs flex items-center gap-1">
                  <RotateCcw className="w-3 h-3" /> clear
                </button>
              )}
            </div>
            <div className="flex-1 overflow-y-auto p-4 font-mono text-xs leading-5 text-gray-300 select-text">
              {cmdRunning ? (
                <div className="flex items-center gap-2 text-gray-500">
                  <Loader2 className="w-4 h-4 animate-spin" /> Running…
                </div>
              ) : cmdOutput ? (
                <pre className="whitespace-pre-wrap break-all">{cmdOutput}</pre>
              ) : (
                <div className="flex flex-col items-center justify-center h-full text-gray-600 gap-2">
                  <Cpu className="w-10 h-10 opacity-20" />
                  <p>Click a command on the left to run it.</p>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
