import React, { useState, useEffect } from 'react';
import axios from 'axios';
import './App.css';

function App() {
  const [tables, setTables] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [syncStatus, setSyncStatus] = useState({});
  const [lastRefresh, setLastRefresh] = useState(null);

  // Fetch status on mount
  useEffect(() => {
    fetchStatus();
  }, []);

  // Fetch current status
  const fetchStatus = async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await axios.get('/api/status');
      setTables(response.data.tables || []);
      setLastRefresh(new Date());
    } catch (err) {
      setError('Failed to fetch status: ' + (err.response?.data?.message || err.message));
      console.error('Error fetching status:', err);
    } finally {
      setLoading(false);
    }
  };

  // Trigger sync for specific table
  const triggerTableSync = async (tableName) => {
    try {
      setSyncStatus(prev => ({ ...prev, [tableName]: 'syncing' }));
      setError(null);
      
      const response = await axios.post('/api/sync', {
        table_name: tableName
      });

      if (response.data.success) {
        setSyncStatus(prev => ({ ...prev, [tableName]: 'success' }));
        setTimeout(() => {
          setSyncStatus(prev => ({ ...prev, [tableName]: null }));
        }, 3000);
        
        // Refresh status after sync
        setTimeout(fetchStatus, 1000);
      } else {
        setSyncStatus(prev => ({ ...prev, [tableName]: 'error' }));
        setError(response.data.message || 'Sync failed');
      }
    } catch (err) {
      setSyncStatus(prev => ({ ...prev, [tableName]: 'error' }));
      setError('Failed to trigger sync: ' + (err.response?.data?.message || err.message));
      console.error('Error triggering sync:', err);
    }
  };

  // Trigger sync for all tables
  const triggerAllSync = async () => {
    try {
      setLoading(true);
      setError(null);
      
      const response = await axios.post('/api/sync', {
        sync_all: true
      });

      if (response.data.success) {
        // Refresh status after sync
        setTimeout(fetchStatus, 2000);
      } else {
        setError(response.data.message || 'Sync failed');
      }
    } catch (err) {
      setError('Failed to trigger sync: ' + (err.response?.data?.message || err.message));
      console.error('Error triggering sync:', err);
    } finally {
      setLoading(false);
    }
  };

  // Get status badge class
  const getStatusBadgeClass = (tableName) => {
    const status = syncStatus[tableName];
    if (status === 'syncing') return 'badge-syncing';
    if (status === 'success') return 'badge-success';
    if (status === 'error') return 'badge-error';
    return '';
  };

  // Get status badge text
  const getStatusBadgeText = (tableName) => {
    const status = syncStatus[tableName];
    if (status === 'syncing') return 'Syncing...';
    if (status === 'success') return 'Success!';
    if (status === 'error') return 'Error';
    return '';
  };

  return (
    <div className="App">
      <div className="container">
        <header className="header">
          <h1>🔄 MSSQL → PostgreSQL Sync Service</h1>
          <p className="subtitle">Real-time database synchronization dashboard</p>
        </header>

        {error && (
          <div className="alert alert-error">
            <span className="alert-icon">⚠️</span>
            {error}
          </div>
        )}

        <div className="actions-bar">
          <button 
            onClick={fetchStatus} 
            className="btn btn-secondary"
            disabled={loading}
          >
            {loading ? '⏳ Loading...' : '🔃 Refresh Status'}
          </button>
          <button 
            onClick={triggerAllSync} 
            className="btn btn-primary"
            disabled={loading}
          >
            {loading ? '⏳ Syncing...' : '⚡ Sync All Tables'}
          </button>
          {lastRefresh && (
            <span className="last-refresh">
              Last updated: {lastRefresh.toLocaleTimeString()}
            </span>
          )}
        </div>

        <div className="table-grid">
          {tables.length === 0 ? (
            <div className="empty-state">
              <p>No tables configured for synchronization.</p>
              <p className="empty-state-hint">Check your configuration file.</p>
            </div>
          ) : (
            tables.map((table, index) => (
              <div key={index} className="table-card">
                <div className="table-card-header">
                  <h3>{table.target_table}</h3>
                  {syncStatus[table.target_table] && (
                    <span className={`badge ${getStatusBadgeClass(table.target_table)}`}>
                      {getStatusBadgeText(table.target_table)}
                    </span>
                  )}
                </div>
                
                <div className="table-card-body">
                  <div className="table-info-row">
                    <span className="label">Source:</span>
                    <span className="value">{table.source_table}</span>
                  </div>
                  <div className="table-info-row">
                    <span className="label">Target:</span>
                    <span className="value">{table.target_table}</span>
                  </div>
                  <div className="table-info-row">
                    <span className="label">Refresh Rate:</span>
                    <span className="value">{table.refresh_rate}s</span>
                  </div>
                  
                  <div className="table-features">
                    <span className={`feature-badge ${table.proto_actor_enabled ? 'enabled' : 'disabled'}`}>
                      {table.proto_actor_enabled ? '✓' : '✗'} Auto Sync
                    </span>
                    <span className={`feature-badge ${table.web_api_enabled ? 'enabled' : 'disabled'}`}>
                      {table.web_api_enabled ? '✓' : '✗'} API Trigger
                    </span>
                  </div>
                </div>

                <div className="table-card-footer">
                  {table.web_api_enabled ? (
                    <button 
                      onClick={() => triggerTableSync(table.target_table)}
                      className="btn btn-sm btn-primary"
                      disabled={syncStatus[table.target_table] === 'syncing'}
                    >
                      {syncStatus[table.target_table] === 'syncing' ? '⏳ Syncing...' : '🔄 Sync Now'}
                    </button>
                  ) : (
                    <span className="disabled-text">Manual sync disabled</span>
                  )}
                </div>
              </div>
            ))
          )}
        </div>

        <footer className="footer">
          <p>Powered by Proto.Actor, Gin, and React.js</p>
        </footer>
      </div>
    </div>
  );
}

export default App;
