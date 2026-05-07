import { Navigate, Route, Routes } from 'react-router-dom';
import { useAuth } from '@/context/auth';
import AuthPage from '@/pages/AuthPage';
import DashboardPage from '@/pages/DashboardPage';
import IdentityRolesPage from '@/pages/IdentityRolesPage';
import RolesPage from '@/pages/RolesPage';
import RolePermissionsPage from '@/pages/RolePermissionsPage';
import OAuthClientsPage from '@/pages/OAuthClientsPage';

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
      <Route
        path="/roles"
        element={
          <ProtectedRoute>
            <RolesPage />
          </ProtectedRoute>
        }
      />
      <Route
        path="/roles/:id/permissions"
        element={
          <ProtectedRoute>
            <RolePermissionsPage />
          </ProtectedRoute>
        }
      />
      <Route
        path="/oauth2/clients"
        element={
          <ProtectedRoute>
            <OAuthClientsPage />
          </ProtectedRoute>
        }
      />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
