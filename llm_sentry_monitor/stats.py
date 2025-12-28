"""
stats.py - GEO 深度洞察报告（增强版）
功能：jieba 分词、SoV 百分比、时间趋势分析
"""
import os
import jieba
import jieba.analyse
from datetime import datetime, timedelta
from dotenv import load_dotenv
from tabulate import tabulate
from collections import Counter
from core.db import get_db_cursor

load_dotenv()

# 自定义词典（行业术语）
CUSTOM_WORDS = [
    "土巴兔", "装修公司", "家装", "软装", "硬装", "全包", "半包",
    "DeepSeek", "豆包", "Kimi", "文心一言", "通义千问"
]

for word in CUSTOM_WORDS:
    jieba.add_word(word)

# 停用词
STOP_WORDS = {
    '什么', '怎么', '如何', '哪些', '为什么', '是否', '可以', '能否',
    '2024', '2025', '的', '了', '在', '是', '和', '与', '或', '等',
    '一个', '这个', '那个', '进行', '问题', '相关', '关于', '有关'
}

def print_header(title):
    """打印漂亮的标题"""
    print("\n" + "="*80)
    print(f"📊 {title}")
    print("="*80 + "\n")

def get_date_range_filter(days=None):
    """生成日期过滤条件"""
    if days:
        cutoff = datetime.now() - timedelta(days=days)
        return f"WHERE created_at >= '{cutoff.strftime('%Y-%m-%d')}'"
    return ""

def analyze_trust_sources(days=None):
    """1. 核心信任源分析（含 SoV 百分比）"""
    print_header("核心信任源分析 - 哪些网站在多个关键词下都被 AI 信任？")
    
    conn, cur = get_db_cursor()
    
    date_filter = get_date_range_filter(days)
    
    cur.execute(f"""
        WITH citation_stats AS (
            SELECT 
                domain,
                COUNT(DISTINCT record_id) as keyword_coverage,
                COUNT(*) as total_citations,
                STRING_AGG(DISTINCT site_name, ' | ') as site_names
            FROM citations
            {date_filter}
            GROUP BY domain
        ),
        total AS (
            SELECT SUM(total_citations) as grand_total FROM citation_stats
        )
        SELECT 
            cs.domain,
            cs.keyword_coverage as "覆盖关键词数",
            cs.total_citations as "总引用次数",
            ROUND(cs.total_citations * 100.0 / t.grand_total, 2) as "SoV(%)",
            cs.site_names as "站点名称"
        FROM citation_stats cs, total t
        ORDER BY cs.keyword_coverage DESC, cs.total_citations DESC
        LIMIT 15
    """)
    
    rows = cur.fetchall()
    print(tabulate(rows, headers=["域名", "覆盖词数", "总引用数", "SoV(%)", "站点名称"], tablefmt="grid"))
    
    conn.close()

def analyze_search_intent():
    """2. AI 搜索意图洞察（jieba 分词版）"""
    print_header("AI 搜索意图洞察 - AI 最关注哪些核心概念？")
    
    conn, cur = get_db_cursor()
    cur.execute("SELECT query FROM search_queries")
    queries = cur.fetchall()
    
    # 使用 jieba 分词
    all_words = []
    for (query_text,) in queries:
        if not query_text:
            continue
        # 使用 TF-IDF 提取关键词（更智能）
        keywords = jieba.analyse.extract_tags(query_text, topK=5, withWeight=False)
        # 或使用普通分词
        words = jieba.lcut(query_text)
        
        # 合并两种方式的结果
        all_words.extend([w for w in words + keywords if len(w) > 1 and w not in STOP_WORDS])
    
    word_counts = Counter(all_words).most_common(20)
    
    print(tabulate(word_counts, headers=["核心概念 (jieba分词)", "出现频次"], tablefmt="grid"))
    
    conn.close()

def analyze_brand_exposure():
    """3. 品牌曝光矩阵"""
    print_header("品牌曝光矩阵 - 每个关键词下排名前 3 的竞争对手")
    
    conn, cur = get_db_cursor()
    cur.execute("SELECT DISTINCT keyword FROM search_records ORDER BY keyword")
    keywords = [r[0] for r in cur.fetchall()]
    
    matrix_data = []
    for kw in keywords:
        cur.execute("""
            SELECT domain, COUNT(*) as count, STRING_AGG(DISTINCT site_name, ',') as names
            FROM citations c
            JOIN search_records r ON c.record_id = r.id
            WHERE r.keyword = %s
            GROUP BY domain
            ORDER BY count DESC
            LIMIT 3
        """, (kw,))
        
        top_sites = []
        for domain, count, names in cur.fetchall():
            name_display = names.split(',')[0] if names else domain
            top_sites.append(f"{name_display}({count})")
        
        matrix_data.append([kw, " | ".join(top_sites)])
    
    print(tabulate(matrix_data, headers=["监控关键词", "头部竞争域名 (引用次数)"], tablefmt="grid"))
    
    conn.close()

