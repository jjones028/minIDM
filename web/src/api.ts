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

export interface LoginResult {
  token: string;
  expires_at: string;
}

const TOKEN_KEY = 'minidm_token';

export const getToken = (): string | null => localStorage.getItem(TOKEN_KEY);
export const setToken = (token: string) => localStorage.setItem(TOKEN_KEY, token);
export const clearToken = () => localStorage.removeItem(TOKEN_KEY);

const api = axios.create({
  baseURL: 'http://localhost:8080/api',
});

api.interceptors.request.use((config) => {
  const token = getToken();
  if (token) config.headers.Authorization = `Bearer ${token}`;
  return config;
});

export const login = (data: LoginData) => api.post<LoginResult>('/login', data);
export const logout = () => api.delete('/session');
export const getIdentities = () => api.get<Identity[]>('/identities');
export const registerIdentity = (data: RegisterIdentityData) => api.post('/register', data);

export const isUnauthorized = (err: unknown) =>
  (err as AxiosError)?.response?.status === 401;
