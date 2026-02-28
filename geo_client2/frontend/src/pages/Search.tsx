import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { Play, Loader2, Combine, CheckSquare, Square } from 'lucide-react';
import { toast } from 'sonner';
import { wailsAPI } from '@/utils/wails-api';
import { MergedTaskViewer } from '@/components/MergedTaskViewer';

interface SearchTask {
  id: number;
  keywords: string;
  platforms: string;
  status: string;
  created_at: string;
}

export default function Search() {
  const navigate = useNavigate();
  const [keywords, setKeywords] = useState('');
  const [selectedPlatforms, setSelectedPlatforms] = useState<string[]>(['deepseek']);
  const [queryCount, setQueryCount] = useState(1);
  const [isSearching, setIsSearching] = useState(false);
  
  const [recentTasks, setRecentTasks] = useState<SearchTask[]>([]);
  const [selectedMergeIds, setSelectedMergeIds] = useState<number[]>([]);
  const [showMergedViewer, setShowMergedViewer] = useState(false);

  useEffect(() => {
    loadRecentTasks();
  }, []);

  const loadRecentTasks = async () => {
    try {
      const res = await wailsAPI.task.getAllTasks({ limit: 50, status: 'completed' });
      if (res.success && res.tasks) {
        setRecentTasks(res.tasks as SearchTask[]);
      }
    } catch (error) {
      console.error('Failed to load recent tasks', error);
    }
  };

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
        navigate('/tasks');
      }
    } catch (error: any) {
      toast.error('创建任务失败', { description: error.message });
    } finally {
      setIsSearching(false);
    }
  };

  const toggleMergeSelection = (id: number) => {
    setSelectedMergeIds(prev => 
      prev.includes(id) ? prev.filter(taskId => taskId !== id) : [...prev, id]
    );
  };

  const handleMergeQuery = () => {
    if (selectedMergeIds.length === 0) {
      toast.error('请至少选择一个已完成的任务');
      return;
    }
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
              {['deepseek', 'doubao', 'yiyan', 'yuanbao'].map((p) => {
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
                w-full md:w-auto px-8 py-3 rounded-xl font-medium flex items-center justify-center gap-2 transition-all shadow-sm hover:shadow-sm
                ${isSearching 
                  ? 'bg-muted text-muted-foreground cursor-not-allowed' 
                  : 'bg-primary text-primary-foreground hover:bg-primary/90'}
              `}
            >
              {isSearching ? <Loader2 className="w-5 h-5 animate-spin" /> : <Play className="w-5 h-5 fill-current" />}
              {isSearching ? '正在创建任务...' : '开始搜索任务'}
            </button>
          </div>
        </div>

        <div className="bg-card border border-border/50 rounded-xl shadow-sm p-6 space-y-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Combine className="w-5 h-5 text-muted-foreground" />
              <h2 className="text-lg font-semibold">结果合并工具</h2>
            </div>
            <button
              onClick={handleMergeQuery}
              className="px-6 py-2.5 bg-secondary text-secondary-foreground font-medium rounded-lg hover:bg-secondary/80 hover:shadow-sm transition-all whitespace-nowrap flex items-center justify-center gap-2"
            >
              <Combine className="w-4 h-4" />
              合并查看
            </button>
          </div>
          
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground">请选择要合并分析的已完成任务：</p>
            {recentTasks.length === 0 ? (
              <div className="text-center py-8 text-sm text-muted-foreground border border-dashed rounded-lg">
                暂无已完成的搜索任务
              </div>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-3 max-h-[300px] overflow-y-auto pr-2 scrollbar-thin">
                {recentTasks.map(task => {
                  const isSelected = selectedMergeIds.includes(task.id);
                  const platforms = task.platforms ? JSON.parse(task.platforms) : [];
                  const platformStr = Array.isArray(platforms) ? platforms.join(', ') : task.platforms;
                  return (
                    <div
                      key={task.id}
                      onClick={() => toggleMergeSelection(task.id)}
                      className={`
                        p-3 rounded-lg border cursor-pointer flex items-start gap-3 transition-colors
                        ${isSelected ? 'border-primary bg-primary/5' : 'border-border hover:border-primary/50 hover:bg-accent'}
                      `}
                    >
                      <div className="mt-0.5">
                        {isSelected ? <CheckSquare className="w-5 h-5 text-primary" /> : <Square className="w-5 h-5 text-muted-foreground" />}
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center justify-between mb-1">
                          <span className="text-sm font-medium truncate">任务 ID: {task.id}</span>
                          <span className="text-xs text-muted-foreground">{new Date(task.created_at).toLocaleString()}</span>
                        </div>
                        <p className="text-xs text-muted-foreground truncate" title={task.keywords}>
                          {task.keywords.replace(/\n/g, ' ')}
                        </p>
                        <p className="text-xs text-muted-foreground mt-1">
                          平台: {platformStr}
                        </p>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        </div>
      </div>

      {showMergedViewer && (
        <MergedTaskViewer
          taskIds={selectedMergeIds}
          onClose={() => setShowMergedViewer(false)}
        />
      )}
    </div>
  );
}
