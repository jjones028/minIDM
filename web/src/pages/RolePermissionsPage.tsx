import { useState, useEffect, useCallback } from 'react';
import { useNavigate, useParams, Link } from 'react-router-dom';
import {
  getResources, getActions, getRolePermissions,
  addRolePermission, removeRolePermission,
  getRoles, isUnauthorized,
  type Resource, type Action, type RolePermission, type Role,
} from '@/api';
import { useAuth } from '@/context/auth';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { AppNav } from '@/components/app-nav';
import { ChevronLeft } from 'lucide-react';

export default function RolePermissionsPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { setAuthenticated } = useAuth();

  const [role, setRole] = useState<Role | null>(null);
  const [resources, setResources] = useState<Resource[]>([]);
  const [actions, setActions] = useState<Action[]>([]);
  const [permissions, setPermissions] = useState<RolePermission[]>([]);
  const [pending, setPending] = useState<Set<string>>(new Set());

  const handleUnauth = useCallback((err: unknown) => {
    if (isUnauthorized(err)) {
      setAuthenticated(false);
      navigate('/login');
      return true;
    }
    return false;
  }, [setAuthenticated, navigate]);

  const fetchAll = useCallback(async () => {
    if (!id) return;
    try {
      const [rolesRes, resourcesRes, actionsRes, permsRes] = await Promise.all([
        getRoles(),
        getResources(),
        getActions(),
        getRolePermissions(id),
      ]);
      const found = rolesRes.data?.find(r => r.id === id) ?? null;
      setRole(found);
      setResources(resourcesRes.data ?? []);
      setActions(actionsRes.data ?? []);
      setPermissions(permsRes.data ?? []);
    } catch (err) {
      if (!handleUnauth(err)) console.error(err);
    }
  }, [id, handleUnauth]);

  useEffect(() => { fetchAll(); }, [fetchAll]);

  const cellKey = (resourceName: string, actionName: string) => `${resourceName}:${actionName}`;

  const permissionMap = new Map<string, string>(); // "resource:action" → permission id
  for (const p of permissions) {
    permissionMap.set(cellKey(p.resource, p.action), p.id);
  }

  const toggle = async (resource: Resource, action: Action) => {
    if (!id || role?.is_builtin) return;
    const key = cellKey(resource.name, action.name);
    if (pending.has(key)) return;

    setPending(prev => new Set(prev).add(key));
    try {
      const existingPermId = permissionMap.get(key);
      if (existingPermId) {
        await removeRolePermission(id, existingPermId);
      } else {
        await addRolePermission(id, resource.id, action.id);
      }
      await fetchAll();
    } catch (err) {
      handleUnauth(err);
    } finally {
      setPending(prev => { const s = new Set(prev); s.delete(key); return s; });
    }
  };

  return (
    <div className="min-h-screen p-4 md:p-12">
      <div className="max-w-4xl mx-auto space-y-10">
        <header className="flex items-start justify-between">
          <div className="space-y-2">
            <h1 className="text-5xl font-extrabold tracking-tight font-heading">minidm</h1>
            <AppNav />
          </div>
          <Button variant="outline" onClick={() => { setAuthenticated(false); navigate('/login'); }}>
            Sign Out
          </Button>
        </header>

        <div>
          <Link
            to="/roles"
            className="-ml-2 mb-4 inline-flex items-center gap-1 rounded-md px-3 py-1.5 text-sm text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
          >
            <ChevronLeft className="w-4 h-4" />
            Back to Roles
          </Link>

          <Card>
            <CardHeader>
              <div className="flex items-center gap-3">
                <CardTitle>
                  Permissions — <span className="font-mono">{role?.name ?? '…'}</span>
                </CardTitle>
                {role?.is_builtin && (
                  <span className="text-xs px-1.5 py-0.5 rounded-sm bg-muted text-muted-foreground font-medium">
                    built-in · read-only
                  </span>
                )}
              </div>
              <CardDescription>
                {role?.is_builtin
                  ? 'Built-in role permissions are fixed and cannot be changed.'
                  : 'Toggle each cell to grant or revoke a resource × action permission for this role.'}
              </CardDescription>
            </CardHeader>
            <CardContent>
              {resources.length === 0 || actions.length === 0 ? (
                <p className="text-sm text-muted-foreground">Loading…</p>
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead>
                      <tr>
                        <th className="text-left font-medium text-muted-foreground pb-3 pr-6 w-32">Resource</th>
                        {actions.map(a => (
                          <th key={a.id} className="text-center font-medium text-muted-foreground pb-3 px-4 capitalize">
                            {a.name}
                          </th>
                        ))}
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-border">
                      {resources.map(res => (
                        <tr key={res.id}>
                          <td className="py-3 pr-6 font-medium capitalize">{res.name}</td>
                          {actions.map(act => {
                            const key = cellKey(res.name, act.name);
                            const checked = permissionMap.has(key);
                            const busy = pending.has(key);
                            return (
                              <td key={act.id} className="py-3 px-4 text-center">
                                <button
                                  type="button"
                                  disabled={role?.is_builtin || busy}
                                  onClick={() => toggle(res, act)}
                                  aria-checked={checked}
                                  role="checkbox"
                                  className={[
                                    'mx-auto flex h-5 w-5 items-center justify-center rounded border transition-colors',
                                    busy ? 'opacity-50 cursor-wait' : '',
                                    role?.is_builtin
                                      ? 'cursor-not-allowed opacity-60'
                                      : 'cursor-pointer hover:border-primary',
                                    checked
                                      ? 'bg-primary border-primary text-primary-foreground'
                                      : 'border-input bg-background',
                                  ].join(' ')}
                                >
                                  {checked && (
                                    <svg viewBox="0 0 12 12" className="w-3 h-3 fill-current" aria-hidden>
                                      <path d="M10 3L5 8.5 2 5.5" stroke="currentColor" strokeWidth="1.5" fill="none" strokeLinecap="round" strokeLinejoin="round"/>
                                    </svg>
                                  )}
                                </button>
                              </td>
                            );
                          })}
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
