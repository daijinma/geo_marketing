import { useEffect, useState } from 'react';
import { LayoutDashboard, ListTodo, CheckCircle2, XCircle } from 'lucide-react';
import { wailsAPI } from '@/utils/wails-api';

export default function Dashboard() {
  const [stats, setStats] = useState({
    queue: 0,
    running: 0,
    completed: 0,
    failed: 0,
  });

  useEffect(() => {
    const loadStats = async () => {
      try {
        const result = await wailsAPI.task.getStats();
        if (result) {
          setStats({
            queue: (result as any).running || 0,
            running: (result as any).running || 0,
            completed: (result as any).completed || 0,
            failed: (result as any).failed || 0,
          });
        }
      } catch (err) {
        console.error('Failed to load stats', err);
      }
    };
    loadStats();
    const interval = setInterval(loadStats, 5000);
    return () => clearInterval(interval);
  }, []);

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold mb-2">仪表板</h1>
        <p className="text-muted-foreground">任务执行概览</p>
      </div>
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
    </div>
  );
}
