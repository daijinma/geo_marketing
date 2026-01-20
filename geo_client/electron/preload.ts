import { contextBridge, ipcRenderer } from 'electron';

// 暴露受保护的方法给渲染进程
contextBridge.exposeInMainWorld('electronAPI', {
  // 认证相关
  auth: {
    login: (username: string, password: string, apiBaseUrl: string) =>
      ipcRenderer.invoke('auth:login', username, password, apiBaseUrl),
    getToken: () => ipcRenderer.invoke('auth:get-token'),
    logout: () => ipcRenderer.invoke('auth:logout'),
    checkTokenValid: () => ipcRenderer.invoke('auth:check-token-valid'),
  },
  
  // 搜索相关
  search: {
    createTask: (params: { keywords: string[]; platforms: string[]; queryCount: number }) =>
      ipcRenderer.invoke('search:createTask', params),
    checkLoginStatus: (platform: string) =>
      ipcRenderer.invoke('search:checkLoginStatus', platform),
    getTasks: () => ipcRenderer.invoke('search:getTasks'),
    cancelTask: (taskId: string) => ipcRenderer.invoke('search:cancelTask', taskId),
    onTaskUpdated: (callback: (data: any) => void) => {
      ipcRenderer.on('search:taskUpdated', (_event, data) => callback(data));
    },
  },

  // 任务管理相关
  task: {
    getAuthorizedPlatforms: () => ipcRenderer.invoke('task:getAuthorizedPlatforms'),
    saveToLocal: (keywords: string[], platforms: string[], queryCount: number, status: string, resultData?: any) =>
      ipcRenderer.invoke('task:saveToLocal', keywords, platforms, queryCount, status, resultData),
    submitToServer: (taskId: string, apiBaseUrl: string, token: string) =>
      ipcRenderer.invoke('task:submitToServer', taskId, apiBaseUrl, token),
  },
  
  // Provider 相关
  provider: {
    openLogin: (platform: string) => ipcRenderer.invoke('provider:openLogin', platform),
    checkLoginAfterAuth: (platform: string) => ipcRenderer.invoke('provider:checkLoginAfterAuth', platform),
    closeLoginView: () => ipcRenderer.invoke('provider:closeLoginView'),
  },
  
  // BrowserView 相关
  browserView: {
    show: (url: string) => ipcRenderer.invoke('browser-view:show', url),
    hide: () => ipcRenderer.invoke('browser-view:hide'),
  },
});
