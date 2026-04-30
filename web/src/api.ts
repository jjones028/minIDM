import axios from 'axios';

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

const api = axios.create({
  baseURL: 'http://localhost:8080/api',
});

export const getIdentities = () => api.get<Identity[]>('/identities');
export const registerIdentity = (data: RegisterIdentityData) => api.post('/register', data);
