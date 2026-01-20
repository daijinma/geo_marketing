import { useAuthStore } from '@/stores/authStore';
import { useNavigate } from 'react-router-dom';
import { LogOut, Settings } from 'lucide-react';
import { Link } from 'react-router-dom';

export default function Header() {
  const navigate = useNavigate();
  const { clearToken } = useAuthStore();

  const handleLogout = async () => {
    await clearToken();
    navigate('/login');
  };

  return (
    <header className="h-14 border-b border-border bg-card flex items-center justify-between px-4">
      <div className="flex items-center gap-4">
        <h1 className="text-lg font-bold"></h1>
      </div>
      
      <div className="flex items-center gap-3">
        <Link
          to="/settings"
          className="p-2 hover:bg-accent rounded-md transition-colors"
          title="设置"
        >
          <Settings className="w-4 h-4" />
        </Link>
        <button
          onClick={handleLogout}
          className="p-2 hover:bg-accent rounded-md transition-colors text-muted-foreground"
          title="退出登录"
        >
          <LogOut className="w-4 h-4" />
        </button>
      </div>
    </header>
  );
}
