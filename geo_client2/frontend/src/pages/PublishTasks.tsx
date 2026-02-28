import { useEffect, useRef, useState } from 'react';
import { Link } from 'react-router-dom';
import { ChevronRight, Trash2, RotateCcw } from 'lucide-react';
import { toast } from 'sonner';
import { wailsAPI } from '@/utils/wails-api';

type LongTaskStatus = 'pending' | 'running' | 'paused' | 'completed' | 'failed' | 'cancelled' | string;

interface PublishArticle {
  title: string;
  content: string;
  cover_image?: string;
}

interface PublishLongTask {
  task_id: string;
  status: LongTaskStatus;
  article?: PublishArticle;
  platforms?: string[];
  current_index?: number;
  total_platforms?: number;
  created_at?: string;
  updated_at?: string;
}

const statusBadge = (status: LongTaskStatus) => {
  const base = 'inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium border';
  switch (status) {
    case 'running':
      return <span className={`${base} border-blue-200 bg-blue-50 text-blue-700`}>运行中</span>;
    case 'pending':
      return <span className={`${base} border-slate-200 bg-slate-50 text-slate-700`}>排队中</span>;
    case 'paused':
      return <span className={`${base} border-yellow-200 bg-yellow-50 text-yellow-800`}>已暂停</span>;
    case 'completed':
      return <span className={`${base} border-green-200 bg-green-50 text-green-700`}>已完成</span>;
    case 'failed':
      return <span className={`${base} border-red-200 bg-red-50 text-red-700`}>失败</span>;
    case 'cancelled':
      return <span className={`${base} border-zinc-200 bg-zinc-50 text-zinc-700`}>已取消</span>;
    default:
      return <span className={`${base} border-border bg-background text-foreground`}>{status}</span>;
  }
};

export default function PublishTasks() {
  const [tasks, setTasks] = useState<PublishLongTask[]>([]);
  const [loading, setLoading] = useState(true);
  const [removing, setRemoving] = useState<string | null>(null);
  const refreshInterval = useRef<ReturnType<typeof setInterval> | null>(null);

  const load = async () => {
    setLoading(true);
    try {
      const res = await wailsAPI.longTask.listRecords();
      if (res?.success && (res as any).tasks) {
        setTasks((res as any).tasks as PublishLongTask[]);
      }
    } catch (e: any) {
      toast.error('加载发文任务失败', { description: e?.message || String(e) });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    refreshInterval.current = setInterval(load, 5000);
    return () => {
      if (refreshInterval.current) clearInterval(refreshInterval.current);
    };
  }, []);

  const handleRemove = async (taskId: string) => {
    setRemoving(taskId);
    try {
      await wailsAPI.longTask.remove(taskId);
      toast.success('任务已删除');
      load();
    } catch (e: any) {
      toast.error('删除任务失败', { description: e?.message || String(e) });
    } finally {
      setRemoving(null);
    }
  };

  const handleRerun = async (taskId: string) => {
    try {
      await wailsAPI.longTask.restart(taskId);
      toast.success('任务已重新开始');
      load();
    } catch (e: any) {
      toast.error('重试失败', { description: e?.message || String(e) });
    }
  };

  return (
    <div className="p-4 md:p-6 space-y-4 max-w-[1200px] mx-auto">
      <div className="flex items-center justify-between gap-3">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">发文任务</h2>
          <p className="text-muted-foreground mt-1">每 5 秒自动刷新一次状态</p>
        </div>
      </div>

      <div className="bg-card border border-border rounded-lg overflow-hidden">
        <div className="grid grid-cols-12 px-4 py-3 text-xs text-muted-foreground border-b">
          <div className="col-span-5">标题</div>
          <div className="col-span-3">平台</div>
          <div className="col-span-2">状态</div>
          <div className="col-span-2 text-right">操作</div>
        </div>

        {loading && tasks.length === 0 ? (
          <div className="p-6 text-center text-muted-foreground">加载中...</div>
        ) : tasks.length === 0 ? (
          <div className="p-8 text-center text-muted-foreground">暂无发文任务</div>
        ) : (
          <div>
            {tasks.map((t) => (
              <div key={t.task_id} className="grid grid-cols-12 px-4 py-3 border-b last:border-b-0 items-center gap-2">
                <div className="col-span-5 min-w-0">
                  <Link
                    to={`/publish-tasks/${t.task_id}`}
                    className="group inline-flex items-center gap-2 hover:text-primary"
                    title={t.task_id}
                  >
                    <span className="truncate font-medium">{t.article?.title || '（无标题）'}</span>
                    <ChevronRight className="w-4 h-4 opacity-0 group-hover:opacity-100 transition-opacity" />
                  </Link>
                  <div className="text-xs text-muted-foreground mt-0.5 truncate">{t.task_id}</div>
                </div>

                <div className="col-span-3 text-sm text-muted-foreground truncate">
                  {(t.platforms || []).join(', ') || '-'}
                </div>

                <div className="col-span-2">{statusBadge(t.status)}</div>

                <div className="col-span-2 flex justify-end gap-2">
                  <button
                    onClick={() => handleRerun(t.task_id)}
                    className="inline-flex items-center gap-1 rounded-md border px-2.5 py-1.5 text-xs hover:bg-accent"
                    title="重试"
                  >
                    <RotateCcw className="w-3.5 h-3.5" />
                    重试
                  </button>
                  <button
                    onClick={() => handleRemove(t.task_id)}
                    disabled={removing === t.task_id}
                    className="inline-flex items-center gap-1 rounded-md border px-2.5 py-1.5 text-xs hover:bg-accent disabled:opacity-60"
                    title="删除"
                  >
                    <Trash2 className="w-3.5 h-3.5" />
                    删除
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
