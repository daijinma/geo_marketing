type Brand = {
  name: string;
  citationCount: number;
  citationRate: number;
  mentionRate: number;
  top3Rate: number;
};

type PlatformData = {
  platform: string;
  totalCitations: number;
  brands: Brand[];
};

type BrandComparisonProps = {
  insight?: string;
  platforms: PlatformData[];
};

const BRAND_COLORS: Record<string, string> = {
  百保力: 'bg-rose-500',
  尤尼克斯: 'bg-blue-500',
  胜利: 'bg-emerald-500',
  李宁: 'bg-amber-500',
  亚狮龙: 'bg-violet-500',
  凯胜: 'bg-cyan-500',
};

function getBrandColor(name: string) {
  return BRAND_COLORS[name] ?? 'bg-slate-400';
}

function Bar({ value, max, colorClass }: { value: number; max: number; colorClass: string }) {
  const pct = max > 0 ? Math.round((value / max) * 100) : 0;
  return (
    <div className="flex items-center gap-2 min-w-0">
      <div className="flex-1 h-2 rounded-full bg-slate-100 overflow-hidden">
        <div
          className={`h-full rounded-full transition-all ${colorClass}`}
          style={{ width: `${pct}%` }}
        />
      </div>
      <span className="text-xs font-mono text-slate-500 w-10 text-right shrink-0">{value}%</span>
    </div>
  );
}

function PlatformBlock({ data }: { data: PlatformData }) {
  const maxCitationRate = Math.max(...data.brands.map((b) => b.citationRate));
  const _ = maxCitationRate;

  return (
    <div className="rounded-xl border border-slate-200 bg-white overflow-hidden shadow-sm">
      <div className="flex items-center justify-between px-5 py-3 bg-slate-50 border-b border-slate-200">
        <span className="text-sm font-bold text-slate-900">{data.platform}</span>
        <span className="text-xs text-slate-500">总引用 {data.totalCitations} 次</span>
      </div>
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-slate-100">
              <th className="text-left px-5 py-2.5 text-xs font-bold uppercase tracking-wider text-slate-400 w-20">品牌</th>
              <th className="px-4 py-2.5 text-xs font-bold uppercase tracking-wider text-slate-400 text-center w-16">引用次数</th>
              <th className="px-4 py-2.5 text-xs font-bold uppercase tracking-wider text-slate-400 min-w-[140px]">引用占比</th>
              <th className="px-4 py-2.5 text-xs font-bold uppercase tracking-wider text-slate-400 min-w-[140px]">提及率</th>
              <th className="px-4 py-2.5 text-xs font-bold uppercase tracking-wider text-slate-400 min-w-[140px]">Top 3 占比</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-50">
            {data.brands.map((brand) => {
              const color = getBrandColor(brand.name);
              const isBabolat = brand.name === '百保力';
              return (
                <tr
                  key={brand.name}
                  className={isBabolat ? 'bg-rose-50/60' : 'hover:bg-slate-50/50 transition-colors'}
                >
                  <td className="px-5 py-3">
                    <div className="flex items-center gap-2">
                      <span className={`h-2.5 w-2.5 rounded-full shrink-0 ${color}`} />
                      <span className={`text-sm font-medium ${isBabolat ? 'text-rose-700 font-bold' : 'text-slate-700'}`}>
                        {brand.name}
                      </span>
                    </div>
                  </td>
                  <td className="px-4 py-3 text-center">
                    <span className="text-sm font-mono text-slate-700">{brand.citationCount}</span>
                  </td>
                  <td className="px-4 py-3">
                    <Bar value={brand.citationRate} max={100} colorClass={color} />
                  </td>
                  <td className="px-4 py-3">
                    <Bar value={brand.mentionRate} max={100} colorClass={color} />
                  </td>
                  <td className="px-4 py-3">
                    <Bar value={brand.top3Rate} max={100} colorClass={color} />
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}

export function BrandComparison({ insight, platforms }: BrandComparisonProps) {
  return (
    <div className="space-y-6">
      {insight && (
        <div className="rounded-lg border border-amber-200 bg-amber-50 px-5 py-4 flex gap-3">
          <span className="mt-0.5 text-amber-500 shrink-0">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
              <circle cx="12" cy="12" r="10" /><line x1="12" y1="8" x2="12" y2="12" /><line x1="12" y1="16" x2="12.01" y2="16" />
            </svg>
          </span>
          <p className="text-sm text-amber-800 leading-relaxed">{insight}</p>
        </div>
      )}
      <div className="space-y-5">
        {platforms.map((p) => (
          <PlatformBlock key={p.platform} data={p} />
        ))}
      </div>
    </div>
  );
}
