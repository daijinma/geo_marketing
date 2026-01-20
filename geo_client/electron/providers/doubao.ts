import { BaseProvider, SearchResult, SearchResultItem } from './base';
import { PageHandle } from '../services/browser-pool';

/**
 * 豆包 Provider
 */
export class DoubaoProvider extends BaseProvider {
  private readonly baseUrl = 'https://www.doubao.com/chat/';

  constructor(headless: boolean = true, timeout: number = 60000) {
    super('doubao', headless, timeout);
  }

  /**
   * 获取登录页面URL
   */
  getLoginUrl(): string {
    return 'https://www.doubao.com';
  }

  /**
   * 检查登录状态
   */
  async checkLoginStatus(): Promise<boolean> {
    let pageHandle: PageHandle | null = null;
    try {
      console.log('[Doubao] 开始检查登录状态');
      pageHandle = await this.getPage(this.baseUrl);
      const page = pageHandle.page;

      // 等待页面加载
      await page.waitForLoadState('networkidle', { timeout: 10000 });
      await this.randomDelay(2000, 3000);

      // 检查是否有登录按钮
      const loginButton = await page.$('button:has-text("登录"), a:has-text("登录")');
      const isLoggedIn = !loginButton;

      console.log(`[Doubao] 登录状态: ${isLoggedIn ? '已登录' : '未登录'}`);
      return isLoggedIn;
    } catch (error: any) {
      console.error('[Doubao] 检查登录状态失败:', error);
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
      console.log(`[Doubao] 开始搜索: ${query}`);
      pageHandle = await this.getPage(this.baseUrl);
      const page = pageHandle.page;

      // 等待页面加载
      await page.waitForLoadState('networkidle', { timeout: 10000 });
      await this.randomDelay(2000, 3000);

      // 检查登录状态
      const loginButton = await page.$('button:has-text("登录"), a:has-text("登录")');
      if (loginButton) {
        throw new Error('需要先登录');
      }

      // 查找输入框（豆包的输入框selector可能不同）
      const inputSelector = 'textarea, input[type="text"]';
      await page.waitForSelector(inputSelector, { timeout: 10000 });

      // 输入搜索内容
      const input = await page.$(inputSelector);
      if (!input) {
        throw new Error('找不到输入框');
      }
      await input.fill(query);
      await this.randomDelay(500, 1000);

      // 发送消息
      await page.keyboard.press('Enter');

      // 等待响应
      await this.randomDelay(3000, 5000);

      // 等待响应完成（等待一定时间或查找完成标志）
      await this.randomDelay(10000, 15000);

      // 提取子查询（豆包特有的query_tokens）
      const queryTokens: string[] = [];
      const queryTokenElements = await page.$$('[data-query-token], .query-token');
      for (const element of queryTokenElements) {
        const text = await element.textContent();
        if (text) {
          queryTokens.push(text.trim());
        }
      }

      // 提取引用链接
      const items: SearchResultItem[] = [];
      const citations = await page.$$('a[href^="http"]');

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
          console.error('[Doubao] 提取引用失败:', error);
        }
      }

      const responseTimeMs = Date.now() - startTime;
      console.log(`[Doubao] 搜索完成，找到 ${items.length} 个结果，耗时 ${responseTimeMs}ms`);

      return {
        query: keyword,
        subQuery: queryTokens.length > 0 ? queryTokens.join(', ') : query,
        items,
        responseTimeMs,
        citationsCount: items.length,
      };
    } catch (error: any) {
      console.error('[Doubao] 搜索失败:', error);
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
