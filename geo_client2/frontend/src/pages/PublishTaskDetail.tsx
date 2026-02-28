import { useEffect, useMemo, useRef, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { ChevronLeft, RotateCcw, Trash2 } from 'lucide-react';
import { toast } from 'sonner';
import { wailsAPI } from '@/utils/wails-api';
import RichContentEditor from '@/components/RichContentEditor';
import { useAccountStore } from '@/stores/accountStore';

type LongTaskStatus = 'pending' | 'running' | 'paused' | 'completed' | 'failed' | 'cancelled' | string;

interface PublishArticle {
  title: string;
  content: string;
  cover_image?: string;
}

interface PlatformState {
  platform: string;
  status: LongTaskStatus;
  started_at?: string;
  completed_at?: string;
  article_url?: string;
  error?: string;
  retries?: number;
  max_retries?: number;
}

interface PublishLongTask {
  task_id: string;
  status: LongTaskStatus;
  article?: PublishArticle;
  platforms?: string[];
  account_ids?: Record<string, string>;
  platform_states?: Record<string, PlatformState> | null;
  current_index?: number;
  total_platforms?: number;
  created_at?: string;
  updated_at?: string;
  started_at?: string | null;
  completed_at?: string | null;
}

const fmtTime = (s?: string | null) => {
  if (!s) return '-';
  const d = new Date(s);
  if (Number.isNaN(d.getTime())) return s;
  return d.toLocaleString();
};

export default function PublishTaskDetail() {
  const { taskId } = useParams();
  const navigate = useNavigate();
  const { accountsByPlatform, loadAccounts } = useAccountStore();
  const [task, setTask] = useState<PublishLongTask | null>(null);
  const refreshInterval = useRef<ReturnType<typeof setInterval> | null>(null);

  const load = async () => {
    if (!taskId) return;
    try {
      const res = await wailsAPI.longTask.getRecord(taskId);
      if (res?.success && (res as any).task) {
        setTask((res as any).task as PublishLongTask);
      }
    } catch (e: any) {
      toast.error('加载任务详情失败', { description: e?.message || String(e) });
    }
  };

  useEffect(() => {
    load();
    refreshInterval.current = setInterval(load, 5000);
    return () => {
      if (refreshInterval.current) clearInterval(refreshInterval.current);
    };
  }, [taskId]);

  useEffect(() => {
    const platforms = task?.platforms || Object.keys(task?.account_ids || {});
    platforms.forEach((p) => {
      // Best effort: store only knows a fixed set of platforms
      loadAccounts(p as any);
    });
  }, [task?.task_id]);

  const handleRerun = async () => {
    if (!taskId) return;
    try {
      await wailsAPI.longTask.restart(taskId);
      toast.success('任务已重新开始');
      load();
    } catch (e: any) {
      toast.error('重试失败', { description: e?.message || String(e) });
    }
  };

  const handleRemove = async () => {
    if (!taskId) return;
    try {
      await wailsAPI.longTask.remove(taskId);
      toast.success('任务已删除');
      navigate('/tasks');
    } catch (e: any) {
      toast.error('删除任务失败', { description: e?.message || String(e) });
    }
  };

  const platformStates = task?.platform_states || {};
  const orderedPlatforms = task?.platforms || Object.keys(platformStates);

  const handleCopy = (text: string) => {
    navigator.clipboard.writeText(text).then(() => {
      toast.success('复制成功');
    });
  };

  const accountNameByPlatformAndId = useMemo(() => {
    const out: Record<string, Record<string, string>> = {};
    for (const [platform, list] of Object.entries(accountsByPlatform as any)) {
      out[platform] = {};
      for (const a of (list as any[])) {
        if (a?.account_id) out[platform][a.account_id] = a.account_name || a.account_id;
      }
    }
    return out;
  }, [accountsByPlatform]);

  return (
    <div className="p-4 md:p-6 space-y-4 max-w-[1200px] mx-auto">
        <div className="flex items-center justify-between gap-3">
        <div className="flex items-center gap-3 min-w-0">
          <Link to="/tasks" className="inline-flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground">
            <ChevronLeft className="w-4 h-4" />
            返回列表
          </Link>
          <div className="min-w-0">
            <h2 className="text-2xl font-bold tracking-tight truncate">{task?.article?.title || '发文任务详情'}</h2>
            <div className="text-xs text-muted-foreground mt-0.5 truncate">{taskId}</div>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <button
            onClick={handleRerun}
            className="inline-flex items-center gap-2 rounded-md border px-3 py-2 text-sm hover:bg-accent"
          >
            <RotateCcw className="w-4 h-4" />
            重试
          </button>
          <button
            onClick={handleRemove}
            className="inline-flex items-center gap-2 rounded-md border px-3 py-2 text-sm hover:bg-accent"
          >
            <Trash2 className="w-4 h-4" />
            删除
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <div className="lg:col-span-2 space-y-4">
          <div className="bg-card border border-border rounded-lg p-4">
            <div className="text-sm font-semibold">输入内容</div>

            <div className="mt-3 space-y-3">
              <div className="border rounded-md p-3">
                <div className="flex items-center justify-between gap-2">
                  <div className="text-xs text-muted-foreground">标题</div>
                  <button
                    onClick={() => handleCopy(task?.article?.title || '')}
                    className="text-xs px-2 py-1 rounded border hover:bg-accent"
                    disabled={!task?.article?.title}
                  >
                    复制
                  </button>
                </div>
                <div className="mt-2 font-medium whitespace-pre-wrap">{task?.article?.title || '（无标题）'}</div>
              </div>

              <div className="border rounded-md p-3">
                <div className="flex items-center justify-between gap-2">
                  <div className="text-xs text-muted-foreground">封面</div>
                  <button
                    onClick={() => handleCopy(task?.article?.cover_image || '')}
                    className="text-xs px-2 py-1 rounded border hover:bg-accent"
                    disabled={!task?.article?.cover_image}
                  >
                    复制
                  </button>
                </div>
                {task?.article?.cover_image ? (
                  <div className="mt-2 space-y-2">
                    <div className="text-xs text-muted-foreground break-all">{task.article.cover_image}</div>
                    {/^https?:\/\//.test(task.article.cover_image) && (
                      <img
                        src={task.article.cover_image}
                        alt="cover"
                        className="w-full max-w-[360px] rounded border"
                      />
                    )}
                  </div>
                ) : (
                  <div className="mt-2 text-sm text-muted-foreground">（未填写，平台侧可能自动生成）</div>
                )}
              </div>

              <div className="border rounded-md p-3">
                <div className="flex items-center justify-between gap-2">
                  <div className="text-xs text-muted-foreground">正文</div>
                  <button
                    onClick={() => handleCopy(task?.article?.content || '')}
                    className="text-xs px-2 py-1 rounded border hover:bg-accent"
                    disabled={!task?.article?.content}
                  >
                    复制
                  </button>
                </div>
                <div className="mt-2">
                  <RichContentEditor
                    value={task?.article?.content || ''}
                    onChange={() => {}}
                    disabled
                    rows={18}
                  />
                </div>
              </div>
            </div>
          </div>

          <div className="bg-card border border-border rounded-lg p-4">
            <div className="text-sm font-semibold">平台进度</div>
            <div className="text-xs text-muted-foreground mt-1">
              状态：{task?.status || '-'}；进度：{task?.current_index ?? 0}/{task?.total_platforms ?? (task?.platforms?.length || 0)}
            </div>

            <div className="mt-3 space-y-2">
              {orderedPlatforms.length === 0 ? (
                <div className="text-sm text-muted-foreground">暂无平台信息</div>
              ) : (
                orderedPlatforms.map((p) => {
                  const ps = (platformStates as any)?.[p] as PlatformState | undefined;
                  const st = ps?.status || 'pending';
                  const err = ps?.error;
                  const url = ps?.article_url;
                  return (
                    <div key={p} className="flex items-start justify-between gap-3 border rounded-md px-3 py-2">
                      <div className="min-w-0">
                        <div className="font-medium">{p}</div>
                        <div className="text-xs text-muted-foreground mt-0.5">
                          {st}{err ? `：${err}` : ''}
                        </div>
                        {url ? (
                          <div className="text-xs mt-1">
                            <a className="text-primary hover:underline" href={url} target="_blank" rel="noreferrer">
                              {url}
                            </a>
                          </div>
                        ) : null}
                      </div>
                      <div className="text-xs text-muted-foreground text-right">
                        <div>开始：{fmtTime(ps?.started_at)}</div>
                        <div>完成：{fmtTime(ps?.completed_at)}</div>
                      </div>
                    </div>
                  );
                })
              )}
            </div>
          </div>
        </div>

        <div className="space-y-4">
          <div className="bg-card border border-border rounded-lg p-4 space-y-2">
            <div className="text-sm font-semibold">任务信息</div>
            <div className="text-sm">
              <div className="flex items-center justify-between gap-2">
                <span className="text-muted-foreground">创建时间</span>
                <span className="font-medium">{fmtTime(task?.created_at)}</span>
              </div>
              <div className="flex items-center justify-between gap-2">
                <span className="text-muted-foreground">更新时间</span>
                <span className="font-medium">{fmtTime(task?.updated_at)}</span>
              </div>
              <div className="flex items-center justify-between gap-2">
                <span className="text-muted-foreground">开始时间</span>
                <span className="font-medium">{fmtTime(task?.started_at)}</span>
              </div>
              <div className="flex items-center justify-between gap-2">
                <span className="text-muted-foreground">完成时间</span>
                <span className="font-medium">{fmtTime(task?.completed_at)}</span>
              </div>
            </div>
          </div>

          <div className="bg-card border border-border rounded-lg p-4 space-y-2">
            <div className="text-sm font-semibold">平台账号</div>
            <div className="text-sm text-muted-foreground">
              {task?.account_ids ? (
                <div className="space-y-1">
                  {Object.entries(task.account_ids).map(([p, id]) => {
                    const name = accountNameByPlatformAndId?.[p]?.[id];
                    return (
                      <div key={p} className="flex items-center justify-between gap-2">
                        <span className="truncate" title={`${p} - ${name || id}`}>{p} - {name || id}</span>
                        <span className="font-mono text-[10px] text-muted-foreground truncate" title={id}>{id}</span>
                      </div>
                    );
                  })}
                </div>
              ) : (
                <div>-</div>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
