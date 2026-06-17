import { useState, useEffect } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import {
  getRoles, createRole, updateRole, deleteRole,
  isUnauthorized, type Role,
} from '@/api';
import { useAuth } from '@/context/auth';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Card, CardContent, CardHeader, CardTitle,
} from '@/components/ui/card';
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from '@/components/ui/table';
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogTrigger,
} from '@/components/ui/dialog';
import { AppNav } from '@/components/app-nav';
import { AxiosError } from 'axios';

export default function RolesPage() {
  const navigate = useNavigate();
  const { setAuthenticated } = useAuth();

  const [roles, setRoles] = useState<Role[]>([]);
  const [open, setOpen] = useState(false);
  const [newName, setNewName] = useState('');
  const [newDescription, setNewDescription] = useState('');

  const [editingId, setEditingId] = useState<string | null>(null);
  const [editName, setEditName] = useState('');
  const [editDescription, setEditDescription] = useState('');

  const fetchRoles = () => {
    let cancelled = false;
    getRoles()
      .then(({ data }) => { if (!cancelled) setRoles(data ?? []); })
      .catch(err => {
        if (isUnauthorized(err)) {
          setAuthenticated(false);
          navigate('/login');
        }
      });
    return () => { cancelled = true; };
  };

  useEffect(fetchRoles, []);

  const handleLogout = async () => {
    setAuthenticated(false);
    navigate('/login');
  };

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await createRole(newName.trim(), newDescription.trim());
      setNewName('');
      setNewDescription('');
      setOpen(false);
      fetchRoles();
    } catch (err) {
      const e = err as AxiosError<string>;
      alert('Failed: ' + (e.response?.data?.trim() ?? e.message));
    }
  };

  const startEdit = (role: Role) => {
    setEditingId(role.id);
    setEditName(role.name);
    setEditDescription(role.description ?? '');
  };

  const handleUpdate = async (id: string) => {
    try {
      await updateRole(id, editName.trim(), editDescription.trim());
      setEditingId(null);
      fetchRoles();
    } catch (err) {
      const e = err as AxiosError<string>;
      alert('Failed: ' + (e.response?.data?.trim() ?? e.message));
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await deleteRole(id);
      fetchRoles();
    } catch (err) {
      const e = err as AxiosError<string>;
      alert('Failed: ' + (e.response?.data?.trim() ?? e.message));
    }
  };

  return (
    <div className="min-h-screen p-4 md:p-12">
      <div className="max-w-5xl mx-auto space-y-10">
        <header className="flex items-start justify-between">
          <div className="space-y-2">
            <h1 className="text-5xl font-extrabold tracking-tight font-heading">minidm</h1>
            <AppNav />
          </div>
          <Button variant="outline" onClick={handleLogout}>Sign Out</Button>
        </header>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle>Roles</CardTitle>
            <Dialog open={open} onOpenChange={setOpen}>
              <DialogTrigger render={<Button size="sm">New Role</Button>} />
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>Create Role</DialogTitle>
                  <DialogDescription>
                    Custom roles can be assigned permissions and identities.
                  </DialogDescription>
                </DialogHeader>
                <form onSubmit={handleCreate} className="space-y-4">
                  <div className="grid gap-1.5">
                    <label className="text-sm font-medium leading-none">Name</label>
                    <Input
                      placeholder="e.g. editor"
                      value={newName}
                      onChange={e => setNewName(e.target.value)}
                      required
                    />
                  </div>
                  <div className="grid gap-1.5">
                    <label className="text-sm font-medium leading-none">Description</label>
                    <Input
                      placeholder="Optional description"
                      value={newDescription}
                      onChange={e => setNewDescription(e.target.value)}
                    />
                  </div>
                  <div className="flex justify-end gap-2 pt-2">
                    <Button type="button" variant="outline" onClick={() => setOpen(false)}>
                      Cancel
                    </Button>
                    <Button type="submit">Create Role</Button>
                  </div>
                </form>
              </DialogContent>
            </Dialog>
          </CardHeader>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="pl-6">Name</TableHead>
                  <TableHead>Description</TableHead>
                  <TableHead className="w-24" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {roles.map(role => (
                  <TableRow key={role.id}>
                    {editingId === role.id ? (
                      <>
                        <TableCell className="pl-6">
                          <Input
                            value={editName}
                            onChange={e => setEditName(e.target.value)}
                            className="h-8"
                          />
                        </TableCell>
                        <TableCell>
                          <Input
                            value={editDescription}
                            onChange={e => setEditDescription(e.target.value)}
                            className="h-8"
                            placeholder="Description"
                          />
                        </TableCell>
                        <TableCell>
                          <div className="flex gap-1">
                            <Button size="sm" onClick={() => handleUpdate(role.id)}>Save</Button>
                            <Button size="sm" variant="ghost" onClick={() => setEditingId(null)}>Cancel</Button>
                          </div>
                        </TableCell>
                      </>
                    ) : (
                      <>
                        <TableCell className="pl-6">
                          <div className="flex items-center gap-2">
                            <span className="font-medium">{role.name}</span>
                            {role.is_builtin && (
                              <span className="text-xs px-1.5 py-0.5 rounded-sm bg-muted text-muted-foreground font-medium">
                                built-in
                              </span>
                            )}
                          </div>
                        </TableCell>
                        <TableCell className="text-muted-foreground">
                          {role.description ?? '—'}
                        </TableCell>
                        <TableCell>
                          <div className="flex gap-1">
                            <Link
                              to={`/roles/${role.id}/permissions`}
                              className="inline-flex items-center justify-center rounded-md px-3 py-1.5 text-sm hover:bg-accent hover:text-accent-foreground transition-colors"
                            >
                              Permissions
                            </Link>
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => startEdit(role)}
                            >
                              Edit
                            </Button>
                            <Button
                              variant="ghost"
                              size="sm"
                              className="text-destructive hover:text-destructive disabled:opacity-30"
                              disabled={role.is_builtin}
                              onClick={() => handleDelete(role.id)}
                            >
                              Delete
                            </Button>
                          </div>
                        </TableCell>
                      </>
                    )}
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
