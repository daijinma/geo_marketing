type TableProps = {
  headers: string[];
  rows: string[][];
  title?: string;
};

export function Table({ headers, rows, title }: TableProps) {
  return (
    <div className="my-10 w-full break-inside-avoid">
      {title && (
        <div className="mb-3 px-1">
          <h4 className="text-xs font-bold uppercase tracking-widest text-slate-500">{title}</h4>
        </div>
      )}
      <div className="relative w-full overflow-x-auto border-t-2 border-slate-800">
        <table className="w-full text-sm text-left border-collapse">
          <thead>
            <tr className="border-b border-slate-300">
              {headers.map((header, index) => (
                <th
                  key={index}
                  className="px-4 py-3 font-bold text-slate-900 uppercase tracking-wide text-[11px] whitespace-nowrap bg-slate-50/50"
                >
                  {header}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-200 border-b border-slate-200">
            {rows.map((row, rowIndex) => (
              <tr
                key={rowIndex}
                className="hover:bg-slate-50 transition-colors"
              >
                {row.map((cell, cellIndex) => (
                  <td
                    key={`${rowIndex}-${cellIndex}`}
                    className="px-4 py-3 align-top font-medium text-slate-700 tabular-nums"
                  >
                    {cell || <span className="text-slate-300">-</span>}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <div className="mt-2 text-[10px] text-slate-400 text-right italic px-1">
        Data source: LLM Sentry Analysis
      </div>
    </div>
  );
}
