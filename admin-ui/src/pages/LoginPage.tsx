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
    <div className="min-h-screen bg-paper text-ink font-serif flex items-center justify-center p-4 transition-colors duration-200">
      <div className="w-full max-w-md">
        {/* Header */}
        <div className="text-center mb-8">
          <div className="inline-flex items-center justify-center w-16 h-16 border-2 border-rule bg-paper-deep shadow-[3px_3px_0px_0px_rgba(26,22,20,0.1)] mb-4">
            <svg viewBox="24 8 52 88" className="h-9 w-auto text-ink" aria-hidden="true">
              <g fill="currentColor">
                <circle cx="50" cy="23" r="12" />
                <rect x="37" y="38" width="26" height="5" />
                <polygon points="43,46 57,46 62,75 38,75" />
                <rect x="31" y="77" width="38" height="6" />
                <rect x="26" y="85" width="48" height="7" />
              </g>
            </svg>
          </div>
          <h1 className="text-3xl text-accent tracking-wide font-display">DARK PAWNS</h1>
          <p className="text-ink-muted uppercase tracking-widest text-[10px] mt-1 font-mono">
            Server administration
          </p>
        </div>

        {/* Connection Status */}
        {serverReachable === false && (
          <div className="mb-6 bg-accent text-paper border-2 border-accent-deep p-3 text-xs font-mono tracking-wide uppercase text-center font-bold">
            Cannot reach the server. Check that Dark Pawns is running on port 4350.
          </div>
        )}

        {/* Login card (Vintage Bookplate style) */}
        <div className="bg-paper-deep border-2 border-rule shadow-[6px_6px_0px_0px_rgba(26,22,20,0.15)] p-8 rounded-none relative">
          {/* Ornamental corner markings */}
          <div className="absolute top-2 left-2 w-2 h-2 border-t-2 border-l-2 border-rule/35" />
          <div className="absolute top-2 right-2 w-2 h-2 border-t-2 border-r-2 border-rule/35" />
          <div className="absolute bottom-2 left-2 w-2 h-2 border-b-2 border-l-2 border-rule/35" />
          <div className="absolute bottom-2 right-2 w-2 h-2 border-b-2 border-r-2 border-rule/35" />

          <h2 className="text-lg font-bold text-ink uppercase tracking-widest font-mono mb-6 border-b border-rule pb-1 text-center">
            Sign in
          </h2>

          <form onSubmit={handleSubmit} className="space-y-5">
            <div>
              <label
                htmlFor="playerName"
                className="block text-xs font-mono uppercase tracking-widest text-ink-muted mb-1.5"
              >
                Character name
              </label>
              <input
                id="playerName"
                type="text"
                value={playerName}
                onChange={(e) => setPlayerName(e.target.value)}
                className="w-full bg-paper border-2 border-rule rounded-none px-3 py-2 text-ink text-sm font-mono focus:outline-none focus:border-accent transition-colors placeholder-ink-muted/50"
                placeholder="Character name"
                autoFocus
                autoComplete="username"
              />
            </div>

            <div>
              <label
                htmlFor="password"
                className="block text-xs font-mono uppercase tracking-widest text-ink-muted mb-1.5"
              >
                Password
              </label>
              <input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full bg-paper border-2 border-rule rounded-none px-3 py-2 text-ink text-sm font-mono focus:outline-none focus:border-accent transition-colors"
                placeholder="••••••••••••"
                autoComplete="current-password"
              />
            </div>

            {error && (
              <div className="bg-paper border border-accent p-3 text-xs text-accent font-mono uppercase font-bold tracking-wide">
                [ERROR] {error}
              </div>
            )}

            <button
              type="submit"
              disabled={loading || !playerName || !password}
              className="w-full bg-accent hover:bg-accent-deep disabled:opacity-40 disabled:cursor-not-allowed text-paper font-mono font-bold uppercase tracking-widest py-3 border-2 border-accent-deep shadow-[3px_3px_0px_0px_rgba(26,22,20,0.1)] transition-all flex items-center justify-center gap-2 rounded-none"
            >
              {loading ? (
                <span className="font-mono text-xs tracking-normal animate-pulse">
                  Signing in
                </span>
              ) : (
                'Sign in'
              )}
            </button>
          </form>
        </div>

        {/* Footer */}
        <p className="text-center text-[10px] uppercase font-mono tracking-widest text-ink-muted mt-8">
          Repository:{' '}
          <a
            href="https://github.com/zax0rz/darkpawns"
            className="text-accent hover:text-accent-deep underline font-bold"
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
