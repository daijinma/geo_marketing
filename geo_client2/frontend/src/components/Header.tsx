import { Link } from 'react-router-dom';
import { Settings } from 'lucide-react';

export default function Header() {
  return (
    <header className="h-14 border-b border-border bg-card flex items-center justify-between px-4">
      <div className="flex items-center gap-4">
        <h1 className="text-lg font-bold">端界 GEO</h1>
      </div>
      <div className="flex items-center gap-3">
        <Link
          to="/settings"
          className="p-2 hover:bg-accent rounded-md transition-colors"
          title="设置"
        >
          <Settings className="w-4 h-4" />
        </Link>
      </div>
    </header>
  );
}
