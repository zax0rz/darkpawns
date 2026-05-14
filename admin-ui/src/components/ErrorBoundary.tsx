import { Component, type ReactNode } from 'react';
import { Link } from 'react-router-dom';

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  handleReset = () => {
    this.setState({ hasError: false, error: null });
  };

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) return this.props.fallback;

      return (
        <div className="flex flex-col items-center justify-center min-h-[40vh] text-center px-4">
          <div className="text-5xl mb-4">💀</div>
          <h2 className="text-xl font-bold text-slate-900 dark:text-white mb-2">
            Something went wrong
          </h2>
          <p className="text-slate-500 dark:text-slate-400 mb-2 text-sm max-w-md">
            {this.state.error?.message || 'An unexpected error occurred.'}
          </p>
          <div className="flex gap-3 mt-4">
            <button
              onClick={this.handleReset}
              className="bg-amber-600 hover:bg-amber-500 text-white px-4 py-2 rounded text-sm font-medium transition-colors"
            >
              Try Again
            </button>
            <Link
              to="/admin/"
              className="bg-slate-200 dark:bg-slate-700 hover:bg-slate-300 dark:hover:bg-slate-600 text-slate-900 dark:text-white px-4 py-2 rounded text-sm transition-colors"
            >
              Go Home
            </Link>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}
