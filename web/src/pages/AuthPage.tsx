import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { registerIdentity, login, setToken, getToken } from '@/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent, CardHeader } from '@/components/ui/card';
import { AxiosError } from 'axios';
import { Navigate } from 'react-router-dom';

type Tab = 'signin' | 'signup';

export default function AuthPage() {
  const navigate = useNavigate();

  if (getToken()) return <Navigate to="/" replace />;

  const [tab, setTab] = useState<Tab>('signin');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');

  const switchTab = (next: Tab) => {
    setTab(next);
    setError('');
    setEmail('');
    setPassword('');
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    try {
      if (tab === 'signup') {
        await registerIdentity({ email, password });
      }
      const { data } = await login({ email, password });
      setToken(data.token);
      navigate('/');
    } catch (err) {
      const axiosError = err as AxiosError<string>;
      const msg = axiosError.response?.data?.trim();
      setError(tab === 'signup' ? (msg || 'Registration failed.') : 'Invalid email or password.');
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center p-4">
      <div className="w-full max-w-sm space-y-6">
        <div className="space-y-1 text-center">
          <h1 className="text-4xl font-extrabold tracking-tight font-heading">minidm</h1>
          <p className="text-sm text-muted-foreground">Identity management.</p>
        </div>
        <Card>
          <CardHeader>
            <div className="flex gap-1 rounded-lg bg-muted p-1 text-sm font-medium">
              <button
                type="button"
                onClick={() => switchTab('signin')}
                className={`flex-1 rounded-md px-3 py-1.5 transition-colors ${
                  tab === 'signin'
                    ? 'bg-background text-foreground shadow-sm'
                    : 'text-muted-foreground hover:text-foreground'
                }`}
              >
                Sign In
              </button>
              <button
                type="button"
                onClick={() => switchTab('signup')}
                className={`flex-1 rounded-md px-3 py-1.5 transition-colors ${
                  tab === 'signup'
                    ? 'bg-background text-foreground shadow-sm'
                    : 'text-muted-foreground hover:text-foreground'
                }`}
              >
                Sign Up
              </button>
            </div>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="grid gap-1.5">
                <label className="text-sm font-medium leading-none">Email</label>
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
              {error && <p className="text-sm text-destructive">{error}</p>}
              <Button type="submit" className="w-full">
                {tab === 'signin' ? 'Sign In' : 'Create Account'}
              </Button>
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
