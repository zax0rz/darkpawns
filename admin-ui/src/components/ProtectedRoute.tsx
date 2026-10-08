import { Navigate, Outlet } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';

export function ProtectedRoute() {
  const { isAuthenticated, ready } = useAuth();

  // Identity now arrives from /admin/session (the cookie is HttpOnly), so
  // wait for that answer before deciding — otherwise a signed-in refresh
  // bounces through the login page.
  if (!ready) {
    return null;
  }
  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  return <Outlet />;
}
