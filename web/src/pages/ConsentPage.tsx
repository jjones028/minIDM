import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { getClientInfo, approveConsent, isUnauthorized, type ClientInfo } from '@/api';
import { useAuth } from '@/context/auth';
import { Button } from '@/components/ui/button';
import {
  Card, CardContent, CardHeader, CardTitle, CardDescription,
} from '@/components/ui/card';

const SCOPE_LABELS: Record<string, string> = {
  openid:  'Verify your identity',
  profile: 'Read your profile information',
  email:   'Read your email address',
};

export default function ConsentPage() {
  const navigate = useNavigate();
  const { setAuthenticated } = useAuth();

  // Read params passed by the authorize redirect.
  const params = new URLSearchParams(window.location.search);
  const clientId           = params.get('client_id') ?? '';
  const redirectUri        = params.get('redirect_uri') ?? '';
  const scope              = params.get('scope') ?? 'openid';
  const state              = params.get('state') ?? '';
  const codeChallenge      = params.get('code_challenge') ?? '';
  const codeChallengeMethod = params.get('code_challenge_method') ?? 'S256';

  const [clientInfo, setClientInfo] = useState<ClientInfo | null>(null);
  const [error, setError]           = useState('');
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (!clientId) { setError('Missing client_id.'); return; }
    getClientInfo(clientId)
      .then(({ data }) => setClientInfo(data))
      .catch(err => {
        if (isUnauthorized(err)) { setAuthenticated(false); navigate('/login'); return; }
        setError('Unknown application. The authorization request may be invalid.');
      });
  }, [clientId]);

  const requestedScopes = scope.split(' ').filter(Boolean);

  const handleApprove = async () => {
    setSubmitting(true);
    try {
      const { data } = await approveConsent({
        client_id:             clientId,
        redirect_uri:          redirectUri,
        scope,
        state,
        code_challenge:        codeChallenge,
        code_challenge_method: codeChallengeMethod,
      });
      window.location.href = data.redirect_url;
    } catch (err) {
      if (isUnauthorized(err)) { setAuthenticated(false); navigate('/login'); return; }
      setError('Authorization failed. Please try again.');
      setSubmitting(false);
    }
  };

  const handleDeny = () => {
    if (!redirectUri) return;
    const dest = new URL(redirectUri);
    dest.searchParams.set('error', 'access_denied');
    dest.searchParams.set('error_description', 'The user denied the authorization request.');
    if (state) dest.searchParams.set('state', state);
    window.location.href = dest.toString();
  };

  if (error) {
    return (
      <div className="min-h-screen flex items-center justify-center p-4">
        <Card className="w-full max-w-md">
          <CardHeader>
            <CardTitle>Authorization Error</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-destructive">{error}</p>
          </CardContent>
        </Card>
      </div>
    );
  }

  if (!clientInfo) {
    return (
      <div className="min-h-screen flex items-center justify-center p-4">
        <p className="text-sm text-muted-foreground">Loading…</p>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex items-center justify-center p-4">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>{clientInfo.name}</CardTitle>
          {clientInfo.description && (
            <CardDescription>{clientInfo.description}</CardDescription>
          )}
        </CardHeader>
        <CardContent className="space-y-6">
          <div>
            <p className="text-sm font-medium mb-3">
              This application is requesting permission to:
            </p>
            <ul className="space-y-2">
              {requestedScopes.map(s => (
                <li key={s} className="flex items-start gap-2 text-sm">
                  <span className="mt-0.5 text-green-500">✓</span>
                  <span>{SCOPE_LABELS[s] ?? s}</span>
                </li>
              ))}
            </ul>
          </div>

          <div className="flex gap-3">
            <Button
              className="flex-1"
              onClick={handleApprove}
              disabled={submitting}
            >
              {submitting ? 'Authorizing…' : 'Allow'}
            </Button>
            <Button
              variant="outline"
              className="flex-1"
              onClick={handleDeny}
              disabled={submitting}
            >
              Deny
            </Button>
          </div>

          <p className="text-xs text-muted-foreground text-center">
            You are logged in. Only grant access to applications you trust.
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
