import { create } from 'zustand';
import { persist } from 'zustand/middleware';

interface AuthState {
  isAuthenticated: boolean;
  csrfToken: string;
  grantList: string[];
  username: string;
  securityContext: number;
  loading: boolean;
  error: string | null;
  login: (username: string, password: string) => Promise<void>;
  refresh: () => Promise<void>;
  logout: () => Promise<void>;
}

const getCsrfToken = () => localStorage.getItem('csrfToken');

const apiFetch = async (url: string, options: RequestInit = {}) => {
  const headers = new Headers(options.headers);
  const csrfToken = getCsrfToken();
  if (csrfToken) {
    headers.set('X-CSRF-Token', csrfToken);
  }
  return fetch(`/api${url}`, {
    ...options,
    headers,
    credentials: 'include',
  });
};

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      isAuthenticated: false,
      csrfToken: '',
      grantList: [],
      username: '',
      securityContext: 0,
      loading: false,
      error: null,
      login: async (username: string, password: string) => {
        set({ loading: true, error: null });
        try {
          const response = await apiFetch('/auth', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ user_name: username, password }),
          });
          if (!response.ok) throw new Error('Login failed');
          const data = await response.json();
          if (data.user) {
            set({
              isAuthenticated: true,
              csrfToken: data.csrfToken,
              grantList: data.user.grantList || [],
              username: data.user.userName || '',
              securityContext: data.user.securityContext || 0,
            });
            localStorage.setItem('csrfToken', data.csrfToken);
          } else {
            set({ error: 'Invalid credentials' });
          }
        } catch (err: any) {
          set({ error: err.message });
        } finally {
          set({ loading: false });
        }
      },
      refresh: async () => {
        set({ loading: true, error: null });
        try {
          const response = await apiFetch('/refresh');
          if (!response.ok) throw new Error('Refresh failed');
          const data = await response.json();
          if (data.user) {
            set({
              isAuthenticated: true,
              csrfToken: data.csrfToken,
              grantList: data.user.grantList || [],
              username: data.user.userName || '',
              securityContext: data.user.securityContext || 0,
            });
            localStorage.setItem('csrfToken', data.csrfToken);
          } else {
            set({ isAuthenticated: false });
          }
        } catch (err) {
          set({ isAuthenticated: false, error: err.message });
        } finally {
          set({ loading: false });
        }
      },
      logout: async () => {
        try {
          await apiFetch('/logoff', { method: 'POST' });
        } catch (err) {
          // Ignore logout errors
        }
        set({
          isAuthenticated: false,
          csrfToken: '',
          grantList: [],
          username: '',
          securityContext: 0,
        });
        localStorage.removeItem('csrfToken');
      },
    }),
    {
      name: 'auth-storage',
      partialize: (state) => ({
        isAuthenticated: state.isAuthenticated,
        csrfToken: state.csrfToken,
        grantList: state.grantList,
        username: state.username,
        securityContext: state.securityContext,
      }),
    }
  )
);