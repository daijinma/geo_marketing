type A2Item = string | { category: string; synonyms: string[] };

type QueryAnalysisData = {
  rootWords: {
    A1: { label: string; description?: string; items: string[] };
    A2: { label: string; description?: string; items: A2Item[] };
    A3: { label: string; description?: string; items: string[] };
  };
  secondaryRoots: { id: string; label: string; desc: string }[];
  queries: { type: string; text: string; priority: string }[];
};

const TYPE_COLOR_MAP: Record<string, { bg: string; text: string; border: string; dot: string }> = {
  A: {
    bg: 'bg-slate-900',
    text: 'text-white',
    border: 'border-slate-900',
    dot: 'bg-white',
  },
  B1: {
    bg: 'bg-blue-600',
    text: 'text-white',
    border: 'border-blue-600',
    dot: 'bg-blue-200',
  },
  B2: {
    bg: 'bg-indigo-600',
    text: 'text-white',
    border: 'border-indigo-600',
    dot: 'bg-indigo-200',
  },
  B3: {
    bg: 'bg-violet-600',
    text: 'text-white',
    border: 'border-violet-600',
    dot: 'bg-violet-200',
  },
  C: {
    bg: 'bg-amber-500',
    text: 'text-white',
    border: 'border-amber-500',
    dot: 'bg-amber-100',
  },
};

const DEFAULT_COLOR = {
  bg: 'bg-slate-500',
  text: 'text-white',
  border: 'border-slate-500',
  dot: 'bg-slate-200',
};

function getColor(type: string) {
  return TYPE_COLOR_MAP[type] ?? DEFAULT_COLOR;
}

function StepLabel({ step, label }: { step: string; label: string }) {
  return (
    <div className="flex items-center gap-3 mb-6">
      <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-slate-900 text-xs font-bold text-white">
        {step}
      </span>
      <h3 className="text-lg font-serif font-semibold text-slate-900 tracking-tight">{label}</h3>
    </div>
  );
}

function RootWordCard({
  id,
  label,
  description,
  children,
}: {
  id: string;
  label: string;
  description?: string;
  children: React.ReactNode;
}) {
  return (
    <div className="rounded-xl border border-slate-200 bg-white overflow-hidden shadow-sm">
      <div className="flex items-start gap-3 px-5 py-4 border-b border-slate-100 bg-slate-50">
        <span className="mt-0.5 flex h-6 w-10 shrink-0 items-center justify-center rounded-md bg-slate-900 text-[11px] font-bold text-white tracking-wider">
          {id}
        </span>
        <div>
          <p className="text-sm font-semibold text-slate-900">{label}</p>
          {description && <p className="text-[12px] text-slate-500 mt-0.5">{description}</p>}
        </div>
      </div>
      <div className="px-5 py-4">{children}</div>
    </div>
  );
}

