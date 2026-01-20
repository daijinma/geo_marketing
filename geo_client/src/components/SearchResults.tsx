import { useState } from 'react';
import { ChevronDown, ChevronRight, ExternalLink } from 'lucide-react';

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

interface SearchResultsProps {
  results: SearchResult[];
}

export default function SearchResults({ results }: SearchResultsProps) {
  const [activeTab, setActiveTab] = useState<'summary' | 'domain' | 'detail'>('summary');
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set());

  const toggleRow = (key: string) => {
    const newExpanded = new Set(expandedRows);
    if (newExpanded.has(key)) {
      newExpanded.delete(key);
    } else {
      newExpanded.add(key);
    }
    setExpandedRows(newExpanded);
  };

  // 汇总表格数据
  const summaryData: Array<{
    query: string;
    subQuery: string;
    citationsCount: number;
    responseTimeMs: number;
  }> = results.map((result) => ({
    query: result.query,
    subQuery: result.subQuery || '',
    citationsCount: result.citationsCount,
    responseTimeMs: result.responseTimeMs,
  }));

  // 计算域名统计
  const domainStats: Record<string, number> = {};
  results.forEach((result) => {
    result.items.forEach((item) => {
      if (item.domain) {
        domainStats[item.domain] = (domainStats[item.domain] || 0) + 1;
      }
    });
  });

  // 详细日志数据
  const detailLogs = results.flatMap((result) =>
    result.items.map((item) => ({
      query: result.query,
      subQuery: result.subQuery || '',
      domain: item.domain || '',
      title: item.title,
      snippet: item.snippet,
      url: item.url,
    }))
  );

  return (
    <div className="bg-card border border-border rounded-lg p-6 space-y-4">
      <h2 className="text-xl font-bold">搜索结果</h2>

      {/* 标签页 */}
      <div className="flex gap-2 border-b border-border">
        <button
          onClick={() => setActiveTab('summary')}
          className={`px-4 py-2 font-medium transition-colors ${
            activeTab === 'summary'
              ? 'border-b-2 border-primary text-primary'
              : 'text-muted-foreground hover:text-foreground'
          }`}
        >
          汇总表格 ({summaryData.length})
        </button>
        <button
          onClick={() => setActiveTab('domain')}
          className={`px-4 py-2 font-medium transition-colors ${
            activeTab === 'domain'
              ? 'border-b-2 border-primary text-primary'
              : 'text-muted-foreground hover:text-foreground'
          }`}
        >
          Domain统计
        </button>
        <button
          onClick={() => setActiveTab('detail')}
          className={`px-4 py-2 font-medium transition-colors ${
            activeTab === 'detail'
              ? 'border-b-2 border-primary text-primary'
              : 'text-muted-foreground hover:text-foreground'
          }`}
        >
          详细日志 ({detailLogs.length})
        </button>
      </div>

      {/* 汇总表格 */}
      {activeTab === 'summary' && (
        <div className="overflow-x-auto">
          <table className="w-full border-collapse">
            <thead>
              <tr className="border-b border-border">
                <th className="text-left p-2">查询词</th>
                <th className="text-left p-2">Sub Query</th>
                <th className="text-left p-2">引用数</th>
                <th className="text-left p-2">响应时间 (ms)</th>
              </tr>
            </thead>
            <tbody>
              {summaryData.map((item, idx) => (
                <tr key={idx} className="border-b border-border hover:bg-accent/50">
                  <td className="p-2">{item.query}</td>
                  <td className="p-2">{item.subQuery}</td>
                  <td className="p-2">{item.citationsCount}</td>
                  <td className="p-2">{item.responseTimeMs}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Domain统计 */}
      {activeTab === 'domain' && (
        <div className="overflow-x-auto">
          <table className="w-full border-collapse">
            <thead>
              <tr className="border-b border-border">
                <th className="text-left p-2">域名</th>
                <th className="text-left p-2">总计</th>
              </tr>
            </thead>
            <tbody>
              {Object.entries(domainStats)
                .sort((a, b) => b[1] - a[1])
                .map(([domain, count]) => (
                  <tr key={domain} className="border-b border-border hover:bg-accent/50">
                    <td className="p-2">{domain}</td>
                    <td className="p-2">{count}</td>
                  </tr>
                ))}
            </tbody>
          </table>
        </div>
      )}

      {/* 详细日志 */}
      {activeTab === 'detail' && (
        <div className="space-y-2 max-h-[600px] overflow-y-auto">
          {detailLogs.map((log, idx) => {
            const rowKey = `detail-${idx}`;
            const isExpanded = expandedRows.has(rowKey);

            return (
              <div key={idx} className="border border-border rounded-md">
                <div
                  className="p-3 hover:bg-accent/50 cursor-pointer flex items-center justify-between"
                  onClick={() => toggleRow(rowKey)}
                >
                  <div className="flex items-center gap-2 flex-1">
                    {isExpanded ? (
                      <ChevronDown className="w-4 h-4" />
                    ) : (
                      <ChevronRight className="w-4 h-4" />
                    )}
                    <span className="text-sm font-medium">{log.query}</span>
                    <span className="text-xs text-muted-foreground">{log.domain}</span>
                    <span className="text-xs text-muted-foreground truncate flex-1">
                      {log.title}
                    </span>
                  </div>
                  <a
                    href={log.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    onClick={(e) => e.stopPropagation()}
                    className="p-1 hover:bg-accent rounded transition-colors"
                  >
                    <ExternalLink className="w-3 h-3" />
                  </a>
                </div>
                {isExpanded && (
                  <div className="p-3 pt-0 border-t border-border space-y-2 text-sm">
                    <div>
                      <span className="font-medium">Sub Query:</span>{' '}
                      <span className="text-muted-foreground">{log.subQuery}</span>
                    </div>
                    <div>
                      <span className="font-medium">标题:</span>{' '}
                      <span className="text-muted-foreground">{log.title}</span>
                    </div>
                    {log.snippet && (
                      <div>
                        <span className="font-medium">摘要:</span>{' '}
                        <span className="text-muted-foreground">{log.snippet}</span>
                      </div>
                    )}
                    <div>
                      <span className="font-medium">网址:</span>{' '}
                      <a
                        href={log.url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-primary hover:underline break-all"
                      >
                        {log.url}
                      </a>
                    </div>
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
