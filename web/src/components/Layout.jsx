import React, { useState } from 'react';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import {
  LayoutDashboard, BarChart3, Globe, ShieldCheck, Key, MailWarning, Network,
  ListOrdered, Webhook, Settings, FileText, Lock, LogOut, Menu, X, ServerCog,
  Wrench, Thermometer, Sliders, Layers, Ban, Bell, BadgeCheck, PieChart,
  ShieldAlert, Server, ChevronDown, Circle, Terminal, Activity, Zap,
  Inbox, Clock, FlaskConical, Router, Share2, Brain
} from 'lucide-react';
import { ThemeToggle } from './ThemeProvider';
import { useAuth } from '../AuthContext';
import { useServer } from '../ServerContext';
import { cn } from '../lib/utils';
import AIAssistant from './AIAssistant'; // Imported Agent

function ServerSwitcher() {
  const { servers, activeServer, switchServer } = useServer();
  const [open, setOpen] = useState(false);
  const isRemote = activeServer?.id !== 'local';

  return (
    <div className="relative px-3 pb-3">
      <button
        onClick={() => setOpen(v => !v)}
        className={cn(
          'w-full flex items-center gap-2 px-3 py-2 rounded-lg border text-xs font-medium transition-colors',
          isRemote
            ? 'border-blue-500/50 bg-blue-500/10 text-blue-600 dark:text-blue-400'
            : 'border-border bg-muted/40 text-muted-foreground hover:bg-muted'
        )}
      >
        <Circle className={cn('w-2 h-2 shrink-0 fill-current', isRemote ? 'text-blue-500' : 'text-green-500')} />
        <span className="flex-1 truncate text-left">{activeServer?.name || 'This Server'}</span>
        <ChevronDown className={cn('w-3.5 h-3.5 shrink-0 transition-transform', open && 'rotate-180')} />
      </button>

      {open && (
        <div className="absolute left-3 right-3 top-full mt-1 z-50 bg-popover border rounded-lg shadow-lg overflow-hidden">
          {servers.map(srv => (
            <button
              key={srv.id}
              onClick={() => { switchServer(srv); setOpen(false); }}
              className={cn(
                'w-full flex items-center gap-2.5 px-3 py-2.5 text-xs hover:bg-accent transition-colors text-left',
                String(activeServer?.id) === String(srv.id) && 'bg-accent font-semibold'
              )}
            >
              <Circle className={cn(
                'w-2 h-2 shrink-0 fill-current',
                srv.status === 'online'  ? 'text-green-500' :
                srv.status === 'offline' ? 'text-red-500'   : 'text-muted-foreground'
              )} />
              <span className="truncate">{srv.name}</span>
            </button>
          ))}
          <Link to="/servers" onClick={() => setOpen(false)}
            className="flex items-center gap-2 px-3 py-2 border-t text-xs text-muted-foreground hover:bg-accent transition-colors">
            <Server className="w-3.5 h-3.5" /> Manage servers
          </Link>
        </div>
      )}
    </div>
  );
}

