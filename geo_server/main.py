import os
import logging
import yaml
import time
from dotenv import load_dotenv
import os
from core.parser import extract_domain
from core.db import get_db_connection, update_domain_stats
from providers.deepseek_web import DeepSeekWebProvider
from providers.doubao_web import DoubaoWebProvider

# 根据 ENV_FILE 环境变量加载不同的 .env 文件
env_file = os.getenv("ENV_FILE", ".env")
load_dotenv(env_file)

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# 平台名称映射（用于向后兼容）
platform_name_map = {}

def load_config():
    config_path = os.path.join(os.path.dirname(__file__), "config.yaml")
    if not os.path.exists(config_path):
        logger.error(f"配置文件未找到: {config_path}")
        return None
    with open(config_path, 'r', encoding='utf-8') as f:
        return yaml.safe_load(f)

def save_to_db(keyword, platform, prompt, result, prompt_type="default", response_time_ms=None, error_message=None):
    """保存搜索结果到数据库"""
    try:
        with get_db_connection() as conn:
            cur = conn.cursor()
            
            # 确定搜索状态
            search_status = 'completed' if result and result.get("full_text") else 'failed'
            if error_message:
                search_status = 'failed'
            
            # 1. 插入搜索记录
            cur.execute("""
                INSERT INTO search_records 
                (keyword, platform, prompt_type, prompt, full_answer, response_time_ms, search_status, error_message) 
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s) 
                RETURNING id
            """, (
                keyword, 
                platform, 
                prompt_type, 
                prompt,
                result.get("full_text", "") if result else "",
                response_time_ms,
                search_status,
                error_message
            ))
            record_id = cur.fetchone()[0]
            
            if not result:
                logger.warning(f"搜索失败，仅保存了记录 ID: {record_id}")
                return
            
            # 2. 插入拓展词 (带顺序)
            for idx, query in enumerate(result.get("queries", []), 1):
                cur.execute(
                    "INSERT INTO search_queries (record_id, query, query_order) VALUES (%s, %s, %s)",
                    (record_id, query, idx)
                )
            
            # 3. 插入引用 (利用唯一约束自动去重)
            citations_count = 0
            for cite in result.get("citations", []):
                url = cite.get("url", "")
                if not url:
                    continue
                    
                domain = extract_domain(url)
                try:
                    cur.execute("""
                        INSERT INTO citations 
                        (record_id, cite_index, url, domain, title, snippet, site_name) 
                        VALUES (%s, %s, %s, %s, %s, %s, %s)
                        ON CONFLICT (record_id, url) DO NOTHING
                    """, (
                        record_id, 
                        cite.get("cite_index", 0), 
                        url, 
                        domain, 
                        cite.get("title", ""), 
                        cite.get("snippet", ""), 
                        cite.get("site_name", "")
                    ))
                    
                    if cur.rowcount > 0:
                        citations_count += 1
                        # 更新域名统计
                        update_domain_stats(conn, domain, platform)
                        
                except Exception as e:
                    logger.debug(f"插入引用失败（可能重复）: {e}")
            
            logger.info(f"✅ 成功保存 {platform} 的数据，记录 ID: {record_id}")
            logger.info(f"  - 拓展词: {len(result.get('queries', []))} 个")
            logger.info(f"  - 参考网页: {citations_count} 个")
            if response_time_ms:
                logger.info(f"  - 响应时间: {response_time_ms/1000:.2f} 秒")
                
    except Exception as e:
        logger.error(f"❌ 保存到数据库失败: {e}", exc_info=True)

def run_tasks():
    config = load_config()
    if not config:
        return

    tasks = config.get("tasks", [])
    settings = config.get("settings", {})
    headless = settings.get("headless", False)
    timeout = settings.get("timeout", 60000)
    delay = settings.get("delay_between_tasks", 5)
    
    providers = {
        "deepseek": DeepSeekWebProvider(headless=headless, timeout=timeout),
        "doubao": DoubaoWebProvider(headless=headless, timeout=timeout)
    }
    
    
    platforms_to_run = os.getenv("PLATFORMS", "deepseek").split(",")
    
    for task in tasks:
        keyword = task.get("keyword")
        prompt = task.get("keyword") # task.get("prompt") or f"请搜索并详细分析：{keyword}。请列出你参考的主要网站来源。"
        
        if not keyword:
            continue
            
        for name in platforms_to_run:
            original_name = name.strip()
            name = original_name
            
            # 尝试直接匹配
            if name not in providers:
                # 尝试通过映射查找
                normalized_name = platform_name_map.get(name)
                if normalized_name and normalized_name in providers:
                    name = normalized_name
                else:
                    # 大小写不敏感匹配
                    name_lower = name.lower()
                    matched = False
                    for key in providers.keys():
                        if key.lower() == name_lower:
                            name = key
                            matched = True
                            break
                    
                    if not matched:
                        logger.warning(f"未找到平台 [{original_name}] 的 Provider")
                        logger.info(f"可用平台: {', '.join(providers.keys())}")
                        logger.info(f"支持的别名: {', '.join(platform_name_map.keys())}")
                        continue
                
            provider = providers[name]
            logger.info(f"\n{'='*60}")
            logger.info(f"🚀 开始执行任务: [{keyword}] 在平台 [{name}]")
            logger.info(f"{'='*60}")
            
            start_time = time.time()
            result = None
            error_message = None
            
            try:
                result = provider.search(keyword, prompt)
                response_time_ms = int((time.time() - start_time) * 1000)
                
                if result and result.get("full_text"):
                    save_to_db(keyword, name, prompt, result, prompt_type="config_task", response_time_ms=response_time_ms)
                    logger.info(f"✅ {name} 任务完成")
                else:
                    error_message = "未返回有效结果"
                    logger.warning(f"⚠️ {name} {error_message}")
                    save_to_db(keyword, name, prompt, None, prompt_type="config_task", error_message=error_message)
                    
            except Exception as e:
                response_time_ms = int((time.time() - start_time) * 1000)
                error_message = str(e)
                logger.error(f"❌ 执行任务失败: {e}", exc_info=True)
                save_to_db(keyword, name, prompt, None, prompt_type="config_task", 
                          response_time_ms=response_time_ms, error_message=error_message)
            
            # 任务间延迟
            if delay > 0:
                logger.info(f"⏳ 等待 {delay} 秒后执行下一个任务...\n")
                time.sleep(delay)
    
    logger.info("\n" + "="*60)
    logger.info("🎉 所有任务执行完成！")
    logger.info("="*60)

if __name__ == "__main__":
    import sys
    
    # 支持通过命令行参数启动 API 服务器
    if len(sys.argv) > 1 and sys.argv[1] == "api":
        import uvicorn
        port = int(os.getenv("API_PORT", "8000"))
        logger.info(f"启动 API 服务器，端口: {port}")
        uvicorn.run("api.app:app", host="0.0.0.0", port=port, reload=False)
    else:
        # 默认行为：运行配置文件中的任务
        run_tasks()
