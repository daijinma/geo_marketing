import { useState, useEffect, useMemo } from 'react';
import { X, Play, ChevronDown, ChevronRight, ExternalLink, BarChart3, Link2, Search, Globe, Download } from 'lucide-react';
import { toast } from 'sonner';
import { wailsAPI } from '@/utils/wails-api';
import { exportMultipleRecordsCitations } from '@/utils/excelExport';

interface LocalTaskDetailProps {
  taskId: number;
  onClose: () => void;
}

interface OverviewStats {
  recordCount: number;
  subQueryCount: number;
  citationCount: number;
  uniqueDomainCount: number;
}

interface SearchStatsItem {
  keyword: string;
  platform: string;
  subQueryCount: number;
  citationCount: number;
}

interface DomainStatsItem {
  domain: string;
  total: number;
  byKeyword: Record<string, number>;
}

function getDomainFromUrl(urlStr: string): string {
  if (!urlStr) return '';
  try {
    const url = new URL(urlStr);
    return url.hostname;
  } catch (e) {
    return '';
  }
}

function computeStats(records: any[]): {
  overview: OverviewStats;
  searchStats: SearchStatsItem[];
  domainStats: { domains: DomainStatsItem[]; keywords: string[] };
} {
  // 总体统计
  let subQueryCount = 0;
  let citationCount = 0;
  const allDomains = new Set<string>();
  
  // 按关键词+平台聚合
  const searchStatsMap = new Map<string, SearchStatsItem>();
  
  // 域名统计
  const domainStatsMap = new Map<string, { total: number; byKeyword: Record<string, number> }>();
  const allKeywords = new Set<string>();
  
  records.forEach(record => {
    const queriesLen = record.queries?.length || 0;
    const citationsLen = record.citations?.length || 0;
    
    subQueryCount += queriesLen;
    citationCount += citationsLen;
    
    // 收集域名
    record.citations?.forEach((cite: any) => {
      const domain = getDomainFromUrl(cite.url) || cite.domain;
      if (domain) {
        allDomains.add(domain);
      }
    });
    
    // 按关键词+平台聚合
    const key = `${record.keyword}|${record.platform}`;
    if (!searchStatsMap.has(key)) {
      searchStatsMap.set(key, {
        keyword: record.keyword,
        platform: record.platform,
        subQueryCount: 0,
        citationCount: 0,
      });
    }
    const item = searchStatsMap.get(key)!;
    item.subQueryCount += queriesLen;
    item.citationCount += citationsLen;
    
    // 域名按关键词统计
    const keyword = record.keyword || '(未知)';
    allKeywords.add(keyword);
    
    record.citations?.forEach((cite: any) => {
      const domain = getDomainFromUrl(cite.url) || cite.domain;
      if (!domain) return;
      
      if (!domainStatsMap.has(domain)) {
        domainStatsMap.set(domain, { total: 0, byKeyword: {} });
      }
      const domainItem = domainStatsMap.get(domain)!;
      domainItem.total += 1;
      domainItem.byKeyword[keyword] = (domainItem.byKeyword[keyword] || 0) + 1;
    });
  });
  
  // 转换为数组并排序
  const searchStats = Array.from(searchStatsMap.values());
  
  const domains = Array.from(domainStatsMap.entries())
    .map(([domain, data]) => ({ domain, ...data }))
    .sort((a, b) => b.total - a.total);
  
  const keywords = Array.from(allKeywords).sort();
  
  return {
    overview: {
      recordCount: records.length,
      subQueryCount,
      citationCount,
      uniqueDomainCount: allDomains.size,
    },
    searchStats,
    domainStats: { domains, keywords },
  };
}

