import { DeepSeekProvider } from './deepseek';
import { DoubaoProvider } from './doubao';
import { BaseProvider } from './base';

/**
 * Provider 工厂
 */
export class ProviderFactory {
  private static providers: Map<string, BaseProvider> = new Map();

  /**
   * 获取 Provider 实例
   */
  static getProvider(platformName: string, headless: boolean = true): BaseProvider {
    const key = `${platformName}_${headless}`;
    
    if (this.providers.has(key)) {
      return this.providers.get(key)!;
    }

    let provider: BaseProvider;
    switch (platformName.toLowerCase()) {
      case 'deepseek':
        provider = new DeepSeekProvider(headless);
        break;
      case 'doubao':
      case '豆包':
        provider = new DoubaoProvider(headless);
        break;
      default:
        throw new Error(`不支持的平台: ${platformName}`);
    }

    this.providers.set(key, provider);
    return provider;
  }

  /**
   * 清理所有 Provider
   */
  static clear(): void {
    this.providers.clear();
  }
}

export * from './base';
export * from './deepseek';
export * from './doubao';
