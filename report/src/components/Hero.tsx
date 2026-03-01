import { Badge } from './Badge';

type HeroMetric = {
  label: string;
  value: string;
};

type HeroProps = {
  title: string;
  subtitle?: string;
  date?: string;
  client?: string;
  tags?: string[];
  summary?: string;
  metrics?: HeroMetric[];
};

export function Hero({ title, subtitle, date, client, tags, summary, metrics }: HeroProps) {
  return (
    <section className="relative overflow-hidden bg-slate-900 text-white py-20 print:bg-transparent print:text-black print:py-8 border-b-8 border-slate-700/50 print:border-b-2 print:border-black">
      <div className="absolute inset-0 opacity-10 pointer-events-none no-print">
         <svg className="absolute inset-0 h-full w-full" xmlns="http://www.w3.org/2000/svg">
            <defs>
              <pattern id="grid-pattern" width="40" height="40" patternUnits="userSpaceOnUse">
                <path d="M0 40L40 0H20L0 20M40 40V20L20 40" stroke="currentColor" strokeWidth="1" fill="none"/>
              </pattern>
            </defs>
            <rect width="100%" height="100%" fill="url(#grid-pattern)" />
         </svg>
      </div>
      
      <div className="relative z-10 mx-auto max-w-6xl px-6 lg:px-8">
        <div className="grid gap-12 lg:grid-cols-[1.5fr,1fr] lg:gap-20 items-start">
          
          <div className="space-y-8">
            <div className="flex items-center gap-4 text-sm font-medium tracking-widest uppercase text-slate-300 print:text-black">
              {client && (
                <>
                  <span className="font-bold text-white print:text-black">{client}</span>
                  <span className="h-1 w-1 rounded-full bg-slate-500"></span>
                </>
              )}
              {date && <span>{date}</span>}
            </div>
            
            <div className="space-y-6">
              <h1 className="text-5xl font-serif font-bold tracking-tight sm:text-6xl lg:text-7xl leading-[1.05] text-white print:text-black">
                {title}
              </h1>
              {subtitle && (
                <p className="text-2xl font-light text-slate-300 font-serif italic max-w-xl print:text-slate-600">{subtitle}</p>
              )}
            </div>

            {summary && (
              <div className="border-l-4 border-slate-500/50 pl-6 py-1 print:border-l-2 print:border-black">
                <p className="text-lg leading-relaxed text-slate-200 print:text-black max-w-prose">{summary}</p>
              </div>
            )}

            {tags && tags.length > 0 && (
              <div className="flex flex-wrap gap-2 pt-2 print:hidden">
                {tags.map((tag) => (
                  <Badge key={tag} className="bg-slate-800/50 text-slate-200 border border-slate-700 hover:bg-slate-700 font-medium px-3 py-1 rounded-sm text-xs uppercase tracking-wider transition-colors">{tag}</Badge>
                ))}
              </div>
            )}
          </div>

          {metrics && metrics.length > 0 && (
            <div className="bg-slate-800/30 backdrop-blur-sm rounded-lg p-8 border border-slate-700/50 print:bg-transparent print:border print:border-black print:rounded-none lg:mt-4">
              <div className="text-xs font-bold uppercase tracking-widest text-slate-400 border-b border-slate-700 pb-4 mb-6 print:text-black print:border-black">Executive Summary Metrics</div>
              <div className="grid gap-8 grid-cols-1 sm:grid-cols-2 lg:grid-cols-1">
                {metrics.map((metric) => (
                  <div key={metric.label} className="group">
                    <div className="text-xs font-medium uppercase tracking-wider text-slate-400 mb-1 print:text-slate-600">{metric.label}</div>
                    <div className="text-5xl font-serif font-bold text-white print:text-black tracking-tight group-hover:text-blue-200 transition-colors">
                      {metric.value}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </section>
  );
}
