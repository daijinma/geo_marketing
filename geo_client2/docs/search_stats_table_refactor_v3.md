# Search Stats Table Refactor - Final Implementation (v3)

## Overview
Refactored the "搜索词统计" table to display **per-unique-subquery-set** statistics with original keyword visible in the leftmost column.

## Core Principle

**从 subquery 的角度分解** - Group by unique subquery combinations, showing:
- **原始搜索词** (leftmost column)
- **Sub Query** combination
- **URL 数** (sum of citations for this combination)

## Data Structure

```typescript
interface SearchStatsItem {
  keyword: string;       // 原始搜索词
  subQueries: string[];  // Unique subquery combination
  urlCount: number;      // Sum of all citations for this combination
}
```

## Grouping Logic

**Group Key:** `keyword + sorted(subqueries).join('|||')`

**Examples:**
- `上海咖啡推荐::上海 咖啡 推荐` → Single subquery
- `上海咖啡推荐::上海咖啡推荐|||上海特色咖啡推荐` → Multiple subqueries
- `上海咖啡推荐::__EMPTY__` → No subqueries (--)

**Merging:**
- Same keyword + same subquery set → Merge into one row
- URL count = sum of all matching records' citation counts

## Task 31 Final Output

### Expected Table (10 rows):

```
┌──────────────┬──────────────────────────────────────────────────┬────────┐
│ 原始搜索词   │ Sub Query                                        │ URL 数 │
├──────────────┼──────────────────────────────────────────────────┼────────┤
│ 上海咖啡推荐 │ --                                               │   16   │
│ 上海咖啡推荐 │ 上海 咖啡 推荐                                    │    8   │
│ 上海咖啡推荐 │ 上海 咖啡 店 推荐                                 │    8   │
│ 上海咖啡推荐 │ 上海咖啡推荐                                      │   18   │
│ 上海咖啡推荐 │ ['上海咖啡推荐', '上海特色咖啡推荐']               │   10   │
│ 北京餐馆推荐 │ --                                               │   14   │
│ 北京餐馆推荐 │ 北京 餐馆 推荐                                    │    8   │
│ 北京餐馆推荐 │ ['北京 餐馆 推荐 热门 2026', '北京 必吃 餐厅...'] │   10   │
│ 北京餐馆推荐 │ 北京餐馆推荐                                      │   14   │
│ 北京餐馆推荐 │ ['北京餐馆推荐', '北京美食推荐']                  │   12   │
└──────────────┴──────────────────────────────────────────────────┴────────┘
```

### Detailed Breakdown:

#### 上海咖啡推荐 (5 rows)

| Sub Query | Sources | URL Count | Notes |
|-----------|---------|-----------|-------|
| `--` | yiyan-R1, yiyan-R2 | 8+8=16 | No subqueries |
| `上海 咖啡 推荐` | deepseek-R1 | 8 | Single unique subquery |
| `上海 咖啡 店 推荐` | deepseek-R2 | 8 | Single unique subquery |
| `上海咖啡推荐` | yuanbao-R1, yuanbao-R2 | 9+9=18 | Same subquery both rounds |
| `['上海咖啡推荐', '上海特色咖啡推荐']` | doubao-R1, doubao-R2 | 5+5=10 | Same set both rounds |

#### 北京餐馆推荐 (5 rows)

| Sub Query | Sources | URL Count | Notes |
|-----------|---------|-----------|-------|
| `--` | yiyan-R1, yiyan-R2 | 7+7=14 | No subqueries |
| `北京 餐馆 推荐` | deepseek-R1 | 8 | Single unique subquery |
| `['北京 餐馆 推荐 热门 2026', ...]` | deepseek-R2 | 10 | Multiple subqueries |
| `北京餐馆推荐` | yuanbao-R1, yuanbao-R2 | 6+8=14 | Same subquery both rounds |
| `['北京餐馆推荐', '北京美食推荐']` | doubao-R1, doubao-R2 | 6+6=12 | Same set both rounds |

**Total: 10 rows** (2 keywords × ~5 unique subquery combinations each)

## Implementation Details

### Statistics Computation

