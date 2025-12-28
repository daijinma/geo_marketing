"""
stats_full.py - GEO 深度洞察报告（完整版）
功能：包含所有基础分析 + 域名类型分布、引用位置分析、跨平台一致性分析
"""
import os
import jieba
import jieba.analyse
from datetime import datetime, timedelta
from dotenv import load_dotenv
from tabulate import tabulate
from collections import Counter
from core.db import get_db_cursor
from core.parser import classify_domain_type

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
    """4. 时间趋势分析"""
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
    """5. 平台对比分析"""
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
    """6. 响应性能分析"""
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

def analyze_domain_types(days=None):
    """7. 域名类型分布分析 - 分析引用来源的网站类型分布"""
    print_header("域名类型分布分析 - AI 对不同类型网站的偏好")
    
    conn, cur = get_db_cursor()
    
    date_filter = get_date_range_filter(days)
    
    # 获取所有引用及其 URL
    if date_filter:
        cur.execute(f"""
            SELECT url, COUNT(*) as citation_count
            FROM citations
            {date_filter}
            GROUP BY url
        """)
    else:
        cur.execute("""
            SELECT url, COUNT(*) as citation_count
            FROM citations
            GROUP BY url
        """)
    
    citations_data = cur.fetchall()
    
    # 分类统计
    type_stats = Counter()
    type_citation_counts = Counter()
    
    for url, count in citations_data:
        domain_type = classify_domain_type(url)
        type_stats[domain_type] += 1
        type_citation_counts[domain_type] += count
    
    # 计算总数和占比
    total_types = sum(type_stats.values())
    total_citations = sum(type_citation_counts.values())
    
    if total_types == 0:
        print("⚠️ 暂无数据")
        conn.close()
        return
    
    # 准备表格数据
    table_data = []
    for domain_type in ['官网', '知乎', '自媒体', '新闻站', '论坛', '其他']:
        if domain_type in type_stats:
            type_count = type_stats[domain_type]
            citation_count = type_citation_counts[domain_type]
            type_percentage = round(type_count * 100.0 / total_types, 2)
            citation_percentage = round(citation_count * 100.0 / total_citations, 2) if total_citations > 0 else 0
            table_data.append([domain_type, type_count, type_percentage, citation_count, citation_percentage])
    
    print(tabulate(table_data, headers=["网站类型", "域名数量", "域名占比(%)", "引用次数", "引用占比(%)"], tablefmt="grid"))
    
    conn.close()

def analyze_citation_positions():
    """8. 引用位置分析 - 分析引用在回答中的位置分布"""
    print_header("引用位置分析 - 哪些域名更常出现在回答的开头/结尾？")
    
    conn, cur = get_db_cursor()
    
    # 获取每个记录的引用总数和位置信息
    cur.execute("""
        SELECT 
            c.record_id,
            c.cite_index,
            c.domain,
            c.url,
            (SELECT COUNT(*) FROM citations WHERE record_id = c.record_id) as total_citations
        FROM citations c
        ORDER BY c.record_id, c.cite_index
    """)
    
    citations = cur.fetchall()
    
    # 分类统计
    position_stats = {
        '开头': Counter(),  # cite_index <= 3
        '中间': Counter(),  # 3 < cite_index <= (total - 3)
        '结尾': Counter()   # cite_index > (total - 3)
    }
    
    # 按 record_id 分组处理
    current_record_id = None
    record_citations = []
    
    for record_id, cite_index, domain, url, total_citations in citations:
        if current_record_id != record_id:
            # 处理上一个记录
            if current_record_id is not None and record_citations:
                for rec_cite_index, rec_domain, rec_total in record_citations:
                    if rec_total <= 6:
                        # 如果总引用数 <= 6，全部算作中间
                        position_stats['中间'][rec_domain] += 1
                    else:
                        if rec_cite_index <= 3:
                            position_stats['开头'][rec_domain] += 1
                        elif rec_cite_index > (rec_total - 3):
                            position_stats['结尾'][rec_domain] += 1
                        else:
                            position_stats['中间'][rec_domain] += 1
            
            # 开始新记录
            current_record_id = record_id
            record_citations = []
        
        record_citations.append((cite_index, domain, total_citations))
    
    # 处理最后一个记录
    if record_citations:
        for rec_cite_index, rec_domain, rec_total in record_citations:
            if rec_total <= 6:
                position_stats['中间'][rec_domain] += 1
            else:
                if rec_cite_index <= 3:
                    position_stats['开头'][rec_domain] += 1
                elif rec_cite_index > (rec_total - 3):
                    position_stats['结尾'][rec_domain] += 1
                else:
                    position_stats['中间'][rec_domain] += 1
    
    # 统计每个位置的前10个域名
    print("\n📍 开头位置（前3个引用）Top 10 域名：")
    top_start = position_stats['开头'].most_common(10)
    if top_start:
        print(tabulate(top_start, headers=["域名", "出现次数"], tablefmt="grid"))
    else:
        print("  暂无数据")
    
    print("\n📍 中间位置 Top 10 域名：")
    top_middle = position_stats['中间'].most_common(10)
    if top_middle:
        print(tabulate(top_middle, headers=["域名", "出现次数"], tablefmt="grid"))
    else:
        print("  暂无数据")
    
    print("\n📍 结尾位置（后3个引用）Top 10 域名：")
    top_end = position_stats['结尾'].most_common(10)
    if top_end:
        print(tabulate(top_end, headers=["域名", "出现次数"], tablefmt="grid"))
    else:
        print("  暂无数据")
    
    # 汇总统计
    print("\n📊 位置分布汇总：")
    summary_data = [
        ['开头', sum(position_stats['开头'].values())],
        ['中间', sum(position_stats['中间'].values())],
        ['结尾', sum(position_stats['结尾'].values())]
    ]
    total_positions = sum([sum(c.values()) for c in position_stats.values()])
    if total_positions > 0:
        for row in summary_data:
            row.append(round(row[1] * 100.0 / total_positions, 2))
        print(tabulate(summary_data, headers=["位置", "引用数", "占比(%)"], tablefmt="grid"))
    
    conn.close()

