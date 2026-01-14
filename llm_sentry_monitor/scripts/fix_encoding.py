#!/usr/bin/env python3
"""
fix_encoding.py - 修复数据库中已存在的乱码数据

使用方法:
    python scripts/fix_encoding.py [--dry-run] [--table TABLE_NAME]

选项:
    --dry-run: 只检测不修复，显示将要修复的数据
    --table: 指定要修复的表 (citations, search_queries, search_records)
"""
import sys
import os
import argparse

# 添加项目根目录到路径
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from core.db import get_db_connection
from providers.doubao_web import ensure_utf8_string


def detect_garbled_text(text):
    """
    检测文本是否是乱码（UTF-8 被当作 Latin-1 读取）
    
    返回: (is_garbled, fixed_text)
    """
    if not text or not isinstance(text, str):
        return False, text
    
    # 检测乱码特征：包含 Latin-1 高字节字符（128-255），但实际应该是 UTF-8
    has_high_bytes = any(ord(c) > 127 for c in text)
    if not has_high_bytes:
        return False, text
    
    # 尝试修复
    try:
        fixed = text.encode('latin-1').decode('utf-8')
        # 如果修复后的文本包含中文字符，说明修复成功
        has_chinese = any('\u4e00' <= c <= '\u9fff' for c in fixed)
        if has_chinese or len(fixed) > 0:
            return True, fixed
    except (UnicodeEncodeError, UnicodeDecodeError):
        pass
    
    # 使用增强的修复函数
    fixed = ensure_utf8_string(text)
    if fixed != text:
        return True, fixed
    
    return False, text


def fix_citations_table(conn, dry_run=False):
    """修复 citations 表中的乱码数据"""
    cur = conn.cursor()
    
    # 查询所有需要检查的字段
    cur.execute("""
        SELECT id, title, snippet, site_name, url
        FROM citations
        WHERE title IS NOT NULL OR snippet IS NOT NULL OR site_name IS NOT NULL
    """)
    
    rows = cur.fetchall()
    fixed_count = 0
    
    print(f"\n检查 citations 表: {len(rows)} 条记录")
    
    for row in rows:
        record_id, title, snippet, site_name, url = row
        updates = {}
        
        # 检查并修复 title
        if title:
            is_garbled, fixed = detect_garbled_text(title)
            if is_garbled:
                updates['title'] = fixed
                if not dry_run:
                    print(f"  [ID {record_id}] 修复 title: {title[:50]}... -> {fixed[:50]}...")
                else:
                    print(f"  [ID {record_id}] 将修复 title: {title[:50]}...")
        
        # 检查并修复 snippet
        if snippet:
            is_garbled, fixed = detect_garbled_text(snippet)
            if is_garbled:
                updates['snippet'] = fixed
                if not dry_run:
                    print(f"  [ID {record_id}] 修复 snippet: {snippet[:50]}... -> {fixed[:50]}...")
                else:
                    print(f"  [ID {record_id}] 将修复 snippet: {snippet[:50]}...")
        
        # 检查并修复 site_name
        if site_name:
            is_garbled, fixed = detect_garbled_text(site_name)
            if is_garbled:
                updates['site_name'] = fixed
                if not dry_run:
                    print(f"  [ID {record_id}] 修复 site_name: {site_name} -> {fixed}")
                else:
                    print(f"  [ID {record_id}] 将修复 site_name: {site_name} -> {fixed}")
        
        # 更新数据库
        if updates and not dry_run:
            set_clauses = []
            params = []
            for field, value in updates.items():
                set_clauses.append(f"{field} = %s")
                params.append(value)
            params.append(record_id)
            
            cur.execute(f"""
                UPDATE citations
                SET {', '.join(set_clauses)}
                WHERE id = %s
            """, params)
            fixed_count += 1
    
    if not dry_run:
        conn.commit()
        print(f"\n✅ 已修复 citations 表: {fixed_count} 条记录")
    else:
        print(f"\n📊 检测到 citations 表需要修复: {fixed_count} 条记录")
    
    return fixed_count


