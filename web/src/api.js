import axios from 'axios';

const api = axios.create({
  baseURL: 'http://localhost:8080/api',
});

export const getIdentities = () => api.get('/identities');
export const registerIdentity = (data) => api.post('/register', data);
