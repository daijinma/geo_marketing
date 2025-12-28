import tldextract
import logging
from urllib.parse import urlparse, urlunparse

logger = logging.getLogger(__name__)

def extract_domain(url):
    """
    从 URL 中提取主域名，例如 https://www.zhihu.com/question/123 -> zhihu.com
    """
    if not url or not isinstance(url, str):
        return "unknown"
    
    try:
        ext = tldextract.extract(url)
        if ext.suffix:
            return f"{ext.domain}.{ext.suffix}".lower()
        return ext.domain.lower()
    except Exception as e:
        logger.warning(f"提取域名失败: {url}, 错误: {e}")
        return "unknown"

def clean_url(url):
    """
    清理 URL，去除多余参数，规范化格式
    """
    if not url or not isinstance(url, str):
        return ""
        
    try:
        parsed = urlparse(url)
        # 移除常见的追踪参数
        # 这里可以根据需要扩展
        return urlunparse((parsed.scheme, parsed.netloc, parsed.path, '', '', ''))
    except Exception:
        return url

def classify_domain_type(url):
    """
    根据 URL 识别网站类型
    返回类型：官网、知乎、自媒体、新闻站、论坛、其他
    """
    if not url or not isinstance(url, str):
        return "其他"
    
    url_lower = url.lower()
    domain = extract_domain(url)
    domain_lower = domain.lower()
    
    # 知乎
    if "zhihu.com" in domain_lower:
        return "知乎"
    
    # 自媒体平台
    if any(keyword in domain_lower for keyword in ["weixin.qq.com", "mp.weixin.qq.com", "weibo.com", "toutiao.com", "douyin.com"]):
        return "自媒体"
    
    # 新闻站
    if any(keyword in domain_lower for keyword in ["sina.com.cn", "163.com", "sohu.com", "qq.com", "ifeng.com", "xinhuanet.com", "people.com.cn", "news"]):
        return "新闻站"
    
    # 论坛
    if any(keyword in domain_lower for keyword in ["bbs", "forum", "tieba.baidu.com", "douban.com"]):
        return "论坛"
    
    # 官网识别（包含常见官网特征）
    # 注意：这里需要根据实际监控的品牌调整
    if any(keyword in domain_lower for keyword in ["tuba", "official", "company", "corp", "inc"]):
        return "官网"
    
    # 默认返回其他
    return "其他"
