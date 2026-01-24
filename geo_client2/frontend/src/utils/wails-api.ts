/**
 * Wails API bridge - replaces electronAPI
 * Maps electronAPI calls to Wails App methods
 */

import { Greet, EmitTestEvent } from '@/wailsjs/go/main/App';
import { EventsOn } from '@/wailsjs/runtime/runtime';

// Get the Wails App instance (will be set when Wails loads)
declare global {
  interface Window {
    go?: {
      main?: { App: WailsApp };
      geo_client2?: { App: WailsApp };
      backend?: { App: WailsApp };
    };
    runtime?: {
      EventsOn: (event: string, callback: (...args: unknown[]) => void) => () => void;
      EventsEmit: (event: string, ...args: unknown[]) => void;
    };
  }
}

interface WailsApp {
  // Auth
  Login(username: string, password: string, apiBaseURL: string): Promise<{ success: boolean; token?: string; expires_at?: string; error?: string }>;

  // Account (replaces Auth)
  CreateAccount(platform: string, accountName: string): Promise<{ success: boolean; account: Account }>;
  ListAccounts(platform: string): Promise<{ success: boolean; accounts: Account[] }>;
  GetActiveAccount(platform: string): Promise<{ success: boolean; account: Account | null }>;
  SetActiveAccount(platform: string, accountID: string): Promise<void>;
  DeleteAccount(accountID: string): Promise<void>;
  UpdateAccountName(accountID: string, name: string): Promise<void>;
  StartLogin(platform: string, accountID: string): Promise<void>;
  StopLogin(): Promise<void>;

  // Settings
  GetSetting(key: string): Promise<string>;
  SetSetting(key: string, value: string): Promise<void>;

  // Task
  CreateLocalSearchTask(keywordsJSON: string, platformsJSON: string, queryCount: number): Promise<{ success: boolean; taskId: number }>;
  GetAllTasks(limit: number, offset: number, filtersJSON: string): Promise<{ success: boolean; tasks: unknown[] }>;
  GetTaskDetail(taskID: number): Promise<{ success: boolean; data: unknown }>;
  CancelTask(taskID: number): Promise<void>;
  RetryTask(taskID: number): Promise<void>;
  ContinueTask(taskID: number): Promise<void>;
  DeleteTask(taskID: number): Promise<void>;
  UpdateTaskName(taskID: number, name: string): Promise<void>;
  GetSearchRecords(taskID: number): Promise<{ success: boolean; records: any[] }>;
  GetMergedSearchRecords(taskIDsJSON: string): Promise<{ success: boolean; records: any[] }>;
  GetStats(): Promise<unknown>;

  // Search
  CheckLoginStatus(platform: string): Promise<{ success: boolean; isLoggedIn: boolean }>;

  // Logs
  GetLogs(limit: number, offset: number, filtersJSON: string): Promise<{ success: boolean; logs: LogEntry[]; total: number }>;
  AddLog(level: string, source: string, message: string, detailsJSON: string, sessionID: string, correlationID: string, component: string, userAction: string, performanceMS: number | null, taskID: number | null): Promise<void>;
  ClearOldLogs(daysToKeep: number): Promise<{ success: boolean; deleted: number }>;
  DeleteAllLogs(): Promise<{ success: boolean; deleted: number }>;

  // Version
  GetVersionInfo(): Promise<{ version: string; buildTime: string }>;

  // Test
  Greet(name: string): Promise<string>;
  EmitTestEvent(): Promise<void>;
}

interface LogEntry {
  id: number;
  level: string;
  source: string;
  message: string;
  details?: string;
  task_id?: number;
  session_id?: string;
  correlation_id?: string;
  component?: string;
  user_action?: string;
  performance_ms?: number;
  timestamp: string;
}

interface Account {
  id: number;
  platform: string;
  account_id: string;
  account_name: string;
  user_data_dir: string;
  is_active: boolean;
  category: string;
  created_at: string;
  updated_at: string;
}

function getApp(): WailsApp | undefined {
  const go = window.go;
  return go?.main?.App ?? go?.geo_client2?.App ?? go?.backend?.App;
}

