import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { authApi } from '@/utils/api';

interface AuthState {
  token: string | null;
  expiresAt: number | null;
  setToken: (token: string, expiresAt: number) => void;
  clearToken: () => Promise<void>;
  isTokenValid: () => boolean;
  checkAndRefreshToken: () => Promise<boolean>;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      token: null,
      expiresAt: null,
      setToken: (token: string, expiresAt: number) => {
        set({ token, expiresAt });
      },
      clearToken: async () => {
        // 同时清除数据库和localStorage
        await authApi.logout();
        set({ token: null, expiresAt: null });
      },
      isTokenValid: () => {
        const { token, expiresAt } = get();
        if (!token || !expiresAt) return false;
        return Date.now() < expiresAt;
      },
      // 检查并刷新token
      checkAndRefreshToken: async () => {
        // 先检查localStorage中的token
        const { token, expiresAt } = get();
        if (token && expiresAt && Date.now() < expiresAt) {
          return true;
        }
        
        // 如果localStorage中的token无效，检查数据库中的token
        try {
          const isValid = await authApi.checkToken();
          if (isValid) {
            // 从数据库同步token到localStorage（如果需要）
            // 这里可以添加从数据库读取token并更新store的逻辑
            return true;
          }
        } catch (error) {
          console.warn('检查token失败:', error);
        }
        
        return false;
      },
    }),
    {
      name: 'auth-storage',
    }
  )
);
