import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Send,
  CheckCircle2,
  Circle,
  AlertTriangle,
  Loader2,
  Play,
  StopCircle,
  RotateCcw,
  ChevronRight,
  Brain,
  Share2,
  Image,
} from 'lucide-react';
import { toast } from 'sonner';
import { wailsAPI } from '@/utils/wails-api';
import { useAccountStore } from '@/stores/accountStore';
import { EventsOn } from '@/wailsjs/runtime/runtime';
import RichContentEditor from '@/components/RichContentEditor';

interface PlatformOption {
  id: string;
  name: string;
  category: 'ai' | 'social_media';
}

const SOCIAL_PLATFORMS: PlatformOption[] = [
  { id: 'zhihu',       name: '知乎',   category: 'social_media' },
  { id: 'sohu',        name: '搜狐号', category: 'social_media' },
  { id: 'csdn',        name: 'CSDN',   category: 'social_media' },
  { id: 'qie',         name: '企鹅号', category: 'social_media' },
  { id: 'baijiahao',   name: '百家号', category: 'social_media' },
  // { id: 'xiaohongshu', name: '小红书', category: 'social_media' },
];

type PublishStatus = 'idle' | 'running' | 'waiting_manual' | 'completed' | 'failed';

interface PlatformPublishState {
  status: PublishStatus;
  message: string;
  articleUrl?: string;
  interventionPrompt?: string;
  taskId?: string;
}

