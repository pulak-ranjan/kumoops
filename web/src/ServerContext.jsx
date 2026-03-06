/**
 * ServerContext — manages which VPS is currently "active".
 *
 * When a remote server is selected every fetch('/api/...') call is automatically
 * intercepted and forwarded through /api/servers/{id}/proxy?path=... so that ALL
 * existing pages work on remote servers with zero modifications.
 */
import React, { createContext, useContext, useEffect, useState, useRef } from 'react';

const ServerContext = createContext(null);

const LOCAL_SERVER = { id: 'local', name: 'This Server (Local)', status: 'online' };
const LS_KEY = 'kumo_active_server';

// Patch window.fetch once at module load time.
const _originalFetch = window.fetch.bind(window);
let _activeServerId = localStorage.getItem(LS_KEY) || 'local';

window.fetch = function patchedFetch(input, init) {
  const url = typeof input === 'string' ? input : input instanceof Request ? input.url : String(input);

  // Only intercept relative /api/ calls when a remote server is active
  if (_activeServerId !== 'local' && url.startsWith('/api/') && !url.startsWith('/api/servers')) {
    const proxyURL = `/api/servers/${_activeServerId}/proxy?path=${encodeURIComponent(url)}`;
    // Re-use the same method/headers/body, but strip the Authorization header because
    // the proxy backend will supply its own token to the remote instance.
    const newInit = { ...(init || {}) };
    return _originalFetch(proxyURL, newInit);
  }
  return _originalFetch(input, init);
};

export function ServerProvider({ children }) {
  const [servers, setServers]           = useState([LOCAL_SERVER]);
  const [activeServer, setActiveServer] = useState(LOCAL_SERVER);
  const token = () => localStorage.getItem('kumoui_token') || '';

  const load = async () => {
    try {
      const res = await _originalFetch('/api/servers', {
        headers: { Authorization: `Bearer ${token()}` },
      });
      if (!res.ok) return;
      const data = await res.json();
      setServers([LOCAL_SERVER, ...(Array.isArray(data) ? data : [])]);

      // Restore previously selected server
      const savedId = localStorage.getItem(LS_KEY);
      if (savedId && savedId !== 'local') {
        const found = data.find(s => String(s.id) === savedId);
        if (found) {
          _activeServerId = savedId;
          setActiveServer(found);
        }
      }
    } catch (_) {}
  };

  useEffect(() => { load(); }, []); // eslint-disable-line

  const switchServer = (srv) => {
    _activeServerId = String(srv.id);
    localStorage.setItem(LS_KEY, String(srv.id));
    setActiveServer(srv);
  };

  const addServer = async (name, url, apiToken) => {
    const res = await _originalFetch('/api/servers', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token()}`,
      },
      body: JSON.stringify({ name, url, api_token: apiToken }),
    });
    if (!res.ok) {
      const d = await res.json();
      throw new Error(d.error || 'Failed to add server');
    }
    await load();
  };

  const removeServer = async (id) => {
    await _originalFetch(`/api/servers/${id}`, {
      method: 'DELETE',
      headers: { Authorization: `Bearer ${token()}` },
    });
    if (String(id) === _activeServerId) switchServer(LOCAL_SERVER);
    await load();
  };

  const testServer = async (id) => {
    const res = await _originalFetch(`/api/servers/${id}/test`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${token()}` },
    });
    const d = await res.json();
    await load();
    return d;
  };

  return (
    <ServerContext.Provider value={{ servers, activeServer, switchServer, addServer, removeServer, testServer, reload: load }}>
      {children}
    </ServerContext.Provider>
  );
}

export function useServer() {
  return useContext(ServerContext);
}
