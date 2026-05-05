import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  getIdentities, registerIdentity, logout, isUnauthorized, type Identity,
} from '@/api';
import { useAuth } from '@/context/auth';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Card, CardContent, CardHeader, CardTitle, CardDescription,
} from '@/components/ui/card';
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from '@/components/ui/table';
import { AxiosError } from 'axios';
import { AppNav } from '@/components/app-nav';

export default function DashboardPage() {
  const navigate = useNavigate();
  const { setAuthenticated } = useAuth();
  const [identities, setIdentities] = useState<Identity[]>([]);
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
      await registerIdentity({ email, password });
      setEmail('');
      setPassword('');
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
          <CardHeader>
            <CardTitle>Create Identity</CardTitle>
            <CardDescription>New identities are automatically granted the viewer role.</CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleCreateIdentity} className="flex flex-col md:flex-row gap-4 items-end">
              <div className="grid w-full gap-1.5">
                <label className="text-sm font-medium leading-none">Email Address</label>
                <Input
                  type="email"
                  placeholder="name@company.com"
                  value={email}
                  onChange={e => setEmail(e.target.value)}
                  required
                />
              </div>
              <div className="grid w-full gap-1.5">
                <label className="text-sm font-medium leading-none">Password</label>
                <Input
                  type="password"
                  placeholder="••••••••"
                  value={password}
                  onChange={e => setPassword(e.target.value)}
                  required
                />
              </div>
              <Button type="submit" className="w-full md:w-auto">Create Identity</Button>
            </form>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Identity Registry</CardTitle>
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
                    <TableCell className="pl-6 font-medium">{id.email}</TableCell>
                    <TableCell className="font-mono text-sm text-muted-foreground">{id.subject_id}</TableCell>
                    <TableCell>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => navigate(`/identities/${id.id}/roles`, { state: { email: id.email } })}
                      >
                        Manage Roles
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
