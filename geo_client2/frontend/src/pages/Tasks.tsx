import { useState, useEffect } from 'react';
import { RefreshCw, ChevronDown, ChevronRight, Play } from 'lucide-react';
import { toast } from 'sonner';
import { LocalTaskCreator } from '@/components/LocalTaskCreator';
import { LocalTaskDetail } from '@/components/LocalTaskDetail';
import { wailsAPI } from '@/utils/wails-api';

interface Task {
  id: number;
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
      // Reload tasks whenever we get an update to ensure status changes are reflected
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

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">任务列表</h1>
        <div className="flex gap-2">
          <button
            onClick={() => setShowCreator(true)}
            className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
          >
            创建任务
          </button>
          <button
            onClick={loadTasks}
            className="p-2 border border-border rounded-md hover:bg-accent"
          >
            <RefreshCw className="w-4 h-4" />
          </button>
        </div>
      </div>

      <div className="flex gap-2">
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
          className="px-3 py-2 border border-border rounded-md bg-background"
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
        <div className="text-center py-8 text-muted-foreground">加载中...</div>
      ) : tasks.length === 0 ? (
        <div className="text-center py-8 text-muted-foreground">暂无任务</div>
      ) : (
        <div className="space-y-2">
          {tasks.map((task) => (
            <div
              key={task.id}
              className="p-4 bg-card border border-border rounded-lg"
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => setExpandedTask(expandedTask === task.id ? null : task.id)}
                    className="p-1 hover:bg-accent rounded"
                  >
                    {expandedTask === task.id ? (
                      <ChevronDown className="w-4 h-4" />
                    ) : (
                      <ChevronRight className="w-4 h-4" />
                    )}
                  </button>
                  <span className="font-medium">任务 #{task.id}</span>
                  <span className={`text-xs px-2 py-1 rounded ${
                    task.status === 'completed' ? 'bg-green-100 text-green-800' :
                    task.status === 'running' ? 'bg-blue-100 text-blue-800' :
                    task.status === 'failed' ? 'bg-red-100 text-red-800' :
                    task.status === 'partial_completed' ? 'bg-yellow-100 text-yellow-800' :
                    'bg-gray-100 text-gray-800'
                  }`}>
                    {task.status === 'partial_completed' ? '部分完成' : 
                     task.status === 'completed' ? '已完成' :
                     task.status === 'running' ? '运行中' :
                     task.status === 'failed' ? '失败' :
                     task.status === 'pending' ? '等待中' :
                     task.status === 'cancelled' ? '已取消' : task.status}
                  </span>
                </div>
                <div className="flex gap-4 items-center">
                  <div className="flex gap-2">
                    {task.status !== 'running' && (
                      <button
                        onClick={() => handleRetry(task.id)}
                        className="flex items-center gap-1 text-sm text-primary hover:underline"
                        title="重新执行"
                      >
                        <Play className="w-3 h-3" />
                        重试
                      </button>
                    )}
                    <button
                      onClick={() => setSelectedTaskId(task.id)}
                      className="text-sm text-primary hover:underline"
                    >
                      查看详情
                    </button>
                    {task.status === 'running' && (
                      <button
                        onClick={() => handleCancel(task.id)}
                        className="text-sm text-destructive hover:underline"
                      >
                        取消
                      </button>
                    )}
                  </div>
                </div>
              </div>
              {expandedTask === task.id && (
                <div className="mt-4 space-y-2 text-sm text-muted-foreground">
                  <div>关键词: {task.keywords}</div>
                  <div>平台: {task.platforms}</div>
                  <div>查询次数: {task.query_count}</div>
                  <div>创建时间: {new Date(task.created_at).toLocaleString()}</div>
                </div>
              )}
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
    </div>
  );
}
