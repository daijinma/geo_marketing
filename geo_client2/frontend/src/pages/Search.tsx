import { useState, useEffect } from 'react';
import { Play, Loader2, Combine } from 'lucide-react';
import { toast } from 'sonner';
import { wailsAPI } from '@/utils/wails-api';
import { MergedTaskViewer } from '@/components/MergedTaskViewer';

export default function Search() {
  const [keywords, setKeywords] = useState('');
  const [selectedPlatforms, setSelectedPlatforms] = useState<string[]>(['deepseek']);
  const [queryCount, setQueryCount] = useState(1);
  const [isSearching, setIsSearching] = useState(false);
  const [mergeTaskIds, setMergeTaskIds] = useState('');
  const [showMergedViewer, setShowMergedViewer] = useState(false);
  const [currentMergedIds, setCurrentMergedIds] = useState<number[]>([]);

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
      
      const result = await wailsAPI.task.createLocalSearchTask({
        keywords: keywordList,
        platforms: selectedPlatforms,
        query_count: queryCount,
      });
      
      if (result.success) {
        toast.success('搜索任务已创建', { description: `任务ID: ${result.taskId}` });
      }
    } catch (error: any) {
      toast.error('创建任务失败', { description: error.message });
    } finally {
      setIsSearching(false);
    }
  };

  const handleMergeQuery = () => {
    const ids = mergeTaskIds
      .split(',')
      .map(id => parseInt(id.trim()))
      .filter(id => !isNaN(id) && id > 0);
    
    if (ids.length === 0) {
      toast.error('请输入有效的任务ID');
      return;
    }

    setCurrentMergedIds(ids);
    setShowMergedViewer(true);
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
          <div className="flex gap-4 flex-wrap">
            {['deepseek', 'doubao', 'yiyan', 'yuanbao', 'xiaohongshu'].map((p) => (
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
                <span>
                  {p === 'deepseek' ? 'DeepSeek' : 
                   p === 'doubao' ? '豆包' : 
                   p === 'yiyan' ? '文心一言' : 
                   p === 'xiaohongshu' ? '小红书' : '腾讯元宝'}
                </span>
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

      <div className="border-t border-border pt-6 mt-8">
        <h2 className="text-lg font-semibold mb-4">合并查询</h2>
        <div className="space-y-3">
          <div>
            <label className="block text-sm font-medium mb-2">
              输入任务ID（多个ID用逗号分隔，如：1,2,3）
            </label>
            <input
              type="text"
              value={mergeTaskIds}
              onChange={(e) => setMergeTaskIds(e.target.value)}
              placeholder="例如: 1,2,3"
              className="w-full px-3 py-2 border border-border rounded-md bg-background"
            />
          </div>
          <button
            onClick={handleMergeQuery}
            className="px-4 py-2 bg-secondary text-secondary-foreground rounded-md hover:bg-secondary/90 flex items-center gap-2"
          >
            <Combine className="w-4 h-4" />
            合并查询
          </button>
        </div>
      </div>

      {showMergedViewer && (
        <MergedTaskViewer
          taskIds={currentMergedIds}
          onClose={() => setShowMergedViewer(false)}
        />
      )}
    </div>
  );
}
