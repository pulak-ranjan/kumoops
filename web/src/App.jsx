import React from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { ThemeProvider } from './components/ThemeProvider';
import { AuthProvider, useAuth } from './AuthContext';
import { ServerProvider } from './ServerContext';
import Layout from './components/Layout';

// Pages
import LoginRegister from './pages/LoginRegister';
import Dashboard from './pages/Dashboard';
import StatsPage from './pages/StatsPage';
import Domains from './pages/Domains';
import DMARCPage from './pages/DMARCPage';
import DKIMPage from './pages/DKIMPage';
import BouncePage from './pages/BouncePage';
import IPsPage from './pages/IPsPage';
import QueuePage from './pages/QueuePage';
import WebhooksPage from './pages/WebhooksPage';
import ConfigPage from './pages/ConfigPage';
import LogsPage from './pages/LogsPage';
import SecurityPage from './pages/SecurityPage';
import Settings from './pages/Settings';
import ToolsPage from './pages/ToolsPage'; // NEW
import WarmupPage from './pages/WarmupPage';
import APIKeysPage from './pages/APIKeysPage';
import TrafficShapingPage from './pages/TrafficShapingPage';
import IPPoolPage from './pages/IPPoolPage';
import SuppressionPage from './pages/SuppressionPage';
import AlertsPage from './pages/AlertsPage';
import EmailAuthPage from './pages/EmailAuthPage';
import BounceAnalyticsPage from './pages/BounceAnalyticsPage';
import DeliveryLogPage from './pages/DeliveryLogPage';
import ReputationPage from './pages/ReputationPage';
import ServersPage from './pages/ServersPage';
import LiveLogsPage from './pages/LiveLogsPage';
import FBLPage from './pages/FBLPage';
import ISPIntelPage from './pages/ISPIntelPage';
import AnomalyPage from './pages/AnomalyPage';
import InboxPlacementPage from './pages/InboxPlacementPage';
import SendTimePage from './pages/SendTimePage';
import ABTestPage from './pages/ABTestPage';
import RelayPage from './pages/RelayPage';
import ClusterPage from './pages/ClusterPage';
import AIAdvisorPage from './pages/AIAdvisorPage';

function ProtectedRoute({ children }) {
  const { user, loading } = useAuth();
  
  if (loading) return <div className="flex items-center justify-center min-h-screen bg-background text-muted-foreground">Loading...</div>;
  if (!user) return <Navigate to="/login" replace />;
  
  return <Layout>{children}</Layout>;
}

export default function App() {
  return (
    <AuthProvider>
      <ServerProvider>
      <ThemeProvider defaultTheme="dark" storageKey="kumoui-theme">
        <BrowserRouter>
          <Routes>
            <Route path="/login" element={<LoginRegister />} />
            
            <Route path="/" element={<ProtectedRoute><Dashboard /></ProtectedRoute>} />
            <Route path="/tools" element={<ProtectedRoute><ToolsPage /></ProtectedRoute>} />
            <Route path="/stats" element={<ProtectedRoute><StatsPage /></ProtectedRoute>} />
            <Route path="/domains" element={<ProtectedRoute><Domains /></ProtectedRoute>} />
            <Route path="/dmarc" element={<ProtectedRoute><DMARCPage /></ProtectedRoute>} />
            <Route path="/dkim" element={<ProtectedRoute><DKIMPage /></ProtectedRoute>} />
            <Route path="/bounce" element={<ProtectedRoute><BouncePage /></ProtectedRoute>} />
            <Route path="/ips" element={<ProtectedRoute><IPsPage /></ProtectedRoute>} />
            <Route path="/queue" element={<ProtectedRoute><QueuePage /></ProtectedRoute>} />
            <Route path="/webhooks" element={<ProtectedRoute><WebhooksPage /></ProtectedRoute>} />
            <Route path="/warmup" element={<ProtectedRoute><WarmupPage /></ProtectedRoute>} />
            <Route path="/apikeys" element={<ProtectedRoute><APIKeysPage /></ProtectedRoute>} /> 
            <Route path="/config" element={<ProtectedRoute><ConfigPage /></ProtectedRoute>} />
            <Route path="/logs" element={<ProtectedRoute><LogsPage /></ProtectedRoute>} />
            <Route path="/security" element={<ProtectedRoute><SecurityPage /></ProtectedRoute>} />
            <Route path="/settings" element={<ProtectedRoute><Settings /></ProtectedRoute>} />
            <Route path="/shaping" element={<ProtectedRoute><TrafficShapingPage /></ProtectedRoute>} />
            <Route path="/ippools" element={<ProtectedRoute><IPPoolPage /></ProtectedRoute>} />
            <Route path="/suppression" element={<ProtectedRoute><SuppressionPage /></ProtectedRoute>} />
            <Route path="/alerts" element={<ProtectedRoute><AlertsPage /></ProtectedRoute>} />
            <Route path="/emailauth" element={<ProtectedRoute><EmailAuthPage /></ProtectedRoute>} />
            <Route path="/bounce-analytics" element={<ProtectedRoute><BounceAnalyticsPage /></ProtectedRoute>} />
            <Route path="/delivery-log" element={<ProtectedRoute><DeliveryLogPage /></ProtectedRoute>} />
            <Route path="/reputation" element={<ProtectedRoute><ReputationPage /></ProtectedRoute>} />
            <Route path="/servers" element={<ProtectedRoute><ServersPage /></ProtectedRoute>} />
            <Route path="/live-logs" element={<ProtectedRoute><LiveLogsPage /></ProtectedRoute>} />
            <Route path="/fbl" element={<ProtectedRoute><FBLPage /></ProtectedRoute>} />
            <Route path="/isp-intel" element={<ProtectedRoute><ISPIntelPage /></ProtectedRoute>} />
            <Route path="/anomalies" element={<ProtectedRoute><AnomalyPage /></ProtectedRoute>} />
            <Route path="/inbox-placement" element={<ProtectedRoute><InboxPlacementPage /></ProtectedRoute>} />
            <Route path="/send-time" element={<ProtectedRoute><SendTimePage /></ProtectedRoute>} />
            <Route path="/ab-testing" element={<ProtectedRoute><ABTestPage /></ProtectedRoute>} />
            <Route path="/relay" element={<ProtectedRoute><RelayPage /></ProtectedRoute>} />
            <Route path="/cluster" element={<ProtectedRoute><ClusterPage /></ProtectedRoute>} />
            <Route path="/ai-advisor" element={<ProtectedRoute><AIAdvisorPage /></ProtectedRoute>} />

            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </BrowserRouter>
      </ThemeProvider>
      </ServerProvider>
    </AuthProvider>
  );
}
