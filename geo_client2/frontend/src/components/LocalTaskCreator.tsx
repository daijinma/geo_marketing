import { useState } from 'react';
import { X } from 'lucide-react';
import { toast } from 'sonner';
import { wailsAPI } from '@/utils/wails-api';

interface LocalTaskCreatorProps {
  onClose: () => void;
  onCreated?: () => void;
}

export function LocalTaskCreator({ onClose, onCreated }: LocalTaskCreatorProps) {
  const [keywords, setKeywords] = useState('');
  const [platforms, setPlatforms] = useState<string[]>(['deepseek']);
  const [queryCount, setQueryCount] = useState(1);
  const [creating, setCreating] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const keywordList = keywords.split('\n').map(k => k.trim()).filter(k => k);
    if (keywordList.length === 0) {
      toast.error('请至少输入一个关键词');
      return;
    }
    if (platforms.length === 0) {
      toast.error('请至少选择一个平台');
      return;
    }

    setCreating(true);
    try {
      for (const platform of platforms) {
        const response = await wailsAPI.search.checkLoginStatus(platform);
        if (!response.success || !response.isLoggedIn) {
          toast.error(`${platform} 未登录`);
          setCreating(false);
          return;
        }
      }

      const result = await wailsAPI.task.createLocalSearchTask({
        keywords: keywordList,
        platforms,
        query_count: queryCount,
      });

      if (result.success) {
        toast.success('任务已创建', { description: `任务ID: ${result.taskId}` });
        onCreated?.();
        onClose();
      }
    } catch (error: any) {
      toast.error('创建任务失败', { description: error.message });
    } finally {
      setCreating(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-card border border-border rounded-lg w-full max-w-2xl mx-4 max-h-[90vh] overflow-y-auto">
        <div className="p-6 border-b border-border flex items-center justify-between">
          <h2 className="text-xl font-bold">创建本地搜索任务</h2>
          <button onClick={onClose} className="p-2 hover:bg-accent rounded-md">
            <X className="w-5 h-5" />
          </button>
        </div>
        <form onSubmit={handleSubmit} className="p-6 space-y-4">
          <div>
            <label className="block text-sm font-medium mb-2">关键词（每行一个）</label>
            <textarea
              value={keywords}
              onChange={(e) => setKeywords(e.target.value)}
              className="w-full px-3 py-2 border border-border rounded-md bg-background"
              rows={5}
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-2">平台</label>
            <div className="flex gap-4">
              {['deepseek', 'doubao'].map((p) => (
                <label key={p} className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    checked={platforms.includes(p)}
                    onChange={(e) => {
                      if (e.target.checked) {
                        setPlatforms([...platforms, p]);
                      } else {
                        setPlatforms(platforms.filter(x => x !== p));
                      }
                    }}
                  />
                  <span>{p === 'deepseek' ? 'DeepSeek' : '豆包'}</span>
                </label>
              ))}
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium mb-2">查询次数</label>
            <input
              type="number"
              min={1}
              max={10}
              value={queryCount}
              onChange={(e) => setQueryCount(parseInt(e.target.value) || 1)}
              className="w-32 px-3 py-2 border border-border rounded-md bg-background"
            />
          </div>
          <div className="flex gap-2 justify-end">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 border border-border rounded-md hover:bg-accent"
            >
              取消
            </button>
            <button
              type="submit"
              disabled={creating}
              className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 disabled:opacity-50"
            >
              {creating ? '创建中...' : '创建任务'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
