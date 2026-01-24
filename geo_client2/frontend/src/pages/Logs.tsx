import { useState, useEffect } from 'react';
import { wailsAPI, LogEntry } from '@/utils/wails-api';
import { Filter, RefreshCw, X, ChevronDown, ChevronRight, Trash2, AlertTriangle, Download } from 'lucide-react';
import { toast } from 'sonner';

export default function Logs() {
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [filters, setFilters] = useState({
    level: '',
    source: '',
    task_id: undefined as number | undefined,
  });
  const [showFilters, setShowFilters] = useState(false);
  const [expandedLog, setExpandedLog] = useState<number | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  
  // Deletion states
  const [showDeleteMenu, setShowDeleteMenu] = useState(false);
  const [showConfirmDialog, setShowConfirmDialog] = useState(false);
  const [deleteMode, setDeleteMode] = useState<'old' | 'all' | null>(null);
  const [confirmInput, setConfirmInput] = useState('');
  const [deleting, setDeleting] = useState(false);

  // Export states
  const [showExportMenu, setShowExportMenu] = useState(false);
  const [exporting, setExporting] = useState(false);

  const PAGE_SIZE = 50;

  useEffect(() => {
    loadLogs();
  }, [page, filters]);

  const loadLogs = async () => {
    setLoading(true);
    try {
      const result = await wailsAPI.logs.getAll({
        limit: PAGE_SIZE,
        offset: (page - 1) * PAGE_SIZE,
        level: filters.level || undefined,
        source: filters.source || undefined,
        task_id: filters.task_id,
      });

      if (result.success) {
        setLogs(result.logs || []);
      }
    } catch (error) {
      console.error('Failed to load logs:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleRefresh = () => {
    setPage(1);
    loadLogs();
  };

  const handleClearFilters = () => {
    setFilters({ level: '', source: '', task_id: undefined });
    setSearchTerm('');
    setPage(1);
  };

  const handleDeleteOld = () => {
    setDeleteMode('old');
    setConfirmInput('');
    setShowConfirmDialog(true);
    setShowDeleteMenu(false);
  };

  const handleDeleteAll = () => {
    setDeleteMode('all');
    setConfirmInput('');
    setShowConfirmDialog(true);
    setShowDeleteMenu(false);
  };

  const executeDelete = async () => {
    const expectedInput = deleteMode === 'all' ? '确认删除所有日志' : '确认删除';
    if (confirmInput !== expectedInput) {
      toast.error(`请输入 "${expectedInput}" 以确认操作`);
      return;
    }

    setDeleting(true);
    try {
      let result;
      if (deleteMode === 'all') {
        result = await wailsAPI.logs.deleteAll();
      } else {
        result = await wailsAPI.logs.clearOld(30);
      }

      if (result.success) {
        toast.success(`成功删除 ${result.deleted} 条日志`);
        setShowConfirmDialog(false);
        setPage(1);
        loadLogs();
      } else {
        toast.error('删除操作失败');
      }
    } catch (error) {
      console.error('Delete logs error:', error);
      toast.error('删除操作发生错误');
    } finally {
      setDeleting(false);
    }
  };

  const handleExport = async (timeRange: string) => {
    setExporting(true);
    setShowExportMenu(false);
    try {
      const result = await wailsAPI.logs.export(timeRange);
      if (result.success) {
        toast.success(`成功导出 ${result.count} 条日志到 ${result.path}`);
      } else {
        if (result.message && result.message !== 'Export cancelled') {
          toast.error(result.message || '导出失败');
        } else if (result.message === 'Export cancelled') {
          // ignore
        } else {
          toast.error('导出失败');
        }
      }
    } catch (error) {
      console.error('Export logs error:', error);
      toast.error('导出操作发生错误');
    } finally {
      setExporting(false);
    }
  };

  const getLevelColor = (level: string) => {
    switch (level) {
      case 'ERROR':
        return 'text-red-600 bg-red-50';
      case 'WARN':
        return 'text-yellow-600 bg-yellow-50';
      case 'INFO':
        return 'text-blue-600 bg-blue-50';
      case 'DEBUG':
        return 'text-gray-600 bg-gray-50';
      default:
        return 'text-gray-600 bg-gray-50';
    }
  };

  const formatTimestamp = (timestamp: string) => {
    try {
      const date = new Date(timestamp);
      return date.toLocaleString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
      });
    } catch {
      return timestamp;
    }
  };

  const parseDetails = (details?: string) => {
    if (!details) return null;
    try {
      const parsed = JSON.parse(details);
      if (typeof parsed === 'string' && (parsed.startsWith('{') || parsed.startsWith('['))) {
        return JSON.parse(parsed);
      }
      return parsed;
    } catch {
      return details;
    }
  };

  const filteredLogs = logs.filter((log) => {
    if (!searchTerm) return true;
    const searchLower = searchTerm.toLowerCase();
    return (
      log.message.toLowerCase().includes(searchLower) ||
      log.source.toLowerCase().includes(searchLower) ||
      log.details?.toLowerCase().includes(searchLower)
    );
  });

  return (
    <div className="p-6 h-full flex flex-col">
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-2xl font-bold">系统日志</h1>
        <div className="flex gap-2 relative">
          <button
            onClick={() => setShowFilters(!showFilters)}
            className="px-3 py-2 bg-white border rounded-md hover:bg-gray-50 flex items-center gap-2"
          >
            <Filter className="w-4 h-4" />
            筛选
          </button>
          <button
            onClick={handleRefresh}
            className="px-3 py-2 bg-white border rounded-md hover:bg-gray-50 flex items-center gap-2"
            disabled={loading}
          >
            <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
            刷新
          </button>
          <div className="relative">
            <button
              onClick={() => setShowExportMenu(!showExportMenu)}
              className="px-3 py-2 bg-white border border-gray-300 text-gray-700 rounded-md hover:bg-gray-50 flex items-center gap-2"
              disabled={exporting}
            >
              <Download className={`w-4 h-4 ${exporting ? 'animate-bounce' : ''}`} />
              导出
            </button>
            {showExportMenu && (
              <div className="absolute right-0 mt-2 w-48 bg-white border rounded-md shadow-lg z-50">
                <div className="py-1">
                  <button
                    onClick={() => handleExport('today')}
                    className="w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-100"
                  >
                    导出今日日志
                  </button>
                  <button
                    onClick={() => handleExport('3days')}
                    className="w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-100"
                  >
                    导出近3天日志
                  </button>
                  <button
                    onClick={() => handleExport('7days')}
                    className="w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-100"
                  >
                    导出近7天日志
                  </button>
                </div>
              </div>
            )}
          </div>
          <div className="relative">
            <button
              onClick={() => setShowDeleteMenu(!showDeleteMenu)}
              className="px-3 py-2 bg-white border border-red-200 text-red-600 rounded-md hover:bg-red-50 flex items-center gap-2"
            >
              <Trash2 className="w-4 h-4" />
              删除日志
            </button>
            {showDeleteMenu && (
              <div className="absolute right-0 mt-2 w-56 bg-white border rounded-md shadow-lg z-50">
                <div className="py-1">
                  <button
                    onClick={handleDeleteOld}
                    className="w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 flex items-center gap-2"
                  >
                    <Trash2 className="w-4 h-4" />
                    删除30天前的日志
                  </button>
                  <button
                    onClick={handleDeleteAll}
                    className="w-full text-left px-4 py-2 text-sm text-red-600 hover:bg-red-50 flex items-center gap-2"
                  >
                    <Trash2 className="w-4 h-4" />
                    删除所有日志
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>

      {showFilters && (
        <div className="bg-white border rounded-md p-4 mb-4">
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <div>
              <label className="block text-sm font-medium mb-1">级别</label>
              <select
                value={filters.level}
                onChange={(e) => setFilters({ ...filters, level: e.target.value })}
                className="w-full px-3 py-2 border rounded-md"
              >
                <option value="">全部</option>
                <option value="ERROR">ERROR</option>
                <option value="WARN">WARN</option>
                <option value="INFO">INFO</option>
                <option value="DEBUG">DEBUG</option>
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">来源</label>
              <input
                type="text"
                value={filters.source}
                onChange={(e) => setFilters({ ...filters, source: e.target.value })}
                placeholder="frontend, backend..."
                className="w-full px-3 py-2 border rounded-md"
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">任务ID</label>
              <input
                type="number"
                value={filters.task_id || ''}
                onChange={(e) =>
                  setFilters({ ...filters, task_id: e.target.value ? parseInt(e.target.value) : undefined })
                }
                placeholder="输入任务ID"
                className="w-full px-3 py-2 border rounded-md"
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">搜索</label>
              <input
                type="text"
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                placeholder="搜索消息内容..."
                className="w-full px-3 py-2 border rounded-md"
              />
            </div>
          </div>
          <div className="mt-3 flex justify-end">
            <button
              onClick={handleClearFilters}
              className="px-3 py-1 text-sm text-gray-600 hover:text-gray-800 flex items-center gap-1"
            >
              <X className="w-4 h-4" />
              清除筛选
            </button>
          </div>
        </div>
      )}

      <div className="flex-1 bg-white border rounded-md overflow-hidden flex flex-col">
        <div className="overflow-auto flex-1">
          <table className="w-full">
            <thead className="bg-gray-50 sticky top-0 z-10">
              <tr className="border-b">
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase w-8"></th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase w-40">时间</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase w-20">级别</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase w-32">来源</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">消息</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase w-24">组件</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase w-20">耗时</th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {loading ? (
                <tr>
                  <td colSpan={7} className="px-4 py-8 text-center text-gray-500">
                    加载中...
                  </td>
                </tr>
              ) : filteredLogs.length === 0 ? (
                <tr>
                  <td colSpan={7} className="px-4 py-8 text-center text-gray-500">
                    暂无日志
                  </td>
                </tr>
              ) : (
                filteredLogs.map((log) => {
                  const details = parseDetails(log.details);
                  const isExpanded = expandedLog === log.id;

                  return (
                    <>
                      <tr
                        key={log.id}
                        className="hover:bg-gray-50 cursor-pointer"
                        onClick={() => setExpandedLog(isExpanded ? null : log.id)}
                      >
                        <td className="px-4 py-3">
                          {details ? (
                            isExpanded ? (
                              <ChevronDown className="w-4 h-4 text-gray-400" />
                            ) : (
                              <ChevronRight className="w-4 h-4 text-gray-400" />
                            )
                          ) : null}
                        </td>
                        <td className="px-4 py-3 text-sm text-gray-500">{formatTimestamp(log.timestamp)}</td>
                        <td className="px-4 py-3">
                          <span className={`px-2 py-1 text-xs font-medium rounded ${getLevelColor(log.level)}`}>
                            {log.level}
                          </span>
                        </td>
                        <td className="px-4 py-3 text-sm text-gray-700 truncate max-w-xs" title={log.source}>
                          {log.source}
                        </td>
                        <td className="px-4 py-3 text-sm text-gray-900">
                          <div className="flex items-center gap-2">
                            {log.message}
                            {details && (
                              <span className="px-1.5 py-0.5 text-[10px] font-bold bg-indigo-100 text-indigo-700 rounded border border-indigo-200">
                                DATA
                              </span>
                            )}
                          </div>
                        </td>
                        <td className="px-4 py-3 text-sm text-gray-500">{log.component || '-'}</td>
                        <td className="px-4 py-3 text-sm text-gray-500">
                          {log.performance_ms ? `${log.performance_ms}ms` : '-'}
                        </td>
                      </tr>
                      {isExpanded && details && (
                        <tr key={`${log.id}-details`}>
                          <td colSpan={7} className="px-4 py-3 bg-gray-50">
                            <div className="space-y-2">
                              {log.session_id && (
                                <div className="text-sm">
                                  <span className="font-medium text-gray-700">Session ID:</span>
                                  <span className="ml-2 text-gray-600 font-mono text-xs">{log.session_id}</span>
                                </div>
                              )}
                              {log.correlation_id && (
                                <div className="text-sm">
                                  <span className="font-medium text-gray-700">Correlation ID:</span>
                                  <span className="ml-2 text-gray-600 font-mono text-xs">{log.correlation_id}</span>
                                </div>
                              )}
                              {log.user_action && (
                                <div className="text-sm">
                                  <span className="font-medium text-gray-700">用户操作:</span>
                                  <span className="ml-2 text-gray-600">{log.user_action}</span>
                                </div>
                              )}
                              {log.task_id && (
                                <div className="text-sm">
                                  <span className="font-medium text-gray-700">任务ID:</span>
                                  <span className="ml-2 text-gray-600">{log.task_id}</span>
                                </div>
                              )}
                              <div className="text-sm">
                                <span className="font-medium text-gray-700">详细信息:</span>
                                <pre className="mt-1 p-3 bg-white rounded border text-xs overflow-auto max-h-[500px]">
                                  {typeof details === 'object' ? JSON.stringify(details, null, 2) : details}
                                </pre>
                              </div>
                            </div>
                          </td>
                        </tr>
                      )}
                    </>
                  );
                })
              )}
            </tbody>
          </table>
        </div>

        <div className="border-t px-4 py-3 flex items-center justify-between bg-gray-50">
          <div className="text-sm text-gray-700">
            显示 {(page - 1) * PAGE_SIZE + 1} - {Math.min(page * PAGE_SIZE, (page - 1) * PAGE_SIZE + filteredLogs.length)} 条
          </div>
          <div className="flex gap-2">
            <button
              onClick={() => setPage(Math.max(1, page - 1))}
              disabled={page === 1}
              className="px-3 py-1 border rounded-md hover:bg-white disabled:opacity-50 disabled:cursor-not-allowed"
            >
              上一页
            </button>
            <span className="px-3 py-1 border rounded-md bg-white">第 {page} 页</span>
            <button
              onClick={() => setPage(page + 1)}
              disabled={filteredLogs.length < PAGE_SIZE}
              className="px-3 py-1 border rounded-md hover:bg-white disabled:opacity-50 disabled:cursor-not-allowed"
            >
              下一页
            </button>
          </div>
        </div>
      </div>

      {/* Confirmation Dialog */}
      {showConfirmDialog && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-[100] p-4">
          <div className="bg-white rounded-lg shadow-xl max-w-md w-full overflow-hidden">
            <div className="p-6">
              <div className="flex items-center gap-3 mb-4 text-red-600">
                <AlertTriangle className="w-6 h-6" />
                <h2 className="text-xl font-bold">
                  {deleteMode === 'all' ? '确认删除所有日志？' : '确认删除旧日志？'}
                </h2>
              </div>
              
              <p className="text-gray-600 mb-4">
                {deleteMode === 'all' 
                  ? '此操作将彻底删除本地存储的所有日志，且不可恢复。' 
                  : '此操作将彻底删除本地存储的 30 天之前的日志，且不可恢复。'}
              </p>

              <div className="bg-amber-50 border border-amber-200 rounded p-3 mb-4">
                <p className="text-sm text-amber-800">
                  请输入 <strong>{deleteMode === 'all' ? '确认删除所有日志' : '确认删除'}</strong> 以确认操作：
                </p>
                <input
                  type="text"
                  value={confirmInput}
                  onChange={(e) => setConfirmInput(e.target.value)}
                  className="w-full mt-2 px-3 py-2 border rounded focus:ring-2 focus:ring-red-500 outline-none"
                  placeholder="请输入确认文字"
                  autoFocus
                />
              </div>

              <div className="flex justify-end gap-3 mt-6">
                <button
                  onClick={() => setShowConfirmDialog(false)}
                  className="px-4 py-2 border rounded-md hover:bg-gray-50 text-gray-700"
                  disabled={deleting}
                >
                  取消
                </button>
                <button
                  onClick={executeDelete}
                  className="px-4 py-2 bg-red-600 text-white rounded-md hover:bg-red-700 disabled:opacity-50 flex items-center gap-2"
                  disabled={deleting || (deleteMode === 'all' ? confirmInput !== '确认删除所有日志' : confirmInput !== '确认删除')}
                >
                  {deleting && <RefreshCw className="w-4 h-4 animate-spin" />}
                  确认删除
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
