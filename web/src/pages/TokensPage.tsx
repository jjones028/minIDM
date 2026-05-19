import { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  listOAuthTokens, adminRevokeOAuthToken, inspectOAuthToken,
  isUnauthorized, type OAuthToken, type TokenInspectResult,
} from '@/api';
import { useAuth } from '@/context/auth';
import { Button } from '@/components/ui/button';
import {
  Card, CardContent, CardHeader, CardTitle, CardDescription,
} from '@/components/ui/card';
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from '@/components/ui/table';
import { AppNav } from '@/components/app-nav';

// ---- Token list -------------------------------------------------------

export default function TokensPage() {
  const navigate = useNavigate();
  const { setAuthenticated } = useAuth();
  const [tokens, setTokens] = useState<OAuthToken[]>([]);
  const [revoking, setRevoking] = useState<string | null>(null);

  const fetchTokens = useCallback(() => {
    let cancelled = false;
    listOAuthTokens()
      .then(({ data }) => { if (!cancelled) setTokens(data ?? []); })
      .catch(err => {
        if (isUnauthorized(err)) {
          setAuthenticated(false);
          navigate('/login');
        }
      });
    return () => { cancelled = true; };
  }, [navigate, setAuthenticated]);

  useEffect(() => { return fetchTokens(); }, [fetchTokens]);

  const handleRevoke = async (token: OAuthToken) => {
    if (!confirm(`Revoke token for ${token.identity_email}? The session will become invalid immediately.`)) return;
    setRevoking(token.id);
    try {
      await adminRevokeOAuthToken(token.id);
      setTokens(prev => prev.filter(t => t.id !== token.id));
    } catch {
      alert('Failed to revoke token.');
    } finally {
      setRevoking(null);
    }
  };

  const handleLogout = () => { setAuthenticated(false); navigate('/login'); };

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

        {/* Inspector */}
        <TokenInspector />

        {/* Active token list */}
        <Card>
          <CardHeader>
            <CardTitle>Active Tokens</CardTitle>
            <CardDescription>
              Non-revoked, non-expired refresh token grants.
              Revoking a row invalidates both the access token and refresh token.
            </CardDescription>
          </CardHeader>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="pl-6">Identity</TableHead>
                  <TableHead>Client</TableHead>
                  <TableHead>Scopes</TableHead>
                  <TableHead>JTI</TableHead>
                  <TableHead>Issued</TableHead>
                  <TableHead>Expires</TableHead>
                  <TableHead className="w-24" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {tokens.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={7} className="pl-6 text-muted-foreground">
                      No active tokens.
                    </TableCell>
                  </TableRow>
                )}
                {tokens.map(token => (
                  <TableRow key={token.id}>
                    <TableCell className="pl-6 font-medium">{token.identity_email}</TableCell>
                    <TableCell>
                      <code className="text-xs bg-muted rounded px-1.5 py-0.5">{token.client_id}</code>
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {token.scopes.join(', ')}
                    </TableCell>
                    <TableCell>
                      <code className="text-xs bg-muted rounded px-1.5 py-0.5" title={token.jti}>
                        {token.jti.slice(0, 8)}…
                      </code>
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground whitespace-nowrap">
                      {new Date(token.created_at).toLocaleString()}
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground whitespace-nowrap">
                      {new Date(token.expires_at).toLocaleString()}
                    </TableCell>
                    <TableCell>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-destructive hover:text-destructive"
                        disabled={revoking === token.id}
                        onClick={() => handleRevoke(token)}
                      >
                        {revoking === token.id ? 'Revoking…' : 'Revoke'}
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

// ---- Inspector --------------------------------------------------------

function TokenInspector() {
  const [raw, setRaw] = useState('');
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<TokenInspectResult | null>(null);
  const [apiError, setApiError] = useState<string | null>(null);

  const handleInspect = async (e: React.FormEvent) => {
    e.preventDefault();
    const token = raw.trim();
    if (!token) return;
    setLoading(true);
    setResult(null);
    setApiError(null);
    try {
      const { data } = await inspectOAuthToken(token);
      setResult(data);
    } catch {
      setApiError('Request failed — check that you are signed in with sufficient permissions.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>Token Inspector</CardTitle>
        <CardDescription>
          Paste a JWT access token to decode its claims and check its live status.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <form onSubmit={handleInspect} className="flex gap-2">
          <textarea
            className="flex-1 min-h-[72px] rounded-md border border-input bg-background px-3 py-2 text-sm font-mono resize-none ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
            placeholder="eyJhbGciOiJSUzI1NiIs…"
            value={raw}
            onChange={e => setRaw(e.target.value)}
            spellCheck={false}
          />
          <Button type="submit" disabled={loading || !raw.trim()} className="self-start">
            {loading ? 'Inspecting…' : 'Inspect'}
          </Button>
        </form>

        {apiError && (
          <p className="text-sm text-destructive">{apiError}</p>
        )}

        {result && <InspectResult result={result} />}
      </CardContent>
    </Card>
  );
}

// ---- Result display ---------------------------------------------------

function InspectResult({ result }: { result: TokenInspectResult }) {
  const { header, claims, status } = result;

  if (status.error) {
    return (
      <div className="rounded-md border border-destructive/30 bg-destructive/5 px-4 py-3">
        <p className="text-sm text-destructive font-medium">{status.error}</p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Status badges */}
      <div className="flex flex-wrap gap-2">
        <StatusBadge
          label="Signature"
          value={status.signature_valid ? 'Valid' : 'Invalid'}
          variant={status.signature_valid ? 'green' : 'red'}
        />
        <StatusBadge
          label="Expiry"
          value={status.expired ? 'Expired' : 'Valid'}
          variant={status.expired ? 'red' : 'green'}
        />
        <StatusBadge
          label="DB State"
          value={dbStatusLabel(status.db_status)}
          variant={dbStatusVariant(status.db_status)}
        />
        <StatusBadge
          label="Overall"
          value={status.active ? 'Active' : 'Inactive'}
          variant={status.active ? 'green' : 'red'}
        />
      </div>

      {/* Header + Claims */}
      <div className="grid md:grid-cols-2 gap-4">
        {header && (
          <div className="space-y-2">
            <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">Header</p>
            <ClaimsTable entries={Object.entries(header)} />
          </div>
        )}
        {claims && Object.keys(claims).length > 0 && (
          <div className="space-y-2">
            <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">Claims</p>
            <ClaimsTable entries={formatClaims(claims)} />
          </div>
        )}
      </div>
    </div>
  );
}

function ClaimsTable({ entries }: { entries: [string, unknown][] }) {
  return (
    <div className="rounded-md border divide-y text-sm">
      {entries.map(([key, value]) => (
        <div key={key} className="flex px-3 py-2 gap-3 min-w-0">
          <span className="font-mono text-muted-foreground shrink-0 w-16 truncate" title={key}>{key}</span>
          <span className="font-mono break-all min-w-0">{renderValue(value)}</span>
        </div>
      ))}
    </div>
  );
}

function StatusBadge({ label, value, variant }: {
  label: string;
  value: string;
  variant: 'green' | 'red' | 'yellow' | 'gray';
}) {
  const colors: Record<string, string> = {
    green: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400',
    red:   'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400',
    yellow:'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400',
    gray:  'bg-muted text-muted-foreground',
  };
  return (
    <div className="flex items-center gap-1.5">
      <span className="text-xs text-muted-foreground">{label}:</span>
      <span className={`text-xs px-2 py-0.5 rounded-full font-medium ${colors[variant]}`}>
        {value}
      </span>
    </div>
  );
}

// ---- Helpers ----------------------------------------------------------

function dbStatusLabel(s: string): string {
  return { active: 'Active', revoked: 'Revoked', not_found: 'Not Found', unknown: 'Unknown' }[s] ?? s;
}

function dbStatusVariant(s: string): 'green' | 'red' | 'yellow' | 'gray' {
  return ({ active: 'green', revoked: 'red', not_found: 'yellow', unknown: 'gray' } as const)[s as 'active' | 'revoked' | 'not_found' | 'unknown'] ?? 'gray';
}

function formatClaims(claims: TokenInspectResult['claims']): [string, unknown][] {
  if (!claims) return [];
  return Object.entries(claims).map(([k, v]) => {
    if ((k === 'exp' || k === 'iat') && typeof v === 'number') {
      return [k, `${new Date(v * 1000).toLocaleString()} (${v})`];
    }
    return [k, v];
  });
}

function renderValue(v: unknown): string {
  if (Array.isArray(v)) return v.join(', ');
  if (v === null || v === undefined) return '—';
  return String(v);
}
