import { useTaskStore } from '@/stores/taskStore';
import { LayoutDashboard, ListTodo, CheckCircle2, XCircle } from 'lucide-react';

export default function Dashboard() {
  const { queue, current, history } = useTaskStore();

  const stats = {
    queue: queue.length,
    running: current ? 1 : 0,
    completed: history.filter((t) => t.status === 'completed').length,
    failed: history.filter((t) => t.status === 'failed').length,
  };

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold mb-2">仪表板</h1>
        <p className="text-muted-foreground">任务执行概览</p>
      </div>

      {/* 统计卡片 */}
      <div className="grid grid-cols-4 gap-4">
        <div className="p-4 bg-card border border-border rounded-lg">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-muted-foreground">待执行</p>
              <p className="text-2xl font-bold">{stats.queue}</p>
            </div>
            <ListTodo className="w-8 h-8 text-muted-foreground" />
          </div>
        </div>
        <div className="p-4 bg-card border border-border rounded-lg">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-muted-foreground">执行中</p>
              <p className="text-2xl font-bold">{stats.running}</p>
            </div>
            <LayoutDashboard className="w-8 h-8 text-primary" />
          </div>
        </div>
        <div className="p-4 bg-card border border-border rounded-lg">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-muted-foreground">已完成</p>
              <p className="text-2xl font-bold">{stats.completed}</p>
            </div>
            <CheckCircle2 className="w-8 h-8 text-green-500" />
          </div>
        </div>
        <div className="p-4 bg-card border border-border rounded-lg">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-muted-foreground">失败</p>
              <p className="text-2xl font-bold">{stats.failed}</p>
            </div>
            <XCircle className="w-8 h-8 text-destructive" />
          </div>
        </div>
      </div>

      {/* 最近任务 */}
      {current && (
        <div className="p-4 bg-card border border-border rounded-lg">
          <h2 className="text-lg font-semibold mb-3">当前执行任务</h2>
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <span className="text-sm">任务 ID: #{current.id}</span>
              <span className="text-xs px-2 py-1 bg-primary/10 text-primary rounded">
                {current.status}
              </span>
            </div>
            <div className="text-sm text-muted-foreground">
              平台: {current.platforms.join(', ')}
            </div>
            <div className="text-sm text-muted-foreground">
              关键词: {current.keywords.length} 个
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
