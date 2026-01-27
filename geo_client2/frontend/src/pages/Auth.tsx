import { useEffect, useState } from 'react';
import { useAccountStore } from '@/stores/accountStore';
import { wailsAPI } from '@/utils/wails-api';
import { 
  Shield, 
  Plus, 
  Trash2, 
  CheckCircle2, 
  Circle, 
  ExternalLink, 
  AlertTriangle, 
  Brain, 
  Share2, 
  Pencil, 
  X, 
  Check,
  LayoutGrid,
  RefreshCw,
  XCircle,
  CheckCheck
} from 'lucide-react';
import { toast } from 'sonner';

type PlatformKey = 'deepseek' | 'doubao' | 'xiaohongshu' | 'yiyan' | 'yuanbao';

interface PlatformConfig {
  id: PlatformKey;
  name: string;
  category: 'ai' | 'platform';
}

export default function Auth() {
  const { 
    accountsByPlatform, 
    activeAccounts, 
    loading, 
    loadAccounts, 
    loadActiveAccount, 
    createAccount, 
    setActiveAccount, 
    deleteAccount,
    startLogin,
    stopLogin,
    updateAccountName
  } = useAccountStore();
  
  const [newAccountName, setNewAccountName] = useState('');
  const [addingPlatform, setAddingPlatform] = useState<string | null>(null);
  const [isLoginOpen, setIsLoginOpen] = useState(false);
  const [currentLogin, setCurrentLogin] = useState<{platform: string, accountID: string} | null>(null);
  const [deleteConfirm, setDeleteConfirm] = useState<{accountID: string, accountName: string} | null>(null);
  const [editingAccountID, setEditingAccountID] = useState<string | null>(null);
  const [editingName, setEditingName] = useState('');
  const [checkingLogin, setCheckingLogin] = useState(false);
  const [checkingStatus, setCheckingStatus] = useState<Record<string, boolean>>({});
  const [loginStatus, setLoginStatus] = useState<Record<string, boolean | null>>({});
  const [batchChecking, setBatchChecking] = useState(false);
  const [batchProgress, setBatchProgress] = useState<{ checked: number; total: number }>({ checked: 0, total: 0 });

  const platforms: PlatformConfig[] = [
    { id: 'deepseek', name: 'DeepSeek', category: 'ai' },
    { id: 'doubao', name: '豆包', category: 'ai' },
    { id: 'yiyan', name: '文心一言', category: 'ai' },
    { id: 'yuanbao', name: '腾讯元宝', category: 'ai' },
    { id: 'xiaohongshu', name: '小红书', category: 'platform' },
  ];

  useEffect(() => {
    platforms.forEach(p => {
      loadAccounts(p.id);
      loadActiveAccount(p.id);
    });

    const unsubStart = wailsAPI.search.onBatchCheckStarted((data) => {
      setBatchChecking(true);
      setBatchProgress({ checked: 0, total: 0 });
      toast.info(data.message);
    });

    const unsubProgress = wailsAPI.search.onBatchCheckProgress((data) => {
      setBatchProgress({ checked: data.checked, total: data.total });
      
      if (data.skipped) {
        toast.info(data.message, { duration: 2000 });
      } else if (data.error) {
        toast.error(data.message, { duration: 3000 });
        setLoginStatus(prev => ({ ...prev, [data.platform]: false }));
      } else if (data.checking) {
        setCheckingStatus(prev => ({ ...prev, [data.platform]: true }));
        toast.info(data.message, { duration: 1000 });
      } else if (data.is_logged_in !== undefined) {
        setCheckingStatus(prev => ({ ...prev, [data.platform]: false }));
        setLoginStatus(prev => ({ ...prev, [data.platform]: data.is_logged_in! }));
        if (data.is_logged_in) {
          toast.success(data.message, { duration: 2000 });
        } else {
          toast.warning(data.message, { duration: 3000 });
        }
      }
    });

    const unsubComplete = wailsAPI.search.onBatchCheckCompleted((data) => {
      setBatchChecking(false);
      setBatchProgress({ checked: 0, total: 0 });
      toast.success(data.message, { duration: 3000 });
    });

    return () => {
      unsubStart();
      unsubProgress();
      unsubComplete();
    };
  }, []);

  const handleLogin = async (platform: string, accountID: string) => {
    try {
      await setActiveAccount(platform as any, accountID);
      setCurrentLogin({ platform, accountID });
      await startLogin(platform as any, accountID);
      setIsLoginOpen(true);
    } catch (error) {
      toast.error('启动浏览器失败');
    }
  };

  const handleConfirmLogin = async () => {
    try {
      setCheckingLogin(true);
      try {
        await stopLogin();
      } catch (error) {
        // 用户可能已经手动关闭浏览器，忽略错误
        console.log('Browser might already be closed by user');
      }
      toast.success('登录完成，浏览器状态已保存');
    } catch (error) {
      toast.error('操作失败');
    } finally {
      setCheckingLogin(false);
      setIsLoginOpen(false);
      setCurrentLogin(null);
    }
  };

  const handleAddAccount = async (platform: string) => {
    if (!newAccountName.trim()) {
      toast.error('请输入账号备注名称');
      return;
    }
    try {
      const account = await createAccount(platform as any, newAccountName);
      toast.success('账号创建成功');
      setNewAccountName('');
      setAddingPlatform(null);
      if (confirm('账号创建成功，是否立即打开浏览器进行登录？')) {
        handleLogin(platform, account.account_id);
      }
    } catch (error) {
      toast.error('账号创建失败');
    }
  };

  const handleDelete = async (accountID: string) => {
    if (!deleteConfirm) return;
    try {
      await deleteAccount(accountID);
      toast.success('账号删除成功');
      setDeleteConfirm(null);
    } catch (error) {
      toast.error('账号删除失败');
    }
  };

  const handleStartEdit = (account: { account_id: string, account_name: string }) => {
    setEditingAccountID(account.account_id);
    setEditingName(account.account_name);
  };

  const handleSaveEdit = async () => {
    if (!editingAccountID || !editingName.trim()) {
      setEditingAccountID(null);
      return;
    }
    try {
      await updateAccountName(editingAccountID, editingName);
      toast.success('账号备注更新成功');
    } catch (error) {
      toast.error('账号备注更新失败');
    } finally {
      setEditingAccountID(null);
    }
  };

  const handleCheckLoginStatus = async (platform: string) => {
    const activeAccount = activeAccounts[platform as PlatformKey];
    if (!activeAccount) {
      toast.error('请先设置该平台的活跃账号');
      return;
    }

    setCheckingStatus(prev => ({ ...prev, [platform]: true }));
    setLoginStatus(prev => ({ ...prev, [platform]: null }));

    try {
      const result = await wailsAPI.search.checkLoginStatus(platform);
      const isLoggedIn = result.isLoggedIn;
      setLoginStatus(prev => ({ ...prev, [platform]: isLoggedIn }));
      
      if (isLoggedIn) {
        toast.success(`${platform} 登录状态正常`);
      } else {
        toast.error(`${platform} 登录已过期，请重新登录`);
      }
    } catch (error) {
      toast.error('检测登录状态失败');
      setLoginStatus(prev => ({ ...prev, [platform]: false }));
    } finally {
      setCheckingStatus(prev => ({ ...prev, [platform]: false }));
    }
  };

  const handleBatchCheckLoginStatus = async () => {
    const hasAnyActiveAccount = platforms.some(p => activeAccounts[p.id as PlatformKey]);
    if (!hasAnyActiveAccount) {
      toast.error('请先至少添加并激活一个账号');
      return;
    }

    try {
      await wailsAPI.search.batchCheckLoginStatus();
    } catch (error) {
      toast.error('启动批量检测失败');
      setBatchChecking(false);
    }
  };

  const renderPlatformCard = (platform: PlatformConfig) => {
    const isAI = platform.category === 'ai';
    const accentClass = isAI ? 'border-l-blue-500' : 'border-l-purple-500';
    const bgHeaderClass = isAI ? 'bg-blue-50/50 dark:bg-blue-950/20' : 'bg-purple-50/50 dark:bg-purple-950/20';
    const iconColorClass = isAI ? 'text-blue-500' : 'text-purple-500';
    const isChecking = checkingStatus[platform.id] || false;
    const status = loginStatus[platform.id];

    return (
      <div key={platform.id} className={`bg-card border border-border rounded-lg overflow-hidden border-l-4 ${accentClass} shadow-sm flex flex-col`}>
        <div className={`px-3 py-2 ${bgHeaderClass} border-b border-border flex justify-between items-center shrink-0`}>
          <div className="flex items-center gap-2 overflow-hidden">
            {isAI ? <Brain className={`w-4 h-4 shrink-0 ${iconColorClass}`} /> : <Share2 className={`w-4 h-4 shrink-0 ${iconColorClass}`} />}
            <span className="font-bold text-sm truncate">{platform.name}</span>
            <span className={`hidden sm:inline-block text-[10px] px-1.5 py-0.5 rounded-full uppercase font-bold tracking-wider shrink-0 ${isAI ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300' : 'bg-purple-100 text-purple-700 dark:bg-purple-900/40 dark:text-purple-300'}`}>
              {isAI ? 'AI' : 'Platform'}
            </span>
            {status !== null && !isChecking && (
              <span title={status ? "登录状态正常" : "登录已过期"}>
                {status ? (
                  <CheckCircle2 className="w-3.5 h-3.5 text-green-500 shrink-0" />
                ) : (
                  <XCircle className="w-3.5 h-3.5 text-red-500 shrink-0" />
                )}
              </span>
            )}
          </div>
          <div className="flex items-center gap-1 shrink-0">
            <button
              onClick={() => handleCheckLoginStatus(platform.id)}
              className={`p-1 hover:bg-background rounded-md transition-colors text-muted-foreground hover:text-primary ${isChecking ? 'animate-spin' : ''}`}
              disabled={isChecking || !activeAccounts[platform.id as PlatformKey]}
              title="检测登录状态"
            >
              <RefreshCw className="w-4 h-4" />
            </button>
            <button
              onClick={() => setAddingPlatform(platform.id)}
              className="p-1 hover:bg-background rounded-md transition-colors text-muted-foreground hover:text-primary shrink-0"
              disabled={loading[`${platform.id}_create`]}
              title="添加账号"
            >
              <Plus className="w-4 h-4" />
            </button>
          </div>
        </div>
        
        {addingPlatform === platform.id && (
          <div className="p-2 border-b border-border bg-accent/20">
            <div className="flex gap-1.5">
              <input 
                autoFocus
                placeholder="账号备注"
                className="flex-1 min-w-0 px-2 py-1 rounded border border-input bg-background text-xs"
                value={newAccountName}
                onChange={e => setNewAccountName(e.target.value)}
                onKeyDown={e => e.key === 'Enter' && handleAddAccount(platform.id)}
              />
              <button 
                onClick={() => handleAddAccount(platform.id)}
                className="p-1 bg-primary text-primary-foreground rounded hover:bg-primary/90"
              >
                <Check className="w-3.5 h-3.5" />
              </button>
              <button 
                onClick={() => { setAddingPlatform(null); setNewAccountName(''); }}
                className="p-1 border border-input rounded hover:bg-accent"
              >
                <X className="w-3.5 h-3.5" />
              </button>
            </div>
          </div>
        )}

        <div className="flex-1 min-h-[60px] max-h-[160px] overflow-y-auto custom-scrollbar">
          {accountsByPlatform[platform.id as PlatformKey]?.length === 0 ? (
            <div className="h-full flex items-center justify-center p-4 text-center text-muted-foreground text-xs italic">
              暂无账号
            </div>
          ) : (
            <div className="divide-y divide-border/50">
              {accountsByPlatform[platform.id as PlatformKey]?.map(account => {
                const isActive = activeAccounts[platform.id as PlatformKey]?.account_id === account.account_id;
                return (
                  <div key={account.account_id} className={`group px-3 py-1.5 flex items-center justify-between hover:bg-accent/30 transition-colors ${isActive ? 'bg-accent/20' : ''}`}>
                    <div className="flex items-center gap-2 overflow-hidden flex-1">
                      <button 
                        onClick={() => !isActive && setActiveAccount(platform.id as any, account.account_id)}
                        className={`shrink-0 transition-colors ${isActive ? 'text-green-500' : 'text-muted-foreground hover:text-primary opacity-40 group-hover:opacity-100'}`}
                      >
                        {isActive ? <CheckCircle2 className="w-4 h-4" /> : <Circle className="w-4 h-4" />}
                      </button>
                      <div className="flex flex-col min-w-0 flex-1">
                        {editingAccountID === account.account_id ? (
                          <input 
                            autoFocus
                            className="px-1 py-0 rounded border border-input bg-background text-xs w-full"
                            value={editingName}
                            onChange={(e) => setEditingName(e.target.value)}
                            onBlur={handleSaveEdit}
                            onKeyDown={(e) => {
                              if (e.key === 'Enter') handleSaveEdit();
                              if (e.key === 'Escape') setEditingAccountID(null);
                            }}
                          />
                        ) : (
                          <div className="flex items-center gap-1.5 min-w-0">
                            <span className={`text-xs font-medium truncate ${isActive ? 'text-foreground' : 'text-muted-foreground'}`}>
                              {account.account_name}
                            </span>
                            {isActive && <span className="shrink-0 text-[9px] bg-green-500/10 text-green-600 px-1 rounded border border-green-500/20 font-bold">使用中</span>}
                          </div>
                        )}
                        <span className="text-[9px] text-muted-foreground/50 font-mono truncate">
                          ID: {account.account_id.slice(0, 6)}
                        </span>
                      </div>
                    </div>
                    
                    <div className="flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity shrink-0 ml-2">
                      <button 
                        onClick={() => handleLogin(platform.id, account.account_id)}
                        className="p-1 text-muted-foreground hover:text-primary hover:bg-primary/10 rounded transition-colors"
                        title="登录"
                      >
                        <ExternalLink className="w-3.5 h-3.5" />
                      </button>
                      <button 
                        onClick={() => handleStartEdit(account)}
                        className="p-1 text-muted-foreground hover:text-primary hover:bg-primary/10 rounded transition-colors"
                        title="重命名"
                      >
                        <Pencil className="w-3.5 h-3.5" />
                      </button>
                      <button 
                        onClick={() => setDeleteConfirm({accountID: account.account_id, accountName: account.account_name})}
                        className="p-1 text-muted-foreground hover:text-destructive hover:bg-destructive/10 rounded transition-colors"
                        title="删除"
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </div>
    );
  };

  const aiPlatforms = platforms.filter(p => p.category === 'ai');
  const socialPlatforms = platforms.filter(p => p.category === 'platform');

  return (
    <div className="p-4 md:p-6 space-y-8 max-w-[1600px] mx-auto">
      {deleteConfirm && (
        <div className="fixed inset-0 z-[100] bg-background/80 backdrop-blur-sm flex items-center justify-center p-4">
          <div className="bg-card border border-border p-5 rounded-xl shadow-2xl max-w-sm w-full space-y-4 animate-in fade-in zoom-in duration-200">
            <div className="flex items-start gap-3 text-destructive">
              <AlertTriangle className="w-5 h-5 mt-0.5 shrink-0" />
              <div>
                <h3 className="font-bold">确认删除账号</h3>
                <p className="text-xs text-muted-foreground mt-1 leading-relaxed">
                  确认删除 <span className="font-semibold text-foreground">{deleteConfirm.accountName}</span> 吗？此操作将清除该账号的所有本地浏览器数据且无法恢复。
                </p>
              </div>
            </div>
            <div className="flex justify-end gap-2 pt-2">
              <button onClick={() => setDeleteConfirm(null)} className="px-4 py-1.5 text-xs border border-input rounded-md hover:bg-accent transition-colors">取消</button>
              <button onClick={() => handleDelete(deleteConfirm.accountID)} className="px-4 py-1.5 text-xs bg-destructive text-destructive-foreground rounded-md hover:bg-destructive/90 transition-colors font-medium">确认删除</button>
            </div>
          </div>
        </div>
      )}

      {isLoginOpen && (
        <div className="fixed inset-0 z-[100] bg-background/80 backdrop-blur-sm flex items-center justify-center p-4">
          <div className="bg-card border border-border p-8 rounded-2xl shadow-2xl max-w-md w-full text-center space-y-6 animate-in fade-in zoom-in duration-200">
            <div className="w-16 h-16 bg-primary/10 rounded-full flex items-center justify-center mx-auto">
              <Shield className="w-8 h-8 text-primary animate-pulse" />
            </div>
            <div className="space-y-2">
              <h3 className="text-xl font-bold">账号登录校验中</h3>
              <p className="text-sm text-muted-foreground leading-relaxed px-4">
                请在开启的浏览器窗口中完成登录操作。<br/>完成登录后，点击下方按钮关闭浏览器并保存状态。
              </p>
            </div>
            <button
              onClick={handleConfirmLogin}
              disabled={checkingLogin}
              className="w-full py-3 bg-primary text-primary-foreground rounded-xl hover:bg-primary/90 transition-all font-bold shadow-lg shadow-primary/20 disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
            >
              {checkingLogin ? (
                <>
                  <div className="w-4 h-4 border-2 border-primary-foreground/30 border-t-primary-foreground rounded-full animate-spin"></div>
                  <span>保存中...</span>
                </>
              ) : (
                '我已完成登录'
              )}
            </button>
          </div>
        </div>
      )}

      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="p-2 bg-primary/10 rounded-lg">
            <LayoutGrid className="w-5 h-5 text-primary" />
          </div>
          <div>
            <h1 className="text-2xl font-bold tracking-tight">账号管理</h1>
            <p className="text-xs text-muted-foreground mt-0.5">管理您的 AI 模型账户与社交媒体发布平台</p>
          </div>
        </div>
        
        <div className="flex items-center gap-2">
          {batchChecking && (
            <div className="flex items-center gap-2 px-3 py-1.5 bg-blue-50 dark:bg-blue-950/20 border border-blue-200 dark:border-blue-800 rounded-lg">
              <div className="w-3 h-3 border-2 border-blue-500 border-t-transparent rounded-full animate-spin"></div>
              <span className="text-xs text-blue-700 dark:text-blue-300 font-medium">
                检测中 {batchProgress.checked}/{batchProgress.total}
              </span>
            </div>
          )}
          
          <button
            onClick={handleBatchCheckLoginStatus}
            disabled={batchChecking}
            className="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50 disabled:cursor-not-allowed text-sm font-medium shadow-sm"
            title="一键检测所有已登录账号"
          >
            <CheckCheck className="w-4 h-4" />
            <span>批量检测登录状态</span>
          </button>
        </div>
      </div>

      <div className="space-y-10">
        <section className="space-y-4">
          <div className="flex items-center gap-2">
            <Brain className="w-4 h-4 text-blue-500" />
            <h2 className="text-xs font-bold text-muted-foreground uppercase tracking-[0.2em]">大模型智能体 (AI Models)</h2>
            <div className="h-px flex-1 bg-blue-500/20 ml-2"></div>
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5 gap-4">
            {aiPlatforms.map(renderPlatformCard)}
          </div>
        </section>

        <section className="space-y-4">
          <div className="flex items-center gap-2">
            <Share2 className="w-4 h-4 text-purple-500" />
            <h2 className="text-xs font-bold text-muted-foreground uppercase tracking-[0.2em]">社交媒体平台 (Publishers)</h2>
            <div className="h-px flex-1 bg-purple-500/20 ml-2"></div>
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5 gap-4">
            {socialPlatforms.map(renderPlatformCard)}
          </div>
        </section>
      </div>
    </div>
  );
}
