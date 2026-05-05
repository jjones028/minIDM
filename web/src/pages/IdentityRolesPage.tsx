import { useState, useEffect } from 'react';
import { useNavigate, useParams, useLocation } from 'react-router-dom';
import {
  getRoles, getIdentityRoles, assignRole, removeRole, isUnauthorized, type Role,
} from '@/api';
import { useAuth } from '@/context/auth';
import { Button } from '@/components/ui/button';
import { X } from 'lucide-react';
import {
  Card, CardContent, CardHeader, CardTitle,
} from '@/components/ui/card';
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select';
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from '@/components/ui/table';
import { AxiosError } from 'axios';

export default function IdentityRolesPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const location = useLocation();
  const { setAuthenticated } = useAuth();
  const email: string = location.state?.email ?? id ?? '';

  const [allRoles, setAllRoles] = useState<Role[]>([]);
  const [assignedRoles, setAssignedRoles] = useState<Role[]>([]);
  const [selectedRoleId, setSelectedRoleId] = useState('');

  const fetchRoles = () => {
    let cancelled = false;
    Promise.all([getRoles(), getIdentityRoles(id!)])
      .then(([all, assigned]) => {
        if (!cancelled) {
          setAllRoles(all.data ?? []);
          setAssignedRoles(assigned.data ?? []);
        }
      })
      .catch(err => {
        if (isUnauthorized(err)) {
          setAuthenticated(false);
          navigate('/login');
        }
      });
    return () => { cancelled = true; };
  };

  useEffect(fetchRoles, [id]);

  const assignedIds = new Set(assignedRoles.map(r => r.id));
  const availableRoles = allRoles.filter(r => !assignedIds.has(r.id));

  const handleAssign = async () => {
    if (!selectedRoleId) return;
    try {
      await assignRole(id!, selectedRoleId);
      setSelectedRoleId('');
      fetchRoles();
    } catch (err) {
      const e = err as AxiosError<string>;
      alert('Failed: ' + (e.response?.data?.trim() ?? e.message));
    }
  };

  const handleRemove = async (roleId: string) => {
    try {
      await removeRole(id!, roleId);
      fetchRoles();
    } catch (err) {
      const e = err as AxiosError<string>;
      alert('Failed: ' + (e.response?.data?.trim() ?? e.message));
    }
  };

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
          <h1 className="text-4xl font-extrabold tracking-tight font-heading">Manage Roles</h1>
          <p className="text-muted-foreground font-mono text-sm">{email}</p>
        </header>

        <Card>
          <CardHeader>
            <CardTitle>Assigned Roles</CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            {assignedRoles.length === 0 ? (
              <p className="text-sm text-muted-foreground px-6 py-4">No roles assigned.</p>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="pl-6">Role</TableHead>
                    <TableHead>Description</TableHead>
                    <TableHead className="w-24" />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {assignedRoles.map(role => (
                    <TableRow key={role.id}>
                      <TableCell className="pl-6 font-medium">{role.name}</TableCell>
                      <TableCell className="text-muted-foreground">{role.description ?? '—'}</TableCell>
                      <TableCell>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="text-destructive hover:text-destructive"
                          onClick={() => handleRemove(role.id)}
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

        {availableRoles.length > 0 && (
          <Card>
            <CardHeader>
              <CardTitle>Assign Role</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex gap-3 items-center">
                <div className="relative w-full">
                  <Select value={selectedRoleId || undefined} onValueChange={setSelectedRoleId}>
                    <SelectTrigger className="h-9 w-full text-sm">
                      <SelectValue placeholder="Select a role…" />
                    </SelectTrigger>
                    <SelectContent position="popper">
                      {availableRoles.map(r => (
                        <SelectItem key={r.id} value={r.id}>{r.name}</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  {selectedRoleId && (
                    <button
                      type="button"
                      onClick={() => setSelectedRoleId('')}
                      className="absolute right-7 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
                      aria-label="Clear selection"
                    >
                      <X className="size-3.5" />
                    </button>
                  )}
                </div>
                <Button onClick={handleAssign} disabled={!selectedRoleId} className="shrink-0">
                  Assign
                </Button>
              </div>
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  );
}
