import { create } from 'zustand';
import { persist } from 'zustand/middleware';

interface AuthState {
  token: string | null;
  expiresAt: number | null;
  username: string | null;
  userId: string | null;
  isAdmin: boolean;
  setToken: (token: string, expiresAt: string, username?: string, userId?: string, isAdmin?: boolean) => void;
  clearToken: () => Promise<void>;
  isTokenValid: () => boolean;
  checkAndRefreshToken: () => Promise<boolean>;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      token: null,
      expiresAt: null,
      username: null,
      userId: null,
      isAdmin: false,

      setToken: (token, expiresAt, username, userId, isAdmin) => {
        const t = new Date(expiresAt).getTime();
        set({
          token,
          expiresAt: t,
          username: username ?? null,
          userId: userId ?? null,
          isAdmin: isAdmin ?? false,
        });
      },

      clearToken: async () => {
        set({
          token: null,
          expiresAt: null,
          username: null,
          userId: null,
          isAdmin: false,
        });
      },

      isTokenValid: () => {
        const { token, expiresAt } = get();
        if (!token || !expiresAt) return false;
        return Date.now() < expiresAt;
      },

      checkAndRefreshToken: async () => {
        if (!get().isTokenValid()) return false;
        return true;
      },
    }),
    { name: 'geo_client2-auth' }
  )
);
