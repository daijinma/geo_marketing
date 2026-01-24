import { useState, useEffect } from 'react';
import { Play, Loader2 } from 'lucide-react';
import { toast } from 'sonner';
import { wailsAPI } from '@/utils/wails-api';

export default function Search() {
  const [keywords, setKeywords] = useState('');
  const [selectedPlatforms, setSelectedPlatforms] = useState<string[]>(['deepseek']);
  const [queryCount, setQueryCount] = useState(1);
  const [isSearching, setIsSearching] = useState(false);

  const handleSearch = async () => {
    if (!keywords.trim()) {
      toast.error('请输入关键词');
      return;
    }
    if (selectedPlatforms.length === 0) {
      toast.error('请至少选择一个平台');
      return;
    }

    setIsSearching(true);
    try {
      const keywordList = keywords.split('\n').map(k => k.trim()).filter(k => k);
      const result = await wailsAPI.search.createTask({
        keywords: keywordList,
        platforms: selectedPlatforms,
        queryCount,
      });
      if (result.success) {
        toast.success('搜索任务已创建');
      }
    } catch (error: any) {
      toast.error('创建任务失败', { description: error.message });
    } finally {
      setIsSearching(false);
    }
  };

  return (
    <div className="p-6 space-y-6">
      <h1 className="text-2xl font-bold">大模型搜索</h1>
      <div className="space-y-4">
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
                  checked={selectedPlatforms.includes(p)}
                  onChange={(e) => {
                    if (e.target.checked) {
                      setSelectedPlatforms([...selectedPlatforms, p]);
                    } else {
                      setSelectedPlatforms(selectedPlatforms.filter(x => x !== p));
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
        <button
          onClick={handleSearch}
          disabled={isSearching}
          className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 disabled:opacity-50 flex items-center gap-2"
        >
          {isSearching ? <Loader2 className="w-4 h-4 animate-spin" /> : <Play className="w-4 h-4" />}
          开始搜索
        </button>
      </div>
    </div>
  );
}
