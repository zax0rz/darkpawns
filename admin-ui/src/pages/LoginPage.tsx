import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';
import { api } from '../api/client';

export function LoginPage() {
  const [playerName, setPlayerName] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [serverReachable, setServerReachable] = useState<boolean | null>(null);
  const { login, isAuthenticated } = useAuth();
  const navigate = useNavigate();

  // Redirect if already authenticated
  useEffect(() => {
    if (isAuthenticated) {
      navigate('/admin/');
    }
  }, [isAuthenticated, navigate]);

  // Check server reachability on mount
  useEffect(() => {
    let cancelled = false;
    api.health()
      .then(() => { if (!cancelled) setServerReachable(true); })
      .catch(() => { if (!cancelled) setServerReachable(false); });
    return () => { cancelled = true; };
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      await login(playerName, password);
      navigate('/admin/');
    } catch (err) {
      const msg = (err as Error).message;
      if (msg.includes('401') || msg.includes('Unauthorized')) {
        setError('Incorrect player name or password.');
      } else if (msg.includes('Failed to fetch') || msg.includes('NetworkError')) {
        setError('Cannot reach server. Check your connection and try again.');
      } else {
        setError(msg);
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-gradient-to-b from-slate-900 via-slate-800 to-slate-900 flex items-center justify-center p-4">
      <div className="w-full max-w-md">
        {/* Header */}
        <div className="text-center mb-8">
          <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-amber-600/20 border border-amber-600/40 mb-4">
            <span className="text-3xl">⚔️</span>
          </div>
          <h1 className="text-3xl font-bold text-amber-400 tracking-wide">Dark Pawns</h1>
          <p className="text-slate-400 mt-2 text-sm">Admin Panel — Zone Editor &amp; Management</p>
          <p className="text-slate-500 text-xs mt-1">v3.0 · Go Port</p>
        </div>

        {/* Connection Status */}
        {serverReachable === false && (
          <div className="mb-4 bg-red-900/20 border border-red-700/50 rounded-lg p-3 text-sm text-red-300 text-center">
            Server unreachable. Check that the game server is running on port 4350.
          </div>
        )}

        {/* Login card */}
        <div className="bg-slate-800/80 backdrop-blur rounded-lg border border-slate-700/80 shadow-xl shadow-black/20 p-8">
          <h2 className="text-xl font-semibold text-white mb-6">Sign In</h2>

          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label
                htmlFor="playerName"
                className="block text-sm text-slate-400 mb-1"
              >
                Player Name
              </label>
              <input
                id="playerName"
                type="text"
                value={playerName}
                onChange={(e) => setPlayerName(e.target.value)}
                className="w-full bg-slate-900 border border-slate-600 rounded-lg px-3 py-2.5 text-white text-sm focus:outline-none focus:border-amber-500 focus:ring-1 focus:ring-amber-500 placeholder-slate-500 transition-colors"
                placeholder="Enter your character name"
                autoFocus
                autoComplete="username"
              />
            </div>

            <div>
              <label
                htmlFor="password"
                className="block text-sm text-slate-400 mb-1"
              >
                Password
              </label>
              <input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full bg-slate-900 border border-slate-600 rounded-lg px-3 py-2.5 text-white text-sm focus:outline-none focus:border-amber-500 focus:ring-1 focus:ring-amber-500 placeholder-slate-500 transition-colors"
                placeholder="Enter your password"
                autoComplete="current-password"
              />
            </div>

            {error && (
              <div className="bg-red-900/30 border border-red-700 rounded-lg p-3 text-sm text-red-300">
                {error}
              </div>
            )}

            <button
              type="submit"
              disabled={loading || !playerName || !password}
              className="w-full bg-amber-600 hover:bg-amber-500 disabled:opacity-50 disabled:cursor-not-allowed text-white font-medium py-2.5 rounded-lg transition-colors flex items-center justify-center gap-2"
            >
              {loading ? (
                <>
                  <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24" fill="none">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                  </svg>
                  Signing in...
                </>
              ) : (
                'Sign In'
              )}
            </button>
          </form>
        </div>

        {/* Footer */}
        <p className="text-center text-xs text-slate-600 mt-6">
          Built on{' '}
          <a
            href="https://github.com/zax0rz/darkpawns"
            className="text-amber-600 hover:text-amber-500 underline"
            target="_blank"
            rel="noopener noreferrer"
          >
            zax0rz/darkpawns
          </a>
        </p>
      </div>
    </div>
  );
}
