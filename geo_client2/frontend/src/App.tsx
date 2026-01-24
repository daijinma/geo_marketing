import { useEffect, useRef } from 'react';
import { BrowserRouter, Routes, Route, useLocation } from 'react-router-dom';
import { Toaster, toast } from 'sonner';
import Layout from './components/Layout';
import Dashboard from './pages/Dashboard';
import Search from './pages/Search';
import Auth from './pages/Auth';
import Tasks from './pages/Tasks';
import Logs from './pages/Logs';
import Settings from './pages/Settings';
import { ErrorBoundary } from './components/ErrorBoundary';
import { EventsOn } from './wailsjs/runtime/runtime';
import { logger } from './utils/logger';

// Navigation logger component
function NavigationLogger() {
  const location = useLocation();
  const prevPath = useRef(location.pathname);

  useEffect(() => {
    if (prevPath.current !== location.pathname) {
      logger.logNavigation(prevPath.current, location.pathname);
      prevPath.current = location.pathname;
    }
  }, [location]);

  return null;
}

function App() {
  useEffect(() => {
    // Log app startup
    logger.info('Application started', {
      component: 'App',
      userAction: 'startup',
    });

    // Listen for test events
    const unsubTest = EventsOn('test', (data: unknown) => {
      if (typeof data === 'object' && data !== null && 'message' in data) {
        toast.info((data as { message: string }).message);
      }
    });

    // Listen for task updates
    const unsubTask = EventsOn('search:taskUpdated', (data: unknown) => {
      logger.debug('Task update received', {
        component: 'App',
        details: { data },
      });
    });

    // Listen for login status changes
    const unsubLogin = EventsOn('login-status-changed', (data: unknown) => {
      if (typeof data === 'object' && data !== null) {
        const d = data as { platform?: string; isLoggedIn?: boolean };
        logger.info(`Login status changed: ${d.platform}`, {
          component: 'App',
          details: { platform: d.platform, isLoggedIn: d.isLoggedIn },
        });
        
        if (d.platform && !d.isLoggedIn) {
          const platformName = d.platform === 'deepseek' ? 'DeepSeek' : d.platform === 'doubao' ? '豆包' : d.platform;
          toast.warning(`${platformName} 登录已过期`, {
            description: '请重新登录以继续使用',
            duration: 5000,
          });
        }
      }
    });

    // Listen for task login required
    const unsubTaskLogin = EventsOn('task-login-required', (data: unknown) => {
      if (typeof data === 'object' && data !== null) {
        const d = data as { platformName?: string; keyword?: string };
        logger.warn(`Task login required: ${d.platformName}`, {
          component: 'App',
          details: { platform: d.platformName, keyword: d.keyword },
        });
        
        toast.error(`${d.platformName || '平台'} 未登录`, {
          description: `任务执行失败：关键词 "${d.keyword || ''}" 需要先完成登录授权`,
          duration: 5000,
        });
      }
    });

    // Log unhandled errors
    const handleError = (event: ErrorEvent) => {
      logger.error('Unhandled error', {
        component: 'Window',
        details: {
          message: event.message,
          filename: event.filename,
          lineno: event.lineno,
          colno: event.colno,
        },
      });
    };

    const handleRejection = (event: PromiseRejectionEvent) => {
      logger.error('Unhandled promise rejection', {
        component: 'Window',
        details: {
          reason: event.reason?.message || String(event.reason),
        },
      });
    };

    window.addEventListener('error', handleError);
    window.addEventListener('unhandledrejection', handleRejection);

    return () => {
      unsubTest();
      unsubTask();
      unsubLogin();
      unsubTaskLogin();
      window.removeEventListener('error', handleError);
      window.removeEventListener('unhandledrejection', handleRejection);
    };
  }, []);

  return (
    <ErrorBoundary>
      <Toaster position="top-right" richColors closeButton />
      <BrowserRouter>
        <NavigationLogger />
        <Routes>
          <Route path="/" element={<Layout />}>
            <Route index element={<Dashboard />} />
            <Route path="search" element={<Search />} />
            <Route path="auth" element={<Auth />} />
            <Route path="tasks" element={<Tasks />} />
            <Route path="logs" element={<Logs />} />
            <Route path="settings" element={<Settings />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </ErrorBoundary>
  );
}

export default App;
