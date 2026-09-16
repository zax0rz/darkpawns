import { Icon } from '../components/Icon';
import { Link } from 'react-router-dom';

export function NotFoundPage() {
  return (
    <div className="flex flex-col items-center justify-center min-h-[60vh] text-center">
      <div className="text-6xl mb-4 flex justify-center"><Icon name="pawn" className="h-12 w-12 text-ink" /></div>
      <h1 className="text-3xl font-bold text-ink mb-2">404</h1>
      <p className="text-ink-muted mb-6">
        This room doesn't exist. You feel a strange sense of disorientation.
      </p>
      <Link
        to="/admin/"
        className="bg-accent hover:bg-accent text-ink px-4 py-2 rounded transition-colors text-sm"
      >
        Return to Dashboard
      </Link>
    </div>
  );
}
