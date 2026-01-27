import { useState, useEffect } from 'react';
import { Switch } from '@/components/ui/switch';
import { wailsAPI } from '@/utils/wails-api';

export default function Settings() {
  const [headless, setHeadless] = useState(true);

  useEffect(() => {
    loadSettings();
  }, []);

  const loadSettings = async () => {
    try {
      const value = await wailsAPI.settings.get('browser_headless');
      setHeadless(value !== 'false'); // Default to true if not set or not 'false'
    } catch (error) {
      console.error('Load settings failed', error);
    }
  };

  const handleHeadlessChange = async (checked: boolean) => {
    setHeadless(checked);
    try {
      await wailsAPI.settings.set('browser_headless', checked.toString());
    } catch (error) {
      console.error('Save settings failed', error);
    }
  };

  return (
    <div className="p-8 space-y-8 max-w-4xl mx-auto">
      <div className="space-y-2">
        <h2 className="text-3xl font-bold tracking-tight">系统设置</h2>
        <p className="text-muted-foreground">管理应用行为和偏好设置。</p>
      </div>
      
      <div className="space-y-6">
        <div className="rounded-lg border bg-card text-card-foreground shadow-sm">
          <div className="flex flex-col space-y-1.5 p-6">
            <h3 className="text-2xl font-semibold leading-none tracking-tight">浏览器配置</h3>
            <p className="text-sm text-muted-foreground">控制后台浏览器的运行方式。</p>
          </div>
          <div className="p-6 pt-0 space-y-6">
            <div className="flex items-center justify-between space-x-2">
                <div className="space-y-1">
                    <label htmlFor="headless-mode" className="text-base font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">无头模式 (Headless)</label>
                    <p className="text-sm text-muted-foreground">
                        启用后浏览器将在后台静默运行，不会显示操作界面。
                        <br/>
                        <span className="text-xs opacity-70">注: 部分平台登录可能需要关闭此选项。</span>
                    </p>
                </div>
                <Switch 
                    id="headless-mode"
                    checked={headless} 
                    onCheckedChange={handleHeadlessChange} 
                />
            </div>
          </div>
        </div>

        <div className="rounded-lg border bg-card text-card-foreground shadow-sm">
            <div className="flex flex-col space-y-1.5 p-6">
                <h3 className="text-2xl font-semibold leading-none tracking-tight">关于应用</h3>
            </div>
            <div className="p-6 pt-0">
                <div className="text-sm text-muted-foreground">
                    <p>当前版本: v0.1.0</p>
                    <p>Powered by Wails + React</p>
                </div>
            </div>
        </div>
      </div>
    </div>
  );
}
