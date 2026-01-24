import { useEffect, useState } from 'react';
import { useAccountStore } from '@/stores/accountStore';
import { wailsAPI } from '@/utils/wails-api';
import { Shield, Plus, Trash2, CheckCircle2, Circle, LogIn, ExternalLink } from 'lucide-react';
import { toast } from 'sonner';

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
    stopLogin
  } = useAccountStore();
  
  const [newAccountName, setNewAccountName] = useState('');
  const [addingPlatform, setAddingPlatform] = useState<string | null>(null);
  const [isLoginOpen, setIsLoginOpen] = useState(false);
  const [currentLogin, setCurrentLogin] = useState<{platform: string, accountID: string} | null>(null);

  useEffect(() => {
    loadAccounts('deepseek');
    loadActiveAccount('deepseek');
    loadAccounts('doubao');
    loadActiveAccount('doubao');
  }, []);

  const handleLogin = async (platform: string, accountID: string) => {
    try {
      // Set as active account to ensure verification works correctly
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
      } else {
        toast.success('登录流程结束');
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

      // Auto start login
      if (confirm('账号创建成功，是否立即打开浏览器进行登录？')) {
        handleLogin(platform, account.account_id);
      }
    } catch (error) {
      toast.error('账号创建失败');
    }
  };

  const handleDelete = async (accountID: string) => {
    if (confirm('确定要删除该账号吗？相关的浏览器缓存将被清除。')) {
      try {
        await deleteAccount(accountID);
        toast.success('账号删除成功');
      } catch (error) {
        toast.error('账号删除失败');
      }
    }
  };

  const platforms = [
    { id: 'deepseek', name: 'DeepSeek' },
    { id: 'doubao', name: '豆包' },
  ];

  return (
    <div className="p-6 space-y-6 relative">
      {isLoginOpen && (
        <div className="fixed inset-0 z-50 bg-background/80 backdrop-blur-sm flex items-center justify-center">
          <div className="bg-card border border-border p-6 rounded-lg shadow-lg max-w-md w-full text-center space-y-4">
            <h3 className="text-xl font-bold">正在登录...</h3>
            <p className="text-muted-foreground">
              请在弹出的浏览器窗口中完成登录操作。登录完成后，请点击下方按钮确认。
            </p>
            <div className="flex justify-center gap-4 pt-4">
              <button
                onClick={handleConfirmLogin}
                className="px-6 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
              >
                我已完成登录，关闭浏览器
              </button>
            </div>
          </div>
        </div>
      )}

      <div className="flex justify-between items-center">
         <h1 className="text-2xl font-bold">账号管理</h1>
      </div>

      <div className="grid gap-6">
        {platforms.map((platform) => (
          <div key={platform.id} className="bg-card border border-border rounded-lg overflow-hidden">
            <div className="p-4 bg-muted/50 border-b border-border flex justify-between items-center">
              <div className="flex items-center gap-2">
                <Shield className="w-5 h-5" />
                <span className="font-semibold">{platform.name}</span>
              </div>
              <button
                onClick={() => setAddingPlatform(platform.id)}
                className="text-sm flex items-center gap-1 hover:text-primary transition-colors"
                disabled={loading[`${platform.id}_create`]}
              >
                <Plus className="w-4 h-4" />
                添加账号
              </button>
            </div>
            
            {addingPlatform === platform.id && (
              <div className="p-4 border-b border-border bg-accent/20 flex gap-2 items-center">
                <input 
                  autoFocus
                  placeholder="请输入账号备注（如：主账号）"
                  className="flex-1 px-3 py-1.5 rounded-md border border-input bg-background text-sm"
                  value={newAccountName}
                  onChange={e => setNewAccountName(e.target.value)}
                  onKeyDown={e => e.key === 'Enter' && handleAddAccount(platform.id)}
                />
                <button 
                  onClick={() => handleAddAccount(platform.id)}
                  className="px-3 py-1.5 bg-primary text-primary-foreground text-sm rounded-md"
                  disabled={loading[`${platform.id}_create`]}
                >
                  确定
                </button>
                <button 
                  onClick={() => { setAddingPlatform(null); setNewAccountName(''); }}
                  className="px-3 py-1.5 border border-input text-sm rounded-md hover:bg-accent"
                >
                  取消
                </button>
              </div>
            )}

            <div className="divide-y divide-border">
              {accountsByPlatform[platform.id as 'deepseek'|'doubao']?.length === 0 ? (
                <div className="p-8 text-center text-muted-foreground text-sm">
                  暂无账号，请添加
                </div>
              ) : (
                accountsByPlatform[platform.id as 'deepseek'|'doubao']?.map(account => {
                    const isActive = activeAccounts[platform.id as 'deepseek'|'doubao']?.account_id === account.account_id;
                    return (
                        <div key={account.account_id} className={`p-4 flex items-center justify-between hover:bg-accent/50 transition-colors ${isActive ? 'bg-accent/10' : ''}`}>
                            <div className="flex items-center gap-3">
                                <button 
                                    onClick={() => !isActive && setActiveAccount(platform.id as any, account.account_id)}
                                    className={`transition-colors ${isActive ? 'text-green-500 cursor-default' : 'text-muted-foreground hover:text-primary'}`}
                                    title={isActive ? '当前激活' : '设为激活'}
                                >
                                    {isActive ? <CheckCircle2 className="w-5 h-5" /> : <Circle className="w-5 h-5" />}
                                </button>
                                <div>
                                    <div className="font-medium flex items-center gap-2">
                                        {account.account_name}
                                        {isActive && <span className="text-xs bg-green-500/10 text-green-600 px-1.5 py-0.5 rounded">使用中</span>}
                                    </div>
                                    <div className="text-xs text-muted-foreground font-mono mt-0.5">ID: {account.account_id.slice(0, 8)}...</div>
                                </div>
                            </div>
                            <div className="flex items-center gap-2">
                                <button 
                                    onClick={() => handleLogin(platform.id, account.account_id)}
                                    className="p-2 text-muted-foreground hover:text-primary hover:bg-primary/10 rounded-md transition-colors"
                                    title="打开浏览器登录"
                                    disabled={loading[`login_${account.account_id}`]}
                                >
                                    <ExternalLink className="w-4 h-4" />
                                </button>
                                <button 
                                    onClick={() => handleDelete(account.account_id)}
                                    className="p-2 text-muted-foreground hover:text-destructive hover:bg-destructive/10 rounded-md transition-colors"
                                    title="删除账号"
                                >
                                    <Trash2 className="w-4 h-4" />
                                </button>
                            </div>
                        </div>
                    );
                })
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
