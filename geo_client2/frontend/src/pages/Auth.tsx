import { useEffect, useState } from 'react';
import { useAccountStore } from '@/stores/accountStore';
import { wailsAPI } from '@/utils/wails-api';
import { Shield, Plus, Trash2, CheckCircle2, Circle, LogIn, ExternalLink, AlertTriangle } from 'lucide-react';
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

  useEffect(() => {
    loadAccounts('deepseek');
    loadActiveAccount('deepseek');
    loadAccounts('doubao');
    loadActiveAccount('doubao');
    loadAccounts('xiaohongshu');
    loadActiveAccount('xiaohongshu');
    loadAccounts('yiyan');
    loadActiveAccount('yiyan');
    loadAccounts('yuanbao');
    loadActiveAccount('yuanbao');
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

  const handleCancelEdit = () => {
    setEditingAccountID(null);
    setEditingName('');
  };

  const largeModelPlatforms = [
    { id: 'deepseek', name: 'DeepSeek' },
    { id: 'doubao', name: '豆包' },
    { id: 'yiyan', name: '文心一言' },
    { id: 'yuanbao', name: '腾讯元宝' },
  ];

  const publisherPlatforms = [
    { id: 'xiaohongshu', name: '小红书' },
  ];

  const renderPlatformCard = (platform: { id: string, name: string }) => (
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
        {accountsByPlatform[platform.id as PlatformKey]?.length === 0 ? (
          <div className="p-8 text-center text-muted-foreground text-sm">
            暂无账号，请添加
          </div>
        ) : (
          accountsByPlatform[platform.id as PlatformKey]?.map(account => {
              const isActive = activeAccounts[platform.id as PlatformKey]?.account_id === account.account_id;
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
                              <div 
                                className="font-medium flex items-center gap-2 cursor-pointer" 
                                onDoubleClick={() => handleStartEdit(account)}
                                title="双击修改备注"
                              >
                                  {editingAccountID === account.account_id ? (
                                    <input 
                                      autoFocus
                                      className="px-2 py-0.5 rounded border border-input bg-background text-sm w-32"
                                      value={editingName}
                                      onChange={(e) => setEditingName(e.target.value)}
                                      onBlur={handleSaveEdit}
                                      onKeyDown={(e) => {
                                        if (e.key === 'Enter') handleSaveEdit();
                                        if (e.key === 'Escape') handleCancelEdit();
                                      }}
                                      onClick={(e) => e.stopPropagation()}
                                    />
                                  ) : (
                                    <>
                                      {account.account_name}
                                      {isActive && <span className="text-xs bg-green-500/10 text-green-600 px-1.5 py-0.5 rounded">使用中</span>}
                                    </>
                                  )}
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
                              onClick={() => setDeleteConfirm({accountID: account.account_id, accountName: account.account_name})}
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
  );

  type PlatformKey = 'deepseek' | 'doubao' | 'xiaohongshu' | 'yiyan' | 'yuanbao';

  return (
    <div className="p-6 space-y-6 relative">
      {deleteConfirm && (
        <div className="fixed inset-0 z-50 bg-background/80 backdrop-blur-sm flex items-center justify-center">
          <div className="bg-card border border-border p-6 rounded-lg shadow-lg max-w-md w-full space-y-4">
            <div className="flex items-start gap-3">
              <AlertTriangle className="w-6 h-6 text-destructive mt-0.5" />
              <div className="flex-1">
                <h3 className="text-lg font-bold">确认删除账号</h3>
                <p className="text-sm text-muted-foreground mt-2">
                  确定要删除账号 <span className="font-semibold text-foreground">{deleteConfirm.accountName}</span> 吗？
                </p>
                <p className="text-sm text-muted-foreground mt-1">
                  此操作将：
                </p>
                <ul className="text-sm text-muted-foreground mt-1 list-disc list-inside">
                  <li>删除账号数据</li>
                  <li>清除浏览器缓存</li>
                  <li>清除登录会话</li>
                </ul>
                <p className="text-sm text-destructive font-semibold mt-2">
                  此操作不可恢复！
                </p>
              </div>
            </div>
            <div className="flex justify-end gap-3 pt-2">
              <button
                onClick={() => setDeleteConfirm(null)}
                className="px-4 py-2 border border-input rounded-md hover:bg-accent"
              >
                取消
              </button>
              <button
                onClick={() => handleDelete(deleteConfirm.accountID)}
                className="px-4 py-2 bg-destructive text-destructive-foreground rounded-md hover:bg-destructive/90"
              >
                确认删除
              </button>
            </div>
          </div>
        </div>
      )}

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

      <div className="space-y-6">
        <div>
          <h2 className="text-lg font-semibold mb-3">大模型类</h2>
          <div className="grid gap-6">
            {largeModelPlatforms.map(renderPlatformCard)}
          </div>
        </div>

        <div>
          <h2 className="text-lg font-semibold mb-3">平台发布类</h2>
          <div className="grid gap-6">
            {publisherPlatforms.map(renderPlatformCard)}
          </div>
        </div>
      </div>
    </div>
  );
}
