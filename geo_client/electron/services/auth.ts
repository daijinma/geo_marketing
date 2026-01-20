import { ipcMain } from 'electron';
import * as db from '../database/index';
// 使用 Node.js 内置的 fetch (Node 18+)
const fetch = global.fetch;

/**
 * 初始化认证相关的 IPC 处理程序
 */
export function initAuthHandlers(): void {
  // 初始化数据库
  db.initDb();

  // 登录
  ipcMain.handle('auth:login', async (event, username: string, password: string, apiBaseUrl: string) => {
    try {
      const loginUrl = `${apiBaseUrl}/client/auth/login`;
      
      const response = await fetch(loginUrl, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          username,
          password,
        }),
      });

      if (!response.ok) {
        if (response.status === 401) {
          throw new Error('登录失败：用户名或密码错误');
        }
        throw new Error(`登录失败: HTTP ${response.status}`);
      }

      const loginResponse = await response.json() as {
        success: boolean;
        token: string;
        expires_at: string;
      };

      if (loginResponse.success) {
        // 保存token到数据库
        db.saveAuthToken(loginResponse.token, loginResponse.expires_at);
        return loginResponse;
      } else {
        throw new Error('登录失败：用户名或密码错误');
      }
    } catch (error: any) {
      throw new Error(error.message || '登录失败');
    }
  });

  // 获取token
  ipcMain.handle('auth:get-token', async () => {
    try {
      const tokenInfo = db.getAuthToken();
      if (!tokenInfo) {
        return null;
      }

      const isValid = !db.isTokenExpired();
      return {
        token: tokenInfo.token,
        expires_at: tokenInfo.expires_at,
        is_valid: isValid,
      };
    } catch (error: any) {
      throw new Error(`获取token失败: ${error.message}`);
    }
  });

  // 退出登录
  ipcMain.handle('auth:logout', async () => {
    try {
      db.deleteAuthToken();
    } catch (error: any) {
      throw new Error(`删除token失败: ${error.message}`);
    }
  });

  // 检查token是否有效
  ipcMain.handle('auth:check-token-valid', async () => {
    try {
      return !db.isTokenExpired();
    } catch (error: any) {
      return false;
    }
  });
}
