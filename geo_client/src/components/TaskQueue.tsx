import { useTaskStore } from '@/stores/taskStore';
import { Pause, X } from 'lucide-react';

export default function TaskQueue() {
  const { queue, current } = useTaskStore();

  return (
    <div className="border-t border-border pt-4">
      <h3 className="text-sm font-semibold text-muted-foreground mb-2">任务队列</h3>
      <div className="space-y-2">
        {current && (
          <div className="p-2 bg-primary/10 rounded-md border border-primary/20">
            <div className="flex items-center justify-between mb-1">
              <span className="text-xs font-medium">执行中</span>
              <span className="text-xs text-muted-foreground">任务 #{current.id}</span>
            </div>
            <div className="text-xs text-muted-foreground">
              {current.platforms.join(', ')} - {current.keywords.length} 个关键词
            </div>
          </div>
        )}
        {queue.length > 0 && (
          <div className="space-y-1">
            {queue.map((task) => (
              <div
                key={task.id}
                className="p-2 bg-muted/50 rounded-md text-xs"
              >
                <div className="flex items-center justify-between">
                  <span>任务 #{task.id}</span>
                  <div className="flex items-center gap-1">
                    <button
                      className="p-1 hover:bg-accent rounded transition-colors"
                      title="暂停"
                    >
                      <Pause className="w-3 h-3" />
                    </button>
                    <button
                      className="p-1 hover:bg-accent rounded transition-colors"
                      title="取消"
                    >
                      <X className="w-3 h-3" />
                    </button>
                  </div>
                </div>
                <div className="text-muted-foreground mt-1">
                  {task.platforms.join(', ')}
                </div>
              </div>
            ))}
          </div>
        )}
        {queue.length === 0 && !current && (
          <div className="text-xs text-muted-foreground text-center py-4">
            暂无待执行任务
          </div>
        )}
      </div>
    </div>
  );
}
