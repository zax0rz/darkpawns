import { lazy, Suspense } from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import { Layout } from './components/Layout';
import { ProtectedRoute } from './components/ProtectedRoute';
import { ErrorBoundary } from './components/ErrorBoundary';
import { ToastProvider } from './components/Toast';
import { LoginPage } from './pages/LoginPage';
import { DashboardPage } from './pages/DashboardPage';
import { ZonesPage } from './pages/ZonesPage';
import { ZoneDetailPage } from './pages/ZoneDetailPage';
import { ShopsPage } from './pages/ShopsPage';
import { ShopEditorPage } from './pages/ShopEditorPage';
import { ZoneEditorPage } from './pages/ZoneEditorPage';
import { MobsPage } from './pages/MobsPage';
import { MobDetailPage } from './pages/MobDetailPage';
import { ObjectsPage } from './pages/ObjectsPage';
import { ObjectDetailPage } from './pages/ObjectDetailPage';
import { RoomDetailPage } from './pages/RoomDetailPage';
import { RoomEditorPage } from './pages/RoomEditorPage';
import { MobEditorPage } from './pages/MobEditorPage';
import { ObjectEditorPage } from './pages/ObjectEditorPage';
import { AgentsPage } from './pages/AgentsPage';
import { DecisionsPage } from './pages/DecisionsPage';
import { MindReaderPage } from './pages/MindReaderPage';
import { TerminalPage } from './pages/TerminalPage';
import { OperationsPage } from './pages/OperationsPage';
import { WorkshopPage } from './pages/WorkshopPage';
import { OlcHelpPage } from './pages/OlcHelpPage';
import { NotFoundPage } from './pages/NotFoundPage';
import { Skeleton } from './components/Skeleton';

// CodeMirror ships only with the file editor, not in the main bundle.
const FileEditorPage = lazy(() => import('./pages/FileEditorPage').then((module) => ({ default: module.FileEditorPage })));
const fileEditorFallback = <Skeleton className="h-[38rem] w-full" />;

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
            <Route path="/admin/game/zones/:zone/edit" element={<ErrorBoundary><ZoneEditorPage /></ErrorBoundary>} />
            <Route path="/admin/game/shops" element={<ErrorBoundary><ShopsPage /></ErrorBoundary>} />
            <Route path="/admin/game/shops/:vnum/edit" element={<ErrorBoundary><ShopEditorPage /></ErrorBoundary>} />
            <Route path="/admin/game/mobs" element={<ErrorBoundary><MobsPage /></ErrorBoundary>} />
            <Route path="/admin/game/mobs/:vnum" element={<ErrorBoundary><MobDetailPage /></ErrorBoundary>} />
            <Route path="/admin/game/mobs/:vnum/edit" element={<ErrorBoundary><MobEditorPage /></ErrorBoundary>} />
            <Route path="/admin/game/objects" element={<ErrorBoundary><ObjectsPage /></ErrorBoundary>} />
            <Route path="/admin/game/objects/:vnum" element={<ErrorBoundary><ObjectDetailPage /></ErrorBoundary>} />
            <Route path="/admin/game/objects/:vnum/edit" element={<ErrorBoundary><ObjectEditorPage /></ErrorBoundary>} />
            <Route path="/admin/game/rooms/:vnum" element={<ErrorBoundary><RoomDetailPage /></ErrorBoundary>} />
            <Route path="/admin/game/rooms/:vnum/edit" element={<ErrorBoundary><RoomEditorPage /></ErrorBoundary>} />
            <Route path="/admin/agents" element={<ErrorBoundary><AgentsPage /></ErrorBoundary>} />
            <Route path="/admin/decisions" element={<ErrorBoundary><DecisionsPage /></ErrorBoundary>} />
            <Route path="/admin/narrative" element={<ErrorBoundary><MindReaderPage /></ErrorBoundary>} />
            <Route path="/admin/operations" element={<ErrorBoundary><OperationsPage /></ErrorBoundary>} />
            <Route path="/admin/workshop" element={<ErrorBoundary><WorkshopPage /></ErrorBoundary>} />
            <Route path="/admin/workshop/help" element={<ErrorBoundary><OlcHelpPage /></ErrorBoundary>} />
            <Route path="/admin/workshop/scripts" element={<ErrorBoundary><Suspense fallback={fileEditorFallback}><FileEditorPage root="lua" /></Suspense></ErrorBoundary>} />
            <Route path="/admin/workshop/text" element={<ErrorBoundary><Suspense fallback={fileEditorFallback}><FileEditorPage root="tedit" /></Suspense></ErrorBoundary>} />
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
