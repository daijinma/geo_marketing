import { BrowserWindow, BrowserView } from 'electron';

/**
 * BrowserView 管理器
 * 用于在 Electron 窗口内显示浏览器页面
 */
export class BrowserViewManager {
  private static instance: BrowserViewManager;
  private browserView: BrowserView | null = null;
  private mainWindow: BrowserWindow | null = null;

  private constructor() {}

  static getInstance(): BrowserViewManager {
    if (!BrowserViewManager.instance) {
      BrowserViewManager.instance = new BrowserViewManager();
    }
    return BrowserViewManager.instance;
  }

  /**
   * 设置主窗口
   */
  setMainWindow(window: BrowserWindow): void {
    this.mainWindow = window;
  }

  /**
   * 显示 BrowserView
   */
  show(url: string): void {
    if (!this.mainWindow) {
      console.error('[BrowserViewManager] 主窗口未设置');
      return;
    }

    // 如果已存在，先销毁
    if (this.browserView) {
      this.hide();
    }

    this.browserView = new BrowserView({
      webPreferences: {
        nodeIntegration: false,
        contextIsolation: true,
        sandbox: false,
      },
    });

    this.mainWindow.addBrowserView(this.browserView);

    // 设置 BrowserView 的位置和大小（占据窗口右侧 50%）
    this.updateBounds();

    // 加载 URL
    this.browserView.webContents.loadURL(url);

    // 监听窗口大小变化，调整 BrowserView
    this.mainWindow.on('resize', () => {
      this.updateBounds();
    });
  }

  /**
   * 隐藏 BrowserView
   */
  hide(): void {
    if (this.browserView && this.mainWindow) {
      this.mainWindow.removeBrowserView(this.browserView);
      this.browserView = null;
    }
  }

  /**
   * 更新 BrowserView 的边界
   */
  private updateBounds(): void {
    if (!this.browserView || !this.mainWindow) {
      return;
    }

    const bounds = this.mainWindow.getBounds();
    const viewWidth = Math.floor(bounds.width * 0.5);
    this.browserView.setBounds({
      x: bounds.width - viewWidth,
      y: 0,
      width: viewWidth,
      height: bounds.height,
    });
  }

  /**
   * 获取当前的 BrowserView
   */
  getBrowserView(): BrowserView | null {
    return this.browserView;
  }

  /**
   * 检查 BrowserView 是否显示
   */
  isVisible(): boolean {
    return this.browserView !== null;
  }
}