export function QueryAnalysis({ data }: { data: QueryAnalysisData }) {
  const { rootWords, secondaryRoots, queries } = data;

  const queryGroups: Record<string, typeof queries> = {};
  for (const q of queries) {
    if (!queryGroups[q.type]) queryGroups[q.type] = [];
    queryGroups[q.type].push(q);
  }

  return (
    <div className="space-y-16">
      <div>
        <StepLabel step="1" label="确定根词" />

        <div className="grid gap-5 md:grid-cols-3">
          <RootWordCard id="A1" label={rootWords.A1.label} description={rootWords.A1.description}>
            <ul className="space-y-2">
              {rootWords.A1.items.map((item) => (
                <li key={item} className="flex items-center gap-2 text-sm text-slate-700">
                  <span className="h-1.5 w-1.5 rounded-full bg-slate-400 shrink-0" />
                  {item}
                </li>
              ))}
            </ul>
          </RootWordCard>

          <RootWordCard id="A2" label={rootWords.A2.label} description={rootWords.A2.description}>
            <ul className="space-y-3">
              {rootWords.A2.items.map((item) => {
                if (typeof item === 'string') {
                  return (
                    <li key={item} className="flex items-center gap-2 text-sm text-slate-700">
                      <span className="h-1.5 w-1.5 rounded-full bg-slate-400 shrink-0" />
                      {item}
                    </li>
                  );
                }
                return (
                  <li key={item.category}>
                    <p className="text-sm font-semibold text-slate-800 mb-1">{item.category}</p>
                    <div className="flex flex-wrap gap-1 pl-1">
                      {item.synonyms.map((s) => (
                        <span
                          key={s}
                          className="inline-block rounded bg-slate-100 px-2 py-0.5 text-[11px] text-slate-600 font-medium"
                        >
                          {s}
                        </span>
                      ))}
                    </div>
                  </li>
                );
              })}
            </ul>
          </RootWordCard>

          <RootWordCard id="A3" label={rootWords.A3.label} description={rootWords.A3.description}>
            <ul className="space-y-2">
              {rootWords.A3.items.map((item) => (
                <li key={item} className="flex items-center gap-2 text-sm font-medium text-slate-800">
                  <span className="h-2 w-2 rounded-full bg-slate-900 shrink-0" />
                  {item}
                </li>
              ))}
            </ul>
            <p className="mt-4 text-[11px] font-medium uppercase tracking-widest text-slate-400">
              百保力实际覆盖品类
            </p>
          </RootWordCard>
        </div>
      </div>

      <div>
        <StepLabel step="2" label={'根词转化为大模型 Query（以\u201c羽毛球拍\u201d为例）'} />

        <div className="mb-8">
          <p className="text-xs font-bold uppercase tracking-widest text-slate-400 mb-4">
            二级根词维度
          </p>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
            {secondaryRoots.map((root) => (
              <div
                key={root.id}
                className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm"
              >
                <div className="flex items-center gap-2 mb-2">
                  <span className="inline-flex h-6 min-w-[2.25rem] items-center justify-center rounded-md bg-slate-900 text-[11px] font-bold text-white px-2">
                    {root.id}
                  </span>
                  <span className="text-sm font-semibold text-slate-900">{root.label}</span>
                </div>
                <p className="text-[12px] text-slate-500 leading-relaxed">{root.desc}</p>
              </div>
            ))}
          </div>
        </div>

        <div>
          <p className="text-xs font-bold uppercase tracking-widest text-slate-400 mb-4">
            P0 Query 列表（共 {queries.length} 条）
          </p>
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {Object.entries(queryGroups).map(([type, items]) => {
              const color = getColor(type);
              return (
                <div key={type} className="rounded-xl border border-slate-200 bg-white shadow-sm overflow-hidden">
                  <div className={`px-4 py-2.5 flex items-center gap-2 ${color.bg}`}>
                    <span className={`text-xs font-bold tracking-widest uppercase ${color.text}`}>
                      {type === 'A' ? '品牌 Query' : type === 'C' ? '参数科普 Query' : `${type} 类 Query`}
                    </span>
                    <span className={`ml-auto text-[10px] font-bold px-2 py-0.5 rounded-full bg-white/20 ${color.text}`}>
                      {items.length} 条
                    </span>
                  </div>
                  <ul className="divide-y divide-slate-100">
                    {items.map((q, i) => (
                      <li key={i} className="flex items-start gap-3 px-4 py-3 hover:bg-slate-50 transition-colors">
                        <span className={`mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full ${color.bg}`} />
                        <span className="text-sm text-slate-700 leading-relaxed">{q.text}</span>
                        <span className="ml-auto shrink-0 mt-0.5 text-[10px] font-bold text-slate-400 bg-slate-100 rounded px-1.5 py-0.5">
                          {q.priority}
                        </span>
                      </li>
                    ))}
                  </ul>
                </div>
              );
            })}
          </div>
        </div>
      </div>
    </div>
  );
}
