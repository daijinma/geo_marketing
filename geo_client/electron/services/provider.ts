import { ipcMain } from 'electron';
import { ProviderFactory } from '../providers';
import { saveLoginStatus } from '../database';
import { BrowserViewManager } from './browser-view-manager';

/**
 * Provider 服务
 * 负责处理大模型平台的登录授权
 */

/**
 * 初始化 Provider IPC 处理程序
 */
export function initProviderHandlers(): void {
  const browserViewManager = BrowserViewManager.getInstance();

  // 打开登录窗口
  ipcMain.handle('provider:openLogin', async (event, platform: string) => {
    try {
      console.log(`[ProviderService] 打开登录窗口: ${platform}`);
      
      // 获取 provider 实例
      const provider = ProviderFactory.getProvider(platform, false); // 非 headless 模式
      
      // 获取登录 URL
      const loginUrl = provider.getLoginUrl();
      console.log(`[ProviderService] 登录 URL: ${loginUrl}`);
      
      // 显示 BrowserView
      browserViewManager.show(loginUrl);
      
      return { success: true };
    } catch (error: any) {
      console.error('[ProviderService] 打开登录窗口失败:', error);
      return { success: false, error: error.message };
    }
  });

  // 登录后检查状态
  ipcMain.handle('provider:checkLoginAfterAuth', async (event, platform: string) => {
    try {
      console.log(`[ProviderService] 检查登录状态: ${platform}`);
      
      // 获取 provider 实例
      const provider = ProviderFactory.getProvider(platform, false); // 非 headless 模式
      
      // 检查登录状态
      const isLoggedIn = await provider.checkLoginStatus();
      console.log(`[ProviderService] 登录状态: ${isLoggedIn ? '已登录' : '未登录'}`);
      
      // 保存到数据库
      saveLoginStatus('llm', platform, isLoggedIn, new Date().toISOString());
      
      // 隐藏 BrowserView
      browserViewManager.hide();
      
      return { success: true, isLoggedIn };
    } catch (error: any) {
      console.error('[ProviderService] 检查登录状态失败:', error);
      return { success: false, isLoggedIn: false, error: error.message };
    }
  });

  // 关闭登录窗口
  ipcMain.handle('provider:closeLoginView', async () => {
    try {
      console.log('[ProviderService] 关闭登录窗口');
      browserViewManager.hide();
      return { success: true };
    } catch (error: any) {
      console.error('[ProviderService] 关闭登录窗口失败:', error);
      return { success: false, error: error.message };
    }
  });

  console.log('[ProviderService] IPC 处理程序已注册');
}
