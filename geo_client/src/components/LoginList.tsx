import { CheckCircle2, Circle, RefreshCw } from 'lucide-react';
import { useLoginStatusStore } from '@/stores/loginStatusStore';
import { useState, useEffect } from 'react';

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
  const { isLoggedIn, updateLoginStatus } = useLoginStatusStore();
  const [loading, setLoading] = useState<string | null>(null);
  const [authorizedPlatforms, setAuthorizedPlatforms] = useState<Set<string>>(new Set());
  const [loginInProgress, setLoginInProgress] = useState<string | null>(null);

  // 加载已授权平台
  useEffect(() => {
    loadAuthorizedPlatforms();
  }, []);

  const loadAuthorizedPlatforms = async () => {
    try {
      const response = await window.electronAPI.task.getAuthorizedPlatforms();
      if (response.success && response.platforms) {
        const platformNames = new Set(response.platforms.map((p) => p.platform_name));
        setAuthorizedPlatforms(platformNames);
        
        // 更新登录状态store
        response.platforms.forEach((p) => {
          updateLoginStatus(p.platform_name as any, p.is_logged_in);
        });
      }
    } catch (error) {
      console.error('加载已授权平台失败:', error);
    }
  };

  const handleLogin = async (platformName: string) => {
    setLoading(platformName);
    try {
      console.log(`打开 ${platformName} 登录窗口`);
      const response = await window.electronAPI.provider.openLogin(platformName);
      if (response.success) {
        setLoginInProgress(platformName);
        console.log(`${platformName} 登录窗口已打开`);
      } else {
        console.error('打开登录窗口失败:', response.error);
        alert(`打开登录窗口失败: ${response.error}`);
      }
    } catch (error) {
      console.error('登录失败:', error);
      alert(`登录失败: ${error}`);
    } finally {
      setLoading(null);
    }
  };

  const handleCompleteLogin = async (platformName: string) => {
    setLoading(platformName);
    try {
      console.log(`检查 ${platformName} 登录状态`);
      const response = await window.electronAPI.provider.checkLoginAfterAuth(platformName);
      if (response.success) {
        updateLoginStatus(platformName as any, response.isLoggedIn);
        if (response.isLoggedIn) {
          setAuthorizedPlatforms((prev) => new Set([...prev, platformName]));
          alert(`${platformName} 登录成功！`);
        } else {
          setAuthorizedPlatforms((prev) => {
            const newSet = new Set(prev);
            newSet.delete(platformName);
            return newSet;
          });
          alert(`${platformName} 未登录，请先完成登录操作`);
        }
        setLoginInProgress(null);
        // 重新加载授权平台列表
        await loadAuthorizedPlatforms();
      } else {
        console.error('检查登录状态失败:', response.error);
        alert(`检查登录状态失败: ${response.error}`);
      }
    } catch (error) {
      console.error('完成登录失败:', error);
      alert(`完成登录失败: ${error}`);
    } finally {
      setLoading(null);
    }
  };

  const handleCancelLogin = async () => {
    try {
      await window.electronAPI.provider.closeLoginView();
      setLoginInProgress(null);
      console.log('已取消登录');
    } catch (error) {
      console.error('取消登录失败:', error);
    }
  };

  const handleCheckStatus = async (platformName: string) => {
    setLoading(platformName);
    try {
      const response = await window.electronAPI.search.checkLoginStatus(platformName);
      if (response.success && response.isLoggedIn !== undefined) {
        updateLoginStatus(platformName as any, response.isLoggedIn);
        if (response.isLoggedIn) {
          setAuthorizedPlatforms((prev) => new Set([...prev, platformName]));
        } else {
          setAuthorizedPlatforms((prev) => {
            const newSet = new Set(prev);
            newSet.delete(platformName);
            return newSet;
          });
        }
      }
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
                  {loginInProgress === platform.name ? (
                    <>
                      <button
                        onClick={() => handleCompleteLogin(platform.name)}
                        disabled={isLoading}
                        className="text-xs px-2 py-1 bg-green-600 text-white rounded hover:bg-green-700 transition-colors"
                      >
                        完成登录
                      </button>
                      <button
                        onClick={handleCancelLogin}
                        disabled={isLoading}
                        className="text-xs px-2 py-1 bg-gray-500 text-white rounded hover:bg-gray-600 transition-colors"
                      >
                        取消
                      </button>
                    </>
                  ) : (
                    <>
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
                    </>
                  )}
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
