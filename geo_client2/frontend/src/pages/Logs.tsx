import { useState, useEffect, useRef } from 'react';
import { wailsAPI } from '@/utils/wails-api';
import { 
  RefreshCw, 
  Trash2, 
  Download, 
  FolderOpen, 
  Play, 
  Pause,
  ChevronDown
} from 'lucide-react';
import { toast } from 'sonner';

export default function Logs() {
  const [logContent, setLogContent] = useState('');
  const [loading, setLoading] = useState(false);
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [exporting, setExporting] = useState(false);
  const [showExportMenu, setShowExportMenu] = useState(false);
  const [showDeleteMenu, setShowDeleteMenu] = useState(false);
  
  const scrollRef = useRef<HTMLPreElement>(null);
  const refreshInterval = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    loadLogs();
    
    return () => {
      if (refreshInterval.current) {
        clearInterval(refreshInterval.current);
      }
    };
  }, []);

  useEffect(() => {
    if (autoRefresh) {
      refreshInterval.current = setInterval(loadLogs, 2000);
    } else if (refreshInterval.current) {
      clearInterval(refreshInterval.current);
    }
  }, [autoRefresh]);

  useEffect(() => {
    if (autoRefresh && scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [logContent]);

  const loadLogs = async () => {
    try {
      const result = await wailsAPI.logs.getFileContent(1000);
      if (result.success) {
        setLogContent(result.content);
      }
    } catch (error) {
      console.error('Failed to load logs:', error);
    }
  };

  const handleRefresh = async () => {
    setLoading(true);
    await loadLogs();
    setLoading(false);
    toast.success('日志已刷新');
  };

  const handleExport = async (timeRange: string) => {
    setExporting(true);
    setShowExportMenu(false);
    try {
      const result = await wailsAPI.logs.export(timeRange);
      if (result.success) {
        toast.success(`日志已导出: ${result.path}`);
      } else if (result.message !== 'Export cancelled') {
        toast.error(result.message || '导出失败');
      }
    } catch (error) {
      toast.error('导出出错');
    } finally {
      setExporting(false);
    }
  };

  const handleOpenFolder = async () => {
    try {
      await wailsAPI.logs.openFolder();
    } catch (error) {
      toast.error('无法打开目录');
    }
  };

  const handleDeleteAll = async () => {
    if (!confirm('确定要清空所有日志记录吗？此操作不可恢复。')) return;
    
    try {
      const result = await wailsAPI.logs.deleteAll();
      if (result.success) {
        setLogContent('');
        toast.success('日志已清空');
      }
    } catch (error) {
      toast.error('清空失败');
    }
    setShowDeleteMenu(false);
  };

  return (
    <div className="h-full flex flex-col bg-background text-foreground font-mono">
      <div className="flex items-center justify-between p-4 border-b border-border bg-card">
        <div className="flex items-center gap-4">
          <h1 className="text-lg font-bold flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-green-500 animate-pulse"></span>
            系统控制台
          </h1>
          <div className="flex items-center bg-muted rounded-md p-0.5 border border-border">
            <button
              onClick={() => setAutoRefresh(true)}
              className={`px-3 py-1 text-xs rounded-sm transition-colors ${
                autoRefresh ? 'bg-primary text-primary-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'
              }`}
            >
              <Play className="w-3 h-3 inline mr-1" /> 实时
            </button>
            <button
              onClick={() => setAutoRefresh(false)}
              className={`px-3 py-1 text-xs rounded-sm transition-colors ${
                !autoRefresh ? 'bg-primary text-primary-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'
              }`}
            >
              <Pause className="w-3 h-3 inline mr-1" /> 暂停
            </button>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <button
            onClick={handleRefresh}
            className="p-2 hover:bg-accent rounded-md transition-colors text-muted-foreground hover:text-foreground"
            title="刷新"
            disabled={loading}
          >
            <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
          </button>
          
          <button
            onClick={handleOpenFolder}
            className="p-2 hover:bg-accent rounded-md transition-colors text-muted-foreground hover:text-foreground"
            title="打开日志文件夹"
          >
            <FolderOpen className="w-4 h-4" />
          </button>

          <div className="relative">
            <button
              onClick={() => setShowExportMenu(!showExportMenu)}
              className="flex items-center gap-1 p-2 hover:bg-accent rounded-md transition-colors text-muted-foreground hover:text-foreground"
              title="导出日志"
            >
              <Download className="w-4 h-4" />
              <ChevronDown className="w-3 h-3" />
            </button>
            {showExportMenu && (
              <div className="absolute right-0 mt-2 w-40 bg-popover text-popover-foreground border border-border rounded-md shadow-xl z-50 overflow-hidden">
                <button
                  onClick={() => handleExport('today')}
                  className="w-full text-left px-4 py-2 text-xs hover:bg-accent transition-colors"
                >
                  导出今日
                </button>
                <button
                  onClick={() => handleExport('all')}
                  className="w-full text-left px-4 py-2 text-xs hover:bg-accent transition-colors"
                >
                  导出全部
                </button>
              </div>
            )}
          </div>

          <div className="relative">
            <button
              onClick={() => setShowDeleteMenu(!showDeleteMenu)}
              className="p-2 hover:bg-destructive/10 rounded-md transition-colors text-muted-foreground hover:text-destructive"
              title="删除/清空"
            >
              <Trash2 className="w-4 h-4" />
            </button>
            {showDeleteMenu && (
              <div className="absolute right-0 mt-2 w-40 bg-popover text-popover-foreground border border-border rounded-md shadow-xl z-50 overflow-hidden">
                <button
                  onClick={handleDeleteAll}
                  className="w-full text-left px-4 py-2 text-xs hover:bg-destructive hover:text-destructive-foreground transition-colors"
                >
                  清空所有日志
                </button>
              </div>
            )}
          </div>
        </div>
      </div>

      <pre 
        ref={scrollRef}
        className="flex-1 p-4 overflow-auto text-[13px] leading-relaxed selection:bg-primary/20 bg-muted/30 text-foreground/90 scrollbar-thin scrollbar-thumb-border scrollbar-track-transparent"
      >
        {logContent || (
          <div className="h-full flex items-center justify-center text-muted-foreground italic">
            等待日志输出...
          </div>
        )}
      </pre>

      <div className="px-4 py-1.5 text-[10px] bg-card border-t border-border text-muted-foreground flex justify-between">
        <div className="flex gap-4">
          <span>自动滚动: {autoRefresh ? '开启' : '关闭'}</span>
          <span>显示最后 1000 行</span>
        </div>
        <span>GEO System Console v2.0</span>
      </div>
    </div>
  );
}