export default function Layout({ children }) {
  const [isMobileOpen, setIsMobileOpen] = useState(false);
  const { logout } = useAuth();
  const location = useLocation();
  const navigate = useNavigate();

  const navGroups = [
    {
      label: 'Overview',
      links: [
        { path: '/', icon: LayoutDashboard, label: 'Dashboard' },
        { path: '/tools', icon: Wrench, label: 'System Tools' },
        { path: '/stats', icon: BarChart3, label: 'Statistics' },
      ],
    },
    {
      label: 'Sending',
      links: [
        { path: '/domains', icon: Globe, label: 'Domains' },
        { path: '/warmup', icon: Thermometer, label: 'IP Warmup' },
        { path: '/ips', icon: Network, label: 'IP Inventory' },
        { path: '/ippools', icon: Layers, label: 'IP Pools' },
        { path: '/shaping', icon: Sliders, label: 'Traffic Shaping' },
        { path: '/queue', icon: ListOrdered, label: 'Queue' },
      ],
    },
    {
      label: 'Authentication',
      links: [
        { path: '/dmarc', icon: ShieldCheck, label: 'DMARC' },
        { path: '/dkim', icon: Key, label: 'DKIM' },
        { path: '/emailauth', icon: BadgeCheck, label: 'Auth Tools' },
      ],
    },
    {
      label: 'Deliverability',
      links: [
        { path: '/fbl', icon: ShieldAlert, label: 'FBL & Bounces' },
        { path: '/isp-intel', icon: Globe, label: 'ISP Intelligence' },
        { path: '/anomalies', icon: Activity, label: 'Anomaly & Throttle' },
        { path: '/inbox-placement', icon: Inbox, label: 'Inbox Placement' },
        { path: '/bounce', icon: MailWarning, label: 'Bounce Accounts' },
        { path: '/bounce-analytics', icon: PieChart, label: 'Bounce Analytics' },
        { path: '/delivery-log', icon: FileText, label: 'Delivery Log' },
        { path: '/reputation', icon: ShieldAlert, label: 'Reputation' },
        { path: '/suppression', icon: Ban, label: 'Suppression' },
        { path: '/alerts', icon: Bell, label: 'Alerts' },
      ],
    },
    {
      label: 'AI Intelligence',
      links: [
        { path: '/ai-advisor', icon: Brain, label: 'AI Advisor' },
      ],
    },
    {
      label: 'Campaigns',
      links: [
        { path: '/ab-testing', icon: FlaskConical, label: 'A/B Testing' },
        { path: '/send-time', icon: Clock, label: 'Send-Time Optimizer' },
      ],
    },
    {
      label: 'Infrastructure',
      links: [
        { path: '/relay', icon: Router, label: 'SMTP Relay' },
        { path: '/cluster', icon: Share2, label: 'Cluster' },
        { path: '/servers', icon: Server, label: 'Remote Servers' },
      ],
    },
    {
      label: 'System',
      links: [
        { path: '/apikeys', icon: Key, label: 'API Keys' },
        { path: '/webhooks', icon: Webhook, label: 'Webhooks' },
        { path: '/config', icon: ServerCog, label: 'Config Gen' },
        { path: '/logs', icon: FileText, label: 'System Logs' },
        { path: '/live-logs', icon: Terminal, label: 'Live Logs' },
        { path: '/security', icon: Lock, label: 'Security' },
        { path: '/settings', icon: Settings, label: 'Settings' },
      ],
    },
  ];
  const links = navGroups.flatMap(g => g.links);

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  const NavItem = ({ link, onClick }) => {
    const isActive = location.pathname === link.path;
    const Icon = link.icon;
    return (
      <Link
        to={link.path}
        onClick={onClick}
        className={cn(
          "flex items-center gap-3 px-3 py-2.5 rounded-md text-sm font-medium transition-all duration-200",
          isActive
            ? "bg-primary text-primary-foreground shadow-sm"
            : "text-foreground/70 hover:bg-accent hover:text-foreground"
        )}
      >
        <Icon className="w-4 h-4 shrink-0" />
        {link.label}
      </Link>
    );
  };

  return (
    <div className="min-h-screen bg-background flex flex-col md:flex-row">
      {/* Mobile Header */}
      <div className="md:hidden border-b bg-card flex items-center justify-between p-4 sticky top-0 z-30 shadow-sm">
        <div className="font-bold text-lg flex items-center gap-2 text-foreground">
          <div className="w-8 h-8 bg-primary rounded-lg flex items-center justify-center text-primary-foreground font-bold">K</div>
          KumoOps
        </div>
        <button onClick={() => setIsMobileOpen(!isMobileOpen)} className="p-2 -mr-2 text-foreground">
          {isMobileOpen ? <X className="w-6 h-6" /> : <Menu className="w-6 h-6" />}
        </button>
      </div>

      {/* Sidebar */}
      <aside className={cn(
        "fixed inset-y-0 left-0 z-40 w-64 bg-card border-r shadow-sm transform transition-transform duration-300 ease-in-out md:translate-x-0 md:static md:h-screen flex flex-col",
        isMobileOpen ? "translate-x-0" : "-translate-x-full"
      )}>
        {/* Brand */}
        <div className="p-5 border-b flex items-center gap-3">
          <div className="w-9 h-9 bg-primary rounded-lg flex items-center justify-center text-primary-foreground font-black text-lg shrink-0 shadow-sm">K</div>
          <div className="min-w-0">
            <div className="font-bold text-foreground leading-tight">KumoOps</div>
            <div className="flex items-center gap-1.5 mt-0.5">
              <span className="text-xs text-muted-foreground">Admin Panel</span>
              <span className="text-[10px] font-medium bg-primary/10 text-primary px-1.5 py-0.5 rounded-full leading-none">v0.2.0</span>
            </div>
          </div>
        </div>

        <ServerSwitcher />

        <nav className="flex-1 overflow-y-auto p-3 space-y-4">
          {navGroups.map((group) => (
            <div key={group.label}>
              <p className="px-3 mb-1.5 text-[10px] font-bold uppercase tracking-widest text-muted-foreground/70">{group.label}</p>
              <div className="space-y-0.5">
                {group.links.map((link) => (
                  <NavItem key={link.path} link={link} onClick={() => setIsMobileOpen(false)} />
                ))}
              </div>
            </div>
          ))}
        </nav>

        <div className="p-4 border-t space-y-3">
          <div className="flex items-center justify-between px-2">
            <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Theme</span>
            <ThemeToggle />
          </div>
          <button onClick={handleLogout} className="w-full flex items-center gap-3 px-3 py-2.5 rounded-md text-sm font-medium text-destructive hover:bg-destructive/10 transition-colors">
            <LogOut className="w-4 h-4" /> Logout
          </button>
        </div>
      </aside>

      {/* Main Content */}
      <main className="flex-1 overflow-auto h-[calc(100vh-65px)] md:h-screen bg-background relative">
        <div className="p-4 md:p-8 max-w-7xl mx-auto">{children}</div>

        {/* The Agent is mounted here */}
        <AIAssistant />
      </main>

      {/* Mobile Overlay */}
      {isMobileOpen && <div className="fixed inset-0 bg-background/80 backdrop-blur-sm z-30 md:hidden" onClick={() => setIsMobileOpen(false)} />}
    </div>
  );
}
