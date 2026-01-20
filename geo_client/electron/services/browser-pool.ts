import { chromium, Browser, BrowserContext, Page, BrowserType } from 'playwright';
import { join } from 'path';
import { homedir } from 'os';
import { existsSync, mkdirSync, rmSync } from 'fs';

/**
 * Provider 配置
 */
export interface ProviderConfig {
  headless: boolean;
  timeout: number; // 秒
  userDataDir: string;
}

/**
 * 浏览器池项
 */
export class BrowserPoolItem {
  public browser: Browser;
  public context: BrowserContext;
  public domain: string;
  public activeTabs: number = 0;
  public createdAt: Date = new Date();

  constructor(
    browser: Browser,
    context: BrowserContext,
    domain: string
  ) {
    this.browser = browser;
    this.context = context;
    this.domain = domain;
  }

  incrementTabs(): void {
    this.activeTabs++;
  }

  decrementTabs(): number {
    this.activeTabs = Math.max(0, this.activeTabs - 1);
    return this.activeTabs;
  }

  async close(): Promise<void> {
    try {
      await this.context.close();
      await this.browser.close();
    } catch (error) {
      console.error('[BrowserPoolItem] 关闭浏览器失败:', error);
    }
  }
}

/**
 * 页面句柄
 */
export class PageHandle {
  public page: Page;
  private _releaseCallback: () => void;
  private _released: boolean = false;

  constructor(page: Page, releaseCallback: () => void) {
    this.page = page;
    this._releaseCallback = releaseCallback;
  }

  release(): void {
    if (this._released) {
      return;
    }
    this._released = true;
    this._releaseCallback();
  }
}

/**
 * 浏览器池管理器
 * 只支持一个浏览器实例，一次只允许一个任务使用浏览器
 */
export class BrowserPoolManager {
  private static instance: BrowserPoolManager;
  private browser: BrowserPoolItem | null = null;
  private taskQueue: Array<() => void> = []; // 任务队列
  private isProcessingTask: boolean = false; // 是否正在处理任务
  private browserLock: Promise<void> = Promise.resolve(); // 浏览器创建锁

  private constructor() {}

  static getInstance(): BrowserPoolManager {
    if (!BrowserPoolManager.instance) {
      BrowserPoolManager.instance = new BrowserPoolManager();
    }
    return BrowserPoolManager.instance;
  }

  /**
   * 提取基础域名（去除www、chat等子域名前缀）
   */
  static extractBaseDomain(url: string): string {
    try {
      const urlObj = new URL(url);
      const host = urlObj.hostname;
      const parts = host.split('.');
      
      if (parts.length >= 2) {
        // 处理 .co.uk, .com.cn 等情况
        if (parts.length >= 3) {
          const lastTwo = `${parts[parts.length - 2]}.${parts[parts.length - 1]}`;
          if (['co.uk', 'com.cn', 'com.au'].includes(lastTwo)) {
            if (parts.length >= 4) {
              return `${parts[parts.length - 3]}.${lastTwo}`;
            }
          }
        }
        return `${parts[parts.length - 2]}.${parts[parts.length - 1]}`;
      }
      return host;
    } catch {
      return 'unknown';
    }
  }

  /**
   * 获取任务许可（确保一次只有一个任务使用浏览器）
   */
  async acquireTaskPermit(): Promise<() => void> {
    console.log('[BrowserPoolManager] 等待获取任务许可...');
    const startTime = Date.now();

    // 如果正在处理任务，等待
    while (this.isProcessingTask) {
      await new Promise(resolve => setTimeout(resolve, 100));
    }

    // 标记为正在处理
    this.isProcessingTask = true;
    const elapsed = Date.now() - startTime;
    console.log(`[BrowserPoolManager] 成功获取任务许可，等待时间: ${elapsed}ms`);

    // 返回释放函数
    return () => {
      this.isProcessingTask = false;
      console.log('[BrowserPoolManager] 任务许可已释放');
    };
  }

