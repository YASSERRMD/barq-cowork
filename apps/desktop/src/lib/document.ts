/**
 * document.ts — helpers for generating branded Word (.docx) and PDF files
 * from the barq-cowork UI.
 *
 * Both generation paths go through the Go backend's tool-invocation endpoint
 * (`POST /api/v1/tools/invoke`) which delegates to:
 *   - write_html_docx  →  Pandoc + UAE AI Safety reference.docx
 *   - write_html_pdf   →  headless Chromium PrintToPDF
 */

import { toolsApi, ToolResult } from "./api";

// ─── Request types ────────────────────────────────────────────────────────────

/** Options shared by both DOCX and PDF generation. */
export interface DocumentRequest {
  /** Output filename without extension, e.g. "ai-safety-report". */
  filename: string;
  /** Document title embedded in metadata and cover page. */
  title: string;
  /**
   * HTML body content. May be a fragment or a complete `<body>` payload.
   *
   * Tip: Wrap the first page in `<div class="cover-page">…</div>` to get an
   * automatic page break. Use `.info-box` / `.warning-box` for styled callouts.
   */
  html: string;
  /** Author name shown on the cover page (optional). */
  author?: string;
  /**
   * Custom CSS override. Leave undefined/empty to apply the built-in UAE AI
   * Safety print profile (Crimson Red header, Dark Green footer, shield
   * watermark, Inter/Montserrat typography).
   */
  css?: string;
}

/** Successful response returned by both generation helpers. */
export interface DocumentResult {
  /** Workspace-relative path, e.g. "documents/ai-safety-report.docx". */
  path: string;
  /** File size in bytes. */
  size: number;
}

// ─── Generation helpers ───────────────────────────────────────────────────────

/**
 * Generate a Word document (.docx) from HTML content via the Pandoc pipeline.
 *
 * The Go backend applies the UAE AI Safety Word styles (Crimson Red H1 borders,
 * Dark Green H2, Inter/Montserrat body) through a dynamically built reference.docx.
 *
 * @throws Error when Pandoc is absent or the backend returns an error.
 */
export async function generateDocx(
  workspaceRoot: string,
  req: DocumentRequest
): Promise<DocumentResult> {
  return invokeDocumentTool("write_html_docx", workspaceRoot, req);
}

/**
 * Generate a PDF from HTML content via headless Chromium.
 *
 * The CSS `position:fixed` header/footer bars and the watermark are rendered
 * faithfully on every page because Chrome honours these properties during
 * print-to-PDF.
 *
 * @throws Error when Chrome/Chromium is absent or the backend returns an error.
 */
export async function generatePdf(
  workspaceRoot: string,
  req: DocumentRequest
): Promise<DocumentResult> {
  return invokeDocumentTool("write_html_pdf", workspaceRoot, req);
}

// ─── HTML template helpers ────────────────────────────────────────────────────

/**
 * Build a UAE AI Safety cover page HTML snippet.
 *
 * Wrap it in a full `DocumentRequest.html` together with the body sections
 * before calling `generateDocx` or `generatePdf`.
 *
 * @example
 * ```ts
 * const html =
 *   buildCoverPage({ title: "AI Safety Report", subtitle: "Q2 2025", author: "Alice" }) +
 *   "<h1>Executive Summary</h1><p>…</p>";
 * await generatePdf(wsRoot, { filename: "report", title: "AI Safety Report", html });
 * ```
 */
export function buildCoverPage(opts: {
  title: string;
  subtitle?: string;
  author?: string;
  organization?: string;
  date?: string;
}): string {
  const date =
    opts.date ?? new Date().toLocaleDateString("en-AE", { dateStyle: "long" });
  return `
<div class="cover-page">
  <div class="cover-accent"></div>
  <h1 class="cover-title">${escHtml(opts.title)}</h1>
  ${opts.subtitle ? `<p class="cover-subtitle">${escHtml(opts.subtitle)}</p>` : ""}
  <div class="cover-meta">
    ${opts.author ? `<span><strong>Prepared by:</strong> ${escHtml(opts.author)}</span> &nbsp;|&nbsp; ` : ""}
    ${opts.organization ? `<span>${escHtml(opts.organization)}</span> &nbsp;|&nbsp; ` : ""}
    <span>${escHtml(date)}</span>
  </div>
</div>
`.trim();
}

