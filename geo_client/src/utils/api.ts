import { invoke } from '@tauri-apps/api/core';
import type { LoginRequest, LoginResponse } from '@/types/auth';
import type { Task } from '@/types/task';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://127.0.0.1:8000';

// 检查是否在Tauri环境中
const isTauri = typeof window !== 'undefined' && '__TAURI_INTERNALS__' in window;

// 获取token（优先从数据库获取，如果没有则从localStorage获取）
async function getToken(): Promise<string | null> {
  if (isTauri) {
    try {
      const tokenInfo = await invoke<{ token: string; expires_at: string; is_valid: boolean }>('get_token');
      if (tokenInfo && tokenInfo.is_valid) {
        return tokenInfo.token;
      }
    } catch (error) {
      console.warn('从数据库获取token失败:', error);
    }
  }
  
  // 降级到localStorage
  const authStorage = localStorage.getItem('auth-storage');
  if (authStorage) {
    try {
      const auth = JSON.parse(authStorage);
      return auth.state?.token || null;
    } catch {
      return null;
    }
  }
  return null;
}

// 检查token是否有效
async function checkTokenValid(): Promise<boolean> {
  if (isTauri) {
    try {
      return await invoke<boolean>('check_token_valid');
    } catch {
      return false;
    }
  }
  
  // 降级到localStorage检查
  const authStorage = localStorage.getItem('auth-storage');
  if (authStorage) {
    try {
      const auth = JSON.parse(authStorage);
      if (auth.state?.expiresAt) {
        return Date.now() < auth.state.expiresAt;
      }
    } catch {
      return false;
    }
  }
  return false;
}

// API请求封装
async function apiRequest<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<T> {
  const token = await getToken();
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string>),
  };

  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const response = await fetch(`${API_BASE_URL}${endpoint}`, {
    ...options,
    headers: headers as HeadersInit,
  });

  if (!response.ok) {
    if (response.status === 401) {
      // Token失效，清除本地token
      if (isTauri) {
        try {
          await invoke('logout');
        } catch (error) {
          console.warn('清除数据库token失败:', error);
        }
      }
      localStorage.removeItem('auth-storage');
      throw new Error('认证失败，请重新登录');
    }
    const error = await response.json().catch(() => ({ message: response.statusText }));
    throw new Error(error.message || `HTTP ${response.status}`);
  }

  return response.json();
}

// 认证API
export const authApi = {
  // 使用Tauri命令登录（会保存到数据库）或直接调用API
  login: async (credentials: LoginRequest): Promise<LoginResponse> => {
    if (isTauri) {
      try {
        const response = await invoke<LoginResponse>('login', {
          username: credentials.username,
          password: credentials.password,
          apiBaseUrl: API_BASE_URL,
        });
        return response;
      } catch (error: any) {
        throw new Error(error || '登录失败');
      }
    }
    
    // 降级到直接API调用
    return apiRequest<LoginResponse>('/client/auth/login', {
      method: 'POST',
      body: JSON.stringify(credentials),
    });
  },
  
  // 检查token是否有效
  checkToken: async (): Promise<boolean> => {
    return checkTokenValid();
  },
  
  // 退出登录
  logout: async (): Promise<void> => {
    if (isTauri) {
      try {
        await invoke('logout');
      } catch (error) {
        console.warn('清除数据库token失败:', error);
      }
    }
    localStorage.removeItem('auth-storage');
  },
};

// 任务API
export const taskApi = {
  getPending: async (limit = 10): Promise<{ tasks: Task[] }> => {
    return apiRequest<{ tasks: Task[] }>(`/client/tasks/pending?limit=${limit}`);
  },
  uploadResults: async (taskId: number, results: any): Promise<{ success: boolean }> => {
    return apiRequest<{ success: boolean }>(`/client/tasks/${taskId}/results`, {
      method: 'POST',
      body: JSON.stringify(results),
    });
  },
  create: async (task: Omit<Task, 'id' | 'task_id'>): Promise<{ task_id: number }> => {
    return apiRequest<{ task_id: number }>('/client/tasks', {
      method: 'POST',
      body: JSON.stringify(task),
    });
  },
};
