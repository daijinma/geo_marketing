type InsightItem = {
  label: string;
  value?: string;
  description: string;
};

type OpportunityItem = {
  title: string;
  description: string;
};

type InsightBlockProps = {
  conclusion: string;
  insights?: InsightItem[];
  opportunities?: OpportunityItem[];
};

export function InsightBlock({ conclusion, insights, opportunities }: InsightBlockProps) {
  return (
    <div className="space-y-5 my-6">
      <div className="rounded-lg border border-blue-200 bg-blue-50 px-5 py-4">
        <div className="flex items-start gap-3">
          <span className="mt-0.5 inline-flex h-5 w-14 shrink-0 items-center justify-center rounded bg-blue-600 text-[10px] font-bold uppercase tracking-wider text-white">
            分析
          </span>
          <p className="text-sm text-blue-900 leading-relaxed font-medium">{conclusion}</p>
        </div>
      </div>

      {insights && insights.length > 0 && (
        <div className="grid gap-4 sm:grid-cols-3">
          {insights.map((item) => (
            <div key={item.label} className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
              <p className="text-xs font-bold uppercase tracking-widest text-slate-400 mb-1">{item.label}</p>
              {item.value && (
                <p className="text-2xl font-serif font-bold text-slate-900 mb-1">{item.value}</p>
              )}
              <p className="text-xs text-slate-500 leading-relaxed">{item.description}</p>
            </div>
          ))}
        </div>
      )}

      {opportunities && opportunities.length > 0 && (
        <div className="rounded-xl border border-emerald-200 bg-emerald-50 overflow-hidden">
          <div className="px-5 py-3 border-b border-emerald-200 bg-emerald-100/60">
            <span className="text-xs font-bold uppercase tracking-widest text-emerald-700">机会解读</span>
          </div>
          <ul className="divide-y divide-emerald-100">
            {opportunities.map((opp, i) => (
              <li key={i} className="flex gap-4 px-5 py-4">
                <span className="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-emerald-600 text-[10px] font-bold text-white">
                  {i + 1}
                </span>
                <div className="space-y-0.5 min-w-0">
                  <p className="text-sm font-semibold text-emerald-900">{opp.title}</p>
                  <p className="text-xs text-emerald-700 leading-relaxed">{opp.description}</p>
                </div>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
