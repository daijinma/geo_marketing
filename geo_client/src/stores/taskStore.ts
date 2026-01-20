import { create } from 'zustand';
import type { Task, TaskProgress } from '@/types/task';

interface TaskState {
  queue: Task[];
  current: Task | null;
  history: Task[];
  setQueue: (tasks: Task[]) => void;
  addToQueue: (task: Task) => void;
  setCurrent: (task: Task | null) => void;
  updateProgress: (progress: TaskProgress) => void;
  addToHistory: (task: Task) => void;
  removeFromQueue: (taskId: number) => void;
}

export const useTaskStore = create<TaskState>((set) => ({
  queue: [],
  current: null,
  history: [],
  setQueue: (tasks) => set({ queue: tasks }),
  addToQueue: (task) => set((state) => ({ queue: [...state.queue, task] })),
  setCurrent: (task) => set({ current: task }),
  updateProgress: (progress) => {
    set((state) => {
      if (state.current?.id === progress.task_id) {
        return {
          current: {
            ...state.current,
            status: progress.status as Task['status'],
          },
        };
      }
      return {};
    });
  },
  addToHistory: (task) =>
    set((state) => ({
      history: [task, ...state.history].slice(0, 100), // 保留最近100条
    })),
  removeFromQueue: (taskId) =>
    set((state) => ({
      queue: state.queue.filter((t) => t.id !== taskId),
    })),
}));