export default function Publish() {
  const { accountsByPlatform, activeAccounts, loadAccounts, loadActiveAccount } = useAccountStore();
  const navigate = useNavigate();

  const [selectedPlatforms, setSelectedPlatforms] = useState<string[]>([]);
  const [selectedAccountIds, setSelectedAccountIds] = useState<Record<string, string>>({});
  
  const [title, setTitle] = useState('');
  const [content, setContent] = useState('');
  const [coverImage, setCoverImage] = useState('');
  const [publishStates, setPublishStates] = useState<Record<string, PlatformPublishState>>({});
  const [isPublishing, setIsPublishing] = useState(false);

  // 加载所有社交媒体平台的账号
  useEffect(() => {
    SOCIAL_PLATFORMS.forEach(p => {
      loadAccounts(p.id as any);
      loadActiveAccount(p.id as any);
    });
  }, []);

  // Initialize selected accounts when platforms change or accounts load
  useEffect(() => {
    const updates: Record<string, string> = {};
    let hasUpdates = false;

    selectedPlatforms.forEach(platformId => {
      // If no account selected, or selected account not in current list (optional check)
      if (!selectedAccountIds[platformId]) {
        const platformAccounts = (accountsByPlatform as any)[platformId] || [];
        const activeAccount = (activeAccounts as any)[platformId];
        const lastUsedId = localStorage.getItem(`last_account_${platformId}`);
        
        let targetId = '';
        
        // Priority: Last used (if valid) > Active > First available
        if (lastUsedId && platformAccounts.some((a: any) => a.account_id === lastUsedId)) {
          targetId = lastUsedId;
        } else if (activeAccount) {
          targetId = activeAccount.account_id;
        } else if (platformAccounts.length > 0) {
          targetId = platformAccounts[0].account_id;
        }

        if (targetId) {
          updates[platformId] = targetId;
          hasUpdates = true;
        }
      }
    });

    if (hasUpdates) {
      setSelectedAccountIds(prev => ({ ...prev, ...updates }));
    }
  }, [selectedPlatforms, accountsByPlatform, activeAccounts, selectedAccountIds]);

  const handleAccountChange = (platformId: string, accountId: string) => {
    setSelectedAccountIds(prev => ({ ...prev, [platformId]: accountId }));
    localStorage.setItem(`last_account_${platformId}`, accountId);
  };

  // 监听发布事件
  useEffect(() => {
    const unsubStarted = EventsOn('publish:started', (data: any) => {
      setPublishStates(prev => ({
        ...prev,
        [data.platform]: { status: 'running', message: '正在发布...' },
      }));
    });

    const unsubProgress = EventsOn('publish:progress', (data: any) => {
      setPublishStates(prev => ({
        ...prev,
        [data.platform]: { ...prev[data.platform], status: 'running', message: data.message },
      }));
    });

    const unsubManual = EventsOn('publish:needs_manual', (data: any) => {
      setPublishStates(prev => ({
        ...prev,
        [data.platform]: {
          status: 'waiting_manual',
          message: '等待人工操作',
          interventionPrompt: data.prompt,
          taskId: data.taskId,
        },
      }));
      toast.warning(`${getPlatformName(data.platform)}：需要人工操作`, {
        description: data.prompt,
        duration: 0,
      });
    });

    const unsubCompleted = EventsOn('publish:completed', (data: any) => {
      setPublishStates(prev => ({
        ...prev,
        [data.platform]: {
          status: data.success ? 'completed' : 'failed',
          message: data.success ? '发布成功' : (data.error || '发布失败'),
          articleUrl: data.articleUrl,
        },
      }));
      if (data.success) {
        toast.success(`${getPlatformName(data.platform)} 发布成功`);
      } else {
        toast.error(`${getPlatformName(data.platform)} 发布失败：${data.error}`);
      }
    });

    const unsubAllDone = EventsOn('publish:all_done', () => {
      setIsPublishing(false);
      toast.success('所有平台发布任务已完成');
    });

    return () => {
      unsubStarted();
      unsubProgress();
      unsubManual();
      unsubCompleted();
      unsubAllDone();
    };
  }, []);

  const getPlatformName = (id: string) =>
    SOCIAL_PLATFORMS.find(p => p.id === id)?.name ?? id;

  // Show platforms that have either an active account OR any available accounts
  // This ensures users can select a platform even if no account is currently marked as "active" globally
  const availablePlatforms = SOCIAL_PLATFORMS.filter(p => {
    const hasActive = (activeAccounts as any)[p.id];
    const hasAccounts = ((accountsByPlatform as any)[p.id] || []).length > 0;
    return hasActive || hasAccounts;
  });
  
  const unavailablePlatforms = SOCIAL_PLATFORMS.filter(p => {
    const hasActive = (activeAccounts as any)[p.id];
    const hasAccounts = ((accountsByPlatform as any)[p.id] || []).length > 0;
    return !hasActive && !hasAccounts;
  });

  const togglePlatform = (id: string) => {
    setSelectedPlatforms(prev =>
      prev.includes(id) ? prev.filter(p => p !== id) : [...prev, id]
    );
  };

  const handleStartPublish = async () => {
    if (selectedPlatforms.length === 0) {
      toast.error('请至少选择一个发布平台');
      return;
    }
    if (!title.trim()) {
      toast.error('请输入文章标题');
      return;
    }
    if (!content.trim()) {
      toast.error('请输入文章正文');
      return;
    }

    // 初始化各平台状态
    const initStates: Record<string, PlatformPublishState> = {};
    selectedPlatforms.forEach(p => {
      initStates[p] = { status: 'idle', message: '等待发布' };
    });
    setPublishStates(initStates);
    setIsPublishing(true);

    try {
      const accountIds: Record<string, string> = {};
      selectedPlatforms.forEach(p => {
        // Use specifically selected account if available, fallback to active account
        const selectedId = selectedAccountIds[p];
        if (selectedId) {
          accountIds[p] = selectedId;
        } else {
          const active = (activeAccounts as any)[p];
          if (active) accountIds[p] = active.account_id;
        }
      });

      const res = await wailsAPI.longTask.start(selectedPlatforms, accountIds, {
        title: title.trim(),
        content: content.trim(),
        cover_image: coverImage.trim() || undefined,
      });
      const taskId = (res as any)?.taskId;
      if (taskId) {
        toast.success('发文任务已创建', { description: `任务ID: ${taskId}` });
        setIsPublishing(false);
        navigate(`/publish-tasks/${taskId}`);
      } else {
        toast.success('发文任务已创建');
        setIsPublishing(false);
      }
    } catch (error) {
      toast.error('启动发布任务失败');
      setIsPublishing(false);
    }
  };

  const handleResume = async (platform: string) => {
    const state = publishStates[platform];
    if (!state?.taskId) return;
    try {
      await wailsAPI.publish.resumePublish(state.taskId);
      setPublishStates(prev => ({
        ...prev,
        [platform]: { ...prev[platform], status: 'running', message: '继续发布中...', interventionPrompt: undefined },
      }));
    } catch (error) {
      toast.error('继续操作失败');
    }
  };

  const handleReset = () => {
    setPublishStates({});
    setIsPublishing(false);
    setSelectedPlatforms([]);
  };

  const allDone =
    isPublishing &&
    selectedPlatforms.length > 0 &&
    selectedPlatforms.every(p => ['completed', 'failed'].includes(publishStates[p]?.status ?? ''));

  const statusIcon = (s: PublishStatus) => {
    switch (s) {
      case 'running':      return <Loader2 className="w-4 h-4 animate-spin text-blue-500" />;
      case 'waiting_manual': return <AlertTriangle className="w-4 h-4 text-yellow-500" />;
      case 'completed':    return <CheckCircle2 className="w-4 h-4 text-green-500" />;
      case 'failed':       return <StopCircle className="w-4 h-4 text-red-500" />;
      default:             return <Circle className="w-4 h-4 text-muted-foreground" />;
    }
  };

  const statusText: Record<PublishStatus, string> = {
    idle:           '等待发布',
    running:        '发布中',
    waiting_manual: '等待人工操作',
    completed:      '发布成功',
    failed:         '发布失败',
  };

  return (
    <div className="p-4 md:p-6 space-y-6 max-w-[1200px] mx-auto">
      {/* 页面标题 */}
      <div className="flex items-center gap-3">
        <div className="p-2 bg-primary/10 rounded-lg">
          <Send className="w-5 h-5 text-primary" />
        </div>
        <div>
          <h1 className="text-2xl font-bold tracking-tight">一键发文</h1>
          <p className="text-xs text-muted-foreground mt-0.5">选择平台，填写内容，自动化多平台同步发布</p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* 左侧：编辑区 */}
        <div className="lg:col-span-2 space-y-5">
          {/* 平台选择 */}
          <div className="bg-card border border-border rounded-lg p-4 space-y-3">
            <div className="flex items-center gap-2">
              <Share2 className="w-4 h-4 text-purple-500" />
              <h2 className="text-sm font-semibold">选择发布平台</h2>
              {selectedPlatforms.length > 0 && (
                <span className="text-xs bg-primary/10 text-primary px-2 py-0.5 rounded-full font-medium">
                  已选 {selectedPlatforms.length}
                </span>
              )}
            </div>

            {availablePlatforms.length === 0 ? (
              <div className="text-center py-6 text-muted-foreground text-sm space-y-2">
                <AlertTriangle className="w-8 h-8 mx-auto text-yellow-500 opacity-60" />
                <p>暂无已登录的社交媒体账号</p>
                <p className="text-xs">请先前往「授权列表」添加并登录社交媒体平台账号</p>
              </div>
            ) : (
              <div className="grid grid-cols-2 sm:grid-cols-3 gap-2">
                {availablePlatforms.map(p => {
                  const selected = selectedPlatforms.includes(p.id);
                  const state = publishStates[p.id];
                  const platformAccounts = (accountsByPlatform as any)[p.id] || [];
                  const currentAccountId = selectedAccountIds[p.id] || '';

                  return (
                    <div
                      key={p.id}
                      className={`relative rounded-lg border transition-all
                        ${selected
                          ? 'border-primary bg-primary/5'
                          : 'border-border bg-background hover:border-primary/50 hover:bg-accent'}
                        ${isPublishing ? 'opacity-80' : ''}
                      `}
                    >
                      <div
                        onClick={() => !isPublishing && togglePlatform(p.id)}
                        className={`flex items-center gap-2 px-3 py-2.5 cursor-pointer ${isPublishing ? 'cursor-not-allowed' : ''}`}
                      >
                        {state ? statusIcon(state.status) : (
                          selected
                            ? <CheckCircle2 className="w-4 h-4 text-primary" />
                            : <Circle className="w-4 h-4 text-muted-foreground" />
                        )}
                        <span className={`truncate text-sm font-medium ${selected ? 'text-primary' : 'text-foreground'}`}>
                          {p.name}
                        </span>
                        {state?.status === 'completed' && (
                          <span className="absolute top-2 right-2 w-2 h-2 bg-green-500 rounded-full" />
                        )}
                        {state?.status === 'failed' && (
                          <span className="absolute top-2 right-2 w-2 h-2 bg-red-500 rounded-full" />
                        )}
                      </div>

                      {selected && (
                        <div className="px-3 pb-2.5 pt-0">
                          <select
                            value={currentAccountId}
                            onChange={(e) => handleAccountChange(p.id, e.target.value)}
                            onClick={(e) => e.stopPropagation()}
                            disabled={isPublishing}
                            className="w-full text-xs px-2 py-1.5 rounded border border-input bg-background/50 focus:outline-none focus:ring-1 focus:ring-primary"
                          >
                            {platformAccounts.map((acc: any) => (
                              <option key={acc.account_id} value={acc.account_id}>
                                {acc.account_name || acc.account_id}
                              </option>
                            ))}
                          </select>
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            )}

            {unavailablePlatforms.length > 0 && (
              <div className="pt-2 border-t border-border/50">
                <p className="text-[11px] text-muted-foreground mb-1.5">未登录（不可选）</p>
                <div className="flex flex-wrap gap-1.5">
                  {unavailablePlatforms.map(p => (
                    <span
                      key={p.id}
                      className="text-xs px-2 py-0.5 rounded border border-dashed border-border text-muted-foreground/60"
                    >
                      {p.name}
                    </span>
                  ))}
                </div>
              </div>
            )}
          </div>

          {/* 内容编辑 */}
          <div className="bg-card border border-border rounded-lg p-4 space-y-4">
            <div className="flex items-center gap-2">
              <Brain className="w-4 h-4 text-blue-500" />
              <h2 className="text-sm font-semibold">文章内容</h2>
            </div>

            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">标题</label>
              <input
                type="text"
                placeholder="请输入文章标题"
                value={title}
                onChange={e => setTitle(e.target.value)}
                disabled={isPublishing}
                maxLength={200}
                className="w-full px-3 py-2 rounded-lg border border-input bg-background text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring disabled:opacity-60"
              />
              <div className="text-right text-[10px] text-muted-foreground">{title.length}/200</div>
            </div>

            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground flex items-center gap-1.5">
                <Image className="w-3.5 h-3.5" />
                封面图（可选）
              </label>
              <input
                type="text"
                placeholder="输入封面图 URL，留空则不设置封面"
                value={coverImage}
                onChange={e => setCoverImage(e.target.value)}
                disabled={isPublishing}
                className="w-full px-3 py-2 rounded-lg border border-input bg-background text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring disabled:opacity-60"
              />
              {coverImage && (
                <div className="mt-2">
                  <img
                    src={coverImage}
                    alt="封面预览"
                    className="max-h-32 rounded-md border border-border object-cover"
                    onError={(e) => {
                      (e.target as HTMLImageElement).style.display = 'none';
                    }}
                  />
                </div>
              )}
            </div>

            <RichContentEditor
              value={content}
              onChange={setContent}
              disabled={isPublishing}
              rows={16}
            />
          </div>
        </div>

        {/* 右侧：发布状态面板 */}
        <div className="space-y-4">
          <div className="bg-card border border-border rounded-lg p-4 space-y-3">
            <h2 className="text-sm font-semibold flex items-center gap-2">
              <ChevronRight className="w-4 h-4 text-muted-foreground" />
              发布状态
            </h2>

            {selectedPlatforms.length === 0 ? (
              <p className="text-xs text-muted-foreground text-center py-8">
                请先选择发布平台
              </p>
            ) : (
              <div className="space-y-2">
                {selectedPlatforms.map(platformId => {
                  const state = publishStates[platformId] ?? { status: 'idle' as PublishStatus, message: '等待发布' };
                  const name = getPlatformName(platformId);
                  return (
                    <div
                      key={platformId}
                      className={`rounded-lg border p-3 space-y-2 transition-colors
                        ${state.status === 'waiting_manual' ? 'border-yellow-400 bg-yellow-50/40 dark:bg-yellow-950/20' : ''}
                        ${state.status === 'completed' ? 'border-green-400 bg-green-50/40 dark:bg-green-950/20' : ''}
                        ${state.status === 'failed' ? 'border-red-400 bg-red-50/40 dark:bg-red-950/20' : ''}
                        ${['idle', 'running'].includes(state.status) ? 'border-border bg-background' : ''}
                      `}
                    >
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                          {statusIcon(state.status)}
                          <span className="text-sm font-medium">{name}</span>
                        </div>
                        <span className={`text-[10px] px-1.5 py-0.5 rounded font-medium
                          ${state.status === 'running' ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300' : ''}
                          ${state.status === 'waiting_manual' ? 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/40 dark:text-yellow-300' : ''}
                          ${state.status === 'completed' ? 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300' : ''}
                          ${state.status === 'failed' ? 'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-300' : ''}
                          ${state.status === 'idle' ? 'bg-muted text-muted-foreground' : ''}
                        `}>
                          {statusText[state.status]}
                        </span>
                      </div>

                      <p className="text-xs text-muted-foreground">{state.message}</p>

                      {/* 人工干预提示与继续按钮 */}
                      {state.status === 'waiting_manual' && (
                        <div className="space-y-2 pt-1">
                          {state.interventionPrompt && (
                            <div className="text-xs bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-200 p-2 rounded border border-yellow-200 dark:border-yellow-800 leading-relaxed">
                              {state.interventionPrompt}
                            </div>
                          )}
                          <button
                            onClick={() => handleResume(platformId)}
                            className="w-full flex items-center justify-center gap-1.5 px-3 py-1.5 bg-yellow-500 hover:bg-yellow-600 text-white text-xs font-semibold rounded-md transition-colors"
                          >
                            <Play className="w-3 h-3" />
                            人工操作完成，继续
                          </button>
                        </div>
                      )}

                      {state.status === 'completed' && state.articleUrl && (
                        <a
                          href={state.articleUrl}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="text-xs text-primary underline truncate block"
                        >
                          查看发布文章
                        </a>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
          </div>

          {/* 说明卡片 */}
          <div className="bg-muted/40 border border-border/50 rounded-lg p-4 space-y-2 text-xs text-muted-foreground">
            <p className="font-semibold text-foreground">发布说明</p>
            <ul className="space-y-1.5 list-disc list-inside leading-relaxed">
              <li>RPA 将自动打开浏览器完成发布流程</li>
              <li>各平台需要选择分类/标签时，系统将暂停并等待您手动操作</li>
              <li>完成人工操作后点击「继续」，RPA 将自动提交发布</li>
              <li>发布前请确保对应账号已在「授权列表」完成登录</li>
            </ul>
          </div>
        </div>
      </div>

      {/* 底部悬浮操作栏 */}
      <div className="fixed bottom-0 left-0 right-0 bg-white dark:bg-background border-t border-border shadow-lg z-50">
        <div className="max-w-[1200px] mx-auto px-5 py-5 flex items-center gap-4">
          {!isPublishing ? (
            <button
              onClick={handleStartPublish}
              disabled={selectedPlatforms.length === 0 || !title.trim() || !content.trim()}
              className="flex items-center gap-2 px-6 py-2.5 bg-primary text-primary-foreground rounded-lg font-medium text-sm hover:bg-primary/90 transition-colors disabled:opacity-50 disabled:cursor-not-allowed shadow-sm"
            >
              <Play className="w-4 h-4" />
              开始发布
            </button>
          ) : (
            <button
              onClick={handleReset}
              disabled={!allDone}
              className="flex items-center gap-2 px-6 py-2.5 border border-input rounded-lg font-medium text-sm hover:bg-accent transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <RotateCcw className="w-4 h-4" />
              重置
            </button>
          )}
          {isPublishing && !allDone && (
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Loader2 className="w-4 h-4 animate-spin" />
              <span>发布进行中...</span>
            </div>
          )}
          {selectedPlatforms.length > 0 && (
            <span className="text-xs text-muted-foreground ml-auto">
              已选择 {selectedPlatforms.length} 个平台
            </span>
          )}
        </div>
      </div>

      {/* 底部留白，防止内容被悬浮栏遮挡 */}
      <div className="h-20" />
    </div>
  );
}
