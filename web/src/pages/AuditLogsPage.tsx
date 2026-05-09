import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { listAuditLogs, isUnauthorized, type AuditLog } from '@/api';
import { useAuth } from '@/context/auth';
import {
  Card, CardContent, CardHeader, CardTitle, CardDescription,
} from '@/components/ui/card';
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from '@/components/ui/table';
import { AppNav } from '@/components/app-nav';

// --- JSON syntax highlighter ---

type TokenType = 'key' | 'string' | 'number' | 'boolean' | 'null' | 'punctuation' | 'whitespace';
interface Token { text: string; type: TokenType }

const TOKEN_RE = /("(?:[^"\\]|\\.)*")(\s*:)?|(\btrue\b|\bfalse\b|\bnull\b)|(-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?)|([{}\[\],])|(:)|([ \t\n]+)/g;

function tokenize(json: string): Token[] {
  const tokens: Token[] = [];
  let m: RegExpExecArray | null;
  TOKEN_RE.lastIndex = 0;
  while ((m = TOKEN_RE.exec(json)) !== null) {
    if (m[1] !== undefined) {
      tokens.push({ text: m[1], type: m[2] !== undefined ? 'key' : 'string' });
      if (m[2] !== undefined) tokens.push({ text: m[2], type: 'punctuation' });
    } else if (m[3] !== undefined) {
      tokens.push({ text: m[3], type: m[3] === 'null' ? 'null' : 'boolean' });
    } else if (m[4] !== undefined) {
      tokens.push({ text: m[4], type: 'number' });
    } else if (m[5] !== undefined) {
      tokens.push({ text: m[5], type: 'punctuation' });
    } else if (m[6] !== undefined) {
      tokens.push({ text: m[6], type: 'punctuation' });
    } else if (m[7] !== undefined) {
      tokens.push({ text: m[7], type: 'whitespace' });
    }
  }
  return tokens;
}

const TOKEN_CLASS: Record<TokenType, string> = {
  key:        'text-blue-400',
  string:     'text-green-400',
  number:     'text-amber-400',
  boolean:    'text-purple-400',
  null:       'text-muted-foreground/60',
  punctuation:'text-foreground/60',
  whitespace: '',
};

function JsonHighlight({ value }: { value: Record<string, unknown> }) {
  const formatted = JSON.stringify(value, null, 2);
  const tokens = tokenize(formatted);
  return (
    <pre className="text-xs font-mono leading-relaxed whitespace-pre-wrap break-all">
      {tokens.map((tok, i) => (
        tok.type === 'whitespace'
          ? tok.text
          : <span key={i} className={TOKEN_CLASS[tok.type]}>{tok.text}</span>
      ))}
    </pre>
  );
}

// --- Page ---

function formatTime(iso: string): string {
  return new Date(iso).toLocaleString(undefined, {
    year: 'numeric', month: 'short', day: 'numeric',
    hour: '2-digit', minute: '2-digit', second: '2-digit',
  });
}

function summariseDetails(details: Record<string, unknown> | null): string {
  if (!details) return '—';
  return Object.entries(details)
    .map(([k, v]) => `${k}: ${v}`)
    .join(', ');
}

function AuditRow({ log }: { log: AuditLog }) {
  const [expanded, setExpanded] = useState(false);
  const hasDetails = log.details !== null;

  return (
    <>
      <TableRow
        onClick={() => hasDetails && setExpanded(e => !e)}
        className={hasDetails ? 'cursor-pointer hover:bg-muted/50 select-none' : ''}
      >
        <TableCell className="whitespace-nowrap text-xs text-muted-foreground">
          {formatTime(log.created_at)}
        </TableCell>
        <TableCell>
          <span className="font-mono text-xs bg-muted px-1.5 py-0.5 rounded">
            {log.action}
          </span>
        </TableCell>
        <TableCell className="text-sm">{log.resource_type}</TableCell>
        <TableCell className="font-mono text-xs text-muted-foreground">
          {log.resource_id ?? '—'}
        </TableCell>
        <TableCell className="font-mono text-xs text-muted-foreground">
          {log.actor_id ? log.actor_id.slice(0, 8) + '…' : 'system'}
        </TableCell>
        <TableCell className="text-xs text-muted-foreground">
          {hasDetails ? (
            <span className="flex items-center gap-1">
              <span>{summariseDetails(log.details)}</span>
              <span className="text-muted-foreground/50">{expanded ? '▲' : '▼'}</span>
            </span>
          ) : '—'}
        </TableCell>
      </TableRow>

      {expanded && hasDetails && (
        <TableRow className="bg-muted/30 hover:bg-muted/30">
          <TableCell colSpan={6} className="py-3 px-4">
            <JsonHighlight value={log.details!} />
          </TableCell>
        </TableRow>
      )}
    </>
  );
}

export default function AuditLogsPage() {
  const navigate = useNavigate();
  const { setAuthenticated } = useAuth();
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    listAuditLogs(100, 0)
      .then(({ data }) => { if (!cancelled) setLogs(data ?? []); })
      .catch(err => {
        if (isUnauthorized(err)) {
          setAuthenticated(false);
          navigate('/login');
        }
      })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, []);

  return (
    <div className="min-h-screen p-4 md:p-12">
      <div className="max-w-7xl mx-auto space-y-6">
        <AppNav />

        <Card>
          <CardHeader>
            <CardTitle>Audit Log</CardTitle>
            <CardDescription>
              Recent administrative actions across identities, roles, sessions, and OAuth2 clients.
              Click a row to expand its details.
            </CardDescription>
          </CardHeader>
          <CardContent>
            {loading ? (
              <p className="text-sm text-muted-foreground">Loading…</p>
            ) : logs.length === 0 ? (
              <p className="text-sm text-muted-foreground">No audit log entries yet.</p>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Time</TableHead>
                    <TableHead>Action</TableHead>
                    <TableHead>Resource</TableHead>
                    <TableHead>Resource ID</TableHead>
                    <TableHead>Actor</TableHead>
                    <TableHead>Details</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {logs.map(log => (
                    <AuditRow key={log.id} log={log} />
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
