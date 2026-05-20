import { useState, useEffect } from 'react';
import { Navigate, useNavigate, useSearchParams } from 'react-router-dom';
import { getConfig, registerIdentity, login, type AppConfig } from '@/api';
import { useAuth } from '@/context/auth';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent, CardHeader } from '@/components/ui/card';
import { AxiosError } from 'axios';

type Tab = 'signin' | 'signup';

export default function AuthPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const { checked, authenticated, setAuthenticated } = useAuth();

  const [config, setConfig] = useState<AppConfig | null>(null);
  const [tab, setTab] = useState<Tab>('signin');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [registered, setRegistered] = useState(false);

  useEffect(() => {
    // Extract client_id from the ?next= URL so the config call is scoped to
    // the OAuth2 client that initiated the login redirect.
    let clientId: string | undefined;
    const next = searchParams.get('next');
    if (next) {
      try {
        const nextUrl = new URL(next, window.location.origin);
        clientId = nextUrl.searchParams.get('client_id') ?? undefined;
      } catch { /* ignore malformed URLs */ }
    }
    getConfig(clientId).then(({ data }) => setConfig(data)).catch(() => {});
  }, []);

  if (checked && authenticated) return <Navigate to="/" replace />;

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
        setRegistered(true);
        return;
      }
      await login({ email, password });
      setAuthenticated(true);
      const next = searchParams.get('next');
      if (next) {
        window.location.href = next;
      } else {
        navigate('/');
      }
    } catch (err) {
      const axiosError = err as AxiosError<string>;
      const status = axiosError.response?.status;
      const msg = axiosError.response?.data?.trim();
      if (tab === 'signin') {
        if (status === 403 && msg === 'account_not_active') {
          setError('Your account is not active. If you recently registered, it may be pending admin approval.');
        } else {
          setError('Invalid email or password.');
        }
      } else {
        setError(msg || 'Registration failed.');
      }
    }
  };

  if (registered) {
    return (
      <div className="min-h-screen flex items-center justify-center p-4">
        <div className="w-full max-w-sm space-y-6">
          <div className="space-y-1 text-center">
            <h1 className="text-4xl font-extrabold tracking-tight font-heading">minidm</h1>
            <p className="text-sm text-muted-foreground">Identity management.</p>
          </div>
          <Card>
            <CardContent className="pt-6 space-y-4 text-center">
              <p className="text-sm font-medium">Account created</p>
              <p className="text-sm text-muted-foreground">
                Your account is pending admin approval. You will be able to sign in once an administrator activates it.
              </p>
              <Button variant="outline" className="w-full" onClick={() => { setRegistered(false); switchTab('signin'); }}>
                Back to Sign In
              </Button>
            </CardContent>
          </Card>
        </div>
      </div>
    );
  }

  const showSignUp = config?.registration_enabled === true;

  return (
    <div className="min-h-screen flex items-center justify-center p-4">
      <div className="w-full max-w-sm space-y-6">
        <div className="space-y-1 text-center">
          <h1 className="text-4xl font-extrabold tracking-tight font-heading">minidm</h1>
          <p className="text-sm text-muted-foreground">Identity management.</p>
        </div>
        <Card>
          {showSignUp && (
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
          )}
          <CardContent className={showSignUp ? undefined : 'pt-6'}>
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