// Wails API that mimics electronAPI structure
export const wailsAPI = {
  auth: {
    login: async (username: string, password: string, apiBaseURL: string) => {
      const app = getApp();
      if (!app) throw new Error('Wails backend not available');
      return app.Login(username, password, apiBaseURL);
    },
  },
  account: {
    create: async (platform: string, accountName: string) => {
      const app = getApp();
      if (!app) throw new Error('Wails backend not available');
      return app.CreateAccount(platform, accountName);
    },
    list: async (platform: string) => {
      const app = getApp();
      if (!app) return { success: false, accounts: [] };
      return app.ListAccounts(platform);
    },
    getActive: async (platform: string) => {
      const app = getApp();
      if (!app) return { success: false, account: null };
      return app.GetActiveAccount(platform);
    },
    setActive: async (platform: string, accountID: string) => {
      const app = getApp();
      if (!app) throw new Error('Wails backend not available');
      return app.SetActiveAccount(platform, accountID);
    },
    delete: async (accountID: string) => {
      const app = getApp();
      if (!app) throw new Error('Wails backend not available');
      return app.DeleteAccount(accountID);
    },
    updateName: async (accountID: string, name: string) => {
      const app = getApp();
      if (!app) throw new Error('Wails backend not available');
      return app.UpdateAccountName(accountID, name);
    },
    startLogin: async (platform: string, accountID: string) => {
      const app = getApp();
      if (!app) throw new Error('Wails backend not available');
      return app.StartLogin(platform, accountID);
    },
    stopLogin: async () => {
      const app = getApp();
      if (!app) throw new Error('Wails backend not available');
      return app.StopLogin();
    },
  },
  search: {
    createTask: async (params: { keywords: string[]; platforms: string[]; queryCount: number }) => {
      const app = getApp();
      if (!app) throw new Error('Wails backend not available');
      const keywordsJSON = JSON.stringify(params.keywords);
      const platformsJSON = JSON.stringify(params.platforms);
      return app.CreateLocalSearchTask(keywordsJSON, platformsJSON, params.queryCount);
    },
    checkLoginStatus: async (platform: string) => {
      const app = getApp();
      if (!app) return { success: false, isLoggedIn: false };
      return app.CheckLoginStatus(platform);
    },
    onTaskUpdated: (callback: (data: unknown) => void) => {
      return EventsOn('search:taskUpdated', callback);
    },
  },
  task: {
    createLocalSearchTask: async (config: {
      keywords: string[];
      platforms: string[];
      query_count: number;
    }) => {
      const app = getApp();
      if (!app) throw new Error('Wails backend not available');
      const keywordsJSON = JSON.stringify(config.keywords);
      const platformsJSON = JSON.stringify(config.platforms);
      return app.CreateLocalSearchTask(keywordsJSON, platformsJSON, config.query_count);
    },
    getAllTasks: async (options?: {
      limit?: number;
      offset?: number;
      status?: string;
      platform?: string;
      source?: string;
      taskType?: string;
      createdBy?: string;
    }) => {
      const app = getApp();
      if (!app) return { success: false, tasks: [] };
      const filtersJSON = options ? JSON.stringify(options) : '';
      return app.GetAllTasks(options?.limit ?? 100, options?.offset ?? 0, filtersJSON);
    },
    getTaskDetail: async (taskId: number) => {
      const app = getApp();
      if (!app) return { success: false };
      return app.GetTaskDetail(taskId);
    },
    cancelLocalTask: async (taskId: number) => {
      const app = getApp();
      if (!app) return;
      return app.CancelTask(taskId);
    },
    retryTask: async (taskId: number) => {
      const app = getApp();
      if (!app) return;
      return app.RetryTask(taskId);
    },
    continueTask: async (taskId: number) => {
      const app = getApp();
      if (!app) return;
      return app.ContinueTask(taskId);
    },
    deleteTask: async (taskId: number) => {
      const app = getApp();
      if (!app) throw new Error('Wails backend not available');
      return app.DeleteTask(taskId);
    },
    updateName: async (taskId: number, name: string) => {
      const app = getApp();
      if (!app) throw new Error('Wails backend not available');
      return app.UpdateTaskName(taskId, name);
    },
    getStats: async () => {
      const app = getApp();
      if (!app) return null;
      return app.GetStats();
    },
    getSearchRecords: async (taskId: number) => {
      const app = getApp();
      if (!app) return { success: false, records: [] };
      return app.GetSearchRecords(taskId);
    },
    getMergedSearchRecords: async (taskIds: number[]) => {
      const app = getApp();
      if (!app) return { success: false, records: [] };
      const taskIDsJSON = JSON.stringify(taskIds);
      return app.GetMergedSearchRecords(taskIDsJSON);
    },
  },
  settings: {
    get: async (key: string) => {
      const app = getApp();
      if (!app) return null;
      return app.GetSetting(key);
    },
    set: async (key: string, value: string) => {
      const app = getApp();
      if (!app) return;
      return app.SetSetting(key, value);
    },
  },
  onLoginStatusChanged: (callback: (data: { platform: string; isLoggedIn: boolean }) => void) => {
    return EventsOn('login-status-changed', callback as (...args: unknown[]) => void);
  },
  onTaskLoginRequired: (callback: (data: { platform: string; platformName: string; keyword: string; taskId: number }) => void) => {
    return EventsOn('task-login-required', callback as (...args: unknown[]) => void);
  },
  logs: {
    getAll: async (options?: {
      limit?: number;
      offset?: number;
      level?: string;
      source?: string;
      task_id?: number;
    }) => {
      const app = getApp();
      if (!app) return { success: false, logs: [], total: 0 };
      const filtersJSON = options ? JSON.stringify(options) : '';
      return app.GetLogs(options?.limit ?? 100, options?.offset ?? 0, filtersJSON);
    },
    clearOld: async (daysToKeep: number) => {
      const app = getApp();
      if (!app) return { success: false, deleted: 0 };
      return app.ClearOldLogs(daysToKeep);
    },
    deleteAll: async () => {
      const app = getApp();
      if (!app) return { success: false, deleted: 0 };
      return app.DeleteAllLogs();
    },
  },
  version: {
    get: async () => {
      const app = getApp();
      if (!app) return { version: 'unknown', buildTime: 'unknown' };
      return app.GetVersionInfo();
    },
  },
};

// Export types
export type { Account, LogEntry };
