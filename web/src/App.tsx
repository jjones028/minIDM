import { Navigate, Route, Routes } from 'react-router-dom';
import { useAuth } from '@/context/auth';
import AuthPage from '@/pages/AuthPage';
import DashboardPage from '@/pages/DashboardPage';
import IdentityRolesPage from '@/pages/IdentityRolesPage';

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { checked, authenticated } = useAuth();
  if (!checked) return null;
  if (!authenticated) return <Navigate to="/login" replace />;
  return <>{children}</>;
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<AuthPage />} />
      <Route
        path="/"
        element={
          <ProtectedRoute>
            <DashboardPage />
          </ProtectedRoute>
        }
      />
      <Route
        path="/identities/:id/roles"
        element={
          <ProtectedRoute>
            <IdentityRolesPage />
          </ProtectedRoute>
        }
      />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
