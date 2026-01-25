# Search Stats Table Refactor - Implementation Summary (v2)

## Overview
Refactored the "搜索词统计" (Search Term Statistics) table to display **per-record subqueries** instead of aggregated statistics. Each row now represents a single search execution (search_record), showing its subqueries and corresponding URL count.

## Key Requirement Change

**从 subquery 的角度分解** - Display from the perspective of individual subqueries, showing clearly:
- How many subqueries exist
- How each subquery corresponds to URLs
- The relationship between subqueries and citations for each search execution

## Data Structure Changes

### Before (v1 - Aggregated by Keyword):
```typescript
interface SearchStatsItem {
  keyword: string;
  subQueries: string[];  // All unique subqueries for this keyword
  urlCount: number;      // Total unique URLs for this keyword
}
```
**Grouping:** By `keyword` only (aggregated across all platforms and rounds)

### After (v2 - Per Search Record):
```typescript
interface SearchStatsItem {
  recordId: number;      // search_records.id
  keyword: string;       // Original search keyword
  platform: string;      // Platform (deepseek, doubao, yiyan, yuanbao)
  round: number;         // Round number (1, 2, ...)
  subQueries: string[];  // Subqueries for THIS specific search execution
  urlCount: number;      // URLs for THIS specific search execution
}
```
**Grouping:** By individual `search_record` (one row per search execution)

## Display Logic

| Scenario | Display Format | Example |
|----------|---------------|---------|
| **No subqueries** (e.g., Yiyan) | `--` | `--` |
| **Single subquery** | Plain text | `上海 咖啡 推荐` |
| **Multiple subqueries** | Array format | `['上海咖啡推荐', '上海特色咖啡推荐']` |

**URL Count:** Shows the number of unique citations for that specific search record.

## Task 31 Example

### Database Records (16 search executions):
```
ID  | Keyword      | Platform | Round | SubQueries                                      | URLs
----|--------------|----------|-------|-------------------------------------------------|-----
85  | 上海咖啡推荐 | deepseek | 1     | 上海 咖啡 推荐                                   | 8
93  | 上海咖啡推荐 | deepseek | 2     | 上海 咖啡 店 推荐                                | 8
86  | 上海咖啡推荐 | doubao   | 1     | ['上海咖啡推荐', '上海特色咖啡推荐']              | 5
94  | 上海咖啡推荐 | doubao   | 2     | ['上海咖啡推荐', '上海特色咖啡推荐']              | 5
87  | 上海咖啡推荐 | yiyan    | 1     | --                                              | 8
95  | 上海咖啡推荐 | yiyan    | 2     | --                                              | 8
88  | 上海咖啡推荐 | yuanbao  | 1     | 上海咖啡推荐                                     | 9
96  | 上海咖啡推荐 | yuanbao  | 2     | 上海咖啡推荐                                     | 9
89  | 北京餐馆推荐 | deepseek | 1     | 北京 餐馆 推荐                                   | 8
97  | 北京餐馆推荐 | deepseek | 2     | ['北京 餐馆 推荐 热门 2026', '北京 必吃...']      | 10
90  | 北京餐馆推荐 | doubao   | 1     | ['北京餐馆推荐', '北京美食推荐']                  | 6
98  | 北京餐馆推荐 | doubao   | 2     | ['北京餐馆推荐', '北京美食推荐']                  | 6
91  | 北京餐馆推荐 | yiyan    | 1     | --                                              | 7
99  | 北京餐馆推荐 | yiyan    | 2     | --                                              | 7
92  | 北京餐馆推荐 | yuanbao  | 1     | 北京餐馆推荐                                     | 6
100 | 北京餐馆推荐 | yuanbao  | 2     | 北京餐馆推荐                                     | 8
```