/**
 * Convert a plain Markdown-ish string to a minimal HTML fragment suitable for
 * `DocumentRequest.html`. Supports:
 *   - `# Heading 1` → `<h1>`
 *   - `## Heading 2` → `<h2>`
 *   - `### Heading 3` → `<h3>`
 *   - `- item` / `* item` → `<ul><li>`
 *   - `1. item` → `<ol><li>`
 *   - `**bold**` → `<strong>`
 *   - `_italic_` → `<em>`
 *   - `` `code` `` → `<code>`
 *   - `---` → `<hr>`
 *   - blank line → paragraph break
 *
 * For production use with complex Markdown, pass pre-rendered HTML directly.
 */
export function markdownToHtml(markdown: string): string {
  const lines = markdown.replace(/\r\n/g, "\n").split("\n");
  const out: string[] = [];
  let inUl = false;
  let inOl = false;
  let inParagraph = false;

  const closeList = () => {
    if (inUl) { out.push("</ul>"); inUl = false; }
    if (inOl) { out.push("</ol>"); inOl = false; }
  };
  const closeParagraph = () => {
    if (inParagraph) { out.push("</p>"); inParagraph = false; }
  };

  for (const raw of lines) {
    const line = raw.trimEnd();

    // Heading
    const hm = line.match(/^(#{1,6})\s+(.*)/);
    if (hm) {
      closeList(); closeParagraph();
      const lvl = hm[1].length;
      out.push(`<h${lvl}>${inlineMarkdown(hm[2])}</h${lvl}>`);
      continue;
    }

    // Horizontal rule
    if (/^---+$/.test(line)) {
      closeList(); closeParagraph();
      out.push("<hr/>");
      continue;
    }

    // Unordered list item
    const ulm = line.match(/^[-*]\s+(.*)/);
    if (ulm) {
      closeParagraph();
      if (!inUl) { if (inOl) { out.push("</ol>"); inOl = false; } out.push("<ul>"); inUl = true; }
      out.push(`<li>${inlineMarkdown(ulm[1])}</li>`);
      continue;
    }

    // Ordered list item
    const olm = line.match(/^\d+\.\s+(.*)/);
    if (olm) {
      closeParagraph();
      if (!inOl) { if (inUl) { out.push("</ul>"); inUl = false; } out.push("<ol>"); inOl = true; }
      out.push(`<li>${inlineMarkdown(olm[1])}</li>`);
      continue;
    }

    // Blank line
    if (line === "") {
      closeList(); closeParagraph();
      continue;
    }

    // Regular paragraph line
    closeList();
    if (!inParagraph) { out.push("<p>"); inParagraph = true; }
    else { out.push(" "); }
    out.push(inlineMarkdown(line));
  }

  closeList();
  closeParagraph();
  return out.join("\n");
}

// ─── Internal utilities ───────────────────────────────────────────────────────

async function invokeDocumentTool(
  toolName: "write_html_docx" | "write_html_pdf",
  workspaceRoot: string,
  req: DocumentRequest
): Promise<DocumentResult> {
  if (!workspaceRoot) throw new Error("workspaceRoot is required");
  if (!req.filename)  throw new Error("filename is required");
  if (!req.title)     throw new Error("title is required");
  if (!req.html)      throw new Error("html is required");

  const result: ToolResult = await toolsApi.invoke({
    workspace_root: workspaceRoot,
    tool_name: toolName,
    args_json: JSON.stringify({
      filename: req.filename,
      title:    req.title,
      author:   req.author ?? "",
      html:     req.html,
      css:      req.css ?? "",
    }),
  });

  if (result.status !== "ok") {
    throw new Error(result.error ?? result.content ?? "document generation failed");
  }

  const data = result.data as { path: string; size: number } | undefined;
  if (!data?.path) {
    throw new Error("backend returned no file path");
  }

  return { path: data.path, size: data.size };
}

/** Minimal HTML entity escaping for user-supplied strings in templates. */
function escHtml(s: string): string {
  return s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&#34;");
}

/** Apply inline Markdown transforms to a single line of text. */
function inlineMarkdown(s: string): string {
  return escHtml(s)
    .replace(/\*\*(.+?)\*\*/g, "<strong>$1</strong>")
    .replace(/__(.+?)__/g,     "<strong>$1</strong>")
    .replace(/\*(.+?)\*/g,     "<em>$1</em>")
    .replace(/_(.+?)_/g,       "<em>$1</em>")
    .replace(/`(.+?)`/g,       "<code>$1</code>");
}
