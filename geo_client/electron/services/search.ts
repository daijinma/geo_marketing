import { ipcMain } from 'electron';
import { ProviderFactory, SearchResult } from '../providers';
import { getTaskQueue, TaskStatus } from './task-queue';
import { saveLoginStatus, saveTask } from '../database';

/**
 * 搜索任务参数
 */
export interface SearchTaskParams {
  keywords: string[];
  platforms: string[];
  queryCount: number;
}

/**
 * 搜索任务结果
 */
export interface SearchTaskResult {
  taskId: string;
  status: 'pending' | 'running' | 'completed' | 'failed';
  results: SearchResult[];
  error?: string;
}

/**
 * 初始化搜索服务 IPC 处理程序
 */
export function initSearchHandlers(): void {
  // 检查平台登录状态
  ipcMain.handle('search:checkLoginStatus', async (event, platform: string) => {
    try {
      console.log(`[SearchService] 检查平台登录状态: ${platform}`);
      const provider = ProviderFactory.getProvider(platform, false); // 非headless模式
      const isLoggedIn = await provider.checkLoginStatus();
      
      // 保存到本地数据库
      saveLoginStatus('llm', platform, isLoggedIn, new Date().toISOString());
      
      return { success: true, isLoggedIn };
    } catch (error: any) {
      console.error('[SearchService] 检查登录状态失败:', error);
      return { success: false, error: error.message };
    }
  });

  // 创建搜索任务
  ipcMain.handle('search:createTask', async (event, params: SearchTaskParams) => {
    try {
      console.log('[SearchService] 创建搜索任务:', params);
      const { keywords, platforms, queryCount } = params;

      // 验证参数
      if (!keywords || keywords.length === 0) {
        throw new Error('关键词不能为空');
      }
      if (!platforms || platforms.length === 0) {
        throw new Error('平台不能为空');
      }

      const taskQueue = getTaskQueue();
      const results: SearchResult[] = [];
      const errors: string[] = [];

      // 为每个关键词和平台组合创建任务
      for (const keyword of keywords) {
        for (const platform of platforms) {
          for (let i = 0; i < queryCount; i++) {
            const taskId = taskQueue.addTask({
              keyword,
              prompt: keyword,
              platform,
            });

            // 监听任务开始事件
            taskQueue.once('task:start', async (task) => {
              if (task.id !== taskId) return;

              try {
                console.log(`[SearchService] 开始执行任务 ${taskId}: ${keyword} @ ${platform}`);
                const provider = ProviderFactory.getProvider(platform, true); // headless模式
                const result = await provider.search(keyword, keyword);
                
                results.push(result);
                taskQueue.updateTaskStatus(taskId, TaskStatus.COMPLETED, result);
                
                // 通知前端任务更新
                event.sender.send('search:taskUpdated', {
                  taskId,
                  status: 'completed',
                  result,
                });
              } catch (error: any) {
                console.error(`[SearchService] 任务 ${taskId} 执行失败:`, error);
                errors.push(error.message);
                taskQueue.updateTaskStatus(taskId, TaskStatus.FAILED, null, error.message);
                
                // 通知前端任务失败
                event.sender.send('search:taskUpdated', {
                  taskId,
                  status: 'failed',
                  error: error.message,
                });
              }
            });
          }
        }
      }

      // 保存任务到本地数据库
      saveTask(
        null,
        keywords.join(', '),
        platforms.join(', '),
        queryCount,
        'pending'
      );

      return {
        success: true,
        message: `已创建 ${keywords.length * platforms.length * queryCount} 个搜索任务`,
      };
    } catch (error: any) {
      console.error('[SearchService] 创建搜索任务失败:', error);
      return { success: false, error: error.message };
    }
  });

  // 获取任务列表
  ipcMain.handle('search:getTasks', async () => {
    try {
      const taskQueue = getTaskQueue();
      const tasks = taskQueue.getAllTasks();
      return { success: true, tasks };
    } catch (error: any) {
      console.error('[SearchService] 获取任务列表失败:', error);
      return { success: false, error: error.message };
    }
  });

  // 取消任务
  ipcMain.handle('search:cancelTask', async (event, taskId: string) => {
    try {
      const taskQueue = getTaskQueue();
      const success = taskQueue.cancelTask(taskId);
      return { success };
    } catch (error: any) {
      console.error('[SearchService] 取消任务失败:', error);
      return { success: false, error: error.message };
    }
  });

  console.log('[SearchService] IPC 处理程序已注册');
}
