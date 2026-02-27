import { useState, useRef, useCallback, useEffect } from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { Eye, Code2 } from 'lucide-react';
import { hasNamedCaptureGroups } from '@/utils/browser-features';

type ContentFormat = 'plain' | 'markdown' | 'html';

interface RichContentEditorProps {
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
  placeholder?: string;
  rows?: number;
}

/** Detect whether the text looks like HTML or Markdown */
function detectFormat(text: string): ContentFormat {
  const trimmed = text.trim();
  if (!trimmed) return 'plain';
  // HTML: starts with a tag or contains common block tags
  if (/^<[a-zA-Z]/.test(trimmed) || /<(p|div|h[1-6]|ul|ol|li|strong|em|br|img|a|blockquote|pre|code)\b/i.test(trimmed)) {
    return 'html';
  }
  // Markdown: contains heading, list, bold, italic, code fence, link, blockquote
  if (/(^#{1,6}\s|^\*\s|^-\s|^\d+\.\s|\*\*|__|\[.+\]\(.+\)|^```|^>)/m.test(trimmed)) {
    return 'markdown';
  }
  return 'plain';
}

export default function RichContentEditor({
  value,
  onChange,
  disabled = false,
  placeholder = '请输入文章正文内容...',
  rows = 16,
}: RichContentEditorProps) {
  const [format, setFormat] = useState<ContentFormat>(() => detectFormat(value));
  const [showPreview, setShowPreview] = useState(false);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  // Re-detect format when value changes externally (e.g. cleared)
  useEffect(() => {
    if (!value) {
      setFormat('plain');
      setShowPreview(false);
    }
  }, [value]);

  const handleChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const newVal = e.target.value;
    onChange(newVal);
    setFormat(detectFormat(newVal));
  };

  const handlePaste = useCallback(
    (e: React.ClipboardEvent<HTMLTextAreaElement>) => {
      // Try to get HTML from clipboard first (rich paste from editors)
      const html = e.clipboardData.getData('text/html');
      const plain = e.clipboardData.getData('text/plain');

      if (html && html.trim()) {
        // Strip the wrapping MS-Word / Google Docs meta fragments but keep meaningful HTML
        const cleaned = cleanPastedHTML(html);
        if (cleaned && cleaned !== plain?.trim()) {
          e.preventDefault();
          insertAtCursor(cleaned);
          setFormat('html');
          setShowPreview(true);
          return;
        }
      }

      // Plain text: detect if it's Markdown
      if (plain) {
        const detected = detectFormat(plain);
        if (detected !== 'plain') {
          // Let the default paste happen (plain text into textarea),
          // but update format & show preview
          setFormat(detected);
          setShowPreview(true);
        }
      }
    },
    [value, onChange],
  );

  /** Insert text at cursor position in the textarea */
  const insertAtCursor = (text: string) => {
    const el = textareaRef.current;
    if (!el) {
      onChange(text);
      return;
    }
    const start = el.selectionStart ?? 0;
    const end = el.selectionEnd ?? 0;
    const before = value.slice(0, start);
    const after = value.slice(end);
    const newVal = before + text + after;
    onChange(newVal);
    // Restore cursor after React re-render
    requestAnimationFrame(() => {
      el.selectionStart = el.selectionEnd = start + text.length;
      el.focus();
    });
  };

  const formatBadge: Record<ContentFormat, { label: string; cls: string }> = {
    plain:    { label: '纯文本', cls: 'bg-muted text-muted-foreground' },
    markdown: { label: 'Markdown', cls: 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300' },
    html:     { label: 'HTML', cls: 'bg-orange-100 text-orange-700 dark:bg-orange-900/40 dark:text-orange-300' },
  };

  const canPreview = format !== 'plain' && value.trim().length > 0;

  return (
    <div className="space-y-1.5">
      {/* Toolbar */}
      <div className="flex items-center justify-between">
        <label className="text-xs font-medium text-muted-foreground">正文</label>
        <div className="flex items-center gap-2">
          <span className={`text-[10px] px-1.5 py-0.5 rounded font-medium ${formatBadge[format].cls}`}>
            {formatBadge[format].label}
          </span>
          {canPreview && (
            <button
              type="button"
              onClick={() => setShowPreview(v => !v)}
              className={`flex items-center gap-1 text-[11px] px-2 py-0.5 rounded border transition-colors
                ${showPreview
                  ? 'border-primary bg-primary/10 text-primary'
                  : 'border-border text-muted-foreground hover:border-primary/50 hover:text-foreground'
                }`}
            >
              {showPreview ? <Code2 className="w-3 h-3" /> : <Eye className="w-3 h-3" />}
              {showPreview ? '编辑' : '预览'}
            </button>
          )}
        </div>
      </div>

      {showPreview && canPreview ? (
        /* ---- Preview pane ---- */
        <div
          className="w-full min-h-[200px] px-3 py-2 rounded-lg border border-input bg-background text-sm overflow-auto"
          style={{ minHeight: `${rows * 1.6}em` }}
          onClick={() => setShowPreview(false)}
          title="点击返回编辑"
        >
           {format === 'html' ? (
             <div
               className="prose prose-sm dark:prose-invert max-w-none"
               dangerouslySetInnerHTML={{ __html: value }}
             />
           ) : hasNamedCaptureGroups() ? (
             <div className="prose prose-sm dark:prose-invert max-w-none">
               <ReactMarkdown remarkPlugins={[remarkGfm]}>{value}</ReactMarkdown>
             </div>
           ) : (
             <div className="whitespace-pre-wrap break-words font-sans text-muted-foreground">
               {value}
             </div>
           )}
          <p className="text-[10px] text-muted-foreground/50 mt-2 text-right select-none">点击任意处返回编辑</p>
        </div>
      ) : (
        /* ---- Edit textarea ---- */
        <textarea
          ref={textareaRef}
          placeholder={placeholder}
          value={value}
          onChange={handleChange}
          onPaste={handlePaste}
          disabled={disabled}
          rows={rows}
          className="w-full px-3 py-2 rounded-lg border border-input bg-background text-sm
                     placeholder:text-muted-foreground focus:outline-none focus:ring-2
                     focus:ring-ring resize-y disabled:opacity-60 font-mono leading-relaxed"
        />
      )}

      <div className="flex items-center justify-between">
        {format !== 'plain' && !showPreview && (
          <p className="text-[10px] text-muted-foreground">
            检测到 {formatBadge[format].label} 格式，点击「预览」查看渲染效果
          </p>
        )}
        <div className="ml-auto text-[10px] text-muted-foreground">{value.length} 字符</div>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// HTML paste cleaner
// Removes meta / style / script / conditional comments while keeping content
// PRESERVES inline styles and class attributes for full HTML fidelity
// ---------------------------------------------------------------------------
function cleanPastedHTML(raw: string): string {
  // Remove <!--[if ... ]>...<![endif]--> conditional comments (MS Word)
  let out = raw.replace(/<!--\[if[^\]]*\]>[\s\S]*?<!\[endif\]-->/gi, '');
  // Remove regular HTML comments
  out = out.replace(/<!--[\s\S]*?-->/g, '');
  // Remove <meta>, <link>, <style>, <script> blocks (document-level, not inline)
  out = out.replace(/<(meta|link|style|script)\b[^>]*>[\s\S]*?<\/\1>/gi, '');
  out = out.replace(/<(meta|link)\b[^>]*\/?>/gi, '');
  // Remove XML namespace declarations and MS Office tags
  out = out.replace(/<\/?o:[^>]*>/gi, '');
  out = out.replace(/<\/?w:[^>]*>/gi, '');
  out = out.replace(/<\/?m:[^>]*>/gi, '');
  out = out.replace(/\s+xmlns[^=]*="[^"]*"/gi, '');
  // Remove data-* attributes (editor metadata) but KEEP style and class
  out = out.replace(/\s+data-[a-z0-9-]+="[^"]*"/gi, '');
  // Trim
  return out.trim();
}
