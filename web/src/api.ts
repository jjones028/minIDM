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
  created_at: string;
  updated_at: string;
}

const api = axios.create({ baseURL: '/api' });

export const login = (data: LoginData) => api.post('/login', data);
export const logout = () => api.delete('/session');
export const getMe = () => api.get<{ id: string }>('/me');
export const getIdentities = () => api.get<Identity[]>('/identities');
export const registerIdentity = (data: RegisterIdentityData) => api.post('/register', data);
export const getRoles = () => api.get<Role[]>('/roles');
export const getIdentityRoles = (id: string) => api.get<Role[]>(`/identities/${id}/roles`);
export const assignRole = (identityId: string, roleId: string) =>
  api.post(`/identities/${identityId}/roles`, { role_id: roleId });
export const removeRole = (identityId: string, roleId: string) =>
  api.delete(`/identities/${identityId}/roles/${roleId}`);

export const isUnauthorized = (err: unknown) =>
  (err as AxiosError)?.response?.status === 401;
