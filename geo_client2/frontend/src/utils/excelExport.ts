import * as XLSX from 'xlsx';

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
  }> = [];

  records.forEach((record) => {
    const citations = record.citations || [];
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
      });
    });
  });

  const worksheet = XLSX.utils.json_to_sheet(allCitations);

  const columnWidths = [
    { wch: 10 },
    { wch: 20 },
    { wch: 15 },
    { wch: 12 },
    { wch: 12 },
    { wch: 40 },
    { wch: 50 },
    { wch: 30 },
    { wch: 60 },
    { wch: 25 },
  ];
  worksheet['!cols'] = columnWidths;

  const workbook = XLSX.utils.book_new();
  XLSX.utils.book_append_sheet(workbook, worksheet, 'All Citations');

  const defaultFilename = `citations_merged_${Date.now()}.xlsx`;
  XLSX.writeFile(workbook, filename || defaultFilename);
}
