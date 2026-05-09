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

const api = axios.create({ baseURL: '/api' });

export const login = (data: LoginData) => api.post('/login', data);
export const logout = () => api.delete('/session');
export const getMe = () => api.get<{ id: string }>('/me');
export const getIdentities = () => api.get<Identity[]>('/identities');
export const registerIdentity = (data: RegisterIdentityData) => api.post('/register', data);
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
  created_at: string;
  updated_at: string;
}

export interface CreateOAuthClientData {
  name: string;
  description?: string;
  redirect_uris: string[];
  scopes: string[];
}

export interface UpdateOAuthClientData {
  name: string;
  description?: string;
  redirect_uris: string[];
  scopes: string[];
  is_enabled: boolean;
}

export interface CreateOAuthClientResult {
  client: OAuthClient;
  client_secret: string; // plaintext, shown once
}

export const listOAuthClients = () => api.get<OAuthClient[]>('/oauth2/clients');
export const createOAuthClient = (data: CreateOAuthClientData) =>
  api.post<CreateOAuthClientResult>('/oauth2/clients', data);
export const getOAuthClient = (id: string) => api.get<OAuthClient>(`/oauth2/clients/${id}`);
export const updateOAuthClient = (id: string, data: UpdateOAuthClientData) =>
  api.patch<OAuthClient>(`/oauth2/clients/${id}`, data);
export const deleteOAuthClient = (id: string) => api.delete(`/oauth2/clients/${id}`);

export const isUnauthorized = (err: unknown) =>
  (err as AxiosError)?.response?.status === 401;

export interface AuditLog {
  id: string;
  actor_id: string | null;
  action: string;
  resource_type: string;
  resource_id: string | null;
  details: Record<string, unknown> | null;
  created_at: string;
}

export const listAuditLogs = (limit = 100, offset = 0) =>
  api.get<AuditLog[]>(`/audit-logs?limit=${limit}&offset=${offset}`);
