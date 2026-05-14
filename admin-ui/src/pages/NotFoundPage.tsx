import { Link } from 'react-router-dom';

export function NotFoundPage() {
  return (
    <div className="flex flex-col items-center justify-center min-h-[60vh] text-center">
      <div className="text-6xl mb-4">🏰</div>
      <h1 className="text-3xl font-bold text-white mb-2">404</h1>
      <p className="text-slate-400 mb-6">
        This room doesn't exist. You feel a strange sense of disorientation.
      </p>
      <Link
        to="/admin/"
        className="bg-amber-600 hover:bg-amber-500 text-white px-4 py-2 rounded transition-colors text-sm"
      >
        Return to Dashboard
      </Link>
    </div>
  );
}
