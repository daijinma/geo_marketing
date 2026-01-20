import { app, BrowserWindow, BrowserView, ipcMain, session } from 'electron';
import { join } from 'path';
import { initAuthHandlers } from './services/auth';
import { initSearchHandlers } from './services/search';
import { initTaskManagerHandlers } from './services/task-manager';
import { initProviderHandlers } from './services/provider';
import { BrowserViewManager } from './services/browser-view-manager';
import { initDb } from './database';

// 开发环境判断（使用 process.defaultApp 而不是 app.isPackaged，避免在模块加载时访问 app）
const isDev = process.env.NODE_ENV === 'development' || process.defaultApp || /node_modules[\\/]electron[\\/]/.test(process.execPath);

let mainWindow: BrowserWindow | null = null;

function createWindow() {
  // 创建主窗口
  mainWindow = new BrowserWindow({
    width: 1200,
    height: 800,
    minWidth: 800,
    minHeight: 600,
    webPreferences: {
      preload: join(__dirname, 'preload.js'),
      nodeIntegration: false,
      contextIsolation: true,
      sandbox: false,
    },
    titleBarStyle: process.platform === 'darwin' ? 'hiddenInset' : 'default',
  });

  // 设置 BrowserView 管理器的主窗口
  BrowserViewManager.getInstance().setMainWindow(mainWindow);

  // 加载应用
  if (isDev) {
    mainWindow.loadURL('http://localhost:1420');
    mainWindow.webContents.openDevTools();
  } else {
    mainWindow.loadFile(join(__dirname, '../dist/index.html'));
  }

  mainWindow.on('closed', () => {
    mainWindow = null;
  });
}

// 应用准备就绪
app.whenReady().then(() => {
  // 初始化数据库
  initDb();
  
  // 初始化 IPC 处理程序
  initAuthHandlers();
  initSearchHandlers();
  initTaskManagerHandlers();
  initProviderHandlers();
  
  // 注册 BrowserView IPC 处理程序
  ipcMain.handle('browser-view:show', (event, url: string) => {
    BrowserViewManager.getInstance().show(url);
  });
  
  ipcMain.handle('browser-view:hide', () => {
    BrowserViewManager.getInstance().hide();
  });

  createWindow();

  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      createWindow();
    }
  });
});

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') {
    app.quit();
  }
});

// IPC 处理程序将在后续步骤中添加
// 目前先注册占位符，避免错误

