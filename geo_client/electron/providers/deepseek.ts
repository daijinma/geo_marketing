import { BaseProvider, SearchResult, SearchResultItem } from './base';
import { PageHandle } from '../services/browser-pool';

/**
 * DeepSeek Provider
 */
export class DeepSeekProvider extends BaseProvider {
  private readonly baseUrl = 'https://chat.deepseek.com';

  constructor(headless: boolean = true, timeout: number = 60000) {
    super('deepseek', headless, timeout);
  }

  /**
   * 获取登录页面URL
   */
  getLoginUrl(): string {
    return this.baseUrl;
  }

  /**
   * 检查登录状态
   */
  async checkLoginStatus(): Promise<boolean> {
    let pageHandle: PageHandle | null = null;
    try {
      console.log('[DeepSeek] 开始检查登录状态');
      pageHandle = await this.getPage(this.baseUrl);
      const page = pageHandle.page;

      // 等待页面加载
      await page.waitForLoadState('networkidle', { timeout: 10000 });
      await this.randomDelay(2000, 3000);

      // 检查是否有登录按钮
      const loginButton = await page.$('button:has-text("登录")');
      const isLoggedIn = !loginButton;

      console.log(`[DeepSeek] 登录状态: ${isLoggedIn ? '已登录' : '未登录'}`);
      return isLoggedIn;
    } catch (error: any) {
      console.error('[DeepSeek] 检查登录状态失败:', error);
      return false;
    } finally {
      if (pageHandle) {
        pageHandle.release();
      }
    }
  }

  /**
   * 执行搜索
   */
  async search(keyword: string, query: string): Promise<SearchResult> {
    let pageHandle: PageHandle | null = null;
    const startTime = Date.now();

    try {
      console.log(`[DeepSeek] 开始搜索: ${query}`);
      pageHandle = await this.getPage(this.baseUrl);
      const page = pageHandle.page;

      // 等待页面加载
      await page.waitForLoadState('networkidle', { timeout: 10000 });
      await this.randomDelay(2000, 3000);

      // 检查登录状态
      const loginButton = await page.$('button:has-text("登录")');
      if (loginButton) {
        throw new Error('需要先登录');
      }

      // 查找输入框
      const inputSelector = 'textarea[placeholder*="输入"], textarea[placeholder*="消息"]';
      await page.waitForSelector(inputSelector, { timeout: 10000 });

      // 输入搜索内容
      await page.fill(inputSelector, query);
      await this.randomDelay(500, 1000);

      // 发送消息（查找发送按钮）
      const sendButton = await page.$('button[type="submit"], button:has-text("发送")');
      if (!sendButton) {
        throw new Error('找不到发送按钮');
      }
      await sendButton.click();

      // 等待响应
      await this.randomDelay(3000, 5000);

      // 等待响应完成（查找思考/生成完成的标志）
      await page.waitForSelector('[data-message-role="assistant"]', { timeout: 60000 });
      await this.randomDelay(2000, 3000);

      // 提取引用链接
      const items: SearchResultItem[] = [];
      const citations = await page.$$('[data-citation], .citation, a[href^="http"]');

      for (const citation of citations.slice(0, 10)) {
        try {
          const url = await citation.getAttribute('href');
          const title = await citation.textContent();
          
          if (url && url.startsWith('http')) {
            items.push({
              title: title?.trim() || url,
              url,
              snippet: '',
              domain: this.extractDomain(url),
            });
          }
        } catch (error) {
          console.error('[DeepSeek] 提取引用失败:', error);
        }
      }

      const responseTimeMs = Date.now() - startTime;
      console.log(`[DeepSeek] 搜索完成，找到 ${items.length} 个结果，耗时 ${responseTimeMs}ms`);

      return {
        query: keyword,
        subQuery: query,
        items,
        responseTimeMs,
        citationsCount: items.length,
      };
    } catch (error: any) {
      console.error('[DeepSeek] 搜索失败:', error);
      const responseTimeMs = Date.now() - startTime;
      return {
        query: keyword,
        subQuery: query,
        items: [],
        responseTimeMs,
        citationsCount: 0,
        error: error.message,
      };
    } finally {
      if (pageHandle) {
        pageHandle.release();
      }
    }
  }
}
