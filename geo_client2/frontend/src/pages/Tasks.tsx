import { useState, useEffect, useRef } from 'react';
import { RefreshCw, ChevronDown, ChevronRight, Play, Trash2, MoreHorizontal } from 'lucide-react';
import { toast } from 'sonner';
import { LocalTaskCreator } from '@/components/LocalTaskCreator';
import { LocalTaskDetail } from '@/components/LocalTaskDetail';
import { wailsAPI } from '@/utils/wails-api';

interface Task {
  id: number;
  task_id?: number;
  name?: string;
  keywords: string;
  platforms: string;
  query_count: number;
  status: string;
  task_type: string;
  source: string;
  created_by: string | null;
  created_at: string;
}

export default function Tasks() {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(true);
  const [expandedTask, setExpandedTask] = useState<number | null>(null);
  const [selectedTaskId, setSelectedTaskId] = useState<number | null>(null);
  const [showCreator, setShowCreator] = useState(false);
  const [statusFilter, setStatusFilter] = useState('all');
  const [deletingTaskId, setDeletingTaskId] = useState<number | null>(null);
  const [deleteConfirmationId, setDeleteConfirmationId] = useState<number | null>(null);
  
  const [editingTaskId, setEditingTaskId] = useState<number | null>(null);
  const [editValue, setEditValue] = useState('');
  const editInputRef = useRef<HTMLInputElement>(null);

  const loadTasks = async () => {
    setLoading(true);
    try {
      const response = await wailsAPI.task.getAllTasks({
        limit: 100,
        offset: 0,
        status: statusFilter !== 'all' ? statusFilter : undefined,
      });
      if (response?.success && response.tasks) {
        setTasks(response.tasks as Task[]);
      }
    } catch (error: any) {
      toast.error('加载任务失败', { description: error.message });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadTasks();
    
    const unlisten = wailsAPI.search.onTaskUpdated((data: any) => {
      loadTasks();
    });
    
    return () => {
      if (typeof unlisten === 'function') unlisten();
    };
  }, [statusFilter]);

  const handleCancel = async (taskId: number) => {
    try {
      await wailsAPI.task.cancelLocalTask(taskId);
      toast.success('任务已取消');
      loadTasks();
    } catch (error: any) {
      toast.error('取消任务失败', { description: error.message });
    }
  };

  const handleRetry = async (taskId: number) => {
    try {
      await wailsAPI.task.retryTask(taskId);
      toast.success('任务已重新开始');
      loadTasks();
    } catch (error: any) {
      toast.error('重试任务失败', { description: error.message });
    }
  };

  const handleContinue = async (taskId: number) => {
    try {
      await wailsAPI.task.continueTask(taskId);
      toast.success('任务已继续执行');
      loadTasks();
    } catch (error: any) {
      toast.error('继续任务失败', { description: error.message });
    }
  };

  const handleDelete = async (taskId: number) => {
    setDeleteConfirmationId(taskId);
  };

  const confirmDelete = async () => {
    if (deleteConfirmationId === null) return;
    const taskId = deleteConfirmationId;
    setDeleteConfirmationId(null);
    setDeletingTaskId(taskId);
    
    try {
      await wailsAPI.task.deleteTask(taskId);
      toast.success('任务已删除');
      loadTasks();
    } catch (error: any) {
      toast.error('删除任务失败', { description: error.message });
    } finally {
      setDeletingTaskId(null);
    }
  };

  const handleDoubleClick = (task: Task) => {
    setEditingTaskId(task.id);
    setEditValue(task.name || `任务 #${task.id}`);
    setTimeout(() => {
      editInputRef.current?.focus();
    }, 0);
  };

  const handleSaveName = async () => {
    if (editingTaskId === null) return;
    
    try {
      await wailsAPI.task.updateName(editingTaskId, editValue);
      toast.success('任务名称已更新');
      loadTasks();
    } catch (error: any) {
      toast.error('更新失败', { description: error.message });
    } finally {
      setEditingTaskId(null);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleSaveName();
    } else if (e.key === 'Escape') {
      setEditingTaskId(null);
    }
  };

  return (
    <div className="p-8 space-y-8 max-w-6xl mx-auto">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">任务列表</h2>
          <p className="text-muted-foreground mt-1">管理和监控您的所有搜索任务</p>
        </div>
        <div className="flex items-center gap-3">
            <button 
                onClick={() => setShowCreator(true)}
                className="px-4 py-2 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 shadow-sm transition-all font-medium text-sm"
            >
                创建任务
            </button>
            <button 
                onClick={loadTasks}
                className="p-2 border border-input rounded-lg hover:bg-accent hover:text-accent-foreground transition-colors"
                title="刷新列表"
            >
                <RefreshCw className="h-4 w-4" />
            </button>
        </div>
      </div>

      <div className="flex items-center gap-2">
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
          className="h-10 w-[180px] rounded-lg border border-input bg-background px-3 py-2 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/20 cursor-pointer"
        >
          <option value="all">全部状态</option>
          <option value="pending">待执行</option>
          <option value="running">运行中</option>
          <option value="completed">已完成</option>
          <option value="failed">失败</option>
          <option value="partial_completed">部分完成</option>
        </select>
      </div>

      {loading && tasks.length === 0 ? (
        <div className="text-center py-20 text-muted-foreground animate-pulse">加载中...</div>
      ) : tasks.length === 0 ? (
        <div className="text-center py-20 border-2 border-dashed border-border/50 rounded-xl bg-card/30">
            <p className="text-muted-foreground text-lg mb-2">暂无任务</p>
            <button 
                onClick={() => setShowCreator(true)}
                className="text-primary hover:underline font-medium"
            >
                立即创建第一个任务
            </button>
        </div>
      ) : (
        <div className="space-y-4">
          {tasks.map((task) => (
            <div key={task.id} className="relative group bg-card border border-border/50 rounded-xl shadow-sm hover:shadow-md transition-all duration-200 overflow-hidden">
              {deletingTaskId === task.id && (
                <div className="absolute inset-0 bg-background/80 flex items-center justify-center z-10 backdrop-blur-sm">
                  <div className="flex items-center gap-2 text-muted-foreground">
                    <RefreshCw className="w-4 h-4 animate-spin" />
                    <span>正在删除...</span>
                  </div>
                </div>
              )}
              
              <div className="p-5">
                <div className="flex items-center justify-between gap-6">
                  <div className="flex items-center gap-4 flex-1 min-w-0">
                    <button 
                        className="h-8 w-8 shrink-0 flex items-center justify-center rounded-lg hover:bg-accent text-muted-foreground hover:text-foreground transition-colors"
                        onClick={() => setExpandedTask(expandedTask === task.id ? null : task.id)}
                    >
                        {expandedTask === task.id ? (
                            <ChevronDown className="h-4 w-4" />
                        ) : (
                            <ChevronRight className="h-4 w-4" />
                        )}
                    </button>
                    
                    <div className="flex flex-col min-w-0 space-y-1">
                        <div className="flex items-center gap-3">
                             {editingTaskId === task.id ? (
                                <input
                                    ref={editInputRef}
                                    type="text"
                                    value={editValue}
                                    onChange={(e) => setEditValue(e.target.value)}
                                    onBlur={handleSaveName}
                                    onKeyDown={handleKeyDown}
                                    className="h-6 py-0 px-2 text-sm w-[240px] border border-primary rounded bg-background focus:outline-none focus:ring-2 focus:ring-primary/20"
                                    onClick={(e) => e.stopPropagation()}
                                />
                            ) : (
                                <span 
                                    onDoubleClick={() => handleDoubleClick(task)}
                                    className="font-semibold text-lg truncate cursor-text hover:text-primary transition-colors select-none"
                                    title={task.name || `任务 #${task.id}`}
                                >
                                    {task.name || `任务 #${task.id}`}
                                </span>
                            )}
                            
                            <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold transition-colors
                                ${task.status === 'completed' ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400' :
                                  task.status === 'running' ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400' :
                                  task.status === 'failed' ? 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400' :
                                  task.status === 'partial_completed' ? 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400' :
                                  'bg-secondary text-secondary-foreground'
                                }`
                            }>
                                {task.status === 'partial_completed' ? '部分完成' : 
                                 task.status === 'completed' ? '已完成' :
                                 task.status === 'running' ? '运行中' :
                                 task.status === 'failed' ? '失败' :
                                 task.status === 'pending' ? '等待中' :
                                 task.status === 'cancelled' ? '已取消' : task.status}
                            </span>
                        </div>
                        <div className="text-xs text-muted-foreground flex items-center gap-2">
                            <span className="font-mono">ID: {task.id}</span>
                            <span>•</span>
                            <span>{new Date(task.created_at).toLocaleString()}</span>
                        </div>
                    </div>
                  </div>

                  <div className="flex items-center gap-2 shrink-0 opacity-0 group-hover:opacity-100 transition-opacity duration-200">
                    {task.status !== 'running' && (
                      <>
                        {task.status === 'partial_completed' && (
                          <button 
                            onClick={() => handleContinue(task.id)}
                            className="h-9 px-3 text-sm font-medium border border-input rounded-lg hover:bg-accent hover:text-accent-foreground flex items-center gap-1.5 transition-colors"
                          >
                            <Play className="h-3.5 w-3.5" />
                            继续
                          </button>
                        )}
                        <button 
                            onClick={() => handleRetry(task.id)}
                            className="h-9 px-3 text-sm font-medium border border-input rounded-lg hover:bg-accent hover:text-accent-foreground flex items-center gap-1.5 transition-colors"
                        >
                            <Play className="h-3.5 w-3.5" />
                            重试
                        </button>
                      </>
                    )}
                    
                    <button 
                        onClick={() => setSelectedTaskId(task.id)}
                        className="h-9 px-4 text-sm font-medium bg-secondary text-secondary-foreground rounded-lg hover:bg-secondary/80 transition-colors"
                    >
                        详情
                    </button>

                    {task.status === 'running' ? (
                      <button 
                        onClick={() => handleCancel(task.id)}
                        className="h-9 px-3 text-sm font-medium border border-destructive/20 text-destructive bg-destructive/5 rounded-lg hover:bg-destructive hover:text-destructive-foreground transition-colors"
                      >
                        取消
                      </button>
                    ) : (
                      <button
                        onClick={() => handleDelete(task.id)}
                        className="h-9 w-9 flex items-center justify-center rounded-lg text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-colors"
                        title="删除任务"
                      >
                        <Trash2 className="h-4 w-4" />
                      </button>
                    )}
                  </div>
                </div>

                {expandedTask === task.id && (
                  <div className="mt-5 pt-5 border-t border-border/50 grid grid-cols-1 md:grid-cols-2 gap-6 text-sm animate-in slide-in-from-top-2 duration-200">
                    <div>
                        <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wider block mb-2">关键词</span>
                        <div className="font-mono bg-secondary/50 p-3 rounded-lg text-xs overflow-x-auto border border-border/50">
                            {task.keywords}
                        </div>
                    </div>
                    <div className="space-y-3">
                        <div>
                            <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wider block mb-1">目标平台</span>
                            <div className="font-medium">{task.platforms}</div>
                        </div>
                        <div>
                            <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wider block mb-1">查询设置</span>
                            <div className="font-medium">深度: {task.query_count} 轮</div>
                        </div>
                    </div>
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      {showCreator && (
        <LocalTaskCreator
          onClose={() => setShowCreator(false)}
          onCreated={() => {
            setShowCreator(false);
            loadTasks();
          }}
        />
      )}

      {selectedTaskId && (
        <LocalTaskDetail
          taskId={selectedTaskId}
          onClose={() => setSelectedTaskId(null)}
        />
      )}

      {deleteConfirmationId && (
        <div className="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4 backdrop-blur-sm">
          <div className="bg-background border border-border rounded-xl shadow-lg max-w-sm w-full overflow-hidden">
            <div className="p-6 space-y-4">
                <div className="space-y-2 text-center">
                    <h3 className="text-lg font-semibold">确认删除任务?</h3>
                    <p className="text-muted-foreground text-sm">
                        此操作将永久删除任务及其所有搜索记录，且无法恢复。
                    </p>
                </div>
                <div className="flex justify-center gap-3 pt-2">
                    <button 
                        onClick={() => setDeleteConfirmationId(null)}
                        className="px-4 py-2 border border-input rounded-lg hover:bg-accent font-medium text-sm transition-colors"
                    >
                        取消
                    </button>
                    <button 
                        onClick={confirmDelete}
                        className="px-4 py-2 bg-destructive text-destructive-foreground rounded-lg hover:bg-destructive/90 font-medium text-sm transition-colors"
                    >
                        确认删除
                    </button>
                </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
