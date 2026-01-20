import { Page } from 'playwright';
import { getOrCreateBrowserForUrl, PageHandle, ProviderConfig } from '../services/browser-pool';
import { homedir } from 'os';
import { join } from 'path';

/**
 * 搜索结果项
 */
export interface SearchResultItem {
  title: string;
  url: string;
  snippet: string;
  domain?: string;
  siteName?: string;
}

/**
 * 搜索结果
 */
export interface SearchResult {
  query: string;
  subQuery?: string;
  items: SearchResultItem[];
  responseTimeMs: number;
  citationsCount: number;
  error?: string;
}

/**
 * Provider 基类
 */
export abstract class BaseProvider {
  protected platformName: string;
  protected headless: boolean;
  protected timeout: number;
  protected userDataDir: string;

  constructor(platformName: string, headless: boolean = true, timeout: number = 30000) {
    this.platformName = platformName;
    this.headless = headless;
    this.timeout = timeout;
    this.userDataDir = join(homedir(), '.geo_client', 'browser', platformName);
  }

  /**
   * 获取浏览器配置
   */
  protected getProviderConfig(): ProviderConfig {
    return {
      headless: this.headless,
      timeout: this.timeout / 1000,
      userDataDir: this.userDataDir,
    };
  }

  /**
   * 等待并获取页面句柄
   */
  protected async getPage(url: string): Promise<PageHandle> {
    return await getOrCreateBrowserForUrl(url, this.getProviderConfig());
  }

  /**
   * 获取登录页面URL
   */
  abstract getLoginUrl(): string;

  /**
   * 检查是否已登录
   */
  abstract checkLoginStatus(): Promise<boolean>;

  /**
   * 执行搜索
   */
  abstract search(keyword: string, query: string): Promise<SearchResult>;

  /**
   * 提取域名
   */
  protected extractDomain(url: string): string {
    try {
      const urlObj = new URL(url);
      return urlObj.hostname;
    } catch {
      return '';
    }
  }

  /**
   * 等待随机时间（模拟人类行为）
   */
  protected async randomDelay(minMs: number = 1000, maxMs: number = 3000): Promise<void> {
    const delay = Math.random() * (maxMs - minMs) + minMs;
    await new Promise(resolve => setTimeout(resolve, delay));
  }

  /**
   * 安全地获取文本内容
   */
  protected async safeGetText(page: Page, selector: string): Promise<string> {
    try {
      const element = await page.$(selector);
      if (element) {
        return await element.textContent() || '';
      }
    } catch (error) {
      console.error(`[${this.platformName}] 获取文本失败:`, error);
    }
    return '';
  }

  /**
   * 安全地点击元素
   */
  protected async safeClick(page: Page, selector: string): Promise<boolean> {
    try {
      const element = await page.$(selector);
      if (element) {
        await element.click();
        return true;
      }
    } catch (error) {
      console.error(`[${this.platformName}] 点击元素失败:`, error);
    }
    return false;
  }
}
