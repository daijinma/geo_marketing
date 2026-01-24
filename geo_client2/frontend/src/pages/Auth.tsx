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
  LayoutGrid
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
      await stopLogin();
      if (currentLogin) {
        toast.info('正在验证登录状态...');
        const result = await wailsAPI.search.checkLoginStatus(currentLogin.platform);
        if (result.success && result.isLoggedIn) {
          toast.success('登录验证成功');
        } else {
          toast.warning('未检测到登录状态，请确保您已在浏览器中完成登录');
        }
      }
    } catch (error) {
      toast.error('关闭浏览器失败');
    } finally {
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

  const renderPlatformCard = (platform: PlatformConfig) => {
    const isAI = platform.category === 'ai';
    const accentClass = isAI ? 'border-l-blue-500' : 'border-l-purple-500';
    const bgHeaderClass = isAI ? 'bg-blue-50/50 dark:bg-blue-950/20' : 'bg-purple-50/50 dark:bg-purple-950/20';
    const iconColorClass = isAI ? 'text-blue-500' : 'text-purple-500';

    return (
      <div key={platform.id} className={`bg-card border border-border rounded-lg overflow-hidden border-l-4 ${accentClass} shadow-sm flex flex-col`}>
        <div className={`px-3 py-2 ${bgHeaderClass} border-b border-border flex justify-between items-center shrink-0`}>
          <div className="flex items-center gap-2 overflow-hidden">
            {isAI ? <Brain className={`w-4 h-4 shrink-0 ${iconColorClass}`} /> : <Share2 className={`w-4 h-4 shrink-0 ${iconColorClass}`} />}
            <span className="font-bold text-sm truncate">{platform.name}</span>
            <span className={`hidden sm:inline-block text-[10px] px-1.5 py-0.5 rounded-full uppercase font-bold tracking-wider shrink-0 ${isAI ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300' : 'bg-purple-100 text-purple-700 dark:bg-purple-900/40 dark:text-purple-300'}`}>
              {isAI ? 'AI' : 'Platform'}
            </span>
          </div>
          <button
            onClick={() => setAddingPlatform(platform.id)}
            className="p-1 hover:bg-background rounded-md transition-colors text-muted-foreground hover:text-primary shrink-0"
            disabled={loading[`${platform.id}_create`]}
            title="添加账号"
          >
            <Plus className="w-4 h-4" />
          </button>
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
              className="w-full py-3 bg-primary text-primary-foreground rounded-xl hover:bg-primary/90 transition-all font-bold shadow-lg shadow-primary/20"
            >
              我已完成登录
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
