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
    <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 backdrop-blur-sm animate-in fade-in duration-200">
      <div className="bg-card border border-border rounded-xl w-full max-w-2xl mx-4 max-h-[90vh] overflow-y-auto shadow-2xl animate-in zoom-in-95 duration-200 scale-100">
        <div className="p-6 border-b border-border/50 flex items-center justify-between sticky top-0 bg-card/95 backdrop-blur z-10">
          <div>
            <h2 className="text-xl font-bold tracking-tight">创建本地搜索任务</h2>
            <p className="text-sm text-muted-foreground">设置关键词与目标平台，启动新的自动化搜索。</p>
          </div>
          <button onClick={onClose} className="p-2 hover:bg-accent rounded-full text-muted-foreground hover:text-foreground transition-colors">
            <X className="w-5 h-5" />
          </button>
        </div>
        <form onSubmit={handleSubmit} className="p-6 space-y-6">
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <label className="text-sm font-semibold">关键词</label>
              <span className="text-xs text-muted-foreground bg-secondary px-2 py-0.5 rounded">每行一个</span>
            </div>
            <textarea
              value={keywords}
              onChange={(e) => setKeywords(e.target.value)}
              className="w-full min-h-[120px] px-3 py-2 border border-input rounded-lg bg-background focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all resize-y text-sm outline-none"
              rows={5}
              placeholder="输入搜索关键词..."
            />
          </div>

          <div className="space-y-3">
            <label className="text-sm font-semibold block">选择平台</label>
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
              {['deepseek', 'doubao', 'yiyan', 'yuanbao'].map((p) => {
                const isSelected = platforms.includes(p);
                return (
                  <label 
                    key={p} 
                    className={`
                      relative cursor-pointer border rounded-lg px-3 py-2.5 flex items-center gap-2 transition-all select-none
                      ${isSelected 
                        ? 'border-primary bg-primary/5 ring-1 ring-primary shadow-sm' 
                        : 'border-border hover:border-primary/50 hover:bg-accent/50'}
                    `}
                  >
                    <input
                      type="checkbox"
                      className="sr-only"
                      checked={isSelected}
                      onChange={(e) => {
                        if (e.target.checked) {
                          setPlatforms([...platforms, p]);
                        } else {
                          setPlatforms(platforms.filter(x => x !== p));
                        }
                      }}
                    />
                    <div className={`w-3.5 h-3.5 rounded-full border flex items-center justify-center transition-colors ${isSelected ? 'border-primary bg-primary' : 'border-muted-foreground/30'}`}>
                      {isSelected && <div className="w-1.5 h-1.5 rounded-full bg-primary-foreground" />}
                    </div>
                    <span className={`text-sm font-medium ${isSelected ? 'text-primary' : 'text-foreground'}`}>
                      {p === 'deepseek' ? 'DeepSeek' : 
                       p === 'doubao' ? '豆包' : 
                       p === 'yiyan' ? '文心一言' : '腾讯元宝'}
                    </span>
                  </label>
                );
              })}
            </div>
          </div>

          <div className="space-y-2">
            <label className="text-sm font-semibold block">查询深度</label>
            <div className="flex items-center gap-4">
                <input
                  type="range"
                  min={1}
                  max={10}
                  step={1}
                  value={queryCount}
                  onChange={(e) => setQueryCount(parseInt(e.target.value) || 1)}
                  className="flex-1 h-2 bg-secondary rounded-lg appearance-none cursor-pointer accent-primary"
                />
                <div className="w-12 text-center font-mono font-medium text-sm border rounded px-1.5 py-1 bg-background">
                  {queryCount}
                </div>
            </div>
          </div>

          <div className="pt-2 flex gap-3 justify-end border-t border-border/50">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 border border-input rounded-lg hover:bg-accent hover:text-accent-foreground font-medium text-sm transition-colors"
            >
              取消
            </button>
            <button
              type="submit"
              disabled={creating}
              className="px-6 py-2 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 disabled:opacity-50 font-medium text-sm shadow-sm transition-all"
            >
              {creating ? '创建中...' : '确认创建'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
