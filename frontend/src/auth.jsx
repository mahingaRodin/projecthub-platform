import { createContext, useCallback, useContext, useEffect, useState } from 'react';
import { api } from './api.js';

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!localStorage.getItem('token')) {
      setLoading(false);
      return;
    }
    api('/auth/me')
      .then((d) => setUser(d.user))
      .catch(() => localStorage.removeItem('token'))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    const onLogout = () => setUser(null);
    window.addEventListener('auth:logout', onLogout);
    return () => window.removeEventListener('auth:logout', onLogout);
  }, []);

  const startSession = (d) => {
    localStorage.setItem('token', d.token);
    setUser(d.user);
  };

  const login = useCallback(async (email, password) => {
    startSession(await api('/auth/login', { method: 'POST', body: { email, password } }));
  }, []);

  const register = useCallback(async (name, email, password) => {
    startSession(await api('/auth/register', { method: 'POST', body: { name, email, password } }));
  }, []);

  const logout = useCallback(() => {
    localStorage.removeItem('token');
    setUser(null);
  }, []);

  return (
    <AuthContext.Provider value={{ user, loading, login, register, logout }}>{children}</AuthContext.Provider>
  );
}

export const useAuth = () => useContext(AuthContext);

export const can = {
  createProject: (u) => ['admin', 'manager'].includes(u?.role),
  manageUsers: (u) => u?.role === 'admin',
};
