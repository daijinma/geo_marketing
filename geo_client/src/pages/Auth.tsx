import LoginList from '@/components/LoginList';
import TaskQueue from '@/components/TaskQueue';
import { useState, useEffect } from 'react';
import { CheckCircle2 } from 'lucide-react';

export default function Auth() {
  const [authorizedPlatforms, setAuthorizedPlatforms] = useState<Array<{
    platform_name: string;
    platform_type: string;
  }>>([]);

  useEffect(() => {
    loadAuthorizedPlatforms();
  }, []);

  const loadAuthorizedPlatforms = async () => {
    try {
      const response = await window.electronAPI.task.getAuthorizedPlatforms();
      if (response.success && response.platforms) {
        setAuthorizedPlatforms(response.platforms);
      }
    } catch (error) {
      console.error('加载已授权平台失败:', error);
    }
  };

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold mb-2">授权列表</h1>
        <p className="text-muted-foreground">管理平台登录状态</p>
      </div>

      <div className="grid grid-cols-2 gap-6">
        {/* 登录列表 */}
        <div className="bg-card border border-border rounded-lg p-4">
          <LoginList />
        </div>

        {/* 任务队列 */}
        <div className="bg-card border border-border rounded-lg p-4">
          <TaskQueue />
        </div>
      </div>

      {/* 已授权平台 */}
      <div className="bg-card border border-border rounded-lg p-6">
        <h2 className="text-lg font-semibold mb-4">已授权的平台</h2>
        {authorizedPlatforms.length === 0 ? (
          <p className="text-muted-foreground text-sm">暂无已授权的平台</p>
        ) : (
          <div className="grid grid-cols-3 gap-4">
            {authorizedPlatforms.map((platform) => (
              <div
                key={platform.platform_name}
                className="flex items-center gap-2 p-3 border border-border rounded-md bg-green-500/10"
              >
                <CheckCircle2 className="w-5 h-5 text-green-500" />
                <div>
                  <div className="font-medium">{platform.platform_name}</div>
                  <div className="text-xs text-muted-foreground">
                    {platform.platform_type === 'llm' ? '大模型' : '平台'}
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
