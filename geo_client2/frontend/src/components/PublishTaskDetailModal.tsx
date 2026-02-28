import { useEffect, useMemo, useRef, useState } from 'react';
import { X, RotateCcw, Trash2, RotateCw } from 'lucide-react';
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

export function PublishTaskDetailModal({ taskId, onClose }: { taskId: string; onClose: () => void }) {
  const { accountsByPlatform, loadAccounts } = useAccountStore();
  const [task, setTask] = useState<PublishLongTask | null>(null);
  const [loading, setLoading] = useState(true);
  const refreshInterval = useRef<ReturnType<typeof setInterval> | null>(null);

  const load = async (opts?: { silent?: boolean }) => {
    const silent = opts?.silent === true;
    if (!silent) setLoading(true);
    try {
      const res = await wailsAPI.longTask.getRecord(taskId);
      if (res?.success && (res as any).task) {
        setTask((res as any).task as PublishLongTask);
      }
    } catch (e: any) {
      toast.error('加载任务详情失败', { description: e?.message || String(e) });
    } finally {
      if (!silent) setLoading(false);
    }
  };

  useEffect(() => {
    load();
    refreshInterval.current = setInterval(() => load({ silent: true }), 5000);
    return () => {
      if (refreshInterval.current) clearInterval(refreshInterval.current);
    };
  }, [taskId]);

  useEffect(() => {
    const platforms = task?.platforms || Object.keys(task?.account_ids || {});
    platforms.forEach((p) => loadAccounts(p as any));
  }, [task?.task_id]);

  const handleCopy = (text: string) => {
    navigator.clipboard.writeText(text).then(() => {
      toast.success('复制成功');
    });
  };

  const accountNameByPlatformAndId = useMemo(() => {
    const out: Record<string, Record<string, string>> = {};
    for (const [platform, list] of Object.entries(accountsByPlatform as any)) {
      out[platform] = {};
      for (const a of list as any[]) {
        if (a?.account_id) out[platform][a.account_id] = a.account_name || a.account_id;
      }
    }
    return out;
  }, [accountsByPlatform]);

  const platformStates = task?.platform_states || {};
  const orderedPlatforms = task?.platforms || Object.keys(platformStates);

  const handleRetry = async () => {
    try {
      await wailsAPI.longTask.restart(taskId);
      toast.success('任务已重新开始');
      load();
    } catch (e: any) {
      toast.error('重试失败', { description: e?.message || String(e) });
    }
  };

  const handleRemove = async () => {
    try {
      await wailsAPI.longTask.remove(taskId);
      toast.success('任务已删除');
      onClose();
    } catch (e: any) {
      toast.error('删除任务失败', { description: e?.message || String(e) });
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-card border border-border rounded-lg w-full max-w-6xl max-h-[90vh] flex flex-col overflow-hidden">
        <div className="p-6 border-b border-border flex items-center justify-between">
          <div className="flex items-center gap-4 min-w-0">
            <h2 className="text-xl font-bold truncate">发文任务详情</h2>
            <button
              onClick={() => {
                load();
                toast.success('已刷新');
              }}
              className="p-1.5 hover:bg-accent rounded-md text-muted-foreground hover:text-foreground transition-colors"
              title="刷新数据"
            >
              <RotateCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
            </button>
            <div className="text-xs text-muted-foreground truncate">{taskId}</div>
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={handleRetry}
              className="h-9 px-3 text-sm font-medium border border-input rounded-lg hover:bg-accent hover:text-accent-foreground flex items-center gap-1.5 transition-colors"
            >
              <RotateCcw className="h-3.5 w-3.5" />
              重试
            </button>
            <button
              onClick={handleRemove}
              className="h-9 px-3 text-sm font-medium border border-destructive/20 text-destructive bg-destructive/5 rounded-lg hover:bg-destructive hover:text-destructive-foreground transition-colors flex items-center gap-1.5"
            >
              <Trash2 className="h-3.5 w-3.5" />
              删除
            </button>
            <button
              onClick={onClose}
              className="h-9 w-9 flex items-center justify-center rounded-lg text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
              title="关闭"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
        </div>

        <div className="p-6 overflow-auto">
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
                      <RichContentEditor value={task?.article?.content || ''} onChange={() => {}} disabled rows={18} />
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
      </div>
    </div>
  );
}
