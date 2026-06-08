import { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  listAuditLogs, listAuditResourceTypes, isUnauthorized,
  type AuditLog, type AuditLogsFilter,
} from '@/api';
import { useAuth } from '@/context/auth';
import {
  Card, CardContent, CardHeader, CardTitle, CardDescription,
} from '@/components/ui/card';
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from '@/components/ui/table';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select';
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

// --- Helpers ---

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

function actorLabel(log: AuditLog): string {
  if (log.actor_email) return log.actor_email;
  if (log.actor_id) return log.actor_id.slice(0, 8) + '…';
  return 'system';
}

// --- Row ---

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
        <TableCell className="text-xs text-muted-foreground">
          {actorLabel(log)}
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

// --- Page ---

const PAGE_SIZE = 50;

export default function AuditLogsPage() {
  const navigate = useNavigate();
  const { setAuthenticated } = useAuth();

  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(0);
  const [loading, setLoading] = useState(true);
  const [resourceTypes, setResourceTypes] = useState<string[]>([]);

  // filter inputs (committed on Apply)
  const [resourceType, setResourceType] = useState('');
  const [actionFilter, setActionFilter] = useState('');
  const [actorIdFilter, setActorIdFilter] = useState('');
  const [since, setSince] = useState('');
  const [until, setUntil] = useState('');

  // committed filter used for actual fetch
  const [activeFilter, setActiveFilter] = useState<AuditLogsFilter>({});

  const fetch = useCallback((filter: AuditLogsFilter, pg: number) => {
    let cancelled = false;
    setLoading(true);
    listAuditLogs({ ...filter, limit: PAGE_SIZE, offset: pg * PAGE_SIZE })
      .then(({ data }) => {
        if (cancelled) return;
        setLogs(data.logs ?? []);
        setTotal(data.total);
      })
      .catch(err => {
        if (isUnauthorized(err)) {
          setAuthenticated(false);
          navigate('/login');
        }
      })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, [navigate, setAuthenticated]);

  useEffect(() => {
    listAuditResourceTypes()
      .then(({ data }) => setResourceTypes(data ?? []))
      .catch(() => {});
  }, []);

  useEffect(() => {
    return fetch(activeFilter, page);
  }, [activeFilter, page, fetch]);

  function handleApply(e: React.FormEvent) {
    e.preventDefault();
    const filter: AuditLogsFilter = {};
    if (resourceType) filter.resource_type = resourceType;
    if (actionFilter) filter.action = actionFilter;
    if (actorIdFilter) filter.actor_id = actorIdFilter;
    if (since) filter.since = new Date(since).toISOString();
    if (until) filter.until = new Date(until + 'T23:59:59').toISOString();
    setPage(0);
    setActiveFilter(filter);
  }

  function handleReset() {
    setResourceType('');
    setActionFilter('');
    setActorIdFilter('');
    setSince('');
    setUntil('');
    setPage(0);
    setActiveFilter({});
  }

  const totalPages = Math.ceil(total / PAGE_SIZE);
  const from = total === 0 ? 0 : page * PAGE_SIZE + 1;
  const to = Math.min((page + 1) * PAGE_SIZE, total);

  return (
    <div className="min-h-screen p-4 md:p-12">
      <div className="max-w-7xl mx-auto space-y-6">
        <AppNav />

        <Card>
          <CardHeader>
            <CardTitle>Audit Log</CardTitle>
            <CardDescription>
              Administrative actions across identities, roles, sessions, and OAuth2 clients.
              Click a row to expand its details.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">

            {/* Filter bar */}
            <form onSubmit={handleApply} className="flex flex-wrap gap-2 items-end">
              <div className="flex flex-col gap-1">
                <label className="text-xs text-muted-foreground">Resource type</label>
                <Select
                  value={resourceType || null}
                  onValueChange={(v) => setResourceType((v as string) ?? '')}
                >
                  <SelectTrigger className="h-9 text-sm">
                    <SelectValue>
                      {(value: unknown) => (value as string) || 'All'}
                    </SelectValue>
                  </SelectTrigger>
                  <SelectContent position="popper">
                    {resourceTypes.map(rt => (
                      <SelectItem key={rt} value={rt}>{rt}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="flex flex-col gap-1">
                <label className="text-xs text-muted-foreground">Action prefix</label>
                <Input
                  className="h-9 w-36"
                  placeholder="e.g. identity"
                  value={actionFilter}
                  onChange={e => setActionFilter(e.target.value)}
                />
              </div>

              <div className="flex flex-col gap-1">
                <label className="text-xs text-muted-foreground">Actor ID (UUID)</label>
                <Input
                  className="h-9 w-72 font-mono text-xs"
                  placeholder="xxxxxxxx-xxxx-…"
                  value={actorIdFilter}
                  onChange={e => setActorIdFilter(e.target.value)}
                />
              </div>

              <div className="flex flex-col gap-1">
                <label className="text-xs text-muted-foreground">Since</label>
                <Input
                  type="date"
                  className="h-9 w-36"
                  value={since}
                  onChange={e => setSince(e.target.value)}
                />
              </div>

              <div className="flex flex-col gap-1">
                <label className="text-xs text-muted-foreground">Until</label>
                <Input
                  type="date"
                  className="h-9 w-36"
                  value={until}
                  onChange={e => setUntil(e.target.value)}
                />
              </div>

              <Button type="submit" size="sm" className="h-9">Apply</Button>
              <Button type="button" size="sm" variant="ghost" className="h-9" onClick={handleReset}>
                Reset
              </Button>
            </form>

            {/* Table */}
            {loading ? (
              <p className="text-sm text-muted-foreground">Loading…</p>
            ) : logs.length === 0 ? (
              <p className="text-sm text-muted-foreground">No audit log entries match the current filter.</p>
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

            {/* Pagination */}
            {total > PAGE_SIZE && (
              <div className="flex items-center justify-between text-sm text-muted-foreground pt-2">
                <span>{from}–{to} of {total}</span>
                <div className="flex gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={page === 0}
                    onClick={() => setPage(p => p - 1)}
                  >
                    Previous
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={page + 1 >= totalPages}
                    onClick={() => setPage(p => p + 1)}
                  >
                    Next
                  </Button>
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
