import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { LayoutDashboard, ListTodo, CheckCircle2, XCircle, Users } from 'lucide-react';
import { wailsAPI } from '@/utils/wails-api';

export default function Dashboard() {
  const navigate = useNavigate();
  const [isLoading, setIsLoading] = useState(true);
  const [stats, setStats] = useState({
    queue: 0,
    running: 0,
    completed: 0,
    failed: 0,
  });
  const [accountStats, setAccountStats] = useState({
    total: 0,
    byPlatform: {} as Record<string, number>,
  });

  useEffect(() => {
    const loadStats = async () => {
      try {
        const result = await wailsAPI.task.getStats();
        if (result) {
          setStats({
            queue: (result as any).pending || 0,
            running: (result as any).running || 0,
            completed: (result as any).completed || 0,
            failed: (result as any).failed || 0,
          });
        }

        const accResult = await wailsAPI.account.getStats();
        if (accResult) {
          setAccountStats({
            total: accResult.total || 0,
            byPlatform: accResult.byPlatform || {},
          });
        }
      } catch (err) {
        console.error('Failed to load stats', err);
      } finally {
        setIsLoading(false);
      }
    };
    loadStats();
    const interval = setInterval(loadStats, 5000);
    return () => clearInterval(interval);
  }, []);

  const navigateToTasks = (status: string) => {
    navigate('/tasks', { state: { status } });
  };

  const navigateToAuth = () => {
    navigate('/auth');
  };

  return (
    <div className="p-8 space-y-8 max-w-7xl mx-auto">
      <div>
        <h1 className="text-3xl font-bold tracking-tight mb-2">仪表板</h1>
        <p className="text-muted-foreground text-lg">实时监控任务执行状态与资源概览</p>
      </div>
      
      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-6">
        <div 
          onClick={navigateToAuth}
          className="p-6 bg-card border border-border/50 rounded-xl shadow-sm hover:shadow-md hover:border-primary/30 transition-all duration-200 group cursor-pointer"
        >
          <div className="flex flex-col h-full justify-between space-y-4">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium text-muted-foreground">账户总数</span>
              <div className="p-2 bg-blue-50 dark:bg-blue-900/20 rounded-lg group-hover:bg-blue-100 dark:group-hover:bg-blue-900/40 transition-colors">
                <Users className="w-5 h-5 text-blue-600 dark:text-blue-400" />
              </div>
            </div>
            <div>
              {isLoading ? (
                <div className="h-9 w-16 bg-muted animate-pulse rounded-md mt-1 mb-1"></div>
              ) : (
                <div className="text-3xl font-bold tracking-tight text-foreground">
                  {accountStats.total}
                </div>
              )}
              <p className="text-xs text-muted-foreground mt-1">跨所有平台</p>
            </div>
          </div>
        </div>

        <div 
          onClick={() => navigateToTasks('pending')}
          className="p-6 bg-card border border-border/50 rounded-xl shadow-sm hover:shadow-md hover:border-primary/30 transition-all duration-200 group cursor-pointer"
        >
          <div className="flex flex-col h-full justify-between space-y-4">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium text-muted-foreground">待执行</span>
              <div className="p-2 bg-slate-100 dark:bg-slate-800 rounded-lg group-hover:bg-slate-200 dark:group-hover:bg-slate-700 transition-colors">
                <ListTodo className="w-5 h-5 text-slate-600 dark:text-slate-400" />
              </div>
            </div>
            <div>
              {isLoading ? (
                <div className="h-9 w-16 bg-muted animate-pulse rounded-md mt-1 mb-1"></div>
              ) : (
                <div className="text-3xl font-bold tracking-tight text-foreground">
                  {stats.queue}
                </div>
              )}
              <p className="text-xs text-muted-foreground mt-1">排队中任务</p>
            </div>
          </div>
        </div>

        <div 
          onClick={() => navigateToTasks('running')}
          className="p-6 bg-card border border-border/50 rounded-xl shadow-sm hover:shadow-md hover:border-primary/30 transition-all duration-200 group cursor-pointer"
        >
          <div className="flex flex-col h-full justify-between space-y-4">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium text-muted-foreground">执行中</span>
              <div className="p-2 bg-indigo-50 dark:bg-indigo-900/20 rounded-lg group-hover:bg-indigo-100 dark:group-hover:bg-indigo-900/40 transition-colors">
                <LayoutDashboard className="w-5 h-5 text-indigo-600 dark:text-indigo-400 animate-pulse" />
              </div>
            </div>
            <div>
              {isLoading ? (
                <div className="h-9 w-16 bg-muted animate-pulse rounded-md mt-1 mb-1"></div>
              ) : (
                <div className="text-3xl font-bold tracking-tight text-foreground">
                  {stats.running}
                </div>
              )}
              <p className="text-xs text-muted-foreground mt-1">正在处理</p>
            </div>
          </div>
        </div>

        <div 
          onClick={() => navigateToTasks('completed')}
          className="p-6 bg-card border border-border/50 rounded-xl shadow-sm hover:shadow-md hover:border-primary/30 transition-all duration-200 group cursor-pointer"
        >
          <div className="flex flex-col h-full justify-between space-y-4">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium text-muted-foreground">已完成</span>
              <div className="p-2 bg-green-50 dark:bg-green-900/20 rounded-lg group-hover:bg-green-100 dark:group-hover:bg-green-900/40 transition-colors">
                <CheckCircle2 className="w-5 h-5 text-green-600 dark:text-green-400" />
              </div>
            </div>
            <div>
              {isLoading ? (
                <div className="h-9 w-16 bg-muted animate-pulse rounded-md mt-1 mb-1"></div>
              ) : (
                <div className="text-3xl font-bold tracking-tight text-foreground">
                  {stats.completed}
                </div>
              )}
              <p className="text-xs text-muted-foreground mt-1">累积完成</p>
            </div>
          </div>
        </div>

        <div 
          onClick={() => navigateToTasks('failed')}
          className="p-6 bg-card border border-border/50 rounded-xl shadow-sm hover:shadow-md hover:border-primary/30 transition-all duration-200 group cursor-pointer"
        >
          <div className="flex flex-col h-full justify-between space-y-4">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium text-muted-foreground">失败</span>
              <div className="p-2 bg-red-50 dark:bg-red-900/20 rounded-lg group-hover:bg-red-100 dark:group-hover:bg-red-900/40 transition-colors">
                <XCircle className="w-5 h-5 text-red-600 dark:text-red-400" />
              </div>
            </div>
            <div>
              {isLoading ? (
                <div className="h-9 w-16 bg-muted animate-pulse rounded-md mt-1 mb-1"></div>
              ) : (
                <div className="text-3xl font-bold tracking-tight text-foreground">
                  {stats.failed}
                </div>
              )}
              <p className="text-xs text-muted-foreground mt-1">需人工干预</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
