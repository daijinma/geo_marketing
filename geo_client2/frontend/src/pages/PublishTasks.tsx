import { useEffect, useRef, useState } from 'react';
import { Play, RefreshCw, Trash2 } from 'lucide-react';
import { toast } from 'sonner';
import { PublishTaskDetailModal } from '@/components/PublishTaskDetailModal';
import { PublishTaskCreatorModal } from '@/components/PublishTaskCreatorModal';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { wailsAPI } from '@/utils/wails-api';

interface PublishLongTask {
  task_id: string;
  status: string;
  article?: { title: string; content: string; cover_image?: string };
  platforms?: string[];
  created_at?: string;
  updated_at?: string;
}

export default function PublishTasks() {
  const [publishTasks, setPublishTasks] = useState<PublishLongTask[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreator, setShowCreator] = useState(false);
  const [selectedPublishTaskId, setSelectedPublishTaskId] = useState<string | null>(null);
  const [publishDeleteConfirmationId, setPublishDeleteConfirmationId] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState('all');
  const [publishLimit, setPublishLimit] = useState(50);
  const [hasMorePublish, setHasMorePublish] = useState(false);
  const refreshInterval = useRef<ReturnType<typeof setInterval> | null>(null);

  const loadTasks = async (opts?: { silent?: boolean }) => {
    const silent = opts?.silent === true;
    if (!silent) setLoading(true);
    try {
      const publishRes = await wailsAPI.longTask.listRecords();
      if (publishRes?.success && (publishRes as any).tasks) {
        const list = ((publishRes as any).tasks as PublishLongTask[]) || [];
        const filtered = statusFilter === 'all'
          ? list
          : list.filter(t => t.status === statusFilter);
        setPublishTasks(filtered);
        setHasMorePublish(filtered.length > publishLimit);
      }
    } catch (error: any) {
      toast.error('加载任务失败', { description: error.message });
    } finally {
      if (!silent) setLoading(false);
    }
  };

  useEffect(() => {
    loadTasks();
  }, [statusFilter, publishLimit]);

  useEffect(() => {
    refreshInterval.current = setInterval(() => loadTasks({ silent: true }), 5000);
    return () => {
      if (refreshInterval.current) clearInterval(refreshInterval.current);
    };
  }, [statusFilter]);

  const handlePublishRerun = async (taskId: string) => {
    try {
      await wailsAPI.longTask.restart(taskId);
      toast.success('任务已重新开始');
      setSelectedPublishTaskId(taskId);
      loadTasks();
    } catch (error: any) {
      toast.error('重试失败', { description: error.message });
    }
  };

  const handlePublishCancel = async (taskId: string) => {
    try {
      await wailsAPI.longTask.cancel(taskId);
      toast.success('任务已取消');
      loadTasks();
    } catch (error: any) {
      toast.error('取消任务失败', { description: error.message });
    }
  };

  const handlePublishContinue = async (taskId: string) => {
    try {
      await wailsAPI.longTask.resume(taskId);
      toast.success('任务已继续执行');
      loadTasks();
    } catch (error: any) {
      toast.error('继续任务失败', { description: error.message });
    }
  };

  const handlePublishDelete = async (taskId: string) => {
    try {
      await wailsAPI.longTask.remove(taskId);
      toast.success('发文任务已删除');
      loadTasks();
    } catch (error: any) {
      toast.error('删除失败', { description: error.message });
    }
  };

  const confirmPublishDelete = async () => {
    if (!publishDeleteConfirmationId) return;
    const taskId = publishDeleteConfirmationId;
    setPublishDeleteConfirmationId(null);
    await handlePublishDelete(taskId);
  };

  return (
    <div className="p-8 space-y-8 max-w-6xl mx-auto">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">发文任务</h2>
          <p className="text-muted-foreground mt-1">管理和监控您的发文任务</p>
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={() => loadTasks()}
            className="h-10 w-10 flex items-center justify-center rounded-lg border border-input text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
            title="刷新"
          >
            <RefreshCw className="w-4 h-4" />
          </button>
          <button
            onClick={() => setShowCreator(true)}
            className="px-4 py-2 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 shadow-sm transition-all font-medium text-sm"
          >
            创建任务
          </button>
        </div>
      </div>

      <div className="flex items-center justify-between gap-3 flex-wrap">
        <Select value={statusFilter} onValueChange={setStatusFilter}>
          <SelectTrigger className="w-[180px] bg-background">
            <SelectValue placeholder="全部状态" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部状态</SelectItem>
            <SelectItem value="pending">待执行</SelectItem>
            <SelectItem value="running">运行中</SelectItem>
            <SelectItem value="completed">已完成</SelectItem>
            <SelectItem value="failed">失败</SelectItem>
            <SelectItem value="paused">已暂停</SelectItem>
            <SelectItem value="cancelled">已取消</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div className="bg-card border border-border/50 rounded-xl shadow-sm overflow-hidden">
        {loading && publishTasks.length === 0 ? (
          <div className="text-center py-20 text-muted-foreground animate-pulse">加载中...</div>
        ) : publishTasks.length === 0 ? (
          <div className="px-5 py-10 text-sm text-muted-foreground">暂无发文任务</div>
        ) : (
          <div className="space-y-4 p-5">
            {publishTasks.slice(0, publishLimit).map((t) => (
              <div key={t.task_id} className="relative group bg-card border border-border/50 rounded-xl shadow-sm hover:shadow-sm transition-all duration-200 overflow-hidden">
                <div className="p-5">
                  <div className="flex items-center justify-between gap-6">
                    <div className="flex items-center gap-4 flex-1 min-w-0">
                      <div className="flex flex-col min-w-0 space-y-1">
                        <div className="flex items-center gap-3">
                          <span
                            className="font-semibold text-lg truncate cursor-pointer hover:text-primary transition-colors select-none"
                            title={t.article?.title || t.task_id}
                            onClick={() => setSelectedPublishTaskId(t.task_id)}
                          >
                            {t.article?.title || '（无标题）'}
                          </span>

                          <span
                            className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold transition-colors
                              ${t.status === 'completed' ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400' :
                                t.status === 'running' ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400' :
                                t.status === 'failed' ? 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400' :
                                t.status === 'paused' ? 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400' :
                                'bg-secondary text-secondary-foreground'}
                            `}
                          >
                            {t.status === 'completed' ? '已完成' :
                              t.status === 'running' ? '运行中' :
                              t.status === 'failed' ? '失败' :
                              t.status === 'paused' ? '已暂停' :
                              t.status === 'pending' ? '等待中' :
                              t.status === 'cancelled' ? '已取消' : t.status}
                          </span>
                        </div>

                        <div className="text-xs text-muted-foreground flex items-center gap-2">
                          <span className="font-mono">ID: {t.task_id}</span>
                          {t.platforms?.length ? (
                            <>
                              <span>•</span>
                              <span className="truncate">{t.platforms.join(', ')}</span>
                            </>
                          ) : null}
                        </div>
                      </div>
                    </div>

                    <div className="flex items-center gap-2 shrink-0 opacity-0 group-hover:opacity-100 transition-opacity duration-200">
                      {t.status === 'paused' && (
                        <button
                          onClick={() => handlePublishContinue(t.task_id)}
                          className="h-9 px-3 text-sm font-medium border border-input rounded-lg hover:bg-accent hover:text-accent-foreground flex items-center gap-1.5 transition-colors"
                        >
                          <Play className="h-3.5 w-3.5" />
                          继续
                        </button>
                      )}

                      {t.status !== 'running' && t.status !== 'paused' && (
                        <button
                          onClick={() => handlePublishRerun(t.task_id)}
                          className="h-9 px-3 text-sm font-medium border border-input rounded-lg hover:bg-accent hover:text-accent-foreground flex items-center gap-1.5 transition-colors"
                        >
                          <Play className="h-3.5 w-3.5" />
                          重试
                        </button>
                      )}

                      <button
                        onClick={() => setSelectedPublishTaskId(t.task_id)}
                        className="h-9 px-4 text-sm font-medium bg-secondary text-secondary-foreground rounded-lg hover:bg-secondary/80 transition-colors"
                      >
                        详情
                      </button>

                      {t.status === 'running' ? (
                        <button
                          onClick={() => handlePublishCancel(t.task_id)}
                          className="h-9 px-3 text-sm font-medium border border-destructive/20 text-destructive bg-destructive/5 rounded-lg hover:bg-destructive hover:text-destructive-foreground transition-colors"
                        >
                          取消
                        </button>
                      ) : (
                        <button
                          onClick={() => setPublishDeleteConfirmationId(t.task_id)}
                          className="h-9 w-9 flex items-center justify-center rounded-lg text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-colors"
                          title="删除任务"
                        >
                          <Trash2 className="h-4 w-4" />
                        </button>
                      )}
                    </div>
                  </div>
                </div>
              </div>
            ))}

            {hasMorePublish && (
              <div className="pt-4 pb-2 flex justify-center">
                <button
                  onClick={() => setPublishLimit(prev => prev + 50)}
                  className="px-4 py-2 text-sm text-muted-foreground hover:text-primary transition-colors bg-secondary/50 rounded-lg hover:bg-secondary"
                >
                  加载更多
                </button>
              </div>
            )}
          </div>
        )}
      </div>

      {showCreator && (
        <PublishTaskCreatorModal
          onClose={() => setShowCreator(false)}
          onCreated={(taskId) => {
            setShowCreator(false);
            loadTasks();
            if (taskId) setSelectedPublishTaskId(taskId);
          }}
        />
      )}

      {publishDeleteConfirmationId && (
        <div className="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4 backdrop-blur-sm">
          <div className="bg-background border border-border rounded-xl shadow-sm max-w-sm w-full overflow-hidden">
            <div className="p-6 space-y-4">
              <div className="space-y-2 text-center">
                <h3 className="text-lg font-semibold">确认删除任务?</h3>
                <p className="text-muted-foreground text-sm">
                  此操作将永久删除发文任务记录，且无法恢复。
                </p>
              </div>
              <div className="flex justify-center gap-3 pt-2">
                <button
                  onClick={() => setPublishDeleteConfirmationId(null)}
                  className="px-4 py-2 border border-input rounded-lg hover:bg-accent font-medium text-sm transition-colors"
                >
                  取消
                </button>
                <button
                  onClick={confirmPublishDelete}
                  className="px-4 py-2 bg-destructive text-destructive-foreground rounded-lg hover:bg-destructive/90 font-medium text-sm transition-colors"
                >
                  确认删除
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {selectedPublishTaskId && (
        <PublishTaskDetailModal
          taskId={selectedPublishTaskId}
          onClose={() => setSelectedPublishTaskId(null)}
        />
      )}
    </div>
  );
}
