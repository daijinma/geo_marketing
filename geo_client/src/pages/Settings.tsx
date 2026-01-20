import { useNavigate } from 'react-router-dom';
import { LogOut, User, Settings as SettingsIcon } from 'lucide-react';
import { useAuthStore } from '@/stores/authStore';

export default function Settings() {
  const navigate = useNavigate();
  const { clearToken } = useAuthStore();

  const handleLogout = async () => {
    await clearToken();
    navigate('/login');
  };

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold mb-2 flex items-center gap-2">
          <SettingsIcon className="w-6 h-6" />
          设置
        </h1>
        <p className="text-muted-foreground">系统设置和账户管理</p>
      </div>

      {/* 账户信息 */}
      <div className="p-6 bg-card border border-border rounded-lg space-y-4">
        <h2 className="text-lg font-semibold flex items-center gap-2">
          <User className="w-5 h-5" />
          账户信息
        </h2>
        <div className="space-y-3">
          <div>
            <label className="text-sm text-muted-foreground">状态</label>
            <p className="text-base font-medium mt-1">已登录</p>
          </div>
        </div>
      </div>

      {/* 账户操作 */}
      <div className="p-6 bg-card border border-border rounded-lg space-y-4">
        <h2 className="text-lg font-semibold">账户操作</h2>
        <button
          onClick={handleLogout}
          className="flex items-center gap-2 px-4 py-2 bg-destructive text-destructive-foreground rounded-md hover:bg-destructive/90 transition-colors"
        >
          <LogOut className="w-4 h-4" />
          <span>退出登录</span>
        </button>
      </div>
    </div>
  );
}