def fix_search_queries_table(conn, dry_run=False):
    """修复 search_queries 表中的乱码数据"""
    cur = conn.cursor()
    
    cur.execute("""
        SELECT id, query
        FROM search_queries
        WHERE query IS NOT NULL
    """)
    
    rows = cur.fetchall()
    fixed_count = 0
    
    print(f"\n检查 search_queries 表: {len(rows)} 条记录")
    
    for row in rows:
        record_id, query = row
        if query:
            is_garbled, fixed = detect_garbled_text(query)
            if is_garbled:
                if not dry_run:
                    cur.execute("""
                        UPDATE search_queries
                        SET query = %s
                        WHERE id = %s
                    """, (fixed, record_id))
                    print(f"  [ID {record_id}] 修复 query: {query[:50]}... -> {fixed[:50]}...")
                else:
                    print(f"  [ID {record_id}] 将修复 query: {query[:50]}...")
                fixed_count += 1
    
    if not dry_run:
        conn.commit()
        print(f"\n✅ 已修复 search_queries 表: {fixed_count} 条记录")
    else:
        print(f"\n📊 检测到 search_queries 表需要修复: {fixed_count} 条记录")
    
    return fixed_count


def fix_search_records_table(conn, dry_run=False):
    """修复 search_records 表中的乱码数据"""
    cur = conn.cursor()
    
    cur.execute("""
        SELECT id, keyword, full_answer
        FROM search_records
        WHERE keyword IS NOT NULL OR full_answer IS NOT NULL
    """)
    
    rows = cur.fetchall()
    fixed_count = 0
    
    print(f"\n检查 search_records 表: {len(rows)} 条记录")
    
    for row in rows:
        record_id, keyword, full_answer = row
        updates = {}
        
        # 检查并修复 keyword
        if keyword:
            is_garbled, fixed = detect_garbled_text(keyword)
            if is_garbled:
                updates['keyword'] = fixed
                if not dry_run:
                    print(f"  [ID {record_id}] 修复 keyword: {keyword[:50]}... -> {fixed[:50]}...")
                else:
                    print(f"  [ID {record_id}] 将修复 keyword: {keyword[:50]}...")
        
        # 检查并修复 full_answer
        if full_answer:
            is_garbled, fixed = detect_garbled_text(full_answer)
            if is_garbled:
                updates['full_answer'] = fixed
                if not dry_run:
                    print(f"  [ID {record_id}] 修复 full_answer (长度: {len(full_answer)} -> {len(fixed)})")
                else:
                    print(f"  [ID {record_id}] 将修复 full_answer (长度: {len(full_answer)})")
        
        # 更新数据库
        if updates and not dry_run:
            set_clauses = []
            params = []
            for field, value in updates.items():
                set_clauses.append(f"{field} = %s")
                params.append(value)
            params.append(record_id)
            
            cur.execute(f"""
                UPDATE search_records
                SET {', '.join(set_clauses)}
                WHERE id = %s
            """, params)
            fixed_count += 1
    
    if not dry_run:
        conn.commit()
        print(f"\n✅ 已修复 search_records 表: {fixed_count} 条记录")
    else:
        print(f"\n📊 检测到 search_records 表需要修复: {fixed_count} 条记录")
    
    return fixed_count


def main():
    parser = argparse.ArgumentParser(description='修复数据库中已存在的乱码数据')
    parser.add_argument('--dry-run', action='store_true', help='只检测不修复，显示将要修复的数据')
    parser.add_argument('--table', choices=['citations', 'search_queries', 'search_records', 'all'],
                       default='all', help='指定要修复的表')
    
    args = parser.parse_args()
    
    if args.dry_run:
        print("🔍 运行模式: 只检测不修复 (dry-run)")
    else:
        print("🔧 运行模式: 检测并修复")
        response = input("⚠️  警告: 这将修改数据库中的数据。是否继续? (yes/no): ")
        if response.lower() != 'yes':
            print("已取消操作")
            return
    
    print("\n" + "="*60)
    print("开始修复数据库编码问题")
    print("="*60)
    
    try:
        with get_db_connection() as conn:
            total_fixed = 0
            
            if args.table in ['citations', 'all']:
                total_fixed += fix_citations_table(conn, args.dry_run)
            
            if args.table in ['search_queries', 'all']:
                total_fixed += fix_search_queries_table(conn, args.dry_run)
            
            if args.table in ['search_records', 'all']:
                total_fixed += fix_search_records_table(conn, args.dry_run)
            
            print("\n" + "="*60)
            if args.dry_run:
                print(f"📊 检测完成: 共发现 {total_fixed} 条需要修复的记录")
                print("💡 提示: 运行时不加 --dry-run 参数将执行实际修复")
            else:
                print(f"✅ 修复完成: 共修复 {total_fixed} 条记录")
            print("="*60)
    
    except Exception as e:
        print(f"\n❌ 错误: {e}", file=sys.stderr)
        import traceback
        traceback.print_exc()
        sys.exit(1)


if __name__ == '__main__':
    main()

