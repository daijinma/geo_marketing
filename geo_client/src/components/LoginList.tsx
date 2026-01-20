import { CheckCircle2, Circle, RefreshCw } from 'lucide-react';
import { useLoginStatusStore } from '@/stores/loginStatusStore';
import { useState } from 'react';

interface Platform {
  type: 'llm' | 'platform';
  name: string;
  label: string;
}

const platforms: Platform[] = [
  { type: 'llm', name: 'deepseek', label: 'DeepSeek' },
  { type: 'llm', name: 'doubao', label: '豆包' },
  { type: 'platform', name: 'netease', label: '网易' },
  { type: 'platform', name: 'cnblogs', label: 'cnblogs' },
];

export default function LoginList() {
  const { isLoggedIn } = useLoginStatusStore();
  const [loading, setLoading] = useState<string | null>(null);

  const handleLogin = async (platformName: string) => {
    setLoading(platformName);
    try {
      // TODO: 调用Tauri命令打开浏览器登录
      // await invoke('open_login', { platform: platformName });
      console.log(`打开 ${platformName} 登录窗口`);
    } catch (error) {
      console.error('登录失败:', error);
    } finally {
      setLoading(null);
    }
  };

  const handleCheckStatus = async (platformName: string) => {
    setLoading(platformName);
    try {
      // TODO: 调用Tauri命令检查登录状态
      // const status = await invoke('check_login_status', { platform: platformName });
      console.log(`检查 ${platformName} 登录状态`);
    } catch (error) {
      console.error('检查状态失败:', error);
    } finally {
      setLoading(null);
    }
  };

  const llmPlatforms = platforms.filter((p) => p.type === 'llm');
  const platformPlatforms = platforms.filter((p) => p.type === 'platform');

  return (
    <div className="space-y-4">
      {/* 网页大模型 */}
      <div>
        <h3 className="text-sm font-semibold text-muted-foreground mb-2">网页大模型</h3>
        <div className="space-y-1">
          {llmPlatforms.map((platform) => {
            const loggedIn = isLoggedIn(platform.name as any);
            const isLoading = loading === platform.name;
            return (
              <div
                key={platform.name}
                className="flex items-center justify-between p-2 rounded-md hover:bg-accent/50 transition-colors"
              >
                <div className="flex items-center gap-2">
                  {loggedIn ? (
                    <CheckCircle2 className="w-4 h-4 text-green-500" />
                  ) : (
                    <Circle className="w-4 h-4 text-muted-foreground" />
                  )}
                  <span className="text-sm">{platform.label}</span>
                </div>
                <div className="flex items-center gap-1">
                  <button
                    onClick={() => handleCheckStatus(platform.name)}
                    disabled={isLoading}
                    className="p-1 hover:bg-accent rounded transition-colors"
                    title="刷新状态"
                  >
                    <RefreshCw
                      className={`w-3 h-3 ${isLoading ? 'animate-spin' : ''}`}
                    />
                  </button>
                  <button
                    onClick={() => handleLogin(platform.name)}
                    disabled={isLoading}
                    className="text-xs px-2 py-1 bg-primary text-primary-foreground rounded hover:bg-primary/90 transition-colors"
                  >
                    {loggedIn ? '重新登录' : '登录'}
                  </button>
                </div>
              </div>
            );
          })}
        </div>
      </div>

      {/* 网页平台 */}
      <div>
        <h3 className="text-sm font-semibold text-muted-foreground mb-2">网页平台</h3>
        <div className="space-y-1">
          {platformPlatforms.map((platform) => {
            const loggedIn = isLoggedIn(platform.name as any);
            const isLoading = loading === platform.name;
            return (
              <div
                key={platform.name}
                className="flex items-center justify-between p-2 rounded-md hover:bg-accent/50 transition-colors"
              >
                <div className="flex items-center gap-2">
                  {loggedIn ? (
                    <CheckCircle2 className="w-4 h-4 text-green-500" />
                  ) : (
                    <Circle className="w-4 h-4 text-muted-foreground" />
                  )}
                  <span className="text-sm">{platform.label}</span>
                </div>
                <div className="flex items-center gap-1">
                  <button
                    onClick={() => handleCheckStatus(platform.name)}
                    disabled={isLoading}
                    className="p-1 hover:bg-accent rounded transition-colors"
                    title="刷新状态"
                  >
                    <RefreshCw
                      className={`w-3 h-3 ${isLoading ? 'animate-spin' : ''}`}
                    />
                  </button>
                  <button
                    onClick={() => handleLogin(platform.name)}
                    disabled={isLoading}
                    className="text-xs px-2 py-1 bg-primary text-primary-foreground rounded hover:bg-primary/90 transition-colors opacity-50 cursor-not-allowed"
                    title="功能占位，暂未实现"
                  >
                    登录
                  </button>
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
