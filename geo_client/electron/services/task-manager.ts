import { ipcMain } from 'electron';
import { saveTask, updateTaskStatus, getAllLoginStatus } from '../database';
import { getTaskQueue, Task, TaskStatus } from './task-queue';
import axios from 'axios';

/**
 * 本地任务管理器
 * 负责任务的创建、执行和同步
 */

/**
 * 提交任务结果到服务器
 */
export async function submitTaskToServer(task: Task, apiBaseUrl: string, token: string): Promise<void> {
  try {
    console.log(`[TaskManager] 开始提交任务 ${task.id} 到服务器`);

    // 构造服务器API请求
    const response = await axios.post(
      `${apiBaseUrl}/tasks/create`,
      {
        keywords: [task.keyword],
        platforms: [task.platform],
        query_count: 1,
      },
      {
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
      }
    );

    if (response.data && response.data.task_id) {
      console.log(`[TaskManager] 任务已提交到服务器，服务器任务ID: ${response.data.task_id}`);
      
      // 更新本地数据库，保存服务器任务ID
      // 注意：这里需要更新数据库表结构来保存本地任务ID和服务器任务ID的映射关系
      // 暂时先打印日志
      console.log(`[TaskManager] 本地任务 ${task.id} 对应服务器任务 ${response.data.task_id}`);
    }
  } catch (error: any) {
    console.error(`[TaskManager] 提交任务到服务器失败:`, error);
    throw error;
  }
}

/**
 * 保存任务到本地数据库
 */
export function saveTaskToLocal(
  keywords: string[],
  platforms: string[],
  queryCount: number,
  status: string,
  resultData?: any
): void {
  try {
    saveTask(
      null,
      keywords.join(', '),
      platforms.join(', '),
      queryCount,
      status,
      resultData ? JSON.stringify(resultData) : null
    );
    console.log('[TaskManager] 任务已保存到本地数据库');
  } catch (error) {
    console.error('[TaskManager] 保存任务到本地数据库失败:', error);
  }
}

/**
 * 获取已授权的平台列表
 */
export function getAuthorizedPlatforms(): Array<{
  platform_name: string;
  platform_type: string;
  is_logged_in: boolean;
}> {
  try {
    const loginStatuses = getAllLoginStatus();
    return loginStatuses
      .filter((status) => status.is_logged_in === 1)
      .map((status) => ({
        platform_name: status.platform_name,
        platform_type: status.platform_type,
        is_logged_in: true,
      }));
  } catch (error) {
    console.error('[TaskManager] 获取已授权平台列表失败:', error);
    return [];
  }
}

/**
 * 初始化任务管理器 IPC 处理程序
 */
export function initTaskManagerHandlers(): void {
  // 获取已授权的平台
  ipcMain.handle('task:getAuthorizedPlatforms', async () => {
    try {
      const platforms = getAuthorizedPlatforms();
      return { success: true, platforms };
    } catch (error: any) {
      console.error('[TaskManager] 获取已授权平台失败:', error);
      return { success: false, error: error.message };
    }
  });

  // 保存任务到本地
  ipcMain.handle(
    'task:saveToLocal',
    async (
      event,
      keywords: string[],
      platforms: string[],
      queryCount: number,
      status: string,
      resultData?: any
    ) => {
      try {
        saveTaskToLocal(keywords, platforms, queryCount, status, resultData);
        return { success: true };
      } catch (error: any) {
        console.error('[TaskManager] 保存任务到本地失败:', error);
        return { success: false, error: error.message };
      }
    }
  );

  // 提交任务到服务器
  ipcMain.handle(
    'task:submitToServer',
    async (event, taskId: string, apiBaseUrl: string, token: string) => {
      try {
        const taskQueue = getTaskQueue();
        const task = taskQueue.getTask(taskId);
        
        if (!task) {
          throw new Error(`任务 ${taskId} 不存在`);
        }

        if (task.status !== TaskStatus.COMPLETED) {
          throw new Error(`任务 ${taskId} 尚未完成，无法提交`);
        }

        await submitTaskToServer(task, apiBaseUrl, token);
        return { success: true };
      } catch (error: any) {
        console.error('[TaskManager] 提交任务到服务器失败:', error);
        return { success: false, error: error.message };
      }
    }
  );

  console.log('[TaskManager] IPC 处理程序已注册');
}
