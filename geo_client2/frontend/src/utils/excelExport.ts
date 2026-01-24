import * as XLSX from 'xlsx';
import { wailsAPI } from './wails-api';

interface Citation {
  cite_index?: number;
  url?: string;
  domain?: string;
  title?: string;
  snippet?: string;
  site_name?: string;
}

interface ExportCitationData {
  taskId?: number;
  keyword?: string;
  platform?: string;
  roundNumber?: number;
  citations: Citation[];
}

export function exportCitationsToExcel(data: ExportCitationData, filename?: string) {
  const citationsData = data.citations.map((citation) => ({
    cite_index: citation.cite_index || '',
    title: citation.title || '',
    url: citation.url || '',
    domain: citation.domain || '',
    snippet: citation.snippet || '',
    site_name: citation.site_name || '',
  }));

  const worksheet = XLSX.utils.json_to_sheet(citationsData);

  const columnWidths = [
    { wch: 12 },
    { wch: 40 },
    { wch: 50 },
    { wch: 30 },
    { wch: 60 },
    { wch: 25 },
  ];
  worksheet['!cols'] = columnWidths;

  const workbook = XLSX.utils.book_new();
  XLSX.utils.book_append_sheet(workbook, worksheet, 'Citations');

  const defaultFilename = `citations_task${data.taskId || 'merged'}_${data.keyword || 'all'}_${Date.now()}.xlsx`;
  XLSX.writeFile(workbook, filename || defaultFilename);
}

export function exportMultipleRecordsCitations(
  records: Array<{
    task_id?: number;
    keyword?: string;
    platform?: string;
    round_number?: number;
    citations?: Citation[];
    full_answer?: string;
    response_time_ms?: number;
    search_status?: string;
    created_at?: string;
  }>,
  filename?: string
) {
  const allCitations: Array<{
    task_id: number | string;
    keyword: string;
    platform: string;
    round_number: number | string;
    cite_index: number | string;
    title: string;
    url: string;
    domain: string;
    snippet: string;
    site_name: string;
    full_answer: string;
    response_time_ms: number | string;
    search_status: string;
    created_at: string;
  }> = [];

  records.forEach((record) => {
    const citations = record.citations || [];
    if (citations.length === 0) {
      allCitations.push({
        task_id: record.task_id || '',
        keyword: record.keyword || '',
        platform: record.platform || '',
        round_number: record.round_number || '',
        cite_index: '-',
        title: '-',
        url: '-',
        domain: '-',
        snippet: '-',
        site_name: '-',
        full_answer: record.full_answer || '',
        response_time_ms: record.response_time_ms || '',
        search_status: record.search_status || '',
        created_at: record.created_at ? new Date(record.created_at).toLocaleString() : '',
      });
    } else {
      citations.forEach((citation) => {
        allCitations.push({
          task_id: record.task_id || '',
          keyword: record.keyword || '',
          platform: record.platform || '',
          round_number: record.round_number || '',
          cite_index: citation.cite_index || '',
          title: citation.title || '',
          url: citation.url || '',
          domain: citation.domain || '',
          snippet: citation.snippet || '',
          site_name: citation.site_name || '',
          full_answer: record.full_answer || '',
          response_time_ms: record.response_time_ms || '',
          search_status: record.search_status || '',
          created_at: record.created_at ? new Date(record.created_at).toLocaleString() : '',
        });
      });
    }
  });

  const worksheet = XLSX.utils.json_to_sheet(allCitations);

  XLSX.utils.sheet_add_aoa(worksheet, [[
    "任务ID", "关键词", "平台", "轮次", "引用序号", "标题", "URL", "域名", "摘要", "站点名称", "完整回答", "耗时(ms)", "状态", "创建时间"
  ]], { origin: "A1" });

  const columnWidths = [
    { wch: 10 },
    { wch: 20 },
    { wch: 15 },
    { wch: 10 },
    { wch: 10 },
    { wch: 40 },
    { wch: 50 },
    { wch: 30 },
    { wch: 60 },
    { wch: 25 },
    { wch: 80 },
    { wch: 12 },
    { wch: 10 },
    { wch: 20 },
  ];
  worksheet['!cols'] = columnWidths;

  const workbook = XLSX.utils.book_new();
  XLSX.utils.book_append_sheet(workbook, worksheet, 'Search Records');

  const defaultFilename = `search_records_${Date.now()}.xlsx`;
  const finalFilename = filename || defaultFilename;
  
  try {
    const wbout = XLSX.write(workbook, { bookType: 'xlsx', type: 'base64' });
    wailsAPI.fs.saveExcel(finalFilename, wbout).then(result => {
      if (result.success && result.path) {
        console.log('File saved to:', result.path);
      }
    }).catch(err => {
      console.error('Failed to save file via Wails:', err);
      XLSX.writeFile(workbook, finalFilename);
    });
  } catch (error) {
    console.error('Error generating Excel base64:', error);
    XLSX.writeFile(workbook, finalFilename);
  }
}
