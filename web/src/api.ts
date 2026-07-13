import axios, { type AxiosError } from 'axios';

export interface Identity {
  id: string;
  subject_id: string;
  email: string;
  is_enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface RegisterIdentityData {
  email: string;
  password: string;
}

export interface LoginData {
  email: string;
  password: string;
}

export interface Role {
  id: string;
  name: string;
  description: string | null;
  is_builtin: boolean;
  created_at: string;
  updated_at: string;
}

export interface Resource {
  id: string;
  name: string;
  description: string | null;
  created_at: string;
}

export interface Action {
  id: string;
  name: string;
  description: string | null;
  created_at: string;
}

export interface RolePermission {
  id: string;
  resource: string;
  action: string;
}

export interface AppConfig {
  registration_enabled: boolean;
}

const api = axios.create({ baseURL: '/api' });

export const getConfig = (clientId?: string) => {
  const qs = clientId ? `?client_id=${encodeURIComponent(clientId)}` : '';
  return api.get<AppConfig>(`/config${qs}`);
};
export const login = (data: LoginData) => api.post('/login', data);
export const logout = () => api.delete('/session');
export const getMe = () => api.get<{ id: string }>('/me');
export const getIdentities = () => api.get<Identity[]>('/identities');
export const registerIdentity = (data: RegisterIdentityData) => api.post('/register', data);
export const createIdentity = (data: RegisterIdentityData) => api.post<Identity>('/identities', data);
export const getRoles = () => api.get<Role[]>('/roles');
export const createRole = (name: string, description: string) =>
  api.post<Role>('/roles', { name, description });
export const updateRole = (id: string, name: string, description: string) =>
  api.patch<Role>(`/roles/${id}`, { name, description });
export const deleteRole = (id: string) => api.delete(`/roles/${id}`);
export const getResources = () => api.get<Resource[]>('/resources');
export const getActions = () => api.get<Action[]>('/actions');
export const getRolePermissions = (roleId: string) => api.get<RolePermission[]>(`/roles/${roleId}/permissions`);
export const addRolePermission = (roleId: string, resourceId: string, actionId: string) =>
  api.post<RolePermission>(`/roles/${roleId}/permissions`, { resource_id: resourceId, action_id: actionId });
export const removeRolePermission = (roleId: string, permId: string) =>
  api.delete(`/roles/${roleId}/permissions/${permId}`);
export interface IdentitySession {
  handle: string;
  created_at: string;
  expires_at: string;
}

export const getIdentity = (id: string) => api.get<Identity>(`/identities/${id}`);
export const getIdentitySessions = (id: string) => api.get<IdentitySession[]>(`/identities/${id}/sessions`);
export const revokeIdentitySession = (identityId: string, handle: string) =>
  api.delete(`/identities/${identityId}/sessions/${handle}`);
export const resetIdentityPassword = (identityId: string, password: string) =>
  api.post(`/identities/${identityId}/reset-password`, { password });
export const setIdentityEnabled = (identityId: string, enabled: boolean) =>
  api.patch<{ id: string; is_enabled: boolean; updated_at: string }>(`/identities/${identityId}/enabled`, { enabled });
export const deleteIdentity = (identityId: string) => api.delete(`/identities/${identityId}`);
export const getIdentityRoles = (id: string) => api.get<Role[]>(`/identities/${id}/roles`);
export const assignRole = (identityId: string, roleId: string) =>
  api.post(`/identities/${identityId}/roles`, { role_id: roleId });
export const removeRole = (identityId: string, roleId: string) =>
  api.delete(`/identities/${identityId}/roles/${roleId}`);

// ---- OAuth2 Client Management ----

export interface OAuthClient {
  id: string;
  client_id: string;
  name: string;
  description: string | null;
  redirect_uris: string[];
  scopes: string[];
  is_enabled: boolean;
  is_public: boolean;
  auto_consent: boolean;
  allow_registration: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateOAuthClientData {
  name: string;
  description?: string;
  redirect_uris: string[];
  scopes: string[];
  auto_consent?: boolean;
  is_public?: boolean;
}

export interface UpdateOAuthClientData {
  name: string;
  description?: string;
  redirect_uris: string[];
  scopes: string[];
  is_enabled: boolean;
  auto_consent: boolean;
  allow_registration: boolean;
}

export interface ClientInfo {
  name: string;
  description: string | null;
  scopes: string[];
  auto_consent: boolean;
}

export interface ConsentParams {
  client_id: string;
  redirect_uri: string;
  scope: string;
  state: string;
  code_challenge: string;
  code_challenge_method: string;
  nonce?: string;
}

export const getClientInfo = (clientId: string) =>
  api.get<ClientInfo>(`/oauth2/client-info?client_id=${encodeURIComponent(clientId)}`);

export const approveConsent = (params: ConsentParams) =>
  api.post<{ redirect_url: string }>('/oauth2/consent', params);

export interface CreateOAuthClientResult {
  client: OAuthClient;
  client_secret: string; // plaintext, shown once
}

export interface OAuthToken {
  id: string;
  client_id: string;
  identity_id: string;
  jti: string;
  scopes: string[];
  expires_at: string;
  created_at: string;
  identity_email: string;
}

export const listOAuthTokens = () => api.get<OAuthToken[]>('/oauth2/tokens');
export const adminRevokeOAuthToken = (id: string) => api.delete(`/oauth2/tokens/${id}`);

export interface TokenInspectResult {
  header?: Record<string, unknown>;
  claims?: {
    iss?: string;
    sub?: string;
    aud?: string[];
    exp?: number;
    iat?: number;
    jti?: string;
    email?: string;
    scope?: string;
    nonce?: string;
  };
  status: {
    signature_valid: boolean;
    expired: boolean;
    db_status: 'active' | 'revoked' | 'not_found' | 'unknown';
    active: boolean;
    error?: string;
  };
}

export const inspectOAuthToken = (token: string) =>
  api.post<TokenInspectResult>('/oauth2/tokens/inspect', { token });

export const rotateOAuthClientSecret = (id: string) =>
  api.post<{ client_secret: string }>(`/oauth2/clients/${id}/rotate-secret`);

export const listOAuthClients = () => api.get<OAuthClient[]>('/oauth2/clients');
export const createOAuthClient = (data: CreateOAuthClientData) =>
  api.post<CreateOAuthClientResult>('/oauth2/clients', data);
export const getOAuthClient = (id: string) => api.get<OAuthClient>(`/oauth2/clients/${id}`);
export const updateOAuthClient = (id: string, data: UpdateOAuthClientData) =>
  api.patch<OAuthClient>(`/oauth2/clients/${id}`, data);
export const deleteOAuthClient = (id: string) => api.delete(`/oauth2/clients/${id}`);

export const isUnauthorized = (err: unknown) =>
  (err as AxiosError)?.response?.status === 401;

// ---- Client Roles ----

export interface ClientRole {
  id: string;
  client_id: string;
  name: string;
  description: string | null;
  created_at: string;
}

export interface ClientGroup {
  id: string;
  client_id: string;
  name: string;
  description: string | null;
  created_at: string;
}

export interface RoleMember {
  id: string;
  email: string;
}

export interface RoleGroup {
  id: string;
  name: string;
}

export interface ClientRoleAssignment {
  role_id: string;
  role_name: string;
  description: string | null;
  client_db_id: string;
  client_name: string;
  app_client_id: string;
}

export interface ClientGroupMembership {
  group_id: string;
  group_name: string;
  description: string | null;
  client_db_id: string;
  client_name: string;
  app_client_id: string;
}

// Client roles CRUD
export const listClientRoles = (clientDbId: string) =>
  api.get<ClientRole[]>(`/oauth2/clients/${clientDbId}/roles`);
export const createClientRole = (clientDbId: string, name: string, description: string) =>
  api.post<ClientRole>(`/oauth2/clients/${clientDbId}/roles`, { name, description });
export const updateClientRole = (clientDbId: string, roleId: string, name: string, description: string) =>
  api.patch<ClientRole>(`/oauth2/clients/${clientDbId}/roles/${roleId}`, { name, description });
export const deleteClientRole = (clientDbId: string, roleId: string) =>
  api.delete(`/oauth2/clients/${clientDbId}/roles/${roleId}`);

// Role identity assignments
export const listIdentitiesWithRole = (clientDbId: string, roleId: string) =>
  api.get<RoleMember[]>(`/oauth2/clients/${clientDbId}/roles/${roleId}/identities`);
export const assignIdentityToRole = (clientDbId: string, roleId: string, identityId: string) =>
  api.post(`/oauth2/clients/${clientDbId}/roles/${roleId}/identities`, { identity_id: identityId });
export const removeIdentityFromRole = (clientDbId: string, roleId: string, identityId: string) =>
  api.delete(`/oauth2/clients/${clientDbId}/roles/${roleId}/identities/${identityId}`);

// Role group assignments
export const listGroupsForRole = (clientDbId: string, roleId: string) =>
  api.get<RoleGroup[]>(`/oauth2/clients/${clientDbId}/roles/${roleId}/groups`);
export const assignGroupToRole = (clientDbId: string, roleId: string, groupId: string) =>
  api.post(`/oauth2/clients/${clientDbId}/roles/${roleId}/groups`, { group_id: groupId });
export const removeGroupFromRole = (clientDbId: string, roleId: string, groupId: string) =>
  api.delete(`/oauth2/clients/${clientDbId}/roles/${roleId}/groups/${groupId}`);

// Client groups CRUD
export const listClientGroups = (clientDbId: string) =>
  api.get<ClientGroup[]>(`/oauth2/clients/${clientDbId}/groups`);
export const createClientGroup = (clientDbId: string, name: string, description: string) =>
  api.post<ClientGroup>(`/oauth2/clients/${clientDbId}/groups`, { name, description });
export const updateClientGroup = (clientDbId: string, groupId: string, name: string, description: string) =>
  api.patch<ClientGroup>(`/oauth2/clients/${clientDbId}/groups/${groupId}`, { name, description });
export const deleteClientGroup = (clientDbId: string, groupId: string) =>
  api.delete(`/oauth2/clients/${clientDbId}/groups/${groupId}`);

// Group member management
export const listGroupMembers = (clientDbId: string, groupId: string) =>
  api.get<RoleMember[]>(`/oauth2/clients/${clientDbId}/groups/${groupId}/members`);
export const addGroupMember = (clientDbId: string, groupId: string, identityId: string) =>
  api.post(`/oauth2/clients/${clientDbId}/groups/${groupId}/members`, { identity_id: identityId });
export const removeGroupMember = (clientDbId: string, groupId: string, identityId: string) =>
  api.delete(`/oauth2/clients/${clientDbId}/groups/${groupId}/members/${identityId}`);

// Group role management
export const listGroupRoles = (clientDbId: string, groupId: string) =>
  api.get<ClientRole[]>(`/oauth2/clients/${clientDbId}/groups/${groupId}/roles`);
export const addRoleToGroup = (clientDbId: string, groupId: string, roleId: string) =>
  api.post(`/oauth2/clients/${clientDbId}/groups/${groupId}/roles`, { role_id: roleId });
export const removeRoleFromGroup = (clientDbId: string, groupId: string, roleId: string) =>
  api.delete(`/oauth2/clients/${clientDbId}/groups/${groupId}/roles/${roleId}`);

// Identity perspective
export const listIdentityClientRoles = (identityId: string) =>
  api.get<ClientRoleAssignment[]>(`/identities/${identityId}/client-roles`);
export const removeIdentityClientRole = (identityId: string, roleId: string) =>
  api.delete(`/identities/${identityId}/client-roles/${roleId}`);
export const listIdentityClientGroups = (identityId: string) =>
  api.get<ClientGroupMembership[]>(`/identities/${identityId}/client-groups`);
export const removeIdentityClientGroup = (identityId: string, groupId: string) =>
  api.delete(`/identities/${identityId}/client-groups/${groupId}`);

export interface AuditLog {
  id: string;
  actor_id: string | null;
  actor_email: string | null;
  action: string;
  resource_type: string;
  resource_id: string | null;
  details: Record<string, unknown> | null;
  created_at: string;
}

export interface AuditLogsResponse {
  total: number;
  logs: AuditLog[];
}

export interface AuditLogsFilter {
  resource_type?: string;
  action?: string;
  actor_id?: string;
  since?: string;
  until?: string;
  limit?: number;
  offset?: number;
}

export const listAuditLogs = (filter: AuditLogsFilter = {}) => {
  const params = new URLSearchParams();
  if (filter.resource_type) params.set('resource_type', filter.resource_type);
  if (filter.action) params.set('action', filter.action);
  if (filter.actor_id) params.set('actor_id', filter.actor_id);
  if (filter.since) params.set('since', filter.since);
  if (filter.until) params.set('until', filter.until);
  if (filter.limit !== undefined) params.set('limit', String(filter.limit));
  if (filter.offset !== undefined) params.set('offset', String(filter.offset));
  const qs = params.toString();
  return api.get<AuditLogsResponse>(`/audit-logs${qs ? `?${qs}` : ''}`);
};

export const listAuditResourceTypes = () => api.get<string[]>('/audit-logs/resource-types');