def analyze_time_trends(days=7):
    """4. 时间趋势分析（新增）"""
    print_header(f"时间趋势分析 - 最近 {days} 天的域名引用变化")
    
    conn, cur = get_db_cursor()
    
    # 获取 Top 5 域名
    cur.execute("""
        SELECT domain 
        FROM citations 
        WHERE created_at >= CURRENT_DATE - INTERVAL '%s days'
        GROUP BY domain 
        ORDER BY COUNT(*) DESC 
        LIMIT 5
    """ % days)
    
    top_domains = [r[0] for r in cur.fetchall()]
    
    if not top_domains:
        print("⚠️ 暂无数据")
        conn.close()
        return
    
    # 按天统计
    cur.execute("""
        SELECT 
            DATE(created_at) as date,
            domain,
            COUNT(*) as citation_count
        FROM citations
        WHERE domain = ANY(%s) AND created_at >= CURRENT_DATE - INTERVAL '%s days'
        GROUP BY DATE(created_at), domain
        ORDER BY date DESC, citation_count DESC
    """ % ("%s", days), (top_domains,))
    
    rows = cur.fetchall()
    print(tabulate(rows, headers=["日期", "域名", "引用次数"], tablefmt="grid"))
    
    conn.close()

def analyze_platform_comparison():
    """5. 平台对比分析（新增）"""
    print_header("平台对比分析 - DeepSeek vs 豆包")
    
    conn, cur = get_db_cursor()
    
    cur.execute("""
        SELECT 
            platform as "平台",
            COUNT(DISTINCT keyword) as "监控关键词数",
            COUNT(*) as "总搜索次数",
            ROUND(AVG(response_time_ms)/1000.0, 2) as "平均响应时间(秒)",
            SUM(CASE WHEN search_status = 'completed' THEN 1 ELSE 0 END) as "成功次数",
            SUM(CASE WHEN search_status = 'failed' THEN 1 ELSE 0 END) as "失败次数"
        FROM search_records
        GROUP BY platform
    """)
    
    rows = cur.fetchall()
    print(tabulate(rows, headers=["平台", "关键词数", "搜索次数", "平均响应(秒)", "成功", "失败"], tablefmt="grid"))
    
    conn.close()

def analyze_response_performance():
    """6. 响应性能分析（新增）"""
    print_header("响应性能分析 - 搜索速度统计")
    
    conn, cur = get_db_cursor()
    
    cur.execute("""
        SELECT 
            keyword as "关键词",
            platform as "平台",
            ROUND(response_time_ms/1000.0, 2) as "响应时间(秒)",
            (SELECT COUNT(*) FROM citations WHERE record_id = search_records.id) as "引用数",
            (SELECT COUNT(*) FROM search_queries WHERE record_id = search_records.id) as "拓展词数",
            created_at as "执行时间"
        FROM search_records
        WHERE search_status = 'completed' AND response_time_ms IS NOT NULL
        ORDER BY created_at DESC
        LIMIT 10
    """)
    
    rows = cur.fetchall()
    print(tabulate(rows, headers=["关键词", "平台", "响应时间(秒)", "引用数", "拓展词数", "执行时间"], tablefmt="grid"))
    
    conn.close()

def main():
    """主函数"""
    print("\n" + "🚀 "*20)
    print("    GEO 深度洞察报告 (增强版 v2.0)")
    print("    集成：jieba分词 | SoV分析 | 时间趋势 | 平台对比")
    print("🚀 "*20)
    
    try:
        # 1. 核心信任源（近 7 天）
        analyze_trust_sources(days=7)
        
        # 2. AI 搜索意图（jieba 分词）
        analyze_search_intent()
        
        # 3. 品牌曝光矩阵
        analyze_brand_exposure()
        
        # 4. 时间趋势分析
        analyze_time_trends(days=7)
        
        # 5. 平台对比分析
        analyze_platform_comparison()
        
        # 6. 响应性能分析
        analyze_response_performance()
        
        print("\n" + "="*80)
        print("✅ 报告生成完成！")
        print("="*80 + "\n")
        
    except Exception as e:
        print(f"\n❌ 获取统计数据失败: {e}")
        import traceback
        traceback.print_exc()

if __name__ == "__main__":
    main()
