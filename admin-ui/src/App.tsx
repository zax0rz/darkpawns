import { Routes, Route, Navigate } from 'react-router-dom';
import { Layout } from './components/Layout';
import { ProtectedRoute } from './components/ProtectedRoute';
import { ErrorBoundary } from './components/ErrorBoundary';
import { ToastProvider } from './components/Toast';
import { LoginPage } from './pages/LoginPage';
import { DashboardPage } from './pages/DashboardPage';
import { ZonesPage } from './pages/ZonesPage';
import { ZoneDetailPage } from './pages/ZoneDetailPage';
import { MobsPage } from './pages/MobsPage';
import { MobDetailPage } from './pages/MobDetailPage';
import { ObjectsPage } from './pages/ObjectsPage';
import { ObjectDetailPage } from './pages/ObjectDetailPage';
import { RoomDetailPage } from './pages/RoomDetailPage';
import { RoomEditPage } from './pages/RoomEditPage';
import { MobEditPage } from './pages/MobEditPage';
import { ObjectEditPage } from './pages/ObjectEditPage';
import { ShopEditPage } from './pages/ShopEditPage';
import { AgentsPage } from './pages/AgentsPage';
import { TerminalPage } from './pages/TerminalPage';
import { OperationsPage } from './pages/OperationsPage';
import { NotFoundPage } from './pages/NotFoundPage';

export default function App() {
  return (
    <ToastProvider>
      <Routes>
        {/* Public routes */}
        <Route path="/login" element={<LoginPage />} />

        {/* Protected admin routes */}
        <Route element={<ProtectedRoute />}>
          <Route element={<Layout />}>
            <Route path="/admin/" element={<ErrorBoundary><DashboardPage /></ErrorBoundary>} />
            <Route path="/admin/game/zones" element={<ErrorBoundary><ZonesPage /></ErrorBoundary>} />
            <Route path="/admin/game/zones/:id" element={<ErrorBoundary><ZoneDetailPage /></ErrorBoundary>} />
            <Route path="/admin/game/mobs" element={<ErrorBoundary><MobsPage /></ErrorBoundary>} />
            <Route path="/admin/game/mobs/:vnum" element={<ErrorBoundary><MobDetailPage /></ErrorBoundary>} />
            <Route path="/admin/game/objects" element={<ErrorBoundary><ObjectsPage /></ErrorBoundary>} />
            <Route path="/admin/game/objects/:vnum" element={<ErrorBoundary><ObjectDetailPage /></ErrorBoundary>} />
            <Route path="/admin/game/rooms/:vnum" element={<ErrorBoundary><RoomDetailPage /></ErrorBoundary>} />
            <Route path="/admin/game/rooms/:vnum/edit" element={<ErrorBoundary><RoomEditPage /></ErrorBoundary>} />
            <Route path="/admin/game/mobs/:vnum/edit" element={<ErrorBoundary><MobEditPage /></ErrorBoundary>} />
            <Route path="/admin/game/objects/:vnum/edit" element={<ErrorBoundary><ObjectEditPage /></ErrorBoundary>} />
            <Route path="/admin/game/shops/:keeperVnum" element={<ErrorBoundary><ShopEditPage /></ErrorBoundary>} />
            <Route path="/admin/agents" element={<ErrorBoundary><AgentsPage /></ErrorBoundary>} />
            <Route path="/admin/operations" element={<ErrorBoundary><OperationsPage /></ErrorBoundary>} />
            <Route path="/admin/webclient" element={<ErrorBoundary><TerminalPage /></ErrorBoundary>} />
          </Route>
        </Route>

        {/* Redirect /admin to /admin/ */}
        <Route path="/admin" element={<Navigate to="/admin/" replace />} />

        {/* 404 */}
        <Route path="*" element={<NotFoundPage />} />
      </Routes>
    </ToastProvider>
  );
}
