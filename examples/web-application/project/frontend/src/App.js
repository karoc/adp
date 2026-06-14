import React, { useState, useEffect } from 'react';
import './App.css';
import { apiClient } from './api';

function App() {
  const [users, setUsers] = useState([]);
  const [health, setHealth] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  // Login state
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [token, setToken] = useState(null);

  // Fetch health status on mount
  useEffect(() => {
    fetchHealth();
  }, []);

  const fetchHealth = async () => {
    try {
      const data = await apiClient.getHealth();
      setHealth(data);
    } catch (err) {
      setError('Failed to fetch health status');
    }
  };

  const fetchUsers = async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiClient.getUsers();
      setUsers(data.users);
    } catch (err) {
      setError('Failed to fetch users');
    } finally {
      setLoading(false);
    }
  };

  const handleLogin = async (e) => {
    e.preventDefault();
    setLoading(true);
    setError(null);
    try {
      const data = await apiClient.login(username, password);
      setToken(data.token);
      setPassword('');
    } catch (err) {
      setError(err.message || 'Login failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="App">
      <header className="App-header">
        <h1>ADP Web Application Example</h1>

        {/* Health Status */}
        <div className="status-card">
          <h2>Server Status</h2>
          {health ? (
            <div className="status-ok">
              <p>✓ {health.status}</p>
              <p className="small">Service: {health.service}</p>
            </div>
          ) : (
            <p>Loading...</p>
          )}
        </div>

        {/* Login Form */}
        {!token && (
          <div className="login-card">
            <h2>Login</h2>
            <form onSubmit={handleLogin}>
              <input
                type="text"
                placeholder="Username (try 'alice' or 'bob')"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                disabled={loading}
              />
              <input
                type="password"
                placeholder="Password (any password)"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                disabled={loading}
              />
              <button type="submit" disabled={loading}>
                {loading ? 'Logging in...' : 'Login'}
              </button>
            </form>
          </div>
        )}

        {/* Token Display */}
        {token && (
          <div className="token-card">
            <h2>✓ Logged In</h2>
            <p className="small">Token: {token.substring(0, 30)}...</p>
            <button onClick={() => setToken(null)}>Logout</button>
          </div>
        )}

        {/* Users List */}
        <div className="users-card">
          <h2>Users</h2>
          <button onClick={fetchUsers} disabled={loading}>
            {loading ? 'Loading...' : 'Fetch Users'}
          </button>
          {users.length > 0 && (
            <ul className="users-list">
              {users.map(user => (
                <li key={user.id}>
                  <strong>{user.username}</strong> - {user.email}
                </li>
              ))}
            </ul>
          )}
        </div>

        {/* Error Display */}
        {error && (
          <div className="error-card">
            <p>⚠ {error}</p>
          </div>
        )}
      </header>
    </div>
  );
}

export default App;
