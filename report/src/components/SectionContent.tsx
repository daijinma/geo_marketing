import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { Table } from './Table';
import { QueryAnalysis } from './QueryAnalysis';
import { ChapterDivider } from './ChapterDivider';
import { BrandComparison } from './BrandComparison';
import { InsightBlock } from './InsightBlock';
import placeholderSrc from '../assets/placeholder.svg';

type ImageItem = {
  src: string;
  alt?: string;
};

type TableData = {
  headers: string[];
  rows: string[][];
  title?: string;
};

type UIConfig = {
  recommendationLabel?: string;
  recommendationItemPrefix?: string;
};

type InsightItem = {
  label: string;
  value?: string;
  description: string;
};

type OpportunityItem = {
  title: string;
  description: string;
};

type InsightBlockData = {
  conclusion: string;
  insights?: InsightItem[];
  opportunities?: OpportunityItem[];
};

type BrandData = {
  name: string;
  citationCount: number;
  citationRate: number;
  mentionRate: number;
  top3Rate: number;
};

type BrandComparisonData = {
  platforms: {
    platform: string;
    totalCitations: number;
    brands: BrandData[];
  }[];
};

type SectionContentProps = {
  type: string;
  content?: string[];
  images?: ImageItem[];
  table?: TableData;
  subsections?: {
    title: string;
    content?: string[];
    images?: ImageItem[];
    table?: TableData;
  }[];
  ui?: UIConfig;
  queryAnalysis?: Parameters<typeof QueryAnalysis>[0]['data'];
  chapterNumber?: string;
  chapterSubtitle?: string;
  brandComparison?: BrandComparisonData;
  insightBlock?: InsightBlockData;
  insight?: string;
};

function MarkdownContent({ content }: { content: string[] }) {
  return (
    <div className="report-prose print:text-[14px] print:leading-normal max-w-3xl mx-auto mb-10">
      <ReactMarkdown remarkPlugins={[remarkGfm]}>{content.join('\n\n')}</ReactMarkdown>
    </div>
  );
}

function ImageGrid({ images }: { images: ImageItem[] }) {
  if (!images || images.length === 0) return null;
  return (
    <div className="grid gap-12 md:grid-cols-2 print:grid-cols-2 print:gap-4 my-12 break-inside-avoid">
      {images.map((image, idx) => (
        <figure key={`${image.src}-${idx}`} className="group relative">
          <div className="overflow-hidden border border-slate-200 bg-slate-50">
            <img
              src={image.src}
              alt={image.alt || ''}
              className="h-auto w-full object-contain"
              loading="lazy"
              onError={(e) => {
                const img = e.target as HTMLImageElement;
                if (img.src !== placeholderSrc) {
                  img.src = placeholderSrc;
                  img.classList.add('opacity-50', 'grayscale', 'p-8');
                }
              }}
            />
          </div>
          {image.alt && (
            <figcaption className="mt-3 text-[11px] font-bold uppercase tracking-widest text-slate-500 text-center border-t border-slate-200 pt-2 print:text-[10px] print:mt-1">
              Figure {idx + 1}: {image.alt}
            </figcaption>
          )}
        </figure>
      ))}
    </div>
  );
}

export function SectionContent({
  type,
  content,
  images,
  table,
  subsections,
  ui,
  queryAnalysis,
  chapterNumber,
  chapterSubtitle,
  brandComparison,
  insightBlock,
  insight,
}: SectionContentProps) {
  if (type === 'chapter-divider' && chapterNumber) {
    return (
      <ChapterDivider
        number={chapterNumber}
        title={content?.[0] ?? ''}
        subtitle={chapterSubtitle}
      />
    );
  }

  if (type === 'query-analysis' && queryAnalysis) {
    return (
      <div className="space-y-12">
        <QueryAnalysis data={queryAnalysis} />
        {images && <ImageGrid images={images} />}
      </div>
    );
  }

  if (type === 'brand-comparison' && brandComparison) {
    return (
      <div className="space-y-8">
        <BrandComparison insight={insight} platforms={brandComparison.platforms} />
        {images && <ImageGrid images={images} />}
        {table && <Table headers={table.headers} rows={table.rows} title={table.title} />}
      </div>
    );
  }

  if (type === 'recommendations' && content) {
    return (
      <div className="space-y-8 my-12">
        <h4 className="text-sm font-bold uppercase tracking-widest text-slate-400 border-b border-slate-200 pb-2 mb-6">Strategic Recommendations</h4>
        <div className="grid gap-x-12 gap-y-10 md:grid-cols-2">
          {content.map((item, index) => (
            <div key={index} className="relative pl-0">
              <div className="flex items-baseline gap-4 mb-3">
                <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-slate-900 text-xs font-bold text-white shadow-sm">
                  {index + 1}
                </span>
                <span className="text-xs font-bold uppercase tracking-widest text-slate-500 pt-1">
                  {ui?.recommendationLabel || 'Action Item'}
                </span>
              </div>
              <div className="report-prose text-[15px] leading-relaxed text-slate-700 pl-10 prose-p:mb-0 border-l border-slate-200 pl-4 ml-3">
                <ReactMarkdown remarkPlugins={[remarkGfm]}>{item}</ReactMarkdown>
              </div>
            </div>
          ))}
        </div>
      </div>
    );
  }

  if (type === 'mixed' && subsections) {
    return (
      <div className="space-y-20 divide-y divide-slate-100">
        {subsections.map((section, idx) => (
          <div key={idx} className={idx > 0 ? 'pt-16' : ''}>
            <div className="flex items-baseline gap-4 mb-8">
              <span className="text-4xl font-serif font-bold text-slate-200">{String(idx + 1).padStart(2, '0')}</span>
              <h3 className="text-2xl font-serif font-bold tracking-tight text-slate-900">
                {section.title}
              </h3>
            </div>
            <div className="space-y-10 pl-0 lg:pl-14 border-l border-slate-100 ml-4 lg:ml-[1.1rem]">
              {section.content && <MarkdownContent content={section.content} />}
              {section.images && <ImageGrid images={section.images} />}
              {section.table && (
                <Table
                  headers={section.table.headers}
                  rows={section.table.rows}
                  title={section.table.title}
                />
              )}
            </div>
          </div>
        ))}
      </div>
    );
  }

  return (
    <div className="space-y-8">
      {insightBlock && (
        <InsightBlock
          conclusion={insightBlock.conclusion}
          insights={insightBlock.insights}
          opportunities={insightBlock.opportunities}
        />
      )}
      {content && <MarkdownContent content={content} />}
      {images && <ImageGrid images={images} />}
      {table && <Table headers={table.headers} rows={table.rows} title={table.title} />}
    </div>
  );
}
