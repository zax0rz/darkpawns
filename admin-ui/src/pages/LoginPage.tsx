import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth, type LoginError } from '../hooks/useAuth';
import { api } from '../api/client';
import { Wordmark } from '../components/Wordmark';

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
      const status = (err as LoginError).status;
      const msg = (err as Error).message;
      if (status === 429) {
        // Worth relaying: it tells the user to wait rather than to retype.
        setError(msg);
      } else if (status === 401 || status === 400) {
        // One answer for every auth failure, so the response cannot be used to
        // work out which character names exist.
        setError('Incorrect character name or password.');
      } else if (msg.includes('Failed to fetch') || msg.includes('NetworkError')) {
        setError('Cannot reach the server. Check that Dark Pawns is running.');
      } else {
        setError('Could not sign in. Try again.');
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-paper text-ink font-serif flex items-center justify-center p-4">
      <div className="w-full max-w-md">
        {/* Header */}
        <div className="mb-8 text-center">
          <Wordmark className="justify-center" />
          <p className="text-ink-muted uppercase tracking-widest text-xs mt-1 font-mono">
            Server administration
          </p>
        </div>

        {/* Connection Status */}
        {serverReachable === false && (
          <div className="mb-6 bg-paper-deep text-accent border border-accent p-3 text-xs text-center">
            Cannot reach the server. Check that Dark Pawns is running on port 4350.
          </div>
        )}

        {/* Login card (Vintage Bookplate style) */}
        <div className="bg-paper-deep border-2 border-rule p-8 rounded-none relative">
          {/* Ornamental corner markings */}
          <div className="absolute top-2 left-2 w-2 h-2 border-t-2 border-l-2 border-rule/35" />
          <div className="absolute top-2 right-2 w-2 h-2 border-t-2 border-r-2 border-rule/35" />
          <div className="absolute bottom-2 left-2 w-2 h-2 border-b-2 border-l-2 border-rule/35" />
          <div className="absolute bottom-2 right-2 w-2 h-2 border-b-2 border-r-2 border-rule/35" />

          <h2 className="text-lg text-ink tracking-wide font-display mb-6 border-b border-rule pb-2 text-center">
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
                autoComplete="current-password"
              />
            </div>

            {error && (
              <div className="bg-paper border border-accent p-3 text-xs text-accent" role="alert">
                {error}
              </div>
            )}

            <button
              type="submit"
              disabled={loading || !playerName || !password}
              className="w-full bg-accent hover:bg-accent-deep disabled:bg-paper-deep disabled:text-ink-muted disabled:border-rule disabled:cursor-not-allowed text-paper font-display tracking-wide py-3 border-2 border-accent-deep transition-all flex items-center justify-center gap-2 rounded-none"
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
        <p className="text-center text-xs uppercase font-mono tracking-widest text-ink-muted mt-8">
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
