import { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  listOAuthClients, createOAuthClient, updateOAuthClient, deleteOAuthClient,
  isUnauthorized, type OAuthClient, type UpdateOAuthClientData,
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
import { AppNav } from '@/components/app-nav';
import { AxiosError } from 'axios';

const ALL_SCOPES = ['openid', 'profile', 'email'];

export default function OAuthClientsPage() {
  const navigate = useNavigate();
  const { setAuthenticated } = useAuth();

  const [clients, setClients] = useState<OAuthClient[]>([]);

  // Create form state
  const [newName, setNewName] = useState('');
  const [newDescription, setNewDescription] = useState('');
  const [newRedirectURIs, setNewRedirectURIs] = useState('');
  const [newScopes, setNewScopes] = useState<string[]>(['openid', 'profile', 'email']);

  // Edit state
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editName, setEditName] = useState('');
  const [editDescription, setEditDescription] = useState('');
  const [editRedirectURIs, setEditRedirectURIs] = useState('');
  const [editScopes, setEditScopes] = useState<string[]>([]);
  const [editEnabled, setEditEnabled] = useState(true);

  // Secret reveal modal
  const [revealedSecret, setRevealedSecret] = useState<{ clientId: string; secret: string } | null>(null);

  const fetchClients = useCallback(() => {
    let cancelled = false;
    listOAuthClients()
      .then(({ data }) => { if (!cancelled) setClients(data ?? []); })
      .catch(err => {
        if (isUnauthorized(err)) {
          setAuthenticated(false);
          navigate('/login');
        }
      });
    return () => { cancelled = true; };
  }, [navigate, setAuthenticated]);

  useEffect(() => {
    return fetchClients();
  }, [fetchClients]);

  const handleLogout = () => {
    setAuthenticated(false);
    navigate('/login');
  };

  const parseURIs = (raw: string) =>
    raw.split('\n').map(s => s.trim()).filter(Boolean);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    const redirectUris = parseURIs(newRedirectURIs);
    if (redirectUris.length === 0) {
      alert('At least one redirect URI is required.');
      return;
    }
    try {
      const { data } = await createOAuthClient({
        name: newName.trim(),
        description: newDescription.trim() || undefined,
        redirect_uris: redirectUris,
        scopes: newScopes,
      });
      setRevealedSecret({ clientId: data.client.client_id, secret: data.client_secret });
      setNewName('');
      setNewDescription('');
      setNewRedirectURIs('');
      setNewScopes(['openid', 'profile', 'email']);
      fetchClients();
    } catch (err) {
      const e = err as AxiosError<string>;
      alert('Failed: ' + (e.response?.data?.trim() ?? e.message));
    }
  };

  const startEdit = (c: OAuthClient) => {
    setEditingId(c.id);
    setEditName(c.name);
    setEditDescription(c.description ?? '');
    setEditRedirectURIs(c.redirect_uris.join('\n'));
    setEditScopes(c.scopes);
    setEditEnabled(c.is_enabled);
  };

  const handleUpdate = async (id: string) => {
    const redirectUris = parseURIs(editRedirectURIs);
    if (redirectUris.length === 0) {
      alert('At least one redirect URI is required.');
      return;
    }
    const update: UpdateOAuthClientData = {
      name: editName.trim(),
      description: editDescription.trim() || undefined,
      redirect_uris: redirectUris,
      scopes: editScopes,
      is_enabled: editEnabled,
    };
    try {
      await updateOAuthClient(id, update);
      setEditingId(null);
      fetchClients();
    } catch (err) {
      const e = err as AxiosError<string>;
      alert('Failed: ' + (e.response?.data?.trim() ?? e.message));
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm('Delete this OAuth2 client? This cannot be undone.')) return;
    try {
      await deleteOAuthClient(id);
      fetchClients();
    } catch (err) {
      const e = err as AxiosError<string>;
      alert('Failed: ' + (e.response?.data?.trim() ?? e.message));
    }
  };

  const toggleScope = (scope: string, checked: boolean, current: string[], setFn: (s: string[]) => void) => {
    setFn(checked ? [...current, scope] : current.filter(s => s !== scope));
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

        {/* Secret reveal modal */}
        {revealedSecret && (
          <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
            <Card className="w-full max-w-lg">
              <CardHeader>
                <CardTitle>Client Secret — Save This Now</CardTitle>
                <CardDescription>
                  This is shown only once. Copy and store it securely before closing.
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-1">
                  <p className="text-xs text-muted-foreground font-medium uppercase tracking-wide">Client ID</p>
                  <code className="block text-sm bg-muted rounded-md px-3 py-2 break-all">
                    {revealedSecret.clientId}
                  </code>
                </div>
                <div className="space-y-1">
                  <p className="text-xs text-muted-foreground font-medium uppercase tracking-wide">Client Secret</p>
                  <code className="block text-sm bg-muted rounded-md px-3 py-2 break-all">
                    {revealedSecret.secret}
                  </code>
                </div>
                <Button className="w-full" onClick={() => setRevealedSecret(null)}>
                  I've saved the secret
                </Button>
              </CardContent>
            </Card>
          </div>
        )}

        {/* Create form */}
        <Card>
          <CardHeader>
            <CardTitle>Register OAuth2 Client</CardTitle>
            <CardDescription>
              Register an application that will use minIDM as an identity provider.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleCreate} className="space-y-4">
              <div className="grid md:grid-cols-2 gap-4">
                <div className="grid gap-1.5">
                  <label className="text-sm font-medium leading-none">Name</label>
                  <Input
                    placeholder="My Application"
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
              </div>
              <div className="grid gap-1.5">
                <label className="text-sm font-medium leading-none">
                  Redirect URIs{' '}
                  <span className="text-muted-foreground font-normal">(one per line)</span>
                </label>
                <textarea
                  className="flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                  placeholder={"https://myapp.com/callback\nhttp://localhost:3000/callback"}
                  value={newRedirectURIs}
                  onChange={e => setNewRedirectURIs(e.target.value)}
                  required
                />
              </div>
              <div className="grid gap-2">
                <label className="text-sm font-medium leading-none">Scopes</label>
                <div className="flex gap-4">
                  {ALL_SCOPES.map(scope => (
                    <label key={scope} className="flex items-center gap-1.5 text-sm cursor-pointer">
                      <input
                        type="checkbox"
                        checked={newScopes.includes(scope)}
                        onChange={e => toggleScope(scope, e.target.checked, newScopes, setNewScopes)}
                        className="rounded"
                      />
                      {scope}
                    </label>
                  ))}
                </div>
              </div>
              <Button type="submit" className="w-full md:w-auto">Register Client</Button>
            </form>
          </CardContent>
        </Card>

        {/* Clients table */}
        <Card>
          <CardHeader>
            <CardTitle>OAuth2 Clients</CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="pl-6">Name</TableHead>
                  <TableHead>Client ID</TableHead>
                  <TableHead>Scopes</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="w-32" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {clients.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={5} className="pl-6 text-muted-foreground">
                      No clients registered yet.
                    </TableCell>
                  </TableRow>
                )}
                {clients.map(client => (
                  <TableRow key={client.id}>
                    {editingId === client.id ? (
                      <>
                        <TableCell className="pl-6" colSpan={3}>
                          <div className="space-y-3">
                            <div className="grid md:grid-cols-2 gap-2">
                              <Input
                                value={editName}
                                onChange={e => setEditName(e.target.value)}
                                placeholder="Name"
                                className="h-8"
                              />
                              <Input
                                value={editDescription}
                                onChange={e => setEditDescription(e.target.value)}
                                placeholder="Description"
                                className="h-8"
                              />
                            </div>
                            <textarea
                              className="flex min-h-[60px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                              value={editRedirectURIs}
                              onChange={e => setEditRedirectURIs(e.target.value)}
                              placeholder="Redirect URIs, one per line"
                            />
                            <div className="flex flex-wrap gap-4">
                              {ALL_SCOPES.map(scope => (
                                <label key={scope} className="flex items-center gap-1.5 text-sm cursor-pointer">
                                  <input
                                    type="checkbox"
                                    checked={editScopes.includes(scope)}
                                    onChange={e => toggleScope(scope, e.target.checked, editScopes, setEditScopes)}
                                  />
                                  {scope}
                                </label>
                              ))}
                              <label className="flex items-center gap-1.5 text-sm cursor-pointer">
                                <input
                                  type="checkbox"
                                  checked={editEnabled}
                                  onChange={e => setEditEnabled(e.target.checked)}
                                />
                                Enabled
                              </label>
                            </div>
                          </div>
                        </TableCell>
                        <TableCell />
                        <TableCell>
                          <div className="flex gap-1">
                            <Button size="sm" onClick={() => handleUpdate(client.id)}>Save</Button>
                            <Button size="sm" variant="ghost" onClick={() => setEditingId(null)}>Cancel</Button>
                          </div>
                        </TableCell>
                      </>
                    ) : (
                      <>
                        <TableCell className="pl-6">
                          <span className="font-medium">{client.name}</span>
                          {client.description && (
                            <p className="text-xs text-muted-foreground mt-0.5">{client.description}</p>
                          )}
                        </TableCell>
                        <TableCell>
                          <ClientIDCell clientId={client.client_id} />
                        </TableCell>
                        <TableCell className="text-muted-foreground text-sm">
                          {client.scopes.join(', ')}
                        </TableCell>
                        <TableCell>
                          <span className={`text-xs px-1.5 py-0.5 rounded-sm font-medium ${
                            client.is_enabled
                              ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
                              : 'bg-muted text-muted-foreground'
                          }`}>
                            {client.is_enabled ? 'enabled' : 'disabled'}
                          </span>
                        </TableCell>
                        <TableCell>
                          <div className="flex gap-1">
                            <Button variant="ghost" size="sm" onClick={() => startEdit(client)}>
                              Edit
                            </Button>
                            <Button
                              variant="ghost"
                              size="sm"
                              className="text-destructive hover:text-destructive"
                              onClick={() => handleDelete(client.id)}
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

// ClientIDCell shows the client_id with a copy button.
function ClientIDCell({ clientId }: { clientId: string }) {
  const [copied, setCopied] = useState(false);

  const copy = () => {
    navigator.clipboard.writeText(clientId).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    });
  };

  return (
    <div className="flex items-center gap-1.5">
      <code className="text-xs bg-muted rounded px-1.5 py-0.5 max-w-[140px] truncate" title={clientId}>
        {clientId}
      </code>
      <button
        type="button"
        onClick={copy}
        className="text-xs text-muted-foreground hover:text-foreground transition-colors"
        title="Copy client ID"
      >
        {copied ? '✓' : 'copy'}
      </button>
    </div>
  );
}
