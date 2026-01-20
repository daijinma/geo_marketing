import { Link, useLocation, useNavigate } from 'react-router-dom';
import { 
  LayoutDashboard, 
  Search, 
  ListTodo, 
  FileText,
  LogOut
} from 'lucide-react';
import { useAuthStore } from '@/stores/authStore';
import LoginList from './LoginList';
import TaskQueue from './TaskQueue';

export default function Sidebar() {
  const location = useLocation();
  const navigate = useNavigate();
  const { clearToken } = useAuthStore();

  const handleLogout = () => {
    clearToken();
    navigate('/login');
  };

  const menuItems = [
    { path: '/', icon: LayoutDashboard, label: '仪表板' },
    { path: '/search', icon: Search, label: '关键词搜索' },
    { path: '/tasks', icon: ListTodo, label: '任务列表' },
    { path: '/logs', icon: FileText, label: '日志查看' },
  ];

  return (
    <div className="w-[280px] border-r border-border bg-card flex flex-col">
      {/* Logo/Header */}
      <div className="p-4 border-b border-border">
        <h1 className="text-xl font-bold">端界GEO</h1>
      </div>

      {/* 登录列表 */}
      <div className="flex-1 overflow-auto p-4 space-y-4">
        <LoginList />

        {/* 任务队列 */}
        <TaskQueue />

        {/* 本地功能 */}
        <div className="pt-4 border-t border-border">
          <Link
            to="/search"
            className={`flex items-center gap-2 px-3 py-2 rounded-md transition-colors ${
              location.pathname === '/search'
                ? 'bg-primary text-primary-foreground'
                : 'hover:bg-accent'
            }`}
          >
            <Search className="w-4 h-4" />
            <span>关键词搜索</span>
          </Link>
        </div>
      </div>

      {/* 底部导航 */}
      <div className="p-4 border-t border-border space-y-2">
        {menuItems.map((item) => {
          const Icon = item.icon;
          const isActive = location.pathname === item.path;
          return (
            <Link
              key={item.path}
              to={item.path}
              className={`flex items-center gap-2 px-3 py-2 rounded-md transition-colors ${
                isActive
                  ? 'bg-primary text-primary-foreground'
                  : 'hover:bg-accent'
              }`}
            >
              <Icon className="w-4 h-4" />
              <span>{item.label}</span>
            </Link>
          );
        })}
        <button
          onClick={handleLogout}
          className="w-full flex items-center gap-2 px-3 py-2 rounded-md transition-colors hover:bg-accent text-muted-foreground"
        >
          <LogOut className="w-4 h-4" />
          <span>退出登录</span>
        </button>
      </div>
    </div>
  );
}
