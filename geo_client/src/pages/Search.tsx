import { useState, useEffect } from 'react';
import { Search as SearchIcon, Play, Loader2, Upload } from 'lucide-react';
import SearchResults from '@/components/SearchResults';
import { useAuthStore } from '@/stores/authStore';

interface SearchResult {
  query: string;
  subQuery?: string;
  items: {
    title: string;
    url: string;
    snippet: string;
    domain?: string;
    siteName?: string;
  }[];
  responseTimeMs: number;
  citationsCount: number;
  error?: string;
}

export default function Search() {
  const { token } = useAuthStore();
  const [keywords, setKeywords] = useState('');
  const [selectedPlatforms, setSelectedPlatforms] = useState<string[]>(['deepseek']);
  const [queryCount, setQueryCount] = useState(1);
  const [isSearching, setIsSearching] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [results, setResults] = useState<SearchResult[]>([]);
  const [error, setError] = useState('');
  const [completedTaskIds, setCompletedTaskIds] = useState<string[]>([]);

  const platforms = [
    { id: 'deepseek', name: 'DeepSeek' },
    { id: 'doubao', name: '豆包' },
  ];

  const handlePlatformToggle = (platformId: string) => {
    setSelectedPlatforms((prev) =>
      prev.includes(platformId)
        ? prev.filter((id) => id !== platformId)
        : [...prev, platformId]
    );
  };

  const handleSearch = async () => {
    // 验证输入
    if (!keywords.trim()) {
      setError('请输入关键词');
      return;
    }

    if (selectedPlatforms.length === 0) {
      setError('请至少选择一个平台');
      return;
    }

    setError('');
    setIsSearching(true);
    setResults([]);

    try {
      // 分割关键词
      const keywordList = keywords
        .split('\n')
        .map((k) => k.trim())
        .filter((k) => k.length > 0);

      if (keywordList.length === 0) {
        throw new Error('没有有效的关键词');
      }

      // 调用 Electron IPC 创建搜索任务
      const response = await window.electronAPI.search.createTask({
        keywords: keywordList,
        platforms: selectedPlatforms,
        queryCount,
      });

      if (!response.success) {
        throw new Error(response.error || '创建搜索任务失败');
      }

      // 监听任务更新
      window.electronAPI.search.onTaskUpdated((data: any) => {
        if (data.status === 'completed' && data.result) {
          setResults((prev) => [...prev, data.result]);
          setCompletedTaskIds((prev) => [...prev, data.taskId]);
        }
      });

      // 保存任务到本地数据库
      await window.electronAPI.task.saveToLocal(
        keywordList,
        selectedPlatforms,
        queryCount,
        'running'
      );

      // 提示任务已创建
      alert(response.message || '搜索任务已创建');
    } catch (err: any) {
      console.error('搜索失败:', err);
      setError(err.message || '搜索失败');
    } finally {
      setIsSearching(false);
    }
  };

  const handleSubmitToServer = async () => {
    if (completedTaskIds.length === 0) {
      alert('没有已完成的任务可以提交');
      return;
    }

    if (!token) {
      alert('未登录，无法提交到服务器');
      return;
    }

    setIsSubmitting(true);
    try {
      const apiBaseUrl = localStorage.getItem('apiBaseUrl') || 'http://localhost:8000';
      
      // 提交所有已完成的任务
      const promises = completedTaskIds.map((taskId) =>
        window.electronAPI.task.submitToServer(taskId, apiBaseUrl, token)
      );

      const responses = await Promise.all(promises);
      const failedCount = responses.filter((r) => !r.success).length;

      if (failedCount === 0) {
        alert('所有任务已成功提交到服务器');
        setCompletedTaskIds([]);
      } else {
        alert(`提交完成，${failedCount} 个任务提交失败`);
      }
    } catch (err: any) {
      console.error('提交到服务器失败:', err);
      alert('提交到服务器失败: ' + err.message);
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="p-6 space-y-6">
      {/* 标题 */}
      <div>
        <h1 className="text-2xl font-bold mb-2">大模型搜索</h1>
        <p className="text-muted-foreground">使用大模型进行关键词搜索，提取引用链接</p>
      </div>

      {/* 搜索表单 */}
      <div className="bg-card border border-border rounded-lg p-6 space-y-6">
        {/* 关键词输入 */}
        <div>
          <label className="block text-sm font-medium mb-2">
            关键词 <span className="text-red-500">*</span>
          </label>
          <textarea
            value={keywords}
            onChange={(e) => setKeywords(e.target.value)}
            placeholder="输入关键词，每行一个"
            rows={5}
            className="w-full px-3 py-2 border border-border rounded-md bg-background focus:outline-none focus:ring-2 focus:ring-primary"
            disabled={isSearching}
          />
          <p className="text-xs text-muted-foreground mt-1">
            每行输入一个关键词，支持批量搜索
          </p>
        </div>

        {/* 平台选择 */}
        <div>
          <label className="block text-sm font-medium mb-2">
            平台选择 <span className="text-red-500">*</span>
          </label>
          <div className="flex flex-wrap gap-3">
            {platforms.map((platform) => (
              <label
                key={platform.id}
                className={`flex items-center gap-2 px-4 py-2 border rounded-md cursor-pointer transition-colors ${
                  selectedPlatforms.includes(platform.id)
                    ? 'border-primary bg-primary/10'
                    : 'border-border hover:bg-accent'
                } ${isSearching ? 'opacity-50 cursor-not-allowed' : ''}`}
              >
                <input
                  type="checkbox"
                  checked={selectedPlatforms.includes(platform.id)}
                  onChange={() => handlePlatformToggle(platform.id)}
                  disabled={isSearching}
                  className="w-4 h-4"
                />
                <span className="text-sm">{platform.name}</span>
              </label>
            ))}
          </div>
        </div>

        {/* 查询次数 */}
        <div>
          <label className="block text-sm font-medium mb-2">
            每个关键词查询次数
          </label>
          <input
            type="number"
            value={queryCount}
            onChange={(e) => setQueryCount(Math.max(1, parseInt(e.target.value) || 1))}
            min={1}
            max={10}
            className="w-32 px-3 py-2 border border-border rounded-md bg-background focus:outline-none focus:ring-2 focus:ring-primary"
            disabled={isSearching}
          />
          <p className="text-xs text-muted-foreground mt-1">
            建议设置为 1-3 次，避免过多请求
          </p>
        </div>

        {/* 错误提示 */}
        {error && (
          <div className="p-3 bg-red-500/10 border border-red-500 rounded-md text-red-500 text-sm">
            {error}
          </div>
        )}

        {/* 操作按钮 */}
        <div className="flex gap-4">
          <button
            onClick={handleSearch}
            disabled={isSearching || isSubmitting}
            className="flex items-center gap-2 px-6 py-3 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isSearching ? (
              <>
                <Loader2 className="w-5 h-5 animate-spin" />
                <span>搜索中...</span>
              </>
            ) : (
              <>
                <Play className="w-5 h-5" />
                <span>开始搜索</span>
              </>
            )}
          </button>

          {completedTaskIds.length > 0 && (
            <button
              onClick={handleSubmitToServer}
              disabled={isSearching || isSubmitting}
              className="flex items-center gap-2 px-6 py-3 bg-green-600 text-white rounded-md hover:bg-green-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isSubmitting ? (
                <>
                  <Loader2 className="w-5 h-5 animate-spin" />
                  <span>提交中...</span>
                </>
              ) : (
                <>
                  <Upload className="w-5 h-5" />
                  <span>提交到服务器 ({completedTaskIds.length})</span>
                </>
              )}
            </button>
          )}
        </div>
      </div>

      {/* 搜索结果 */}
      {results.length > 0 && (
        <div className="bg-card border border-border rounded-lg p-6">
          <SearchResults results={results} />
        </div>
      )}
    </div>
  );
}