export function LocalTaskDetail({ taskId, onClose }: LocalTaskDetailProps) {
  const [loading, setLoading] = useState(true);
  const [taskData, setTaskData] = useState<any>(null);
  const [records, setRecords] = useState<any[]>([]);
  const [expandedRecords, setExpandedRecords] = useState<number[]>([]);
  
  // 计算统计数据
  const stats = useMemo(() => computeStats(records), [records]);

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

  const handleExportExcel = () => {
    if (records.length === 0) {
      toast.error('没有可导出的数据');
      return;
    }
    
    try {
      exportMultipleRecordsCitations(records);
      toast.success('数据已导出为 Excel');
    } catch (error: any) {
      toast.error('导出失败', { description: error.message });
    }
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
              <div className="grid grid-cols-2 md:grid-cols-5 gap-4 p-4 bg-accent/30 rounded-lg text-sm">
                <div>
                  <div className="text-muted-foreground">任务ID</div>
                  <div className="font-medium">{taskData.id}</div>
                </div>
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

              {/* Statistics Section */}
              {records.length > 0 && (
                <div className="space-y-4">
                  {/* Overview Cards */}
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                    <div className="p-4 bg-accent/20 rounded-lg border border-border">
                      <div className="flex items-center gap-2 text-muted-foreground text-sm mb-1">
                        <BarChart3 className="w-4 h-4" />
                        搜索记录数
                      </div>
                      <div className="text-2xl font-bold">{stats.overview.recordCount}</div>
                    </div>
                    <div className="p-4 bg-accent/20 rounded-lg border border-border">
                      <div className="flex items-center gap-2 text-muted-foreground text-sm mb-1">
                        <Search className="w-4 h-4" />
                        Sub Query 总数
                      </div>
                      <div className="text-2xl font-bold">{stats.overview.subQueryCount}</div>
                    </div>
                    <div className="p-4 bg-accent/20 rounded-lg border border-border">
                      <div className="flex items-center gap-2 text-muted-foreground text-sm mb-1">
                        <Link2 className="w-4 h-4" />
                        引用链接总数
                      </div>
                      <div className="text-2xl font-bold">{stats.overview.citationCount}</div>
                    </div>
                    <div className="p-4 bg-accent/20 rounded-lg border border-border">
                      <div className="flex items-center gap-2 text-muted-foreground text-sm mb-1">
                        <Globe className="w-4 h-4" />
                        唯一域名数
                      </div>
                      <div className="text-2xl font-bold">{stats.overview.uniqueDomainCount}</div>
                    </div>
                  </div>

                  {/* Search Stats Table */}
                  <div className="border border-border rounded-lg overflow-hidden">
                    <div className="p-3 bg-accent/30">
                      <span className="font-medium text-sm">搜索词统计</span>
                    </div>
                    <div className="max-h-60 overflow-y-auto">
                      <table className="w-full text-sm text-left">
                        <thead className="bg-accent/20 border-b border-border sticky top-0">
                          <tr>
                            <th className="px-4 py-2 font-medium">关键词</th>
                            <th className="px-4 py-2 font-medium">平台</th>
                            <th className="px-4 py-2 font-medium text-right">Sub Query 数</th>
                            <th className="px-4 py-2 font-medium text-right">引用链接数</th>
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-border">
                          {stats.searchStats.map((item, idx) => (
                            <tr key={idx} className="hover:bg-accent/10">
                              <td className="px-4 py-2 font-medium">{item.keyword}</td>
                              <td className="px-4 py-2 capitalize">{item.platform}</td>
                              <td className="px-4 py-2 text-right">{item.subQueryCount}</td>
                              <td className="px-4 py-2 text-right">{item.citationCount}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  </div>

                  {/* Domain Stats Table */}
                  <div className="border border-border rounded-lg overflow-hidden">
                    <div className="p-3 bg-accent/30">
                      <span className="font-medium text-sm">Domain 统计 ({stats.domainStats.domains.length} 个域名)</span>
                    </div>
                    {stats.domainStats.domains.length > 0 ? (
                      <div className="max-h-80 overflow-auto">
                        <table className="w-full text-sm text-left">
                          <thead className="bg-accent/20 border-b border-border sticky top-0">
                            <tr>
                              <th className="px-4 py-2 font-medium sticky left-0 bg-accent/20 z-10 min-w-[200px]">Domain</th>
                              <th className="px-4 py-2 font-medium text-left min-w-[200px]">链接</th>
                              <th className="px-4 py-2 font-medium text-right min-w-[60px]">总计</th>
                              {stats.domainStats.keywords.map(kw => (
                                <th key={kw} className="px-4 py-2 font-medium text-right min-w-[80px] truncate" title={kw}>
                                  {kw.length > 10 ? kw.slice(0, 10) + '...' : kw}
                                </th>
                              ))}
                            </tr>
                          </thead>
                          <tbody className="divide-y divide-border">
                            {stats.domainStats.domains.map((item, idx) => (
                              <tr key={idx} className="hover:bg-accent/10">
                                <td className="px-4 py-2 font-medium sticky left-0 bg-card z-10 truncate max-w-[200px]" title={item.domain}>
                                  {item.domain}
                                </td>
                                <td className="px-4 py-2 text-left">
                                  <button 
                                    onClick={() => wailsAPI.browser.openURL(`https://${item.domain}`)}
                                    className="flex items-center gap-1 text-primary hover:underline truncate max-w-[200px]"
                                  >
                                    https://{item.domain}
                                    <ExternalLink className="w-3 h-3" />
                                  </button>
                                </td>
                                <td className="px-4 py-2 text-right font-semibold">{item.total}</td>
                                {stats.domainStats.keywords.map(kw => (
                                  <td key={kw} className="px-4 py-2 text-right text-muted-foreground">
                                    {item.byKeyword[kw] || '-'}
                                  </td>
                                ))}
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                    ) : (
                      <div className="p-4 text-center text-muted-foreground text-sm">暂无域名数据</div>
                    )}
                  </div>
                </div>
              )}

              {/* Records Table */}
              <div>
                <div className="flex items-center justify-between mb-3">
                  <h3 className="text-lg font-semibold">搜索记录</h3>
                  {records.length > 0 && (
                    <button
                      onClick={handleExportExcel}
                      className="flex items-center gap-2 px-3 py-1.5 text-sm bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
                    >
                      <Download className="w-4 h-4" />
                      导出Excel
                    </button>
                  )}
                </div>
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
                                                        <button 
                                                          onClick={(e) => {
                                                            e.stopPropagation();
                                                            wailsAPI.browser.openURL(cite.url);
                                                          }}
                                                          className="text-primary hover:underline text-xs flex items-center gap-1 truncate"
                                                        >
                                                          {cite.domain || cite.url}
                                                          <ExternalLink className="w-3 h-3 opacity-0 group-hover:opacity-100 transition-opacity" />
                                                        </button>
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
