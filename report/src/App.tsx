import { useEffect, useState } from 'react';
import reportConfig from './config/report.json';
import { Hero } from './components/Hero';
import { Section } from './components/Section';
import { SectionContent } from './components/SectionContent';
import { TOC } from './components/TOC';
import { ReportSchema, type Report } from './lib/reportSchema';
import { z } from 'zod';

export default function App() {
  const [reportData, setReportData] = useState<Report | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    try {
      const parsed = ReportSchema.parse(reportConfig);
      setReportData(parsed);
      document.title = `${parsed.meta.title} - ${parsed.meta.client || 'Report'}`;
    } catch (err) {
      if (err instanceof z.ZodError) {
        setError(`Invalid report configuration: ${err.issues.map((e: z.ZodIssue) => `${e.path.join('.')}: ${e.message}`).join(', ')}`);
      } else {
        setError('Failed to load report configuration');
      }
      console.error(err);
    }
  }, []);

  if (error) {
    return (
      <div className="flex h-screen items-center justify-center bg-red-50 p-4">
        <div className="max-w-md rounded-lg bg-white p-6 shadow-lg border border-red-200">
          <h2 className="text-lg font-bold text-red-700">Configuration Error</h2>
          <p className="mt-2 text-sm text-red-600 font-mono whitespace-pre-wrap break-words">{error}</p>
        </div>
      </div>
    );
  }

  if (!reportData) {
    return <div className="flex h-screen items-center justify-center text-muted-foreground animate-pulse">Loading report...</div>;
  }

  const tocItems = reportData.sections.map((section) => ({
    id: section.id,
    title: section.title,
  }));

  return (
    <div className="min-h-screen bg-background font-sans text-foreground selection:bg-primary/10 selection:text-primary print:bg-white">
      {/* Mobile Header */}
      <div className="lg:hidden print:hidden sticky top-0 z-40 bg-background/80 backdrop-blur-md border-b border-border p-4">
        <details className="group">
          <summary className="flex cursor-pointer list-none items-center justify-between text-sm font-medium text-foreground">
            <div className="flex items-center gap-2">
              <span className="font-serif font-bold text-lg">{reportData.meta.client || 'Report'}</span>
            </div>
            <div className="flex items-center gap-2 text-muted-foreground">
              <span>Menu</span>
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="transition-transform group-open:rotate-180"><polyline points="6 9 12 15 18 9"></polyline></svg>
            </div>
          </summary>
          <div className="absolute left-0 right-0 top-full border-b border-border bg-background px-4 py-4 shadow-lg animate-in slide-in-from-top-2">
            <nav className="flex flex-col gap-2">
              {tocItems.map((item) => (
                <a
                  key={item.id}
                  href={`#${item.id}`}
                  className="block rounded-md py-3 px-4 text-sm text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
                  onClick={(e) => {
                    const details = e.currentTarget.closest('details');
                    if (details) details.removeAttribute('open');
                  }}
                >
                  {item.title}
                </a>
              ))}
            </nav>
          </div>
        </details>
      </div>

      <div className="flex flex-col lg:flex-row print:block">
        
        {/* Desktop Sidebar (TOC) - Moved to left */}
        <TOC items={tocItems} title={reportData.ui?.tocTitle} />

        {/* Main Content */}
        <main className="flex-1 w-full min-w-0 print:pb-0">
          <div className="mx-auto max-w-5xl px-4 py-8 lg:px-12 lg:py-16 space-y-20 print:space-y-10 print:p-0">
            <Hero
              title={reportData.meta.title}
              subtitle={reportData.meta.subtitle}
              date={reportData.meta.date}
              client={reportData.meta.client}
              tags={reportData.meta.tags}
              summary={reportData.meta.summary}
              metrics={reportData.meta.heroMetrics}
            />

            <div className="space-y-24 print:space-y-12">
              {reportData.sections.map((section) => (
                <div key={section.id} className="print:break-inside-avoid scroll-mt-24" id={section.id}>
                  {section.type === 'chapter-divider' ? (
                    <SectionContent
                      type={section.type}
                      chapterNumber={(section as any).chapterNumber}
                      chapterSubtitle={(section as any).chapterSubtitle}
                      content={[section.title]}
                    />
                  ) : (
                    <Section
                      id={section.id}
                      title={section.title}
                      description={section.description}
                    >
                      <SectionContent
                        type={section.type}
                        content={section.content}
                        images={section.images}
                        table={section.table}
                        subsections={section.subsections}
                        ui={reportData.ui}
                        queryAnalysis={(section as any).queryAnalysis}
                        chapterNumber={(section as any).chapterNumber}
                        chapterSubtitle={(section as any).chapterSubtitle}
                        brandComparison={(section as any).brandComparison}
                        insightBlock={(section as any).insightBlock}
                        insight={(section as any).insight}
                      />
                    </Section>
                  )}
                </div>
              ))}
            </div>

            <footer className="mt-32 border-t border-border pt-12 pb-12 text-center text-sm text-muted-foreground print:hidden">
              <div className="flex flex-col items-center gap-4">
                <div className="h-px w-12 bg-border"></div>
                <p className="font-serif text-lg text-foreground">{reportData.meta.client}</p>
                <p className="max-w-md text-muted-foreground/60">{reportData.meta.title}</p>
                <div className="mt-8 text-xs font-mono uppercase tracking-widest opacity-50">
                  {reportData.ui?.footerDataSource} · {reportData.meta.date}
                </div>
              </div>
            </footer>
          </div>
        </main>

      </div>
    </div>
  );
}
