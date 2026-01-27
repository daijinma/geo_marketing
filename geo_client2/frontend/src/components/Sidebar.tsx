import { Link, useLocation } from 'react-router-dom';
import {
  LayoutDashboard,
  Shield,
  ListTodo,
  FileText,
  Search,
} from 'lucide-react';
import { cn } from '@/lib/utils';

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
    <div className="w-[280px] border-r bg-card flex flex-col h-full shadow-sm">
      <div className="flex-1 overflow-auto p-4">
        <nav className="space-y-1">
          {menuItems.map((item) => {
            const Icon = item.icon;
            const isActive = location.pathname === item.path;
            return (
              <Link
                key={item.path}
                to={item.path}
                className={cn(
                  "flex items-center w-full gap-3 h-11 px-4 font-normal rounded-md transition-colors",
                  isActive 
                    ? "bg-secondary text-secondary-foreground font-medium" 
                    : "hover:bg-accent hover:text-accent-foreground"
                )}
              >
                <Icon className="w-4 h-4" />
                <span>{item.label}</span>
              </Link>
            );
          })}
        </nav>
      </div>
    </div>
  );
}