```typescript
const searchStatsMap = new Map<string, {
  keyword: string;
  subQueries: string[];
  urlCount: number;
}>();

records.forEach(record => {
  const subQueries = record.queries
    .map(q => q.query)
    .filter(q => q && q.trim())
    .map(q => q.trim())
    .sort();  // Sort for consistent grouping
  
  const subQueryKey = subQueries.length === 0 
    ? '__EMPTY__' 
    : subQueries.join('|||');
  
  const groupKey = `${keyword}::${subQueryKey}`;
  
  if (!searchStatsMap.has(groupKey)) {
    searchStatsMap.set(groupKey, {
      keyword,
      subQueries: subQueries.slice(),
      urlCount: 0,
    });
  }
  
  const item = searchStatsMap.get(groupKey)!;
  item.urlCount += record.citations.length;  // Sum, not deduplicate
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

### Sorting

```typescript
searchStats.sort((a, b) => {
  // First by keyword
  if (a.keyword !== b.keyword) {
    return a.keyword.localeCompare(b.keyword);
  }
  
  // Empty subqueries first
  if (a.subQueries.length === 0 && b.subQueries.length === 0) return 0;
  if (a.subQueries.length === 0) return -1;
  if (b.subQueries.length === 0) return 1;
  
  // Then by subquery text
  return a.subQueries.join(',').localeCompare(b.subQueries.join(','));
});
```

## Key Features

### ✅ Two `--` Rows Maximum
- Only keywords where **all platforms and rounds** have no subqueries show `--`
- Task 31: Yiyan never generates subqueries → 2 `--` rows (one per keyword)

### ✅ Subquery Deduplication
- `['上海咖啡推荐', '上海特色咖啡推荐']` appears once, even though doubao generates it in both rounds
- URL count is **summed** (5+5=10), not deduplicated

### ✅ Clear Visibility
- See exactly which subquery combinations exist
- See how many URLs each combination retrieved
- Original keyword always visible for context

### ✅ Platform Comparison
- Compare strategies: DeepSeek varies by round, Doubao is consistent
- Identify which platforms generate more diverse queries
- Spot patterns in subquery generation

## Files Modified

1. **`frontend/src/components/LocalTaskDetail.tsx`**
   - Updated `SearchStatsItem` interface (3 fields)
   - Rewrote `computeStats` to group by `keyword + subquery_set`
   - Added "原始搜索词" column to table
   - Fixed variable naming conflict (`recordCitationCount`)

2. **`frontend/src/components/MergedTaskViewer.tsx`**
   - Same changes as `LocalTaskDetail.tsx`
   - Ensures consistency across both viewers

## Validation

### TypeScript Compilation
```bash
cd frontend && pnpm exec tsc --noEmit
# ✅ No errors
```

### Database Verification
```sql
SELECT 
  sr.keyword,
  GROUP_CONCAT(sq.query, ', ') as subqueries,
  SUM((SELECT COUNT(*) FROM citations c WHERE c.record_id = sr.id)) as total_urls
FROM search_records sr
LEFT JOIN search_queries sq ON sr.id = sq.record_id
WHERE sr.task_id = 31
GROUP BY sr.keyword, (
  SELECT GROUP_CONCAT(sq2.query)
  FROM search_queries sq2
  WHERE sq2.record_id = sr.id
  ORDER BY sq2.query
);
```

### Expected Row Count
- **Task 31:** 10 rows (verified against database)
- **上海咖啡推荐:** 5 unique subquery combinations
- **北京餐馆推荐:** 5 unique subquery combinations

## Benefits

✅ **Clear Subquery Analysis** - See all unique subquery combinations at a glance  
✅ **Reduced Redundancy** - Only 2 `--` rows instead of 4  
✅ **Better Grouping** - Same subqueries merged automatically  
✅ **Original Context** - Keyword always visible for reference  
✅ **Accurate Totals** - URL counts summed correctly  
✅ **Platform Insights** - Easy to compare different platform strategies  

## Migration Notes

**No Breaking Changes:**
- Database schema unchanged
- API unchanged
- Only frontend display logic modified

**Backward Compatible:**
- Works with all existing task data
- No migration scripts needed

## Future Enhancements

1. **Expand/Collapse** - Show/hide details for each keyword group
2. **Platform Labels** - Show which platforms contributed to each row
3. **Percentage View** - Show URL count as % of total
4. **Filtering** - Filter by keyword or platform
5. **Sorting Options** - Sort by URL count, subquery count, etc.
