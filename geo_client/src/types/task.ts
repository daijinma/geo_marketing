export interface Task {
  id?: number;
  task_id?: number; // 服务端任务ID
  keywords: string[];
  platforms: string[];
  query_count: number;
  status: 'pending' | 'running' | 'completed' | 'failed';
  result_data?: any;
  created_at?: string;
  updated_at?: string;
}

export interface TaskProgress {
  task_id: number;
  progress: number;
  status: string;
  platform_progress?: {
    [platform: string]: {
      completed: number;
      failed: number;
      pending: number;
      total: number;
    };
  };
}

export interface TaskResult {
  keyword: string;
  platform: string;
  status: 'completed' | 'failed';
  record_id?: number;
  citations_count?: number;
  response_time_ms?: number;
  error_message?: string;
}
