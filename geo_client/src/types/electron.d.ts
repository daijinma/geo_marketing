export interface ElectronAPI {
  auth: {
    login: (username: string, password: string, apiBaseUrl: string) => Promise<{
      success: boolean;
      token: string;
      expires_at: string;
    }>;
    getToken: () => Promise<{
      token: string;
      expires_at: string;
      is_valid: boolean;
    } | null>;
    logout: () => Promise<void>;
    checkTokenValid: () => Promise<boolean>;
  };
  
  search: {
    createTask: (params: {
      keywords: string[];
      platforms: string[];
      queryCount: number;
    }) => Promise<{
      success: boolean;
      message?: string;
      error?: string;
    }>;
    checkLoginStatus: (platform: string) => Promise<{
      success: boolean;
      isLoggedIn?: boolean;
      error?: string;
    }>;
    getTasks: () => Promise<{
      success: boolean;
      tasks?: any[];
      error?: string;
    }>;
    cancelTask: (taskId: string) => Promise<{
      success: boolean;
      error?: string;
    }>;
    onTaskUpdated: (callback: (data: any) => void) => void;
  };

  task: {
    getAuthorizedPlatforms: () => Promise<{
      success: boolean;
      platforms?: Array<{
        platform_name: string;
        platform_type: string;
        is_logged_in: boolean;
      }>;
      error?: string;
    }>;
    saveToLocal: (
      keywords: string[],
      platforms: string[],
      queryCount: number,
      status: string,
      resultData?: any
    ) => Promise<{
      success: boolean;
      error?: string;
    }>;
    submitToServer: (
      taskId: string,
      apiBaseUrl: string,
      token: string
    ) => Promise<{
      success: boolean;
      error?: string;
    }>;
  };

  provider: {
    openLogin: (platform: string) => Promise<{
      success: boolean;
      error?: string;
    }>;
    checkLoginAfterAuth: (platform: string) => Promise<{
      success: boolean;
      isLoggedIn: boolean;
      error?: string;
    }>;
    closeLoginView: () => Promise<{
      success: boolean;
      error?: string;
    }>;
  };
  
  browserView: {
    show: (url: string) => Promise<void>;
    hide: () => Promise<void>;
  };
}

declare global {
  interface Window {
    electronAPI: ElectronAPI;
  }
}
