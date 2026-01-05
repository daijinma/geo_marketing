#!/usr/bin/env python3
"""
分析搜索词与网站关联统计脚本
支持按以下格式输入数据：
搜索词
{"p":"response/fragments/-1/results","v":[{"site_name":"网站名",...},...]}
"""

import json
import sys
from collections import defaultdict
from typing import Dict, List, Set


def parse_input_data(input_text: str) -> Dict[str, List[Dict]]:
    """
    解析输入数据，返回 {搜索词: [结果列表]} 的字典
    """
    queries_data = {}
    lines = input_text.strip().split('\n')
    
    i = 0
    while i < len(lines):
        line = lines[i].strip()
        if not line:
            i += 1
            continue
            
        # 检查是否是搜索词（不以 { 开头）
        if not line.startswith('{'):
            query = line
            i += 1
            
            # 读取下一行的JSON数据
            if i < len(lines):
                json_line = lines[i].strip()
                try:
                    data = json.loads(json_line)
                    if 'v' in data and isinstance(data['v'], list):
                        queries_data[query] = data['v']
                except json.JSONDecodeError as e:
                    print(f"警告: 解析搜索词 '{query}' 的JSON数据失败: {e}", file=sys.stderr)
            i += 1
        else:
            i += 1
    
    return queries_data


def analyze_sites(queries_data: Dict[str, List[Dict]]) -> tuple:
    """
    分析网站关联关系
    返回: (每个搜索词的网站列表, 整体统计分布)
    """
    # 每个搜索词关联的网站
    query_sites: Dict[str, Set[str]] = defaultdict(set)
    
    # 整体统计：每个网站出现的总次数
    site_count: Dict[str, int] = defaultdict(int)
    
    # 每个搜索词中每个网站出现的次数
    query_site_count: Dict[str, Dict[str, int]] = defaultdict(lambda: defaultdict(int))
    
    for query, results in queries_data.items():
        for result in results:
            site_name = result.get('site_name', '未知网站')
            if site_name:
                query_sites[query].add(site_name)
                site_count[site_name] += 1
                query_site_count[query][site_name] += 1
    
    return query_sites, site_count, query_site_count


def print_statistics(queries_data: Dict[str, List[Dict]], 
                     query_sites: Dict[str, Set[str]], 
                     site_count: Dict[str, int],
                     query_site_count: Dict[str, Dict[str, int]]):
    """
    打印统计结果
    """
    print("=" * 80)
    print("搜索词与网站关联统计报告")
    print("=" * 80)
    print()
    
    # 1. 每个搜索词关联的网站
    print("【一、每个搜索词关联的网站】")
    print("-" * 80)
    for query in sorted(queries_data.keys()):
        sites = sorted(query_sites[query])
        print(f"\n搜索词: {query}")
        print(f"  关联网站数量: {len(sites)}")
        print(f"  网站列表:")
        for site in sites:
            count = query_site_count[query][site]
            print(f"    - {site} (出现 {count} 次)")
    print()
    
    # 2. 整体统计分布
    print("【二、整体统计分布】")
    print("-" * 80)
    print(f"\n总搜索词数: {len(queries_data)}")
    print(f"总网站数: {len(site_count)}")
    print(f"总结果数: {sum(len(results) for results in queries_data.values())}")
    print()
    
    # 按出现次数排序
    sorted_sites = sorted(site_count.items(), key=lambda x: x[1], reverse=True)
    
    print("网站出现频次统计（按频次降序）:")
    print(f"{'排名':<6} {'网站名称':<40} {'出现次数':<10} {'占比':<10}")
    print("-" * 80)
    
    total_count = sum(site_count.values())
    for rank, (site, count) in enumerate(sorted_sites, 1):
        percentage = (count / total_count * 100) if total_count > 0 else 0
        print(f"{rank:<6} {site:<40} {count:<10} {percentage:>6.2f}%")
    print()
    
    # 3. 跨搜索词的网站分布
    print("【三、跨搜索词网站分布】")
    print("-" * 80)
    
    # 统计每个网站在哪些搜索词中出现
    site_queries: Dict[str, Set[str]] = defaultdict(set)
    for query, sites in query_sites.items():
        for site in sites:
            site_queries[site].add(query)
    
    print("\n每个网站出现的搜索词:")
    for site in sorted(site_queries.keys()):
        queries = sorted(site_queries[site])
        print(f"  {site}: {', '.join(queries)}")
    print()
    
    # 4. 只出现在单个搜索词中的网站
    print("【四、独占网站（仅出现在单个搜索词中）】")
    print("-" * 80)
    exclusive_sites = {site: queries for site, queries in site_queries.items() if len(queries) == 1}
    if exclusive_sites:
        for site, queries in sorted(exclusive_sites.items()):
            print(f"  {site}: {list(queries)[0]}")
    else:
        print("  无独占网站")
    print()
    
    # 5. 出现在多个搜索词中的网站
    print("【五、共享网站（出现在多个搜索词中）】")
    print("-" * 80)
    shared_sites = {site: queries for site, queries in site_queries.items() if len(queries) > 1}
    if shared_sites:
        sorted_shared = sorted(shared_sites.items(), key=lambda x: len(x[1]), reverse=True)
        for site, queries in sorted_shared:
            print(f"  {site} (出现在 {len(queries)} 个搜索词中): {', '.join(sorted(queries))}")
    else:
        print("  无共享网站")
    print()
    
    print("=" * 80)


def main():
    """
    主函数：从标准输入读取数据并分析
    """
    if len(sys.argv) > 1:
        # 从文件读取
        with open(sys.argv[1], 'r', encoding='utf-8') as f:
            input_text = f.read()
    else:
        # 从标准输入读取
        print("请输入数据（格式：搜索词后跟JSON数据，输入完成后按 Ctrl+D 结束）:")
        print("-" * 80)
        input_text = sys.stdin.read()
    
    # 解析数据
    queries_data = parse_input_data(input_text)
    
    if not queries_data:
        print("错误: 未找到有效的搜索词数据", file=sys.stderr)
        sys.exit(1)
    
    # 分析
    query_sites, site_count, query_site_count = analyze_sites(queries_data)
    
    # 打印统计结果
    print_statistics(queries_data, query_sites, site_count, query_site_count)


if __name__ == '__main__':
    main()