def analyze_cross_platform_consistency():
    """9. 跨平台一致性分析 - 对比同一关键词在不同平台的引用差异"""
    print_header("跨平台一致性分析 - DeepSeek vs 豆包的引用差异")
    
    conn, cur = get_db_cursor()
    
    # 获取所有关键词
    cur.execute("SELECT DISTINCT keyword FROM search_records ORDER BY keyword")
    keywords = [r[0] for r in cur.fetchall()]
    
    if not keywords:
        print("⚠️ 暂无数据")
        conn.close()
        return
    
    # 对每个关键词进行跨平台分析
    comparison_data = []
    
    for keyword in keywords:
        # 获取该关键词在不同平台的引用域名
        cur.execute("""
            SELECT 
                r.platform,
                c.domain
            FROM search_records r
            JOIN citations c ON r.id = c.record_id
            WHERE r.keyword = %s
            GROUP BY r.platform, c.domain
        """, (keyword,))
        
        platform_domains = {}
        for platform, domain in cur.fetchall():
            if platform not in platform_domains:
                platform_domains[platform] = set()
            platform_domains[platform].add(domain)
        
        if len(platform_domains) < 2:
            # 只有一个平台的数据，跳过
            continue
        
        # 计算重叠域名
        platforms = list(platform_domains.keys())
        if len(platforms) >= 2:
            common_domains = platform_domains[platforms[0]] & platform_domains[platforms[1]]
            all_domains = platform_domains[platforms[0]] | platform_domains[platforms[1]]
            
            overlap_count = len(common_domains)
            total_unique = len(all_domains)
            overlap_rate = round(overlap_count * 100.0 / total_unique, 2) if total_unique > 0 else 0
            
            # 平台特有域名
            platform1_unique = platform_domains[platforms[0]] - platform_domains[platforms[1]]
            platform2_unique = platform_domains[platforms[1]] - platform_domains[platforms[0]]
            
            comparison_data.append({
                'keyword': keyword,
                'platform1': platforms[0],
                'platform2': platforms[1],
                'platform1_domains': len(platform_domains[platforms[0]]),
                'platform2_domains': len(platform_domains[platforms[1]]),
                'common_domains': overlap_count,
                'overlap_rate': overlap_rate,
                'platform1_unique': len(platform1_unique),
                'platform2_unique': len(platform2_unique)
            })
    
    if not comparison_data:
        print("⚠️ 暂无跨平台对比数据（需要至少两个平台的数据）")
        conn.close()
        return
    
    # 显示对比表格
    table_data = []
    for comp in comparison_data:
        table_data.append([
            comp['keyword'],
            comp['platform1'],
            comp['platform1_domains'],
            comp['platform2'],
            comp['platform2_domains'],
            comp['common_domains'],
            f"{comp['overlap_rate']}%",
            comp['platform1_unique'],
            comp['platform2_unique']
        ])
    
    print(tabulate(table_data, headers=[
        "关键词", 
        "平台1", "平台1域名数",
        "平台2", "平台2域名数",
        "共同域名", "重叠率",
        "平台1特有", "平台2特有"
    ], tablefmt="grid"))
    
    # 计算平均重叠率
    if comparison_data:
        avg_overlap = sum(c['overlap_rate'] for c in comparison_data) / len(comparison_data)
        print(f"\n📈 平均重叠率: {round(avg_overlap, 2)}%")
    
    conn.close()

def main():
    """主函数 - 完整版分析"""
    print("\n" + "🚀 "*20)
    print("    GEO 深度洞察报告 (完整版 v3.0)")
    print("    集成：基础分析 + 域名类型分布 + 引用位置分析 + 跨平台一致性")
    print("🚀 "*20)
    
    try:
        # 基础分析（6个）
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
        
        # 新增分析（3个）
        # 7. 域名类型分布分析
        analyze_domain_types(days=7)
        
        # 8. 引用位置分析
        analyze_citation_positions()
        
        # 9. 跨平台一致性分析
        analyze_cross_platform_consistency()
        
        print("\n" + "="*80)
        print("✅ 完整版报告生成完成！")
        print("="*80 + "\n")
        
    except Exception as e:
        print(f"\n❌ 获取统计数据失败: {e}")
        import traceback
        traceback.print_exc()

if __name__ == "__main__":
    main()

