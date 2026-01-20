import { EventEmitter } from 'events';

/**
 * 任务状态
 */
export enum TaskStatus {
  PENDING = 'pending',
  RUNNING = 'running',
  COMPLETED = 'completed',
  FAILED = 'failed',
  CANCELLED = 'cancelled',
}

/**
 * 任务接口
 */
export interface Task {
  id: string;
  keyword: string;
  prompt: string;
  platform: string;
  status: TaskStatus;
  result?: any;
  error?: string;
  createdAt: Date;
  updatedAt: Date;
}

/**
 * 任务队列
 * 使用 EventEmitter 实现任务队列和事件通知
 */
export class TaskQueue extends EventEmitter {
  private tasks: Map<string, Task> = new Map();
  private runningTasks: Set<string> = new Set();
  private maxConcurrent: number = 1; // 默认最多1个并发任务

  constructor(maxConcurrent: number = 1) {
    super();
    this.maxConcurrent = maxConcurrent;
  }

  /**
   * 添加任务
   */
  addTask(task: Omit<Task, 'id' | 'status' | 'createdAt' | 'updatedAt'>): string {
    const id = `task_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    const newTask: Task = {
      ...task,
      id,
      status: TaskStatus.PENDING,
      createdAt: new Date(),
      updatedAt: new Date(),
    };

    this.tasks.set(id, newTask);
    this.emit('task:added', newTask);
    this.processQueue();

    return id;
  }

  /**
   * 获取任务
   */
  getTask(id: string): Task | undefined {
    return this.tasks.get(id);
  }

  /**
   * 获取所有任务
   */
  getAllTasks(): Task[] {
    return Array.from(this.tasks.values());
  }

  /**
   * 获取待处理任务
   */
  getPendingTasks(): Task[] {
    return Array.from(this.tasks.values()).filter(
      (task) => task.status === TaskStatus.PENDING
    );
  }

  /**
   * 获取运行中的任务
   */
  getRunningTasks(): Task[] {
    return Array.from(this.tasks.values()).filter(
      (task) => task.status === TaskStatus.RUNNING
    );
  }

  /**
   * 更新任务状态
   */
  updateTaskStatus(id: string, status: TaskStatus, result?: any, error?: string): void {
    const task = this.tasks.get(id);
    if (!task) {
      return;
    }

    task.status = status;
    task.updatedAt = new Date();
    if (result !== undefined) {
      task.result = result;
    }
    if (error !== undefined) {
      task.error = error;
    }

    this.tasks.set(id, task);
    this.emit('task:updated', task);

    if (status === TaskStatus.COMPLETED || status === TaskStatus.FAILED || status === TaskStatus.CANCELLED) {
      this.runningTasks.delete(id);
      this.processQueue(); // 处理下一个任务
    }
  }

  /**
   * 取消任务
   */
  cancelTask(id: string): boolean {
    const task = this.tasks.get(id);
    if (!task) {
      return false;
    }

    if (task.status === TaskStatus.PENDING) {
      this.updateTaskStatus(id, TaskStatus.CANCELLED);
      return true;
    }

    if (task.status === TaskStatus.RUNNING) {
      // 对于运行中的任务，需要外部处理取消逻辑
      this.emit('task:cancel', task);
      return true;
    }

    return false;
  }

  /**
   * 删除任务
   */
  removeTask(id: string): boolean {
    const task = this.tasks.get(id);
    if (task && (task.status === TaskStatus.COMPLETED || task.status === TaskStatus.FAILED || task.status === TaskStatus.CANCELLED)) {
      this.tasks.delete(id);
      this.runningTasks.delete(id);
      this.emit('task:removed', id);
      return true;
    }
    return false;
  }

  /**
   * 处理队列
   */
  private processQueue(): void {
    // 如果已达到最大并发数，不处理新任务
    if (this.runningTasks.size >= this.maxConcurrent) {
      return;
    }

    // 获取待处理的任务
    const pendingTasks = this.getPendingTasks();
    if (pendingTasks.length === 0) {
      return;
    }

    // 按创建时间排序，先处理最早的任务
    pendingTasks.sort((a, b) => a.createdAt.getTime() - b.createdAt.getTime());

    // 启动新任务
    const taskToRun = pendingTasks[0];
    this.runningTasks.add(taskToRun.id);
    this.updateTaskStatus(taskToRun.id, TaskStatus.RUNNING);
    this.emit('task:start', taskToRun);
  }

  /**
   * 设置最大并发数
   */
  setMaxConcurrent(max: number): void {
    this.maxConcurrent = max;
    this.processQueue(); // 重新处理队列
  }

  /**
   * 获取当前并发数
   */
  getCurrentConcurrent(): number {
    return this.runningTasks.size;
  }

  /**
   * 清空已完成的任务
   */
  clearCompleted(): number {
    const completedTasks = Array.from(this.tasks.values()).filter(
      (task) =>
        task.status === TaskStatus.COMPLETED ||
        task.status === TaskStatus.FAILED ||
        task.status === TaskStatus.CANCELLED
    );

    let count = 0;
    for (const task of completedTasks) {
      if (this.removeTask(task.id)) {
        count++;
      }
    }

    return count;
  }
}

// 导出单例
let taskQueueInstance: TaskQueue | null = null;

export function getTaskQueue(): TaskQueue {
  if (!taskQueueInstance) {
    taskQueueInstance = new TaskQueue(1); // 默认最多1个并发任务
  }
  return taskQueueInstance;
}
