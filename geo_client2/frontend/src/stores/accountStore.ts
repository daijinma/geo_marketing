import { create } from 'zustand';
import { wailsAPI, type Account } from '@/utils/wails-api';

type PlatformName = 'deepseek' | 'doubao' | 'xiaohongshu';

interface AccountState {
  // Active accounts per platform
  activeAccounts: Record<PlatformName, Account | null>;
  // All accounts per platform
  accountsByPlatform: Record<PlatformName, Account[]>;
  // Loading states
  loading: Record<string, boolean>;
  
  // Actions
  loadAccounts: (platform: PlatformName) => Promise<void>;
  loadActiveAccount: (platform: PlatformName) => Promise<void>;
  createAccount: (platform: PlatformName, accountName: string) => Promise<Account>;
  setActiveAccount: (platform: PlatformName, accountID: string) => Promise<void>;
  deleteAccount: (accountID: string) => Promise<void>;
  updateAccountName: (accountID: string, name: string) => Promise<void>;
  startLogin: (platform: PlatformName, accountID: string) => Promise<void>;
  stopLogin: () => Promise<void>;
  refreshAccounts: (platform: PlatformName) => Promise<void>;
}

export const useAccountStore = create<AccountState>((set, get) => ({
  activeAccounts: {
    deepseek: null,
    doubao: null,
    xiaohongshu: null,
  },
  accountsByPlatform: {
    deepseek: [],
    doubao: [],
    xiaohongshu: [],
  },
  loading: {},

  loadAccounts: async (platform: PlatformName) => {
    set((state) => ({ loading: { ...state.loading, [`${platform}_list`]: true } }));
    try {
      const response = await wailsAPI.account.list(platform);
      if (response.success) {
        set((state) => ({
          accountsByPlatform: {
            ...state.accountsByPlatform,
            [platform]: response.accounts,
          },
        }));
      }
    } catch (error) {
      console.error(`Failed to load accounts for ${platform}:`, error);
    } finally {
      set((state) => ({ loading: { ...state.loading, [`${platform}_list`]: false } }));
    }
  },

  loadActiveAccount: async (platform: PlatformName) => {
    set((state) => ({ loading: { ...state.loading, [`${platform}_active`]: true } }));
    try {
      const response = await wailsAPI.account.getActive(platform);
      if (response.success) {
        set((state) => ({
          activeAccounts: {
            ...state.activeAccounts,
            [platform]: response.account,
          },
        }));
      }
    } catch (error) {
      console.error(`Failed to load active account for ${platform}:`, error);
    } finally {
      set((state) => ({ loading: { ...state.loading, [`${platform}_active`]: false } }));
    }
  },

  createAccount: async (platform: PlatformName, accountName: string) => {
    set((state) => ({ loading: { ...state.loading, [`${platform}_create`]: true } }));
    try {
      const response = await wailsAPI.account.create(platform, accountName);
      if (response.success && response.account) {
        // Refresh accounts list
        await get().loadAccounts(platform);
        // Refresh active account
        await get().loadActiveAccount(platform);
        return response.account;
      }
      throw new Error('Failed to create account');
    } catch (error) {
      console.error(`Failed to create account for ${platform}:`, error);
      throw error;
    } finally {
      set((state) => ({ loading: { ...state.loading, [`${platform}_create`]: false } }));
    }
  },

  setActiveAccount: async (platform: PlatformName, accountID: string) => {
    set((state) => ({ loading: { ...state.loading, [`${platform}_setActive`]: true } }));
    try {
      await wailsAPI.account.setActive(platform, accountID);
      // Refresh accounts and active account
      await get().loadAccounts(platform);
      await get().loadActiveAccount(platform);
    } catch (error) {
      console.error(`Failed to set active account for ${platform}:`, error);
      throw error;
    } finally {
      set((state) => ({ loading: { ...state.loading, [`${platform}_setActive`]: false } }));
    }
  },

  deleteAccount: async (accountID: string) => {
    set((state) => ({ loading: { ...state.loading, [`delete_${accountID}`]: true } }));
    try {
      await wailsAPI.account.delete(accountID);
      // Find which platform this account belongs to and refresh
      const state = get();
      for (const platform of ['deepseek', 'doubao', 'xiaohongshu'] as PlatformName[]) {
        const account = state.accountsByPlatform[platform].find((a) => a.account_id === accountID);
        if (account) {
          await get().loadAccounts(platform);
          await get().loadActiveAccount(platform);
          break;
        }
      }
    } catch (error) {
      console.error(`Failed to delete account ${accountID}:`, error);
      throw error;
    } finally {
      set((state) => ({ loading: { ...state.loading, [`delete_${accountID}`]: false } }));
    }
  },

  updateAccountName: async (accountID: string, name: string) => {
    set((state) => ({ loading: { ...state.loading, [`update_${accountID}`]: true } }));
    try {
      await wailsAPI.account.updateName(accountID, name);
      // Refresh accounts
      const state = get();
      for (const platform of ['deepseek', 'doubao', 'xiaohongshu'] as PlatformName[]) {
        const account = state.accountsByPlatform[platform].find((a) => a.account_id === accountID);
        if (account) {
          await get().loadAccounts(platform);
          break;
        }
      }
    } catch (error) {
      console.error(`Failed to update account name ${accountID}:`, error);
      throw error;
    } finally {
      set((state) => ({ loading: { ...state.loading, [`update_${accountID}`]: false } }));
    }
  },

  startLogin: async (platform: PlatformName, accountID: string) => {
    set((state) => ({ loading: { ...state.loading, [`login_${accountID}`]: true } }));
    try {
      await wailsAPI.account.startLogin(platform, accountID);
    } catch (error) {
      console.error(`Failed to start login for ${accountID}:`, error);
      throw error;
    } finally {
      set((state) => ({ loading: { ...state.loading, [`login_${accountID}`]: false } }));
    }
  },

  stopLogin: async () => {
    try {
      await wailsAPI.account.stopLogin();
    } catch (error) {
      console.error('Failed to stop login:', error);
      throw error;
    }
  },

  refreshAccounts: async (platform: PlatformName) => {
    await Promise.all([
      get().loadAccounts(platform),
      get().loadActiveAccount(platform),
    ]);
  },
}));
