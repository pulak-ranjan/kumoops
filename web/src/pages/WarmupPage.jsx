import React, { useEffect, useState } from "react";
import {
  Thermometer, Play, Pause, RefreshCw, AlertCircle, Settings, X, Save,
  Loader2, Calendar, Clock, ChevronRight
} from "lucide-react";
import { getWarmupList, updateWarmup } from "../api";
import { cn } from "../lib/utils";

const token = () => localStorage.getItem('kumoui_token');
const hdrs = () => ({ Authorization: `Bearer ${token()}`, 'Content-Type': 'application/json' });

export default function WarmupPage() {
  const [senders, setSenders] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  // Config modal
  const [editing, setEditing] = useState(null);
  const [form, setForm] = useState({ enabled: false, plan: "standard" });
  const [saving, setSaving] = useState(false);

  // Pause modal
  const [pausingId, setPausingId] = useState(null);
  const [pauseReason, setPauseReason] = useState("");
  const [pauseBusy, setPauseBusy] = useState(false);

  // Calendar modal
  const [calendarId, setCalendarId] = useState(null);
  const [calendarData, setCalendarData] = useState(null);
  const [calendarLoading, setCalendarLoading] = useState(false);

  // History modal
  const [historyId, setHistoryId] = useState(null);
  const [historyData, setHistoryData] = useState([]);
  const [historyLoading, setHistoryLoading] = useState(false);

  const load = async () => {
    setLoading(true);
    try {
      const data = await getWarmupList();
      setSenders(data || []);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, []);

  const openEdit = (s) => {
    setEditing(s);
    setForm({ enabled: s.enabled, plan: s.plan || "standard" });
  };

  const handleSave = async (e) => {
    e.preventDefault();
    setSaving(true);
    try {
      await updateWarmup(editing.sender_id, form.enabled, form.plan);
      setEditing(null);
      load();
    } catch (err) {
      setError(err.message);
    } finally {
      setSaving(false);
    }
  };

  // Pause with reason
  const confirmPause = async () => {
    setPauseBusy(true);
    try {
      await fetch(`/api/warmup/${pausingId}/pause`, {
        method: 'POST', headers: hdrs(),
        body: JSON.stringify({ reason: pauseReason })
      });
      setPausingId(null);
      setPauseReason("");
      load();
    } catch (e) { setError(e.message); }
    setPauseBusy(false);
  };

  // Resume
  const resumeSender = async (id) => {
    try {
      await fetch(`/api/warmup/${id}/resume`, { method: 'POST', headers: hdrs() });
      load();
    } catch (e) { setError(e.message); }
  };

  // Load calendar
  const openCalendar = async (id) => {
    setCalendarId(id);
    setCalendarLoading(true);
    try {
      const res = await fetch(`/api/warmup/${id}/calendar`, { headers: hdrs() });
      if (res.ok) setCalendarData(await res.json());
    } catch (e) { console.error(e); }
    setCalendarLoading(false);
  };

  // Load history
  const openHistory = async (id) => {
    setHistoryId(id);
    setHistoryLoading(true);
    try {
      const res = await fetch(`/api/warmup/${id}/logs`, { headers: hdrs() });
      if (res.ok) setHistoryData(await res.json());
    } catch (e) { console.error(e); }
    setHistoryLoading(false);
  };

  const formatDate = (d) => d ? new Date(d).toLocaleString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' }) : '-';

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">IP Warmup</h1>
          <p className="text-muted-foreground">Automated daily rate limiting for new IPs.</p>
        </div>
        <button onClick={load} className="p-2 hover:bg-muted rounded-md border">
          <RefreshCw className={cn("w-5 h-5", loading && "animate-spin")} />
        </button>
      </div>

      {error && (
        <div className="bg-destructive/10 text-destructive p-4 rounded-md flex items-center gap-2">
          <AlertCircle className="w-5 h-5" /> {error}
          <button onClick={() => setError("")} className="ml-auto"><X className="w-4 h-4" /></button>
        </div>
      )}

      <div className="bg-card border rounded-xl overflow-hidden shadow-sm">
        <div className="overflow-x-auto">
          <table className="w-full text-sm text-left">
            <thead className="bg-muted/50 text-muted-foreground uppercase text-xs">
              <tr>
                <th className="px-6 py-3">Sender Identity</th>
                <th className="px-6 py-3">Status</th>
                <th className="px-6 py-3">Plan</th>
                <th className="px-6 py-3">Progress</th>
                <th className="px-6 py-3">Current Limit</th>
                <th className="px-6 py-3 text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {senders.map((s) => (
                <tr key={s.sender_id} className="hover:bg-muted/50 transition-colors">
                  <td className="px-6 py-4">
                    <div className="font-medium">{s.email}</div>
                    <div className="text-xs text-muted-foreground">{s.domain}</div>
                  </td>
                  <td className="px-6 py-4">
                    <span className={cn("px-2 py-1 rounded-full text-xs font-bold flex w-fit items-center gap-1",
                      s.enabled ? "bg-orange-500/10 text-orange-600" : "bg-muted text-muted-foreground")}>
                      {s.enabled ? <Thermometer className="w-3 h-3" /> : null}
                      {s.enabled ? "WARMING" : "OFF"}
                    </span>
                  </td>
                  <td className="px-6 py-4 capitalize">
                    {s.enabled ? s.plan : <span className="text-muted-foreground opacity-50">{s.plan || "Standard"}</span>}
                  </td>
                  <td className="px-6 py-4">
                    {s.enabled ? (
                      <div className="flex items-center gap-2">
                        <span className="font-mono font-bold">Day {s.day}</span>
                      </div>
                    ) : "-"}
                  </td>
                  <td className="px-6 py-4">
                    <span className="font-mono bg-background border px-2 py-1 rounded">
                      {s.current_rate || "Unlimited"}
                    </span>
                  </td>
                  <td className="px-6 py-4 text-right">
                    <div className="flex items-center justify-end gap-1.5">
                      {/* Calendar */}
                      <button onClick={() => openCalendar(s.sender_id)} title="View Schedule"
                        className="p-1.5 rounded-md hover:bg-blue-500/10 border border-blue-200 dark:border-blue-800 text-blue-600 dark:text-blue-400 transition-colors">
                        <Calendar className="w-3.5 h-3.5" />
                      </button>
                      {/* History */}
                      <button onClick={() => openHistory(s.sender_id)} title="View History"
                        className="p-1.5 rounded-md hover:bg-muted border text-muted-foreground transition-colors">
                        <Clock className="w-3.5 h-3.5" />
                      </button>
                      {/* Configure */}
                      <button onClick={() => openEdit(s)} title="Configure Plan"
                        className="p-1.5 rounded-md hover:bg-muted border text-muted-foreground transition-colors">
                        <Settings className="w-3.5 h-3.5" />
                      </button>
                      {/* Pause/Resume */}
                      {s.enabled ? (
                        <button onClick={() => setPausingId(s.sender_id)} title="Pause Warmup"
                          className="p-1.5 rounded-md hover:bg-red-500/10 border border-red-200 dark:border-red-800 text-red-600 transition-colors">
                          <Pause className="w-3.5 h-3.5" />
                        </button>
                      ) : (
                        <button onClick={() => resumeSender(s.sender_id)} title="Resume Warmup"
                          className="p-1.5 rounded-md hover:bg-green-500/10 border border-green-200 dark:border-green-800 text-green-600 transition-colors">
                          <Play className="w-3.5 h-3.5" />
                        </button>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
              {senders.length === 0 && !loading && (
                <tr><td colSpan="6" className="text-center py-8 text-muted-foreground">No senders found. Add senders in Domains first.</td></tr>
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Configuration Modal */}
      {editing && (
        <div className="fixed inset-0 bg-background/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-card w-full max-w-md border rounded-xl shadow-lg p-6 space-y-4">
            <div className="flex justify-between items-center border-b pb-4">
              <div>
                <h3 className="text-lg font-semibold">Configure Warmup</h3>
                <p className="text-xs text-muted-foreground">{editing.email}</p>
              </div>
              <button onClick={() => setEditing(null)}><X className="w-4 h-4" /></button>
            </div>
            <form onSubmit={handleSave} className="space-y-4">
              <div className="space-y-3">
                <label className="text-sm font-medium">Warmup Plan</label>
                <div className="grid grid-cols-1 gap-2">
                  {['conservative', 'standard', 'aggressive'].map((plan) => (
                    <label key={plan} className={cn(
                      "flex items-center justify-between p-3 rounded-md border cursor-pointer transition-all",
                      form.plan === plan ? "border-primary bg-primary/5 ring-1 ring-primary" : "hover:bg-muted"
                    )}>
                      <div className="flex items-center gap-2">
                        <input type="radio" name="plan" value={plan} checked={form.plan === plan}
                          onChange={e => setForm({...form, plan: e.target.value})} className="hidden" />
                        <span className="capitalize font-medium">{plan}</span>
                      </div>
                      {plan === 'conservative' && <span className="text-xs text-muted-foreground">10/hr → 4000/hr (10 days)</span>}
                      {plan === 'standard' && <span className="text-xs text-muted-foreground">25/hr → 12000/hr (10 days)</span>}
                      {plan === 'aggressive' && <span className="text-xs text-muted-foreground">50/hr → 20000/hr (9 days)</span>}
                    </label>
                  ))}
                </div>
              </div>
              <div className="flex items-center gap-2 p-3 bg-muted/30 rounded-md">
                <input type="checkbox" id="enableSwitch" checked={form.enabled}
                  onChange={e => setForm({...form, enabled: e.target.checked})}
                  className="h-4 w-4 rounded" />
                <label htmlFor="enableSwitch" className="text-sm font-medium cursor-pointer">Enable Warmup Schedule</label>
              </div>
              <div className="flex justify-end gap-2 pt-2">
                <button type="button" onClick={() => setEditing(null)} className="px-4 py-2 text-sm rounded-md hover:bg-muted">Cancel</button>
                <button type="submit" disabled={saving} className="flex items-center gap-2 px-4 py-2 text-sm rounded-md bg-primary text-primary-foreground hover:bg-primary/90">
                  {saving ? <Loader2 className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />} Save Configuration
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Pause Modal */}
      {pausingId && (
        <div className="fixed inset-0 bg-background/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-card w-full max-w-sm border rounded-xl shadow-lg p-6 space-y-4">
            <div className="flex justify-between items-center">
              <h3 className="text-lg font-semibold">Pause Warmup</h3>
              <button onClick={() => setPausingId(null)}><X className="w-4 h-4" /></button>
            </div>
            <p className="text-sm text-muted-foreground">Optionally add a reason for pausing the warmup schedule.</p>
            <textarea value={pauseReason} onChange={e => setPauseReason(e.target.value)}
              placeholder="Reason for pause (optional)..."
              rows={3}
              className="w-full rounded-md border bg-background px-3 py-2 text-sm focus:ring-2 focus:ring-ring focus:outline-none resize-none" />
            <div className="flex justify-end gap-2">
              <button onClick={() => setPausingId(null)} className="px-4 py-2 text-sm rounded-md hover:bg-muted">Cancel</button>
              <button onClick={confirmPause} disabled={pauseBusy}
                className="flex items-center gap-2 px-4 py-2 text-sm rounded-md bg-amber-600 text-white hover:bg-amber-700">
                {pauseBusy ? <Loader2 className="w-4 h-4 animate-spin" /> : <Pause className="w-4 h-4" />}
                Pause Warmup
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Calendar Modal */}
      {calendarId && (
        <div className="fixed inset-0 bg-background/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-card w-full max-w-2xl border rounded-xl shadow-lg p-6 space-y-4 max-h-[80vh] overflow-y-auto">
            <div className="flex justify-between items-center border-b pb-4">
              <div>
                <h3 className="text-lg font-semibold">Warmup Schedule</h3>
                {calendarData && <p className="text-xs text-muted-foreground">{calendarData.email} · {calendarData.plan} plan · Current: Day {calendarData.current_day}</p>}
              </div>
              <button onClick={() => { setCalendarId(null); setCalendarData(null); }}><X className="w-4 h-4" /></button>
            </div>
            {calendarLoading ? (
              <div className="flex items-center justify-center py-8"><Loader2 className="w-6 h-6 animate-spin text-muted-foreground" /></div>
            ) : calendarData ? (
              <div className="grid grid-cols-3 sm:grid-cols-5 gap-2">
                {(calendarData.schedule || []).map(entry => (
                  <div key={entry.day} className={cn(
                    "p-3 rounded-lg border text-center transition-all",
                    entry.is_today ? "bg-blue-500 text-white border-blue-500 shadow-md ring-2 ring-blue-300" :
                    entry.is_done ? "bg-green-500/15 border-green-500/30 text-green-700 dark:text-green-300" :
                    "bg-muted/30 border-border text-muted-foreground"
                  )}>
                    <div className={cn("text-xs font-medium mb-1", entry.is_today ? "text-blue-100" : "")}>Day {entry.day}</div>
                    <div className={cn("text-sm font-bold font-mono", entry.is_today ? "text-white" : "")}>{entry.rate}</div>
                    {entry.is_today && <div className="text-xs mt-1 text-blue-100">Today</div>}
                    {entry.is_done && <div className="text-xs mt-1 text-green-600 dark:text-green-400">Done</div>}
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-center text-muted-foreground py-4">No schedule data available</p>
            )}
          </div>
        </div>
      )}

      {/* History Modal */}
      {historyId && (
        <div className="fixed inset-0 bg-background/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-card w-full max-w-xl border rounded-xl shadow-lg p-6 space-y-4 max-h-[80vh] overflow-y-auto">
            <div className="flex justify-between items-center border-b pb-4">
              <h3 className="text-lg font-semibold">Warmup History</h3>
              <button onClick={() => { setHistoryId(null); setHistoryData([]); }}><X className="w-4 h-4" /></button>
            </div>
            {historyLoading ? (
              <div className="flex items-center justify-center py-8"><Loader2 className="w-6 h-6 animate-spin text-muted-foreground" /></div>
            ) : historyData.length === 0 ? (
              <p className="text-center text-muted-foreground py-4">No warmup history recorded</p>
            ) : (
              <div className="space-y-2">
                {historyData.map((log, i) => (
                  <div key={i} className="flex items-start gap-3 p-3 rounded-lg border bg-muted/20">
                    <div className={cn(
                      "mt-0.5 w-2 h-2 rounded-full flex-shrink-0",
                      log.event === 'advanced' ? 'bg-green-500' :
                      log.event === 'paused' ? 'bg-amber-500' :
                      log.event === 'resumed' ? 'bg-blue-500' : 'bg-gray-400'
                    )} />
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <span className="text-sm font-medium capitalize">{log.event}</span>
                        <ChevronRight className="w-3 h-3 text-muted-foreground" />
                        <span className="text-xs text-muted-foreground font-mono">{log.new_rate || 'Unlimited'}</span>
                        {log.old_day !== log.new_day && (
                          <span className="text-xs text-muted-foreground">Day {log.old_day} → {log.new_day}</span>
                        )}
                      </div>
                      {log.reason && <p className="text-xs text-muted-foreground mt-0.5">{log.reason}</p>}
                      <p className="text-xs text-muted-foreground mt-0.5">{formatDate(log.created_at)}</p>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