  /**
   * 获取或创建浏览器实例
   * 注意：调用此函数前必须先获取任务许可
   */
  async acquireBrowser(url: string, config: ProviderConfig): Promise<BrowserPoolItem> {
    console.log(`[BrowserPoolManager] 开始获取浏览器实例，URL: ${url}`);

    // 第一次检查：是否已存在浏览器
    if (this.browser) {
      console.log('[BrowserPoolManager] 发现现有浏览器，复用并增加tab计数');
      this.browser.incrementTabs();
      return this.browser;
    }

    console.log('[BrowserPoolManager] 未发现现有浏览器，创建新浏览器');

    // 使用锁确保只有一个任务创建浏览器
    const currentLock = this.browserLock;
    this.browserLock = currentLock.then(async () => {
      // 双重检查：可能在等待锁时其他任务已经创建了
      if (this.browser) {
        console.log('[BrowserPoolManager] 双重检查：发现其他任务已创建浏览器，复用');
        this.browser.incrementTabs();
        return;
      }

      console.log('[BrowserPoolManager] 双重检查通过，开始创建新浏览器');

      // 提取域名
      const domain = BrowserPoolManager.extractBaseDomain(url);
      console.log(`[BrowserPoolManager] 提取的基础域名: ${domain}`);

      // 清理浏览器锁文件
      this.cleanupBrowserLockFiles(config.userDataDir);

      // 创建浏览器
      const browser = await this.createBrowserInternal(config);
      const context = await browser.newContext({
        viewport: { width: 1280, height: 720 },
        userAgent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
      });

      // 创建浏览器池项
      const item = new BrowserPoolItem(browser, context, domain);
      item.incrementTabs();

      // 存储浏览器实例
      this.browser = item;
      console.log('[BrowserPoolManager] 浏览器实例已存储到池中');
    });

    // 等待锁完成
    await this.browserLock;

    // 再次检查（可能在等待时其他任务创建了）
    if (!this.browser) {
      throw new Error('创建浏览器失败');
    }
    
    const createdBrowser: BrowserPoolItem = this.browser;
    createdBrowser.incrementTabs();
    return createdBrowser;
  }

  /**
   * 释放tab，如果tab计数为0则关闭浏览器
   */
  async releaseTab(): Promise<void> {
    if (!this.browser) {
      return;
    }

    const remainingTabs = this.browser.decrementTabs();
    console.log(`[BrowserPoolManager] 释放tab，剩余tab数: ${remainingTabs}`);

    if (remainingTabs === 0) {
      console.log('[BrowserPoolManager] tab计数为0，关闭浏览器');
      const browser = this.browser;
      this.browser = null;
      await browser.close();
    }
  }

  /**
   * 内部创建浏览器函数
   */
  private async createBrowserInternal(config: ProviderConfig): Promise<Browser> {
    console.log('[BrowserPoolManager] 开始启动浏览器，超时时间: 30秒');

    const browserArgs = [
      '--disable-blink-features=AutomationControlled',
      '--disable-infobars',
      '--lang=zh-CN,zh;q=0.9',
      '--no-first-run',
      '--mute-audio',
      '--disable-features=ProcessSingleton',
      '--disable-background-networking',
      '--disable-background-timer-throttling',
      '--disable-renderer-backgrounding',
      '--disable-backgrounding-occluded-windows',
    ];

    try {
      const browser = await Promise.race([
        chromium.launch({
          headless: config.headless,
          channel: 'chrome', // 使用系统安装的 Chrome
          args: [...browserArgs, `--user-data-dir=${config.userDataDir}`],
        }),
        new Promise<never>((_, reject) =>
          setTimeout(() => reject(new Error('启动浏览器超时（30秒）')), 30000)
        ),
      ]);

      console.log('[BrowserPoolManager] 浏览器启动成功');
      return browser;
    } catch (error: any) {
      console.error('[BrowserPoolManager] 浏览器启动失败:', error);
      throw new Error(`启动浏览器失败: ${error.message}`);
    }
  }

