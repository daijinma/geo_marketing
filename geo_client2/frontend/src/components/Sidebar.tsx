import { Link, useLocation } from 'react-router-dom';
import {
  LayoutDashboard,
  Shield,
  ListTodo,
  FileText,
  Search,
} from 'lucide-react';

export default function Sidebar() {
  const location = useLocation();

  const menuItems = [
    { path: '/', icon: LayoutDashboard, label: '首页' },
    { path: '/search', icon: Search, label: '大模型搜索' },
    { path: '/auth', icon: Shield, label: '授权列表' },
    { path: '/tasks', icon: ListTodo, label: '任务列表' },
    { path: '/logs', icon: FileText, label: '系统控制台' },
  ];

  return (
    <div className="w-[280px] border-r border-border bg-card flex flex-col">
      <div className="flex-1 overflow-auto p-4">
        <nav className="space-y-2">
          {menuItems.map((item) => {
            const Icon = item.icon;
            const isActive = location.pathname === item.path;
            return (
              <Link
                key={item.path}
                to={item.path}
                className={`flex items-center gap-3 px-4 py-3 rounded-md transition-colors ${
                  isActive
                    ? 'bg-primary text-primary-foreground'
                    : 'hover:bg-accent'
                }`}
              >
                <Icon className="w-5 h-5" />
                <span className="font-medium">{item.label}</span>
              </Link>
            );
          })}
        </nav>
      </div>
    </div>
  );
}
