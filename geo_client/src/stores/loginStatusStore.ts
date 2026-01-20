import { create } from 'zustand';
import type { LoginStatus, PlatformName } from '@/types/provider';

interface LoginStatusState {
  statuses: Record<PlatformName, LoginStatus | null>;
  setStatus: (platform: PlatformName, status: LoginStatus) => void;
  getStatus: (platform: PlatformName) => LoginStatus | null;
  isLoggedIn: (platform: PlatformName) => boolean;
  updateLoginStatus: (platform: PlatformName, isLoggedIn: boolean) => void;
}

export const useLoginStatusStore = create<LoginStatusState>((set, get) => ({
  statuses: {
    deepseek: null,
    doubao: null,
    netease: null,
    cnblogs: null,
  },
  setStatus: (platform, status) =>
    set((state) => ({
      statuses: { ...state.statuses, [platform]: status },
    })),
  getStatus: (platform) => get().statuses[platform] || null,
  isLoggedIn: (platform) => {
    const status = get().statuses[platform];
    return status?.is_logged_in || false;
  },
  updateLoginStatus: (platform, isLoggedIn) =>
    set((state) => ({
      statuses: {
        ...state.statuses,
        [platform]: {
          ...(state.statuses[platform] || {}),
          is_logged_in: isLoggedIn,
          last_check_at: new Date().toISOString(),
        } as LoginStatus,
      },
    })),
}));
