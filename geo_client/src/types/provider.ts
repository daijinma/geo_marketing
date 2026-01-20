export type PlatformType = 'llm' | 'platform';
export type PlatformName = 'deepseek' | 'doubao' | 'netease' | 'cnblogs';

export interface LoginStatus {
  platform_type: PlatformType;
  platform_name: PlatformName;
  is_logged_in: boolean;
  last_check_at?: string;
}

export interface ProviderSearchResult {
  full_text: string;
  queries: string[];
  citations: Citation[];
}

export interface Citation {
  url: string;
  title: string;
  snippet: string;
  site_name: string;
  cite_index: number;
  query_indexes?: number[];
}
