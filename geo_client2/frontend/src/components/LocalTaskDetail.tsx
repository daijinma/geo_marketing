import { useState, useEffect } from 'react';
import { X, Play, ChevronDown, ChevronRight, ExternalLink } from 'lucide-react';
import { toast } from 'sonner';
import { wailsAPI } from '@/utils/wails-api';

interface LocalTaskDetailProps {
  taskId: number;
  onClose: () => void;
}

export function LocalTaskDetail({ taskId, onClose }: LocalTaskDetailProps) {
  const [loading, setLoading] = useState(true);
  const [taskData, setTaskData] = useState<any>(null);
  const [records, setRecords] = useState<any[]>([]);
  const [expandedRecords, setExpandedRecords] = useState<number[]>([]);

  useEffect(() => {
    loadTaskDetail();
    loadRecords();
  }, [taskId]);

  const loadTaskDetail = async () => {
    try {
      const result = await wailsAPI.task.getTaskDetail(taskId);
      if (result.success && 'data' in result) {
        setTaskData(result.data);
      }
    } catch (error: any) {
      toast.error('加载任务详情失败', { description: error.message });
    }
  };

  const loadRecords = async () => {
    try {
      const result = await wailsAPI.task.getSearchRecords(taskId);
      if (result.success && result.records) {
        setRecords(result.records);
      }
    } catch (error: any) {
      console.error('Failed to load records', error);
    } finally {
      setLoading(false);
    }
  };

  const handleRetry = async () => {
    try {
      await wailsAPI.task.retryTask(taskId);
      toast.success('任务已重新开始');
      loadTaskDetail();
      loadRecords();
    } catch (error: any) {
      toast.error('重试任务失败', { description: error.message });
    }
  };

  const toggleRecord = (recordId: number) => {
    setExpandedRecords(prev => 
      prev.includes(recordId) 
        ? prev.filter(id => id !== recordId) 
        : [...prev, recordId]
    );
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-card border border-border rounded-lg w-full max-w-6xl mx-4 max-h-[90vh] flex flex-col">
        <div className="p-6 border-b border-border flex items-center justify-between">
          <div className="flex items-center gap-4">
            <h2 className="text-xl font-bold">任务详情 #{taskId}</h2>
            {taskData && taskData.status !== 'running' && (
              <button
                onClick={handleRetry}
                className="flex items-center gap-1 px-3 py-1 bg-primary text-primary-foreground rounded-md text-sm hover:bg-primary/90"
              >
                <Play className="w-3 h-3" />
                重新执行
              </button>
            )}
          </div>
          <button onClick={onClose} className="p-2 hover:bg-accent rounded-md">
            <X className="w-5 h-5" />
          </button>
        </div>
        
        <div className="flex-1 overflow-y-auto p-6">
          {loading && records.length === 0 ? (
            <div className="text-center py-8">加载中...</div>
          ) : taskData ? (
            <div className="space-y-6">
              {/* Task Info Header */}
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4 p-4 bg-accent/30 rounded-lg text-sm">
                <div>
                  <div className="text-muted-foreground">状态</div>
                  <div className="font-medium capitalize">
                    {taskData.status === 'partial_completed' ? '部分完成' : 
                     taskData.status === 'completed' ? '已完成' :
                     taskData.status === 'running' ? '运行中' :
                     taskData.status === 'failed' ? '失败' :
                     taskData.status === 'pending' ? '等待中' :
                     taskData.status === 'cancelled' ? '已取消' : taskData.status}
                  </div>
                </div>
                <div>
                  <div className="text-muted-foreground">关键词</div>
                  <div className="font-medium truncate" title={taskData.keywords}>{taskData.keywords}</div>
                </div>
                <div>
                  <div className="text-muted-foreground">平台</div>
                  <div className="font-medium capitalize">{taskData.platforms}</div>
                </div>
                <div>
                  <div className="text-muted-foreground">进度</div>
                  <div className="font-medium">
                    {taskData.completed_queries} / {taskData.total_queries}
                  </div>
                </div>
              </div>

              {/* Records Table */}
              <div>
                <h3 className="text-lg font-semibold mb-3">搜索记录</h3>
                {records.length === 0 ? (
                  <div className="text-center py-8 border border-dashed rounded-lg text-muted-foreground">
                    暂无记录
                  </div>
                ) : (
                  <div className="border border-border rounded-lg overflow-hidden">
                    <table className="w-full text-sm text-left">
                      <thead className="bg-accent/50 border-b border-border">
                        <tr>
                          <th className="w-10 px-4 py-2"></th>
                          <th className="px-4 py-2 font-medium">轮次</th>
                          <th className="px-4 py-2 font-medium">平台</th>
                          <th className="px-4 py-2 font-medium">关键词</th>
                          <th className="px-4 py-2 font-medium">回答摘要</th>
                          <th className="px-4 py-2 font-medium">耗时</th>
                          <th className="px-4 py-2 font-medium">状态</th>
                          <th className="px-4 py-2 font-medium">时间</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-border">
                        {records.map((record) => (
                          <>
                            <tr 
                              key={record.id} 
                              className="hover:bg-accent/10 transition-colors cursor-pointer"
                              onClick={() => toggleRecord(record.id)}
                            >
                              <td className="px-4 py-3">
                                {expandedRecords.includes(record.id) ? 
                                  <ChevronDown className="w-4 h-4 text-muted-foreground" /> : 
                                  <ChevronRight className="w-4 h-4 text-muted-foreground" />
                                }
                              </td>
                              <td className="px-4 py-3">{record.round_number}</td>
                              <td className="px-4 py-3 capitalize">{record.platform}</td>
                              <td className="px-4 py-3 font-medium">{record.keyword}</td>
                              <td className="px-4 py-3">
                                <div className="max-w-xs truncate" title={record.full_answer}>
                                  {record.full_answer || '-'}
                                </div>
                              </td>
                              <td className="px-4 py-3">{record.response_time_ms}ms</td>
                              <td className="px-4 py-3">
                                <span className={`px-2 py-0.5 rounded-full text-xs ${
                                  record.search_status === 'completed' ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'
                                }`}>
                                  {record.search_status === 'completed' ? '成功' : '失败'}
                                </span>
                              </td>
                              <td className="px-4 py-3 text-muted-foreground whitespace-nowrap">
                                {new Date(record.created_at).toLocaleString()}
                              </td>
                            </tr>
                            
                            {/* Expanded Details Row */}
                            {expandedRecords.includes(record.id) && (
                              <tr key={`${record.id}-details`}>
                                <td colSpan={8} className="bg-accent/5 p-0">
                                  <div className="p-4 space-y-4 border-b border-border">
                                    {/* Full Answer */}
                                    <div>
                                      <h4 className="font-semibold mb-2 text-xs uppercase tracking-wider text-muted-foreground">完整回答</h4>
                                      <div className="bg-background border border-border rounded-md p-4 whitespace-pre-wrap max-h-96 overflow-y-auto text-sm leading-relaxed">
                                        {record.full_answer || '无回答内容'}
                                      </div>
                                    </div>

                                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                                      {/* Search Queries */}
                                      <div>
                                        <h4 className="font-semibold mb-2 text-xs uppercase tracking-wider text-muted-foreground">搜索关键词 ({record.queries?.length || 0})</h4>
                                        <div className="bg-background border border-border rounded-md p-2 max-h-60 overflow-y-auto">
                                          {record.queries && record.queries.length > 0 ? (
                                            <ul className="space-y-1">
                                              {record.queries.map((q: any, idx: number) => (
                                                <li key={idx} className="text-sm px-2 py-1.5 hover:bg-accent/50 rounded flex gap-2">
                                                  <span className="text-muted-foreground w-4 text-right">{idx + 1}.</span>
                                                  <span>{q.query}</span>
                                                </li>
                                              ))}
                                            </ul>
                                          ) : (
                                            <div className="text-muted-foreground text-sm p-2">无搜索关键词记录</div>
                                          )}
                                        </div>
                                      </div>

                                      {/* Citations */}
                                      <div>
                                        <h4 className="font-semibold mb-2 text-xs uppercase tracking-wider text-muted-foreground">引用来源 ({record.citations?.length || 0})</h4>
                                        <div className="bg-background border border-border rounded-md p-2 max-h-60 overflow-y-auto">
                                          {record.citations && record.citations.length > 0 ? (
                                            <ul className="space-y-1">
                                              {record.citations.map((cite: any, idx: number) => (
                                                <li key={idx} className="text-sm border-b border-border/50 last:border-0 pb-2 last:pb-0 mb-2 last:mb-0">
                                                  <div className="flex items-start gap-2 p-1.5 hover:bg-accent/50 rounded group">
                                                    <span className="text-muted-foreground min-w-[1.5rem] text-xs pt-0.5">[{cite.cite_index}]</span>
                                                    <div className="flex-1 overflow-hidden">
                                                      <div className="font-medium truncate mb-0.5" title={cite.title}>{cite.title || '无标题'}</div>
                                                      <a 
                                                        href={cite.url} 
                                                        target="_blank" 
                                                        rel="noopener noreferrer" 
                                                        className="text-primary hover:underline text-xs flex items-center gap-1 truncate"
                                                        onClick={(e) => e.stopPropagation()}
                                                      >
                                                        {cite.domain || cite.url}
                                                        <ExternalLink className="w-3 h-3 opacity-0 group-hover:opacity-100 transition-opacity" />
                                                      </a>
                                                      {cite.snippet && (
                                                        <div className="text-xs text-muted-foreground mt-1 line-clamp-2" title={cite.snippet}>
                                                          {cite.snippet}
                                                        </div>
                                                      )}
                                                    </div>
                                                  </div>
                                                </li>
                                              ))}
                                            </ul>
                                          ) : (
                                            <div className="text-muted-foreground text-sm p-2">无引用来源记录</div>
                                          )}
                                        </div>
                                      </div>
                                    </div>
                                  </div>
                                </td>
                              </tr>
                            )}
                          </>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
            </div>
          ) : (
            <div className="text-center py-8 text-muted-foreground">未找到任务数据</div>
          )}
        </div>
      </div>
    </div>
  );
}
