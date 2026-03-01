import { useState, useEffect } from 'react';
import { Switch } from '@/components/ui/switch';
import { wailsAPI } from '@/utils/wails-api';
import { toast } from 'sonner';
import { ScrapeFlowManager } from '@/components/ScrapeFlowManager';

export default function Settings() {
  const [headless, setHeadless] = useState(true);
  const [aiPublishEnabled, setAIPublishEnabled] = useState(true);
  const [aiBaseURL, setAIBaseURL] = useState('');
  const [aiApiKey, setAIApiKey] = useState('');

  useEffect(() => {
    loadSettings();
  }, []);

  const loadSettings = async () => {
    try {
      const value = await wailsAPI.settings.get('browser_headless');
      setHeadless(value !== 'false'); // Default to true if not set or not 'false'

      const aiEnabledValue = await wailsAPI.settings.get('ai_publish_assist');
      setAIPublishEnabled(aiEnabledValue !== 'false');

      const baseURL = await wailsAPI.settings.get('ai_publish_base_url');
      const apiKey = await wailsAPI.settings.get('ai_publish_api_key');
      setAIBaseURL(baseURL || '');
      setAIApiKey(apiKey || '');
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

  const handleAIPublishToggle = async (checked: boolean) => {
    setAIPublishEnabled(checked);
    try {
      await wailsAPI.settings.set('ai_publish_assist', checked.toString());
    } catch (error) {
      console.error('Save AI publish setting failed', error);
    }
  };

  const handleSaveAIPublishConfig = async () => {
    try {
      await wailsAPI.aiPublish.setConfig(aiBaseURL.trim(), aiApiKey.trim());
      toast.success('AI 发布辅助配置保存成功');
    } catch (error) {
      console.error('Save AI publish config failed', error);
      toast.error('AI 发布辅助配置保存失败');
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
            <h3 className="text-2xl font-semibold leading-none tracking-tight">AI 发布辅助</h3>
            <p className="text-sm text-muted-foreground">调用 Dify Workflow 获取自动化操作指令。</p>
          </div>
          <div className="p-6 pt-0 space-y-6">
            <div className="flex items-center justify-between space-x-2">
              <div className="space-y-1">
                <label htmlFor="ai-publish-assist" className="text-base font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">AI 发布辅助</label>
                <p className="text-sm text-muted-foreground">
                  默认开启。关闭后发布流程将回退为人工操作。
                </p>
              </div>
              <Switch
                id="ai-publish-assist"
                checked={aiPublishEnabled}
                onCheckedChange={handleAIPublishToggle}
              />
            </div>

            <div className="grid grid-cols-1 gap-4">
              <div className="space-y-1">
                <label className="text-sm font-medium">Dify Base URL</label>
                <input
                  value={aiBaseURL}
                  onChange={(e) => setAIBaseURL(e.target.value)}
                  placeholder="http://levi.duanjieai.com/v1"
                  className="w-full px-3 py-2 rounded-md border border-input bg-background text-sm"
                />
              </div>
              <div className="space-y-1">
                <label className="text-sm font-medium">Dify API Key</label>
                <input
                  value={aiApiKey}
                  onChange={(e) => setAIApiKey(e.target.value)}
                  placeholder="app-xxxxxxxxxxxxxxxx"
                  className="w-full px-3 py-2 rounded-md border border-input bg-background text-sm"
                />
              </div>
              <div className="flex justify-end">
                <button
                  onClick={handleSaveAIPublishConfig}
                  className="px-4 py-2 text-sm rounded-md bg-primary text-primary-foreground hover:bg-primary/90"
                >
                  保存配置
                </button>
              </div>
            </div>
          </div>
        </div>

        <div className="rounded-lg border bg-card text-card-foreground shadow-sm">
            <div className="p-6">
                <ScrapeFlowManager />
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
