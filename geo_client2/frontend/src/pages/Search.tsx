import { useState } from 'react';
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
        setKeywords('');
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
    <div className="p-8 space-y-8 max-w-5xl mx-auto">
      <div>
        <h1 className="text-3xl font-bold tracking-tight mb-2">大模型搜索</h1>
        <p className="text-muted-foreground text-lg">创建新的搜索任务以挖掘市场洞察</p>
      </div>

      <div className="grid gap-8">
        <div className="bg-card border border-border/50 rounded-xl shadow-sm p-6 space-y-8">
          
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <label className="text-base font-semibold">搜索关键词</label>
              <span className="text-xs text-muted-foreground bg-secondary px-2 py-1 rounded">每行一个关键词</span>
            </div>
            <textarea
              value={keywords}
              onChange={(e) => setKeywords(e.target.value)}
              className="w-full min-h-[160px] px-4 py-3 border border-input rounded-xl bg-background/50 focus:bg-background focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all resize-y text-base outline-none"
              placeholder="例如：&#10;上海 咖啡店 推荐&#10;2024年 露营装备 评测"
            />
          </div>

          <div className="space-y-4">
            <label className="text-base font-semibold block">目标平台</label>
            <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-3">
              {['deepseek', 'doubao', 'yiyan', 'yuanbao', 'xiaohongshu'].map((p) => {
                const isSelected = selectedPlatforms.includes(p);
                return (
                  <label 
                    key={p} 
                    className={`
                      relative cursor-pointer border rounded-xl px-4 py-3 flex items-center gap-3 transition-all select-none
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
                          setSelectedPlatforms([...selectedPlatforms, p]);
                        } else {
                          setSelectedPlatforms(selectedPlatforms.filter(x => x !== p));
                        }
                      }}
                    />
                    <div className={`w-4 h-4 rounded-full border flex items-center justify-center transition-colors ${isSelected ? 'border-primary bg-primary' : 'border-muted-foreground/30'}`}>
                      {isSelected && <div className="w-2 h-2 rounded-full bg-primary-foreground" />}
                    </div>
                    <span className={`font-medium ${isSelected ? 'text-primary' : 'text-foreground'}`}>
                      {p === 'deepseek' ? 'DeepSeek' : 
                       p === 'doubao' ? '豆包' : 
                       p === 'yiyan' ? '文心一言' : 
                       p === 'xiaohongshu' ? '小红书' : '腾讯元宝'}
                    </span>
                  </label>
                );
              })}
            </div>
          </div>

          <div className="grid md:grid-cols-2 gap-6">
            <div className="space-y-3">
              <label className="text-base font-semibold block">查询深度</label>
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
                <div className="w-16 text-center font-mono font-medium border rounded px-2 py-1 bg-background">
                  {queryCount} 次
                </div>
              </div>
              <p className="text-xs text-muted-foreground">每次搜索执行的查询轮数，次数越多结果越丰富</p>
            </div>
          </div>

          <div className="pt-4">
            <button
              onClick={handleSearch}
              disabled={isSearching}
              className={`
                w-full md:w-auto px-8 py-3 rounded-xl font-medium flex items-center justify-center gap-2 transition-all shadow-sm hover:shadow-md
                ${isSearching 
                  ? 'bg-muted text-muted-foreground cursor-not-allowed' 
                  : 'bg-primary text-primary-foreground hover:bg-primary/90 hover:scale-[1.02] active:scale-[0.98]'}
              `}
            >
              {isSearching ? <Loader2 className="w-5 h-5 animate-spin" /> : <Play className="w-5 h-5 fill-current" />}
              {isSearching ? '正在创建任务...' : '开始搜索任务'}
            </button>
          </div>
        </div>

        <div className="bg-card border border-border/50 rounded-xl shadow-sm p-6 space-y-6 opacity-90 hover:opacity-100 transition-opacity">
          <div className="flex items-center gap-2">
            <Combine className="w-5 h-5 text-muted-foreground" />
            <h2 className="text-lg font-semibold">结果合并工具</h2>
          </div>
          
          <div className="flex flex-col md:flex-row gap-4">
            <div className="flex-1 space-y-2">
              <input
                type="text"
                value={mergeTaskIds}
                onChange={(e) => setMergeTaskIds(e.target.value)}
                placeholder="输入任务ID，用逗号分隔（例如: 1, 2, 3）"
                className="w-full px-4 py-2.5 border border-input rounded-lg bg-background/50 focus:bg-background focus:ring-2 focus:ring-secondary focus:border-secondary transition-all outline-none"
              />
            </div>
            <button
              onClick={handleMergeQuery}
              className="px-6 py-2.5 bg-secondary text-secondary-foreground font-medium rounded-lg hover:bg-secondary/80 hover:shadow-sm transition-all whitespace-nowrap flex items-center justify-center gap-2"
            >
              <Combine className="w-4 h-4" />
              合并查看
            </button>
          </div>
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