  /**
   * 清理浏览器锁文件
   */
  private cleanupBrowserLockFiles(userDataDir: string): void {
    const lockFiles = ['SingletonLock', 'SingletonSocket', 'SingletonCookie'];
    const MAX_RETRIES = 3;
    const RETRY_DELAY_MS = 100;

    for (const lockFile of lockFiles) {
      const lockPath = join(userDataDir, lockFile);
      if (!existsSync(lockPath)) {
        continue;
      }

      console.log(`[BrowserPoolManager] 发现锁文件，尝试清理: ${lockPath}`);

      let success = false;
      for (let attempt = 1; attempt <= MAX_RETRIES; attempt++) {
        if (attempt > 1) {
          // 同步等待（使用阻塞方式）
          const delay = RETRY_DELAY_MS * (attempt - 1);
          const start = Date.now();
          while (Date.now() - start < delay) {
            // 阻塞等待
          }
        }

        try {
          rmSync(lockPath, { force: true });
          console.log(`[BrowserPoolManager] 成功清理锁文件: ${lockPath} (尝试 ${attempt}/${MAX_RETRIES})`);
          success = true;
          break;
        } catch (error: any) {
          console.error(
            `[BrowserPoolManager] 清理锁文件失败: ${lockPath}, 错误: ${error.message} (尝试 ${attempt}/${MAX_RETRIES})`
          );
          if (attempt < MAX_RETRIES && existsSync(lockPath)) {
            continue;
          } else if (!existsSync(lockPath)) {
            console.log('[BrowserPoolManager] 锁文件已不存在，可能已被其他进程清理');
            success = true;
            break;
          }
        }
      }

      if (!success) {
        console.warn(`[BrowserPoolManager] 警告: 无法清理锁文件 ${lockPath}，可能会影响浏览器启动`);
      }
    }
  }

  /**
   * 检查是否有活跃的浏览器
   */
  hasActiveBrowser(): boolean {
    return this.browser !== null;
  }

  /**
   * 获取当前可用的任务槽位
   */
  availableTaskSlots(): number {
    return this.isProcessingTask ? 0 : 1;
  }
}

/**
 * 获取或创建浏览器并创建新tab（使用浏览器池管理器）
 * 这是推荐的新API，支持任务排队（一次只允许一个任务使用浏览器）
 */
export async function getOrCreateBrowserForUrl(
  url: string,
  config: ProviderConfig
): Promise<PageHandle> {
  const MAX_RETRIES = 2;
  const manager = BrowserPoolManager.getInstance();

  // 先获取任务许可（确保一次只有一个任务使用浏览器）
  const releasePermit = await manager.acquireTaskPermit();

  // 重试机制：如果浏览器启动失败，清理锁文件后重试
  let lastError: Error | null = null;
  let pageHandle: PageHandle | null = null;

  try {
    for (let attempt = 1; attempt <= MAX_RETRIES; attempt++) {
      console.log(`[getOrCreateBrowserForUrl] 尝试获取浏览器 (尝试 ${attempt}/${MAX_RETRIES})`);

      try {
        // 从浏览器池获取或创建浏览器
        const poolItem = await manager.acquireBrowser(url, config);
        console.log('[getOrCreateBrowserForUrl] 成功获取浏览器实例');

        // 在浏览器上创建新tab
        const page = await poolItem.context.newPage();
        await page.goto(url, { waitUntil: 'domcontentloaded' });
        console.log('[getOrCreateBrowserForUrl] 成功创建新tab');

        // 创建页面句柄，当释放时会自动减少tab计数
        const releaseCallback = async () => {
          try {
            await page.close();
            await manager.releaseTab();
            releasePermit(); // 释放任务许可
          } catch (error) {
            console.error('[getOrCreateBrowserForUrl] 释放资源失败:', error);
            releasePermit(); // 即使出错也要释放许可
          }
        };

        pageHandle = new PageHandle(page, releaseCallback);
        break; // 成功，退出重试循环
      } catch (error: any) {
        console.error(`[getOrCreateBrowserForUrl] 获取浏览器失败: ${error.message} (尝试 ${attempt}/${MAX_RETRIES})`);
        lastError = error;

        // 如果是最后一次尝试，返回错误
        if (attempt >= MAX_RETRIES) {
          break;
        }

        // 等待一段时间后重试
        await new Promise(resolve => setTimeout(resolve, 500));
      }
    }

    if (!pageHandle) {
      releasePermit(); // 释放任务许可
      throw lastError || new Error('获取浏览器失败：未知错误');
    }

    return pageHandle;
  } catch (error) {
    // 如果出错，确保释放任务许可
    releasePermit();
    throw error;
  }
}

/**
 * 等待页面加载
 */
export async function waitForPageLoad(page: Page, timeout: number): Promise<void> {
  // 等待页面加载完成
  await new Promise(resolve => setTimeout(resolve, 2000));

  // 可以添加更多等待逻辑，如等待特定元素出现
}
