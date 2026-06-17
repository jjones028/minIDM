import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  getIdentities, createIdentity, logout, isUnauthorized, type Identity,
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
import { AxiosError } from 'axios';
import { AppNav } from '@/components/app-nav';

export default function DashboardPage() {
  const navigate = useNavigate();
  const { setAuthenticated } = useAuth();
  const [identities, setIdentities] = useState<Identity[]>([]);
  const [open, setOpen] = useState(false);
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');

  const fetchIdentities = () => {
    let cancelled = false;
    getIdentities()
      .then(({ data }) => { if (!cancelled) setIdentities(data ?? []); })
      .catch(err => {
        if (isUnauthorized(err)) {
          setAuthenticated(false);
          navigate('/login');
        }
      });
    return () => { cancelled = true; };
  };

  useEffect(fetchIdentities, []);

  const handleLogout = async () => {
    try { await logout(); } catch { /* best-effort */ }
    setAuthenticated(false);
    navigate('/login');
  };

  const handleCreateIdentity = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await createIdentity({ email, password });
      setEmail('');
      setPassword('');
      setOpen(false);
      fetchIdentities();
    } catch (err) {
      const axiosError = err as AxiosError<string>;
      alert('Failed: ' + (axiosError.response?.data?.trim() ?? axiosError.message));
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
            <CardTitle>Identity Registry</CardTitle>
            <Dialog open={open} onOpenChange={setOpen}>
              <DialogTrigger render={<Button size="sm">New Identity</Button>} />
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>Create Identity</DialogTitle>
                  <DialogDescription>
                    New identities are automatically granted the viewer role.
                  </DialogDescription>
                </DialogHeader>
                <form onSubmit={handleCreateIdentity} className="space-y-4">
                  <div className="grid gap-1.5">
                    <label className="text-sm font-medium leading-none">Email Address</label>
                    <Input
                      type="email"
                      placeholder="name@company.com"
                      value={email}
                      onChange={e => setEmail(e.target.value)}
                      required
                    />
                  </div>
                  <div className="grid gap-1.5">
                    <label className="text-sm font-medium leading-none">Password</label>
                    <Input
                      type="password"
                      placeholder="••••••••"
                      value={password}
                      onChange={e => setPassword(e.target.value)}
                      required
                    />
                  </div>
                  <div className="flex justify-end gap-2 pt-2">
                    <Button type="button" variant="outline" onClick={() => setOpen(false)}>
                      Cancel
                    </Button>
                    <Button type="submit">Create Identity</Button>
                  </div>
                </form>
              </DialogContent>
            </Dialog>
          </CardHeader>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="pl-6">Email</TableHead>
                  <TableHead>Subject ID</TableHead>
                  <TableHead className="w-32" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {identities.map(id => (
                  <TableRow key={id.id}>
                    <TableCell className="pl-6 font-medium">
                      <span>{id.email}</span>
                      {!id.is_enabled && (
                        <span className="ml-2 inline-flex items-center rounded-md bg-amber-50 px-2 py-0.5 text-xs font-medium text-amber-700 ring-1 ring-inset ring-amber-600/20 dark:bg-amber-400/10 dark:text-amber-400 dark:ring-amber-400/30">
                          pending
                        </span>
                      )}
                    </TableCell>
                    <TableCell className="font-mono text-sm text-muted-foreground">{id.subject_id}</TableCell>
                    <TableCell>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => navigate(`/identities/${id.id}`)}
                      >
                        View
                      </Button>
                    </TableCell>
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
