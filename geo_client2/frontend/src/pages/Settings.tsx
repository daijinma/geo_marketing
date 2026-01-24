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
    <div className="p-6 space-y-6">
      <h1 className="text-2xl font-bold">设置</h1>
      <div className="space-y-4">
        <div className="p-4 bg-card border border-border rounded-lg">
          <h2 className="text-lg font-semibold mb-4">浏览器设置</h2>
          <div className="flex items-center justify-between">
            <div>
              <div className="font-medium">无头模式</div>
              <div className="text-sm text-muted-foreground">启用后浏览器将在后台运行</div>
            </div>
            <Switch checked={headless} onCheckedChange={handleHeadlessChange} />
          </div>
        </div>
      </div>
    </div>
  );
}
