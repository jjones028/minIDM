import { createContext, useContext, useState, useEffect, type ReactNode } from 'react';
import { getMe } from '@/api';

interface AuthContextType {
  checked: boolean;
  authenticated: boolean;
  setAuthenticated: (val: boolean) => void;
}

const AuthContext = createContext<AuthContextType>({
  checked: false,
  authenticated: false,
  setAuthenticated: () => {},
});

export function AuthProvider({ children }: { children: ReactNode }) {
  const [checked, setChecked] = useState(false);
  const [authenticated, setAuthenticatedState] = useState(false);

  useEffect(() => {
    getMe()
      .then(() => { setAuthenticatedState(true); setChecked(true); })
      .catch(() => { setAuthenticatedState(false); setChecked(true); });
  }, []);

  const setAuthenticated = (val: boolean) => {
    setAuthenticatedState(val);
    setChecked(true);
  };

  return (
    <AuthContext.Provider value={{ checked, authenticated, setAuthenticated }}>
      {children}
    </AuthContext.Provider>
  );
}

export const useAuth = () => useContext(AuthContext);
