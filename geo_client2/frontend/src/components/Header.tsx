import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { Settings } from 'lucide-react';
import { wailsAPI } from '@/utils/wails-api';
import pkg from '../../package.json';

export default function Header() {
  const [buildTime, setBuildTime] = useState<string>('');

  useEffect(() => {
    wailsAPI.version.get().then(info => {
      if (info.buildTime && info.buildTime !== 'unknown') {
        setBuildTime(info.buildTime);
      }
    });
  }, []);

  return (
    <header className="h-14 border-b border-border bg-card flex items-center justify-between px-4">
      <div className="flex items-center gap-4">
        <div className="flex items-baseline gap-2">
          <h1 className="text-lg font-bold">端界 GEO</h1>
          <span className="text-[10px] text-muted-foreground font-mono">
            v{pkg.version}
            {buildTime && (
              <span className="ml-1 opacity-70">({buildTime})</span>
            )}
          </span>
        </div>
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