### Expected Table Display:
```
┌──────────────────────────────────────────────────────┬────────┐
│ Sub Query                                            │ URL 数 │
├──────────────────────────────────────────────────────┼────────┤
│ 上海 咖啡 推荐                                        │   8    │
│ 上海 咖啡 店 推荐                                     │   8    │
│ ['上海咖啡推荐', '上海特色咖啡推荐']                   │   5    │
│ ['上海咖啡推荐', '上海特色咖啡推荐']                   │   5    │
│ --                                                   │   8    │
│ --                                                   │   8    │
│ 上海咖啡推荐                                         │   9    │
│ 上海咖啡推荐                                         │   9    │
│ 北京 餐馆 推荐                                        │   8    │
│ ['北京 餐馆 推荐 热门 2026', '北京 必吃 餐厅 2026'...] │   10   │
│ ['北京餐馆推荐', '北京美食推荐']                       │   6    │
│ ['北京餐馆推荐', '北京美食推荐']                       │   6    │
│ --                                                   │   7    │
│ --                                                   │   7    │
│ 北京餐馆推荐                                         │   6    │
│ 北京餐馆推荐                                         │   8    │
└──────────────────────────────────────────────────────┴────────┘
```

**Total: 16 rows** (one per search_record)

## Platform-Specific Behavior

| Platform | SubQuery Pattern | Example |
|----------|-----------------|---------|
| **DeepSeek** | Single spaced query per round | Round 1: `上海 咖啡 推荐`<br>Round 2: `上海 咖啡 店 推荐` |
| **Doubao** | Multiple related queries | `['上海咖啡推荐', '上海特色咖啡推荐']` (same both rounds) |
| **Yiyan** | No subqueries | `--` (all rounds) |
| **Yuanbao** | Single query per round | `上海咖啡推荐` (same both rounds) |

## Implementation Details

### Statistics Computation (`computeStats` function)

**Key Changes:**
1. **No aggregation** - Each `search_record` becomes a row
2. **Direct mapping** - Subqueries from `record.queries[]` displayed as-is
3. **Per-record URL count** - Count unique URLs from `record.citations[]`

**Pseudocode:**
```typescript
records.forEach(record => {
  const subQueries = record.queries
    .map(q => q.query)
    .filter(q => q && q.trim());
  
  const uniqueUrls = new Set(
    record.citations.map(c => c.url)
  );
  
  searchStats.push({
    recordId: record.id,
    keyword: record.keyword,
    platform: record.platform,
    round: record.round_number,
    subQueries,
    urlCount: uniqueUrls.size,
  });
});
```

### Display Logic

```typescript
let displayText: string;
if (item.subQueries.length === 0) {
  displayText = '--';
} else if (item.subQueries.length === 1) {
  displayText = item.subQueries[0];
} else {
  displayText = `[${item.subQueries.map(q => `'${q}'`).join(', ')}]`;
}
```

## Files Modified

1. **`frontend/src/components/LocalTaskDetail.tsx`**
   - Updated `SearchStatsItem` interface
   - Rewrote `computeStats` to process per-record stats
   - Updated table rendering logic

2. **`frontend/src/components/MergedTaskViewer.tsx`**
   - Same changes as `LocalTaskDetail.tsx`
   - Ensures consistency across both viewers

## Benefits

✅ **Clear subquery visibility** - See exactly what each platform generated  
✅ **1:1 mapping** - Each row = one search execution  
✅ **Easy debugging** - Identify which platform/round produced which subqueries  
✅ **URL traceability** - See how many URLs each subquery set retrieved  
✅ **Platform comparison** - Compare subquery strategies across platforms  

## Validation

### TypeScript Compilation:
```bash
cd frontend && pnpm exec tsc --noEmit
# ✅ No errors
```

### Database Query:
```sql
SELECT 
  sr.id,
  sr.keyword,
  sr.platform,
  sr.round_number,
  GROUP_CONCAT(sq.query, '|||') as subqueries,
  COUNT(DISTINCT c.url) as url_count
FROM search_records sr
LEFT JOIN search_queries sq ON sr.id = sq.record_id
LEFT JOIN citations c ON sr.id = c.record_id
WHERE sr.task_id = 31
GROUP BY sr.id
ORDER BY sr.keyword, sr.platform, sr.round_number;
```

## Migration Notes

**No Database Changes Required:**
- Existing schema supports this view
- Only frontend display logic changed
- All existing data compatible

## Future Enhancements

1. **Grouping Toggle** - Allow switching between per-record and aggregated view
2. **Sorting** - Sort by platform, round, URL count
3. **Filtering** - Filter by platform or keyword
4. **Export** - Include detailed subquery mapping in Excel export
5. **Highlighting** - Highlight duplicate subqueries across records
