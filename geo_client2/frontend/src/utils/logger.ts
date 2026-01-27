// Log levels
export type LogLevel = 'DEBUG' | 'INFO' | 'WARN' | 'ERROR';

// Session ID - persisted across app restarts
const SESSION_ID_KEY = 'geo_client2_session_id';

// Simple UUID v4 generator
function generateUUID(): string {
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    const v = c === 'x' ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}

// Get or create session ID
function getSessionId(): string {
  let sessionId = localStorage.getItem(SESSION_ID_KEY);
  if (!sessionId) {
    sessionId = generateUUID();
    localStorage.setItem(SESSION_ID_KEY, sessionId);
  }
  return sessionId;
}

// Generate correlation ID for tracking related operations
export function generateCorrelationId(): string {
  return generateUUID();
}

interface LogContext {
  component?: string;
  userAction?: string;
  details?: Record<string, any>;
  correlationId?: string;
  taskId?: number;
  performanceMs?: number;
}

class Logger {
  private sessionId: string;
  private isWailsAvailable: boolean = false;

  constructor() {
    this.sessionId = getSessionId();
    this.checkWailsAvailability();
  }

  private checkWailsAvailability() {
    // Check if we're running in Wails environment
    this.isWailsAvailable = typeof window !== 'undefined' && 
                           !!(window as any).go?.backend?.App?.AddLog;
  }

  private async sendToBackend(
    level: LogLevel,
    message: string,
    context?: LogContext
  ) {
    if (!this.isWailsAvailable) {
      // Fallback to console if not in Wails
      const consoleMethod = level === 'ERROR' ? 'error' : level === 'WARN' ? 'warn' : 'log';
      console[consoleMethod](`[${level}]`, message, context);
      return;
    }

    try {
      const details = context?.details;
      let detailsJSON = '';
      if (details) {
        try {
          detailsJSON = JSON.stringify(details);
        } catch (e) {
          detailsJSON = JSON.stringify({ error: 'Failed to stringify details', message: String(e) });
        }
      }
      
      const app = (window as any).go?.main?.App ?? 
                  (window as any).go?.geo_client2?.App ?? 
                  (window as any).go?.backend?.App;

      if (!app?.AddLog) {
        console.error('Wails AddLog method not found in any expected location');
        return;
      }

      await app.AddLog(
        level,
        'frontend',
        message,
        detailsJSON,
        context?.correlationId || this.sessionId,
        context?.correlationId || '',
        context?.component || '',
        context?.userAction || '',
        context?.performanceMs ?? null,
        context?.taskId ?? null
      );
    } catch (error) {
      console.error('Failed to send log to backend:', error);
    }
  }

  debug(message: string, context?: LogContext) {
    this.sendToBackend('DEBUG', message, context);
  }

  info(message: string, context?: LogContext) {
    this.sendToBackend('INFO', message, context);
  }

  warn(message: string, context?: LogContext) {
    this.sendToBackend('WARN', message, context);
  }

  error(message: string, context?: LogContext) {
    this.sendToBackend('ERROR', message, context);
  }

  // Log user action
  logUserAction(action: string, component: string, details?: Record<string, any>) {
    this.info(`User action: ${action}`, {
      component,
      userAction: action,
      details,
    });
  }

  // Log API call
  logApiCall(method: string, params: any, correlationId?: string) {
    this.debug(`API call: ${method}`, {
      component: 'API',
      correlationId,
      details: { method, params },
    });
  }

  // Log API response
  logApiResponse(method: string, success: boolean, duration: number, correlationId?: string) {
    this.info(`API response: ${method} - ${success ? 'success' : 'failed'}`, {
      component: 'API',
      correlationId,
      performanceMs: duration,
      details: { method, success },
    });
  }

  // Log API error
  logApiError(method: string, error: any, correlationId?: string) {
    this.error(`API error: ${method}`, {
      component: 'API',
      correlationId,
      details: {
        method,
        error: error?.message || String(error),
        stack: error?.stack,
      },
    });
  }

  // Log performance metric
  logPerformance(operation: string, durationMs: number, component?: string, details?: Record<string, any>) {
    this.info(`Performance: ${operation}`, {
      component: component || 'Performance',
      performanceMs: durationMs,
      details,
    });
  }

  // Log navigation
  logNavigation(from: string, to: string) {
    this.info(`Navigation: ${from} → ${to}`, {
      component: 'Router',
      userAction: 'navigate',
      details: { from, to },
    });
  }

  // Log state change
  logStateChange(store: string, action: string, details?: Record<string, any>) {
    this.debug(`State change: ${store}.${action}`, {
      component: store,
      userAction: action,
      details,
    });
  }

  // Create a performance timer
  startTimer(operation: string, component?: string): PerformanceTimer {
    return new PerformanceTimer(this, operation, component);
  }

  getSessionId(): string {
    return this.sessionId;
  }
}

// Performance timer helper
class PerformanceTimer {
  private startTime: number;
  private operation: string;
  private component?: string;
  private logger: Logger;
  private correlationId: string;

  constructor(logger: Logger, operation: string, component?: string) {
    this.logger = logger;
    this.operation = operation;
    this.component = component;
    this.startTime = performance.now();
    this.correlationId = generateCorrelationId();
  }

  end(details?: Record<string, any>) {
    const duration = Math.round(performance.now() - this.startTime);
    this.logger.logPerformance(this.operation, duration, this.component, details);
  }

  getCorrelationId(): string {
    return this.correlationId;
  }
}

// Export singleton instance
export const logger = new Logger();

// Export timer class for external use
export { PerformanceTimer };
