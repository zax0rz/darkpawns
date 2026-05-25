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
            <span className="text-3xl">⚔️</span>
          </div>
          <h1 className="text-3xl font-extrabold text-accent tracking-widest font-sans">DARK PAWNS</h1>
          <p className="text-ink-muted uppercase tracking-widest text-[10px] mt-1 font-sans">
            Mythic Administrative Console
          </p>
          <p className="text-ink-muted text-[9px] mt-1 font-mono">v3.0.0 · GO ENGINE</p>
        </div>

        {/* Connection Status */}
        {serverReachable === false && (
          <div className="mb-6 bg-accent text-paper border-2 border-accent-deep p-3 text-xs font-sans tracking-wide uppercase text-center font-bold">
            [DANGER] Server offline. Verify MUD is running on port 4350.
          </div>
        )}

        {/* Login card (Vintage Bookplate style) */}
        <div className="bg-paper-deep border-2 border-rule shadow-[6px_6px_0px_0px_rgba(26,22,20,0.15)] p-8 rounded-none relative">
          {/* Ornamental corner markings */}
          <div className="absolute top-2 left-2 w-2 h-2 border-t-2 border-l-2 border-rule/35" />
          <div className="absolute top-2 right-2 w-2 h-2 border-t-2 border-r-2 border-rule/35" />
          <div className="absolute bottom-2 left-2 w-2 h-2 border-b-2 border-l-2 border-rule/35" />
          <div className="absolute bottom-2 right-2 w-2 h-2 border-b-2 border-r-2 border-rule/35" />

          <h2 className="text-lg font-bold text-ink uppercase tracking-widest font-sans mb-6 border-b border-rule pb-1 text-center">
            Operator Credentials
          </h2>

          <form onSubmit={handleSubmit} className="space-y-5">
            <div>
              <label
                htmlFor="playerName"
                className="block text-xs font-sans uppercase tracking-widest text-ink-muted mb-1.5"
              >
                CHARACTER IDENTIFIER
              </label>
              <input
                id="playerName"
                type="text"
                value={playerName}
                onChange={(e) => setPlayerName(e.target.value)}
                className="w-full bg-paper border-2 border-rule rounded-none px-3 py-2 text-ink text-sm font-sans focus:outline-none focus:border-accent transition-colors placeholder-ink-muted/50"
                placeholder="Enter character name..."
                autoFocus
                autoComplete="username"
              />
            </div>

            <div>
              <label
                htmlFor="password"
                className="block text-xs font-sans uppercase tracking-widest text-ink-muted mb-1.5"
              >
                PASSPHRASE
              </label>
              <input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full bg-paper border-2 border-rule rounded-none px-3 py-2 text-ink text-sm font-sans focus:outline-none focus:border-accent transition-colors"
                placeholder="••••••••••••"
                autoComplete="current-password"
              />
            </div>

            {error && (
              <div className="bg-paper border border-accent p-3 text-xs text-accent font-sans uppercase font-bold tracking-wide">
                [ERROR] {error}
              </div>
            )}

            <button
              type="submit"
              disabled={loading || !playerName || !password}
              className="w-full bg-accent hover:bg-accent-deep disabled:opacity-40 disabled:cursor-not-allowed text-paper font-sans font-bold uppercase tracking-widest py-3 border-2 border-accent-deep shadow-[3px_3px_0px_0px_rgba(26,22,20,0.1)] transition-all flex items-center justify-center gap-2 rounded-none"
            >
              {loading ? (
                <span className="font-mono text-xs tracking-normal animate-pulse">
                  TRANSMITTING ENCRYPTED TELEMETRY...
                </span>
              ) : (
                'ESTABLISH LINK'
              )}
            </button>
          </form>
        </div>

        {/* Footer */}
        <p className="text-center text-[10px] uppercase font-sans tracking-widest text-ink-muted mt-8">
          Repository Ledger:{' '}
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
