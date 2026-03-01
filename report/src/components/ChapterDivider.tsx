type ChapterDividerProps = {
  number: string;
  title: string;
  subtitle?: string;
};

export function ChapterDivider({ number, title, subtitle }: ChapterDividerProps) {
  return (
    <div className="relative flex flex-col items-center justify-center py-24 my-8 rounded-2xl overflow-hidden bg-slate-900 text-white select-none print:py-16">
      <div className="absolute inset-0 opacity-10 pointer-events-none">
        <svg className="absolute inset-0 h-full w-full" xmlns="http://www.w3.org/2000/svg">
          <defs>
            <pattern id={`chapter-grid-${number}`} width="40" height="40" patternUnits="userSpaceOnUse">
              <path d="M0 40L40 0H20L0 20M40 40V20L20 40" stroke="currentColor" strokeWidth="1" fill="none" />
            </pattern>
          </defs>
          <rect width="100%" height="100%" fill={`url(#chapter-grid-${number})`} />
        </svg>
      </div>

      <div className="absolute left-6 top-1/2 -translate-y-1/2 flex flex-col gap-3 opacity-20 print:hidden">
        {[0, 1, 2, 3].map((i) => (
          <div
            key={i}
            className="h-5 w-5 rotate-12 rounded-sm bg-blue-400"
            style={{ opacity: 1 - i * 0.2 }}
          />
        ))}
      </div>
      <div className="absolute right-6 top-1/2 -translate-y-1/2 flex flex-col gap-3 opacity-20 print:hidden">
        {[0, 1, 2, 3].map((i) => (
          <div
            key={i}
            className="h-5 w-5 -rotate-12 rounded-sm bg-blue-400"
            style={{ opacity: 1 - i * 0.2 }}
          />
        ))}
      </div>

      <div className="relative z-10 flex flex-col items-center gap-4 text-center px-8">
        <span className="text-[7rem] font-serif font-bold leading-none text-white/10 tracking-tight select-none print:text-[5rem]">
          {number}
        </span>
        <div className="-mt-12 space-y-3">
          <h2 className="text-3xl font-serif font-bold tracking-tight text-white sm:text-4xl print:text-2xl">
            {title}
          </h2>
          {subtitle && (
            <p className="text-base text-slate-300 max-w-md mx-auto font-light">{subtitle}</p>
          )}
        </div>
      </div>
    </div>
  );
}
