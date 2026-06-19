import { useState, useEffect } from 'react';
import { useNavigate, useParams, Link } from 'react-router-dom';
import {
  getIdentity, getIdentityRoles, getIdentitySessions, revokeIdentitySession,
  resetIdentityPassword, setIdentityEnabled,
  listIdentityClientRoles, removeIdentityClientRole,
  listIdentityClientGroups, removeIdentityClientGroup,
  isUnauthorized, type Identity, type Role, type IdentitySession,
  type ClientRoleAssignment, type ClientGroupMembership,
} from '@/api';
import { useAuth } from '@/context/auth';
import { Button, buttonVariants } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Card, CardContent, CardDescription, CardHeader, CardTitle,
} from '@/components/ui/card';
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from '@/components/ui/table';

function formatDate(iso: string) {
  return new Date(iso).toLocaleString(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  });
}

export default function IdentityDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { setAuthenticated } = useAuth();

  const [identity, setIdentity] = useState<Identity | null>(null);
  const [roles, setRoles] = useState<Role[]>([]);
  const [sessions, setSessions] = useState<IdentitySession[]>([]);
  const [revokingHandle, setRevokingHandle] = useState<string | null>(null);
  const [togglingEnabled, setTogglingEnabled] = useState(false);
  const [clientRoles, setClientRoles] = useState<ClientRoleAssignment[]>([]);
  const [clientGroups, setClientGroups] = useState<ClientGroupMembership[]>([]);
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [resetError, setResetError] = useState<string | null>(null);
  const [resetSuccess, setResetSuccess] = useState(false);
  const [resetting, setResetting] = useState(false);

  useEffect(() => {
    let cancelled = false;
    Promise.allSettled([
      getIdentity(id!), getIdentityRoles(id!), getIdentitySessions(id!),
      listIdentityClientRoles(id!), listIdentityClientGroups(id!),
    ]).then(([identRes, rolesRes, sessionsRes, clientRolesRes, clientGroupsRes]) => {
      if (cancelled) return;
      if (identRes.status === 'rejected') {
        if (isUnauthorized(identRes.reason)) { setAuthenticated(false); navigate('/login'); }
        return;
      }
      setIdentity(identRes.value.data);
      if (rolesRes.status === 'fulfilled') setRoles(rolesRes.value.data ?? []);
      if (sessionsRes.status === 'fulfilled') setSessions(sessionsRes.value.data ?? []);
      if (clientRolesRes.status === 'fulfilled') setClientRoles(clientRolesRes.value.data ?? []);
      if (clientGroupsRes.status === 'fulfilled') setClientGroups(clientGroupsRes.value.data ?? []);
    });
    return () => { cancelled = true; };
  }, [id]);

  async function handleRevoke(handle: string) {
    setRevokingHandle(handle);
    try {
      await revokeIdentitySession(id!, handle);
      setSessions(prev => prev.filter(s => s.handle !== handle));
    } catch (err) {
      if (isUnauthorized(err)) {
        setAuthenticated(false);
        navigate('/login');
      }
    } finally {
      setRevokingHandle(null);
    }
  }

  async function handleResetPassword(e: React.FormEvent) {
    e.preventDefault();
    setResetError(null);
    setResetSuccess(false);
    if (newPassword.length < 8) {
      setResetError('Password must be at least 8 characters.');
      return;
    }
    if (newPassword !== confirmPassword) {
      setResetError('Passwords do not match.');
      return;
    }
    setResetting(true);
    try {
      await resetIdentityPassword(id!, newPassword);
      setResetSuccess(true);
      setNewPassword('');
      setConfirmPassword('');
      // All sessions were invalidated server-side — clear the local list.
      setSessions([]);
    } catch (err) {
      if (isUnauthorized(err)) {
        setAuthenticated(false);
        navigate('/login');
      } else {
        setResetError('Failed to reset password. Check permissions and try again.');
      }
    } finally {
      setResetting(false);
    }
  }

  async function handleToggleEnabled() {
    if (!identity) return;
    const next = !identity.is_enabled;
    const verb = next ? 'enable' : 'disable';
    if (!confirm(`${verb.charAt(0).toUpperCase() + verb.slice(1)} ${identity.email}?`)) return;
    setTogglingEnabled(true);
    try {
      const { data } = await setIdentityEnabled(id!, next);
      setIdentity(prev => prev ? { ...prev, is_enabled: data.is_enabled } : prev);
    } catch (err) {
      if (isUnauthorized(err)) { setAuthenticated(false); navigate('/login'); }
      else alert(`Failed to ${verb} identity.`);
    } finally {
      setTogglingEnabled(false);
    }
  }

  if (!identity) return null;

  return (
    <div className="min-h-screen p-4 md:p-12">
      <div className="max-w-3xl mx-auto space-y-8">
        <header className="space-y-2">
          <button
            onClick={() => navigate('/')}
            className="text-sm text-muted-foreground hover:text-foreground transition-colors"
          >
            ← Back to Dashboard
          </button>
          <div className="flex items-center gap-3">
            <h1 className="text-4xl font-extrabold tracking-tight font-heading">{identity.email}</h1>
            <button
              onClick={handleToggleEnabled}
              disabled={togglingEnabled}
              title={identity.is_enabled ? 'Click to disable' : 'Click to enable'}
              className={`inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium transition-opacity hover:opacity-70 disabled:opacity-50 cursor-pointer ${
                identity.is_enabled
                  ? 'bg-primary/10 text-primary'
                  : 'bg-muted text-muted-foreground'
              }`}
            >
              {togglingEnabled ? '…' : identity.is_enabled ? 'Enabled' : 'Disabled'}
            </button>
          </div>
        </header>

        <Card>
          <CardHeader>
            <CardTitle>Details</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-sm">
            <div className="flex gap-2">
              <span className="text-muted-foreground w-28 shrink-0">Subject ID</span>
              <span className="font-mono">{identity.subject_id}</span>
            </div>
            <div className="flex gap-2">
              <span className="text-muted-foreground w-28 shrink-0">Identity ID</span>
              <span className="font-mono text-muted-foreground">{identity.id}</span>
            </div>
            <div className="flex gap-2">
              <span className="text-muted-foreground w-28 shrink-0">Created</span>
              <span>{formatDate(identity.created_at)}</span>
            </div>
            <div className="flex gap-2">
              <span className="text-muted-foreground w-28 shrink-0">Last Updated</span>
              <span>{formatDate(identity.updated_at)}</span>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle>Assigned Roles</CardTitle>
            <Link to={`/identities/${id}/roles`} className={buttonVariants({ variant: 'outline', size: 'sm' })}>
              Manage Roles
            </Link>
          </CardHeader>
          <CardContent className="p-0">
            {roles.length === 0 ? (
              <p className="text-sm text-muted-foreground px-6 py-4">No roles assigned.</p>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="pl-6">Role</TableHead>
                    <TableHead>Description</TableHead>
                    <TableHead>Type</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {roles.map(role => (
                    <TableRow key={role.id}>
                      <TableCell className="pl-6 font-medium">{role.name}</TableCell>
                      <TableCell className="text-muted-foreground">{role.description ?? '—'}</TableCell>
                      <TableCell>
                        {role.is_builtin && (
                          <span className="inline-flex items-center rounded-md border border-border px-2 py-0.5 text-xs text-muted-foreground">
                            built-in
                          </span>
                        )}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Active Sessions</CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            {sessions.length === 0 ? (
              <p className="text-sm text-muted-foreground px-6 py-4">No active sessions.</p>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="pl-6">Created</TableHead>
                    <TableHead>Expires</TableHead>
                    <TableHead />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {sessions.map(s => (
                    <TableRow key={s.handle}>
                      <TableCell className="pl-6">{formatDate(s.created_at)}</TableCell>
                      <TableCell className="text-muted-foreground">{formatDate(s.expires_at)}</TableCell>
                      <TableCell className="text-right pr-4">
                        <Button
                          variant="ghost"
                          size="sm"
                          className="text-destructive hover:text-destructive"
                          disabled={revokingHandle === s.handle}
                          onClick={() => handleRevoke(s.handle)}
                        >
                          {revokingHandle === s.handle ? 'Revoking…' : 'Revoke'}
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>Client Roles</CardTitle>
            <CardDescription>
              Roles assigned to this identity for specific OAuth2 clients. These appear as the <code>roles</code> claim in issued tokens.
            </CardDescription>
          </CardHeader>
          <CardContent className="p-0">
            {clientRoles.length === 0 && clientGroups.length === 0 ? (
              <p className="text-sm text-muted-foreground px-6 py-4">No client roles or group memberships.</p>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="pl-6">Client</TableHead>
                    <TableHead>Role / Group</TableHead>
                    <TableHead>Type</TableHead>
                    <TableHead className="w-20" />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {clientRoles.map(r => (
                    <TableRow key={`role-${r.role_id}`}>
                      <TableCell className="pl-6 text-sm">
                        <span className="font-medium">{r.client_name}</span>
                        <span className="block text-xs text-muted-foreground font-mono">{r.app_client_id}</span>
                      </TableCell>
                      <TableCell className="font-mono text-sm">{r.role_name}</TableCell>
                      <TableCell>
                        <span className="text-xs px-1.5 py-0.5 rounded-sm font-medium bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400">
                          role
                        </span>
                      </TableCell>
                      <TableCell className="text-right pr-4">
                        <Button
                          variant="ghost" size="sm"
                          className="h-7 text-xs text-destructive hover:text-destructive"
                          onClick={async () => {
                            await removeIdentityClientRole(id!, r.role_id);
                            setClientRoles(prev => prev.filter(x => x.role_id !== r.role_id));
                          }}
                        >
                          Remove
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                  {clientGroups.map(g => (
                    <TableRow key={`group-${g.group_id}`}>
                      <TableCell className="pl-6 text-sm">
                        <span className="font-medium">{g.client_name}</span>
                        <span className="block text-xs text-muted-foreground font-mono">{g.app_client_id}</span>
                      </TableCell>
                      <TableCell className="text-sm">{g.group_name}</TableCell>
                      <TableCell>
                        <span className="text-xs px-1.5 py-0.5 rounded-sm font-medium bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400">
                          group
                        </span>
                      </TableCell>
                      <TableCell className="text-right pr-4">
                        <Button
                          variant="ghost" size="sm"
                          className="h-7 text-xs text-destructive hover:text-destructive"
                          onClick={async () => {
                            await removeIdentityClientGroup(id!, g.group_id);
                            setClientGroups(prev => prev.filter(x => x.group_id !== g.group_id));
                          }}
                        >
                          Remove
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Reset Password</CardTitle>
            <CardDescription>
              Set a new password for this identity. All active sessions will be
              invalidated immediately and the user will need to sign in again.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleResetPassword} className="space-y-3 max-w-sm">
              <div className="grid gap-1.5">
                <label className="text-sm font-medium leading-none">New Password</label>
                <Input
                  type="password"
                  value={newPassword}
                  onChange={e => { setNewPassword(e.target.value); setResetSuccess(false); setResetError(null); }}
                  placeholder="Min. 8 characters"
                  autoComplete="new-password"
                />
              </div>
              <div className="grid gap-1.5">
                <label className="text-sm font-medium leading-none">Confirm Password</label>
                <Input
                  type="password"
                  value={confirmPassword}
                  onChange={e => { setConfirmPassword(e.target.value); setResetSuccess(false); setResetError(null); }}
                  placeholder="Re-enter password"
                  autoComplete="new-password"
                />
              </div>
              {resetError && (
                <p className="text-sm text-destructive">{resetError}</p>
              )}
              {resetSuccess && (
                <p className="text-sm text-green-600 dark:text-green-400">
                  Password reset. All sessions have been invalidated.
                </p>
              )}
              <Button type="submit" disabled={resetting || !newPassword || !confirmPassword}>
                {resetting ? 'Resetting…' : 'Reset Password'}
              </Button>
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
