import fs from "node:fs";
import path from "node:path";

import PptxGenJS from "pptxgenjs";

type PptxSlide = any;
type TextOptions = Record<string, unknown>;
type TableRow = any[];

type Manifest = {
  version: number;
  title: string;
  subtitle?: string;
  theme: string;
  palette: Palette;
  deck_plan: DeckPlan;
  narrative?: string;
  layout_mix?: string[];
  slides: SlidePlan[];
};

type Palette = {
  background: string;
  card: string;
  accent: string;
  accent2: string;
  text: string;
  muted: string;
  border: string;
};

type DeckPlan = {
  subject: string;
  audience: string;
  narrative_arc: string;
  visual_direction: string;
  dominant_need: string;
  cover_style?: string;
  color_story?: string;
  motif?: string;
  kicker?: string;
  design?: DeckDesign;
  layout_mix?: string[];
};

type DeckDesign = {
  composition?: string;
  density?: string;
  shape_language?: string;
  accent_mode?: string;
  hero_layout?: string;
};

type SlideDesign = {
  layout_style?: string;
  panel_style?: string;
  accent_mode?: string;
  density?: string;
  visual_focus?: string;
};

type SlidePlan = {
  number: number;
  heading: string;
  layout: string;
  variant: number;
  purpose?: string;
  visual?: string;
  content_source?: string;
  design?: SlideDesign;
  speaker_notes?: string;
  points?: string[];
  stats?: Stat[];
  steps?: string[];
  cards?: Card[];
  chart_type?: string;
  chart_categories?: string[];
  chart_series?: ChartSeries[];
  y_label?: string;
  timeline?: TimelineItem[];
  left_column?: CompareColumn;
  right_column?: CompareColumn;
  table?: TableData;
};

type Stat = {
  value: string;
  label: string;
  desc?: string;
};

type Card = {
  icon: string;
  title: string;
  desc?: string;
};

type ChartSeries = {
  name: string;
  values: number[];
  color?: string;
};

type TimelineItem = {
  date: string;
  title: string;
  desc?: string;
};

type CompareColumn = {
  heading: string;
  points: string[];
};

type TableData = {
  headers: string[];
  rows: string[][];
};

type RenderFamily = "proposal" | "studio" | "playful";

type RenderPalette = {
  bg: string;
  card: string;
  accent: string;
  accent2: string;
  text: string;
  muted: string;
  border: string;
  canvas: string;
  header: string;
  footer: string;
  darkMuted: string;
  lightMuted: string;
};

type Bounds = {
  x: number;
  y: number;
  w: number;
  h: number;
};

const SLIDE_W = 13.333;
const SLIDE_H = 7.5;
const FONT_HEAD = "Aptos Display";
const FONT_BODY = "Aptos";

const legacyEmojiIcons: Record<string, string> = {
  "⚡": "automation",
  "🔒": "shield",
  "📊": "chart",
  "📈": "growth",
  "🧩": "integration",
  "🧠": "strategy",
  "🌐": "integration",
  "👥": "people",
  "🧭": "strategy",
  "📚": "learning",
  "❤️": "health",
  "🌿": "leaf",
};

function normalizeText(value: string): string {
  return ` ${value.toLowerCase().replace(/[^a-z0-9]+/g, " ").trim()} `;
}

function containsAny(value: string, ...terms: string[]): boolean {
  const text = normalizeText(value);
  return terms.some((term) => text.includes(` ${normalizeText(term).trim()} `));
}

function normalizeHex(value: string, fallback = "000000"): string {
  const cleaned = value.trim().replace(/^#/, "").replace(/[^0-9a-fA-F]/g, "").toUpperCase();
  if (cleaned.length === 3) {
    return cleaned.split("").map((ch) => ch + ch).join("");
  }
  if (cleaned.length >= 6) {
    return cleaned.slice(0, 6);
  }
  return fallback;
}

function cssHex(value: string): string {
  return `#${normalizeHex(value)}`;
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value));
}

function mixHex(base: string, overlay: string, overlayWeight: number): string {
  const weight = clamp(overlayWeight, 0, 1);
  const a = normalizeHex(base);
  const b = normalizeHex(overlay);
  const channels = [0, 2, 4].map((offset) => {
    const av = parseInt(a.slice(offset, offset + 2), 16);
    const bv = parseInt(b.slice(offset, offset + 2), 16);
    const mixed = Math.round(av * (1 - weight) + bv * weight);
    return mixed.toString(16).padStart(2, "0");
  });
  return channels.join("").toUpperCase();
}

function proposalCanvasFill(pal: Palette): string {
  return mixHex(pal.background, "F0F4F8", 0.76);
}

function proposalHeaderFill(pal: Palette): string {
  return mixHex("0D1B2A", pal.text, 0.32);
}

function proposalFooterFill(pal: Palette): string {
  return mixHex(proposalHeaderFill(pal), pal.card, 0.12);
}

function proposalMutedOnDark(pal: Palette): string {
  return mixHex(proposalHeaderFill(pal), "FFFFFF", 0.66);
}

function proposalMutedOnLight(pal: Palette): string {
  return mixHex(proposalCanvasFill(pal), pal.text, 0.54);
}

function accentColor(pal: RenderPalette, index: number): string {
  switch (index % 4) {
    case 1:
      return pal.accent2;
    case 2:
      return mixHex(pal.text, pal.border, 0.34);
    case 3:
      return mixHex(pal.accent, pal.accent2, 0.5);
    default:
      return pal.accent;
  }
}

function inferIconToken(raw: string): string {
  const trimmed = raw.trim();
  if (!trimmed) return "";
  if (legacyEmojiIcons[trimmed]) {
    return legacyEmojiIcons[trimmed];
  }
  const text = normalizeText(trimmed);
  if (containsAny(text, "shield", "lock", "security", "secure", "control", "governance", "privacy", "risk")) return "shield";
  if (containsAny(text, "chart", "graph", "analytics", "data", "insight", "metric", "dashboard")) return "chart";
  if (containsAny(text, "growth", "revenue", "sales", "finance", "market", "profit")) return "growth";
  if (containsAny(text, "automation", "workflow", "process", "ops", "operations", "speed", "efficiency", "flow")) return "automation";
  if (containsAny(text, "integration", "connect", "platform", "system", "api", "network")) return "integration";
  if (containsAny(text, "people", "team", "customer", "user", "community", "parent", "educator")) return "people";
  if (containsAny(text, "strategy", "roadmap", "plan", "planning", "direction", "goal")) return "strategy";
  if (containsAny(text, "learning", "education", "school", "student", "teacher", "classroom", "kid", "kids", "children")) return "learning";
  if (containsAny(text, "health", "medical", "patient", "clinical", "care")) return "health";
  if (containsAny(text, "environment", "climate", "green", "sustainability", "carbon", "nature")) return "leaf";
  if (containsAny(text, "logistics", "delivery", "fleet", "shipping", "warehouse", "transport", "supply")) return "logistics";
  if (containsAny(text, "creative", "design", "idea", "brand", "innovation", "spark")) return "spark";
  return "";
}

function iconSvg(token: string, palette: { accent: string; accent2: string; text: string }): string {
  const fg = cssHex(palette.text);
  const accent = cssHex(palette.accent);
  const soft = cssHex(palette.accent2);
  const normalized = inferIconToken(token) || "spark";
  let glyph = "";
  switch (normalized) {
    case "shield":
      glyph = `<polygon points="32,10 48,18 48,34 32,50 16,34 16,18" fill="none" stroke="${fg}" stroke-width="4"/><path d="M24 30h16M32 22v16" stroke="${fg}" stroke-width="4" stroke-linecap="round"/>`;
      break;
    case "chart":
      glyph = `<rect x="16" y="34" width="8" height="14" rx="2" fill="${fg}"/><rect x="28" y="24" width="8" height="24" rx="2" fill="${fg}"/><rect x="40" y="18" width="8" height="30" rx="2" fill="${fg}"/><path d="M14 48h36" stroke="${fg}" stroke-width="3" stroke-linecap="round"/>`;
      break;
    case "growth":
      glyph = `<rect x="16" y="38" width="8" height="10" rx="2" fill="${fg}"/><rect x="28" y="30" width="8" height="18" rx="2" fill="${fg}"/><rect x="40" y="22" width="8" height="26" rx="2" fill="${fg}"/><path d="M23 26l9-7 8 5 9-7" stroke="${fg}" stroke-width="3.5" stroke-linecap="round" stroke-linejoin="round" fill="none"/>`;
      break;
    case "automation":
      glyph = `<path d="M14 30h18l-6-6m6 6-6 6" stroke="${fg}" stroke-width="4" stroke-linecap="round" stroke-linejoin="round" fill="none"/><path d="M30 30h18l-6-6m6 6-6 6" stroke="${fg}" stroke-width="4" stroke-linecap="round" stroke-linejoin="round" fill="none"/><circle cx="49" cy="30" r="5" fill="${soft}" stroke="${fg}" stroke-width="2"/>`;
      break;
    case "integration":
      glyph = `<circle cx="24" cy="32" r="10" fill="none" stroke="${fg}" stroke-width="4"/><circle cx="40" cy="32" r="10" fill="none" stroke="${fg}" stroke-width="4"/><path d="M28 32h8" stroke="${fg}" stroke-width="4" stroke-linecap="round"/>`;
      break;
    case "people":
      glyph = `<circle cx="25" cy="22" r="6" fill="${fg}"/><circle cx="39" cy="22" r="6" fill="${fg}"/><rect x="18" y="31" width="14" height="15" rx="5" fill="${fg}"/><rect x="32" y="31" width="14" height="15" rx="5" fill="${fg}"/>`;
      break;
    case "learning":
      glyph = `<rect x="15" y="17" width="15" height="28" rx="3" fill="none" stroke="${fg}" stroke-width="3.5"/><rect x="34" y="17" width="15" height="28" rx="3" fill="none" stroke="${fg}" stroke-width="3.5"/><path d="M32 18v28" stroke="${fg}" stroke-width="3"/>`;
      break;
    case "health":
      glyph = `<path d="M32 16v32M16 32h32" stroke="${fg}" stroke-width="6" stroke-linecap="round"/>`;
      break;
    case "leaf":
      glyph = `<path d="M22 38c0-11 8-18 18-20 1 10-3 20-14 23-3 1-4-1-4-3Z" fill="${fg}"/><path d="M25 40c6-8 12-13 20-17" stroke="${accent}" stroke-width="3" stroke-linecap="round"/>`;
      break;
    case "logistics":
      glyph = `<rect x="16" y="26" width="24" height="18" rx="3" fill="none" stroke="${fg}" stroke-width="3.5"/><path d="M28 26v18" stroke="${fg}" stroke-width="3"/><path d="M40 22h10l-4-4m4 4-4 4" stroke="${fg}" stroke-width="3.5" stroke-linecap="round" stroke-linejoin="round" fill="none"/>`;
      break;
    default:
      glyph = `<path d="M32 14l5 9 9 5-9 5-5 9-5-9-9-5 9-5 5-9Z" fill="${fg}"/><path d="M15 46l3 5 5 3-5 3-3 5-3-5-5-3 5-3 3-5Z" fill="${soft}"/>`;
      break;
  }
  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64"><circle cx="32" cy="32" r="28" fill="${accent}"/>${glyph}</svg>`;
}

function svgDataUri(svg: string): string {
  return `data:image/svg+xml;base64,${Buffer.from(svg).toString("base64")}`;
}

function previewFamily(manifest: Manifest): RenderFamily {
  const text = [
    manifest.title,
    manifest.subtitle ?? "",
    manifest.theme,
    manifest.deck_plan.subject,
    manifest.deck_plan.audience,
    manifest.deck_plan.narrative_arc,
    manifest.deck_plan.visual_direction,
    manifest.deck_plan.color_story ?? "",
    manifest.deck_plan.dominant_need,
    manifest.deck_plan.cover_style ?? "",
  ].join(" ");
  if (containsAny(text, "playful", "storybook", "collage", "cartoon", "fun") &&
      !containsAny(text, "refined", "structured", "proposal", "report", "executive", "premium")) {
    return "playful";
  }
  if (containsAny(text, "poster", "campaign", "showcase", "gallery", "bold studio") &&
      !containsAny(text, "proposal", "report", "summary", "brief", "guide")) {
    return "studio";
  }
  return "proposal";
}

function buildPalette(manifest: Manifest, family: RenderFamily): RenderPalette {
  const base: RenderPalette = {
    bg: normalizeHex(manifest.palette.background, "F7F8FC"),
    card: normalizeHex(manifest.palette.card, "FFFFFF"),
    accent: normalizeHex(manifest.palette.accent, "0EA5E9"),
    accent2: normalizeHex(manifest.palette.accent2, "67E8F9"),
    text: normalizeHex(manifest.palette.text, "0F172A"),
    muted: normalizeHex(manifest.palette.muted, "64748B"),
    border: normalizeHex(manifest.palette.border, "D6EAF4"),
    canvas: proposalCanvasFill(manifest.palette),
    header: proposalHeaderFill(manifest.palette),
    footer: proposalFooterFill(manifest.palette),
    darkMuted: proposalMutedOnDark(manifest.palette),
    lightMuted: proposalMutedOnLight(manifest.palette),
  };
  switch (family) {
    case "studio":
      return {
        ...base,
        canvas: mixHex(base.bg, base.card, 0.9),
        header: mixHex(base.text, "0B1220", 0.6),
        footer: mixHex(base.text, base.card, 0.12),
        darkMuted: mixHex(base.text, "FFFFFF", 0.62),
        lightMuted: mixHex(base.bg, base.text, 0.56),
      };
    case "playful":
      return {
        ...base,
        canvas: mixHex(base.bg, "FFFFFF", 0.86),
        header: mixHex(base.accent, base.text, 0.56),
        footer: mixHex(base.accent2, base.bg, 0.54),
        darkMuted: mixHex(base.header, "FFFFFF", 0.7),
        lightMuted: mixHex(base.bg, base.text, 0.5),
      };
    default:
      return base;
  }
}

function addFullRect(slide: PptxSlide, fill: string): void {
  slide.addShape("rect", {
    x: 0,
    y: 0,
    w: SLIDE_W,
    h: SLIDE_H,
    line: { color: fill, transparency: 100 },
    fill: { color: fill },
  });
}

function addPanel(slide: PptxSlide, bounds: Bounds, fill: string, border: string, radius = 0): void {
  slide.addShape(radius >= 0.18 ? "roundRect" : "rect", {
    ...bounds,
    rectRadius: radius,
    line: { color: border, pt: 1 },
    fill: { color: fill },
  });
}

function addLine(slide: PptxSlide, x: number, y: number, w: number, h: number, color: string, pt = 1): void {
  slide.addShape("line", {
    x,
    y,
    w,
    h,
    line: { color, pt },
  });
}

function addText(slide: PptxSlide, text: string, bounds: Bounds, options: TextOptions = {}): void {
  slide.addText(text, {
    ...bounds,
    fontFace: FONT_BODY,
    fontSize: 12,
    color: "0F172A",
    margin: 0,
    fit: "shrink",
    valign: "mid",
    breakLine: false,
    ...options,
  });
}

function addMiniRect(slide: PptxSlide, x: number, y: number, w: number, h: number, fill: string): void {
  slide.addShape("rect", {
    x,
    y,
    w,
    h,
    line: { color: fill, transparency: 100 },
    fill: { color: fill },
  });
}

function addMiniLine(slide: PptxSlide, x: number, y: number, w: number, h: number, color: string, pt = 1): void {
  slide.addShape("line", {
    x,
    y,
    w,
    h,
    line: { color, pt },
  });
}

function addIcon(slide: PptxSlide, token: string, x: number, y: number, size: number, pal: RenderPalette): void {
  const accent = pal.accent;
  const soft = pal.accent2;
  const fg = "FFFFFF";
  const normalized = inferIconToken(token) || "spark";

  slide.addShape("ellipse", {
    x,
    y,
    w: size,
    h: size,
    line: { color: accent, transparency: 100 },
    fill: { color: accent },
  });

  const left = x + size * 0.2;
  const top = y + size * 0.2;
  const inner = size * 0.6;

  switch (normalized) {
    case "shield":
      slide.addShape("hexagon", {
        x: x + size * 0.24,
        y: y + size * 0.17,
        w: size * 0.52,
        h: size * 0.6,
        line: { color: fg, pt: 1.2 },
        fill: { color: accent, transparency: 100 },
      });
      addMiniRect(slide, x + size * 0.47, y + size * 0.29, size * 0.06, size * 0.26, fg);
      addMiniRect(slide, x + size * 0.37, y + size * 0.39, size * 0.26, size * 0.06, fg);
      break;
    case "chart":
      addMiniRect(slide, x + size * 0.28, y + size * 0.52, size * 0.08, size * 0.18, fg);
      addMiniRect(slide, x + size * 0.44, y + size * 0.4, size * 0.08, size * 0.3, fg);
      addMiniRect(slide, x + size * 0.6, y + size * 0.28, size * 0.08, size * 0.42, fg);
      addMiniRect(slide, x + size * 0.26, y + size * 0.72, size * 0.48, size * 0.03, fg);
      break;
    case "growth":
      addMiniRect(slide, x + size * 0.28, y + size * 0.56, size * 0.08, size * 0.14, fg);
      addMiniRect(slide, x + size * 0.44, y + size * 0.45, size * 0.08, size * 0.25, fg);
      addMiniRect(slide, x + size * 0.6, y + size * 0.34, size * 0.08, size * 0.36, fg);
      addMiniLine(slide, x + size * 0.29, y + size * 0.44, size * 0.18, -size * 0.1, fg, 1.2);
      addMiniLine(slide, x + size * 0.47, y + size * 0.34, size * 0.18, -size * 0.1, fg, 1.2);
      addMiniLine(slide, x + size * 0.61, y + size * 0.24, size * 0.08, size * 0.06, fg, 1.2);
      addMiniLine(slide, x + size * 0.61, y + size * 0.24, -size * 0.02, size * 0.1, fg, 1.2);
      break;
    case "automation":
      slide.addShape("rightArrow", {
        x: x + size * 0.18,
        y: y + size * 0.34,
        w: size * 0.3,
        h: size * 0.16,
        line: { color: fg, transparency: 100 },
        fill: { color: fg },
      });
      slide.addShape("rightArrow", {
        x: x + size * 0.42,
        y: y + size * 0.34,
        w: size * 0.26,
        h: size * 0.16,
        line: { color: fg, transparency: 100 },
        fill: { color: fg },
      });
      slide.addShape("ellipse", {
        x: x + size * 0.66,
        y: y + size * 0.3,
        w: size * 0.16,
        h: size * 0.16,
        line: { color: fg, pt: 1 },
        fill: { color: soft },
      });
      break;
    case "integration":
      slide.addShape("ellipse", {
        x: x + size * 0.22,
        y: y + size * 0.3,
        w: size * 0.22,
        h: size * 0.22,
        line: { color: fg, pt: 1.2 },
        fill: { color: accent, transparency: 100 },
      });
      slide.addShape("ellipse", {
        x: x + size * 0.46,
        y: y + size * 0.3,
        w: size * 0.22,
        h: size * 0.22,
        line: { color: fg, pt: 1.2 },
        fill: { color: accent, transparency: 100 },
      });
      addMiniRect(slide, x + size * 0.39, y + size * 0.4, size * 0.12, size * 0.03, fg);
      break;
    case "people":
      slide.addShape("ellipse", {
        x: x + size * 0.25,
        y: y + size * 0.23,
        w: size * 0.15,
        h: size * 0.15,
        line: { color: fg, transparency: 100 },
        fill: { color: fg },
      });
      slide.addShape("ellipse", {
        x: x + size * 0.48,
        y: y + size * 0.23,
        w: size * 0.15,
        h: size * 0.15,
        line: { color: fg, transparency: 100 },
        fill: { color: fg },
      });
      slide.addShape("roundRect", {
        x: x + size * 0.21,
        y: y + size * 0.44,
        w: size * 0.22,
        h: size * 0.18,
        rectRadius: 0.2,
        line: { color: fg, transparency: 100 },
        fill: { color: fg },
      });
      slide.addShape("roundRect", {
        x: x + size * 0.44,
        y: y + size * 0.44,
        w: size * 0.22,
        h: size * 0.18,
        rectRadius: 0.2,
        line: { color: fg, transparency: 100 },
        fill: { color: fg },
      });
      break;
    case "learning":
      slide.addShape("roundRect", {
        x: x + size * 0.22,
        y: y + size * 0.25,
        w: size * 0.18,
        h: size * 0.38,
        rectRadius: 0.08,
        line: { color: fg, pt: 1.1 },
        fill: { color: accent, transparency: 100 },
      });
      slide.addShape("roundRect", {
        x: x + size * 0.43,
        y: y + size * 0.25,
        w: size * 0.18,
        h: size * 0.38,
        rectRadius: 0.08,
        line: { color: fg, pt: 1.1 },
        fill: { color: accent, transparency: 100 },
      });
      addMiniRect(slide, x + size * 0.4, y + size * 0.24, size * 0.02, size * 0.41, fg);
      break;
    case "health":
      addMiniRect(slide, x + size * 0.46, y + size * 0.22, size * 0.08, size * 0.46, fg);
      addMiniRect(slide, x + size * 0.27, y + size * 0.41, size * 0.46, size * 0.08, fg);
      break;
    case "leaf":
      slide.addShape("ellipse", {
        x: x + size * 0.27,
        y: y + size * 0.22,
        w: size * 0.2,
        h: size * 0.3,
        rotate: -32,
        line: { color: fg, transparency: 100 },
        fill: { color: fg },
      });
      slide.addShape("ellipse", {
        x: x + size * 0.42,
        y: y + size * 0.26,
        w: size * 0.2,
        h: size * 0.28,
        rotate: 28,
        line: { color: fg, transparency: 100 },
        fill: { color: fg },
      });
      addMiniLine(slide, x + size * 0.5, y + size * 0.44, 0, size * 0.18, soft, 1.2);
      break;
    case "logistics":
      slide.addShape("roundRect", {
        x: x + size * 0.24,
        y: y + size * 0.34,
        w: size * 0.32,
        h: size * 0.22,
        rectRadius: 0.08,
        line: { color: fg, pt: 1.1 },
        fill: { color: accent, transparency: 100 },
      });
      addMiniRect(slide, x + size * 0.39, y + size * 0.34, size * 0.02, size * 0.22, fg);
      slide.addShape("rightArrow", {
        x: x + size * 0.58,
        y: y + size * 0.28,
        w: size * 0.16,
        h: size * 0.14,
        line: { color: fg, transparency: 100 },
        fill: { color: fg },
      });
      break;
    case "strategy":
      slide.addShape("diamond", {
        x: x + size * 0.39,
        y: y + size * 0.2,
        w: size * 0.18,
        h: size * 0.18,
        line: { color: fg, transparency: 100 },
        fill: { color: fg },
      });
      addMiniLine(slide, x + size * 0.48, y + size * 0.38, 0, size * 0.2, fg, 1.2);
      addMiniLine(slide, x + size * 0.48, y + size * 0.58, -size * 0.12, size * 0.1, fg, 1.2);
      addMiniLine(slide, x + size * 0.48, y + size * 0.58, size * 0.12, size * 0.1, fg, 1.2);
      break;
    default:
      slide.addShape("diamond", {
        x: x + size * 0.37,
        y: y + size * 0.21,
        w: size * 0.2,
        h: size * 0.2,
        line: { color: fg, transparency: 100 },
        fill: { color: fg },
      });
      slide.addShape("diamond", {
        x: x + size * 0.22,
        y: y + size * 0.46,
        w: size * 0.12,
        h: size * 0.12,
        line: { color: soft, transparency: 100 },
        fill: { color: soft },
      });
      slide.addShape("diamond", {
        x: x + size * 0.56,
        y: y + size * 0.5,
        w: size * 0.12,
        h: size * 0.12,
        line: { color: soft, transparency: 100 },
        fill: { color: soft },
      });
      break;
  }
}

function shortTitle(title: string): string {
  return title.trim().length <= 30 ? title.trim().toUpperCase() : title.trim();
}

function splitCardText(value: string): { title: string; desc: string } {
  const clean = value.trim().replace(/^[-*•\s]+/, "");
  const separators = [" - ", ": ", " — ", " – "];
  for (const separator of separators) {
    if (clean.includes(separator)) {
      const [title, ...rest] = clean.split(separator);
      return { title: title.trim(), desc: rest.join(separator).trim() };
    }
  }
  const words = clean.split(/\s+/);
  if (words.length > 8) {
    return { title: words.slice(0, 4).join(" "), desc: words.slice(4).join(" ") };
  }
  return { title: clean, desc: "" };
}

function slideIconToken(slide: SlidePlan): string {
  switch (slide.layout) {
    case "stats":
    case "chart":
      return "chart";
    case "steps":
    case "timeline":
      return "strategy";
    case "compare":
      return "integration";
    case "table":
      return "shield";
    case "cards":
      return inferIconToken(slide.cards?.[0]?.icon ?? slide.cards?.[0]?.title ?? "") || "spark";
    default:
      return inferIconToken(`${slide.heading} ${slide.layout}`) || "spark";
  }
}

function coverToken(manifest: Manifest): string {
  return inferIconToken(manifest.deck_plan.motif ?? manifest.deck_plan.subject ?? manifest.title) || "spark";
}

function buildLead(slide: SlidePlan): string {
  const note = slide.speaker_notes?.trim();
  if (note) {
    const sentence = note.split(/(?<=[.!?])\s+/)[0];
    if (sentence.trim()) return sentence.trim();
  }
  const points = slide.points ?? [];
  if (points.length > 0) {
    return splitCardText(points[0]).title;
  }
  const stats = slide.stats ?? [];
  if (stats.length > 0) {
    return `${stats[0].value} reflects ${stats[0].label.toLowerCase()}.`;
  }
  return slide.purpose?.trim() || "";
}

function proposalStatsTakeaway(stats: Stat[]): string {
  if (stats.length === 0) return "Key indicators that frame the decision.";
  const labels = stats
    .map((stat) => stat.label.trim())
    .filter(Boolean)
    .slice(0, 3);
  if (labels.length === 0) return "Key indicators that frame the decision.";
  if (labels.length === 1) return `Primary signal: ${labels[0]}.`;
  if (labels.length === 2) return `Decision signals: ${labels[0]} and ${labels[1]}.`;
  return `Decision signals: ${labels[0]}, ${labels[1]}, and ${labels[2]}.`;
}

function proposalLeadText(plan: SlidePlan, points: string[]): string {
  const notes = plan.speaker_notes?.trim();
  if (notes) return trimSentence(notes, 160);
  for (const point of points) {
    if (point.trim().length >= 72) {
      return trimSentence(point, 160);
    }
  }
  if (points.length >= 2) {
    const left = splitCardText(points[0]).title;
    const right = splitCardText(points[1]).title;
    return `Focus areas include ${left} and ${right}.`;
  }
  return trimSentence(plan.purpose?.trim() || buildLead(plan), 160);
}

function proposalSectionLabel(plan: SlidePlan): string {
  const text = normalizeText(plan.heading || "");
  if (containsAny(text, "deliverable", "scope", "phase", "build", "implementation")) return "KEY DELIVERABLES";
  if (containsAny(text, "assumption", "constraint", "dependency")) return "KEY ASSUMPTIONS";
  if (containsAny(text, "risk", "issue")) return "KEY RISKS";
  if (containsAny(text, "team", "role")) return "TEAM FOCUS";
  return "KEY POINTS";
}

function coverWordmark(title: string): string {
  const words = title.trim().split(/\s+/).filter(Boolean);
  if (words.length === 0) return "";
  const first = words[0].replace(/[^A-Za-z0-9]/g, "");
  if (!first) return "";
  if (first === first.toUpperCase() && first.length <= 8) return first;
  if (first.length <= 4) return first.toUpperCase();
  return "";
}

function sectionKicker(plan: SlidePlan): string {
  return (plan.purpose?.trim() || plan.layout || "summary").toUpperCase();
}

function trimSentence(value: string, limit = 120): string {
  const clean = value.trim();
  if (clean.length <= limit) return clean;
  const clipped = clean.slice(0, limit).trim();
  const parts = clipped.split(/[.!?]/);
  if (parts[0]?.trim()) return `${parts[0].trim()}.`;
  return `${clipped}…`;
}

function addMetaPill(slide: PptxSlide, text: string, x: number, y: number, w: number, pal: RenderPalette): void {
  addPanel(slide, { x, y, w, h: 0.34 }, mixHex(pal.header, pal.card, 0.08), mixHex(pal.header, pal.card, 0.14), 0.08);
  addText(slide, text, { x: x + 0.12, y: y + 0.08, w: w - 0.24, h: 0.16 }, {
    fontFace: FONT_BODY,
    fontSize: 9,
    color: pal.darkMuted,
    bold: true,
    align: "center",
    fit: "shrink",
  });
}

function addCoverSideCard(slide: PptxSlide, label: string, title: string, body: string, bounds: Bounds, token: string, pal: RenderPalette): void {
  addPanel(slide, bounds, mixHex(pal.header, pal.card, 0.06), mixHex(pal.header, pal.card, 0.12), 0.08);
  addIcon(slide, token, bounds.x + 0.18, bounds.y + 0.18, 0.34, {
    ...pal,
    accent: accentColor(pal, label.length),
    accent2: mixHex(accentColor(pal, label.length), pal.card, 0.44),
    text: "FFFFFF",
  });
  addText(slide, label.toUpperCase(), { x: bounds.x + 0.62, y: bounds.y + 0.16, w: bounds.w - 0.82, h: 0.16 }, {
    fontFace: FONT_BODY,
    fontSize: 8.5,
    color: pal.darkMuted,
    charSpace: 0.8,
    bold: true,
  });
  addText(slide, title, { x: bounds.x + 0.18, y: bounds.y + 0.62, w: bounds.w - 0.36, h: 0.34 }, {
    fontFace: FONT_HEAD,
    fontSize: 13,
    bold: true,
    color: "FFFFFF",
    breakLine: true,
    valign: "top",
  });
  if (body.trim()) {
    addText(slide, trimSentence(body, 72), { x: bounds.x + 0.18, y: bounds.y + 1.02, w: bounds.w - 0.36, h: bounds.h - 1.16 }, {
      fontFace: FONT_BODY,
      fontSize: 10,
      color: pal.darkMuted,
      breakLine: true,
      valign: "top",
      fit: "shrink",
    });
  }
}

function addLeadPanel(slide: PptxSlide, plan: SlidePlan, bounds: Bounds, pal: RenderPalette, token?: string): void {
  addPanel(slide, bounds, mixHex(pal.canvas, pal.card, 0.82), mixHex(pal.canvas, pal.border, 0.84), 0.08);
  if (token) {
    addIcon(slide, token, bounds.x + 0.18, bounds.y + 0.18, 0.42, {
      ...pal,
      accent: accentColor(pal, 0),
      accent2: mixHex(accentColor(pal, 0), pal.card, 0.42),
      text: "FFFFFF",
    });
  }
  addText(slide, sectionKicker(plan), { x: bounds.x + (token ? 0.68 : 0.18), y: bounds.y + 0.18, w: bounds.w - (token ? 0.86 : 0.36), h: 0.18 }, {
    fontFace: FONT_BODY,
    fontSize: 8.6,
    bold: true,
    color: pal.lightMuted,
    charSpace: 0.8,
  });
  addText(slide, buildLead(plan), { x: bounds.x + 0.18, y: bounds.y + 0.56, w: bounds.w - 0.36, h: bounds.h - 0.74 }, {
    fontFace: FONT_HEAD,
    fontSize: 16,
    bold: true,
    color: pal.text,
    breakLine: true,
    valign: "top",
    fit: "shrink",
  });
}

function addTopSummaryStrip(slide: PptxSlide, plan: SlidePlan, bounds: Bounds, pal: RenderPalette, token?: string): number {
  const stripH = 0.92;
  addPanel(slide, { x: bounds.x, y: bounds.y, w: bounds.w, h: stripH }, pal.card, mixHex(pal.canvas, pal.border, 0.84), 0.06);
  if (token) {
    addIcon(slide, token, bounds.x + 0.18, bounds.y + 0.18, 0.38, {
      ...pal,
      accent: accentColor(pal, 0),
      accent2: mixHex(accentColor(pal, 0), pal.card, 0.42),
      text: "FFFFFF",
    });
  }
  addText(slide, sectionKicker(plan), { x: bounds.x + (token ? 0.7 : 0.18), y: bounds.y + 0.18, w: 2.8, h: 0.16 }, {
    fontFace: FONT_BODY,
    fontSize: 8.4,
    bold: true,
    color: pal.lightMuted,
    charSpace: 0.8,
  });
  addText(slide, buildLead(plan), { x: bounds.x + (token ? 0.7 : 0.18), y: bounds.y + 0.44, w: bounds.w - (token ? 0.88 : 0.36), h: 0.28 }, {
    fontFace: FONT_HEAD,
    fontSize: 15.5,
    bold: true,
    color: pal.text,
    fit: "shrink",
  });
  return bounds.y + stripH + 0.2;
}

function renderCover(slide: PptxSlide, manifest: Manifest, family: RenderFamily, pal: RenderPalette): void {
  addFullRect(slide, family === "playful" ? mixHex(pal.bg, pal.card, 0.72) : pal.header);
  addPanel(slide, { x: 0, y: 0, w: SLIDE_W, h: 0.18 }, pal.accent, pal.accent, 0);

  const title = manifest.title.trim() || "Presentation";
  const subtitle = manifest.subtitle?.trim() ?? "";
  const kicker = manifest.deck_plan.kicker?.trim() || "LLM-directed presentation";
  const support = manifest.deck_plan.audience?.trim() ? `For ${manifest.deck_plan.audience.trim()}` : "";
  const meta = [manifest.deck_plan.subject.trim(), manifest.deck_plan.color_story?.trim() ?? ""].filter(Boolean).join("  |  ");
  const motif = coverToken(manifest);
  const wordmark = coverWordmark(title);
  const titleRemainder = wordmark
    ? title.replace(new RegExp(`^${wordmark}\\s*`, "i"), "").trim()
    : title;

  if (family === "studio") {
    slide.addShape("rect", {
      x: 8.75,
      y: 0.72,
      w: 3.9,
      h: 5.96,
      line: { color: mixHex(pal.header, pal.accent, 0.38), pt: 1 },
      fill: { color: mixHex(pal.header, pal.card, 0.04), transparency: 4 },
    });
    slide.addShape("rect", {
      x: 9.25,
      y: 1.2,
      w: 3,
      h: 0.12,
      line: { color: pal.accent2, transparency: 100 },
      fill: { color: pal.accent2 },
    });
    addIcon(slide, motif, 10.05, 1.84, 1.62, pal);
  } else if (family === "playful") {
    slide.addShape("ellipse", {
      x: 9.4,
      y: 1.2,
      w: 2.4,
      h: 2.4,
      line: { color: mixHex(pal.accent, pal.card, 0.3), transparency: 35 },
      fill: { color: mixHex(pal.accent2, pal.card, 0.38), transparency: 6 },
    });
    slide.addShape("ellipse", {
      x: 10.6,
      y: 3.2,
      w: 1.55,
      h: 1.55,
      line: { color: mixHex(pal.accent2, pal.card, 0.4), transparency: 40 },
      fill: { color: mixHex(pal.accent, pal.card, 0.24), transparency: 12 },
    });
    addIcon(slide, motif, 10.0, 1.9, 1.2, { ...pal, accent: pal.accent, accent2: pal.accent2, text: pal.header });
  } else {
    slide.addShape("rect", {
      x: 0,
      y: 6.95,
      w: SLIDE_W,
      h: 0.55,
      line: { color: pal.footer, transparency: 100 },
      fill: { color: pal.footer },
    });
  }

  addText(slide, kicker.toUpperCase(), { x: 1.08, y: 1.12, w: 3.8, h: 0.25 }, {
    fontFace: FONT_BODY,
    fontSize: 10,
    color: family === "playful" ? pal.header : pal.darkMuted,
    charSpace: 1.4,
    bold: true,
    valign: "mid",
  });

  if (family === "proposal" && wordmark) {
    addText(slide, wordmark, { x: 1.02, y: 1.72, w: 3.8, h: 0.76 }, {
      fontFace: FONT_HEAD,
      fontSize: 40,
      bold: true,
      color: pal.accent,
      fit: "shrink",
      valign: "top",
    });
  }

  const titleY = family === "proposal" ? (wordmark ? 2.62 : 1.82) : 1.54;
  const titleH = family === "proposal" ? 1.14 : 1.78;
  const titleColor = family === "proposal" ? "FFFFFF" : (family === "studio" ? "FFFFFF" : pal.header);
  addText(slide, family === "proposal" ? (titleRemainder || title) : title, { x: 1.0, y: titleY, w: 7.6, h: titleH }, {
    fontFace: family === "proposal" ? "Georgia" : FONT_HEAD,
    fontSize: family === "proposal" ? 29 : (family === "playful" ? 25 : 30),
    bold: true,
    color: titleColor,
    breakLine: true,
    fit: "shrink",
    valign: "top",
  });

  if (family === "proposal") {
    addLine(slide, 1.06, titleY + titleH + 0.06, 2.72, 0, "D6A33E", 2.5);
  }

  if (subtitle) {
    addText(slide, subtitle, { x: 1.02, y: family === "proposal" ? 4.2 : 3.18, w: 7.2, h: 0.66 }, {
      fontFace: "Georgia",
      fontSize: family === "proposal" ? 17 : 17,
      color: family === "playful" ? mixHex(pal.header, "FFFFFF", 0.22) : "FFFFFF",
      italic: family === "proposal",
      breakLine: true,
    });
  }
  if (support) {
    addText(slide, support, { x: 1.04, y: family === "proposal" ? 5.12 : 4.02, w: 6.2, h: 0.28 }, {
      fontFace: family === "proposal" ? "Georgia" : FONT_BODY,
      fontSize: family === "proposal" ? 10.8 : 11,
      color: family === "playful" ? mixHex(pal.header, "FFFFFF", 0.42) : pal.darkMuted,
      italic: family === "proposal",
    });
  }
  if (family === "proposal") {
    const footer = manifest.deck_plan.subject.trim() || manifest.title;
    addText(slide, footer, { x: 1.08, y: 7.02, w: 7.6, h: 0.22 }, {
      fontFace: "Georgia",
      fontSize: 10.5,
      color: pal.darkMuted,
    });
    addText(slide, "Confidential", { x: 11.05, y: 7.03, w: 1.2, h: 0.22 }, {
      fontFace: "Georgia",
      fontSize: 10,
      color: pal.darkMuted,
      align: "right",
    });
  } else if (meta) {
    addText(slide, meta, { x: 1.04, y: 5.02, w: 7.1, h: 0.32 }, {
      fontFace: "Georgia",
      fontSize: 10.5,
      italic: true,
      color: family === "playful" ? mixHex(pal.header, "FFFFFF", 0.48) : pal.darkMuted,
    });
  }
}

function renderSlideChrome(slide: PptxSlide, content: SlidePlan, totalSlides: number, family: RenderFamily, pal: RenderPalette): Bounds {
  addFullRect(slide, family === "proposal" ? pal.canvas : pal.bg);
  const headFill = family === "playful" ? mixHex(pal.header, "FFFFFF", 0.08) : pal.header;
  addPanel(slide, { x: 0, y: 0, w: SLIDE_W, h: 0.58 }, headFill, headFill, 0);
  if (family !== "playful") {
    addPanel(slide, { x: 0, y: 0.58, w: SLIDE_W, h: 0.06 }, mixHex(pal.canvas, pal.card, 0.2), mixHex(pal.canvas, pal.card, 0.2), 0);
  }

  addIcon(slide, slideIconToken(content), 0.38, 0.12, 0.28, {
    ...pal,
    accent: mixHex(headFill, pal.accent, 0.44),
    accent2: mixHex(headFill, pal.accent2, 0.44),
    text: "FFFFFF",
  });
  addText(slide, shortTitle(content.heading || "Untitled Slide"), { x: 0.78, y: 0.14, w: 8.2, h: 0.22 }, {
    fontFace: FONT_BODY,
    fontSize: 11.5,
    color: "FFFFFF",
    bold: true,
    charSpace: 0.3,
  });
  addPanel(slide, { x: 10.72, y: 0.1, w: 1.98, h: 0.3 }, mixHex(headFill, pal.accent, 0.48), mixHex(headFill, pal.card, 0.1), 0.08);
  addText(slide, `Slide ${content.number} of ${totalSlides}`, { x: 10.89, y: 0.15, w: 1.64, h: 0.16 }, {
    fontFace: FONT_BODY,
    fontSize: 9.5,
    color: "FFFFFF",
    bold: true,
    align: "center",
  });
  return { x: 0.48, y: 0.8, w: 12.37, h: 6.2 };
}

function renderMetricTile(slide: PptxSlide, stat: Stat, bounds: Bounds, fill: string, border: string, accent: string, valueColor: string, textColor: string): void {
  addPanel(slide, bounds, fill, border, 0.12);
  addPanel(slide, { x: bounds.x, y: bounds.y, w: bounds.w, h: 0.04 }, accent, accent, 0);
  addText(slide, stat.value, { x: bounds.x + 0.18, y: bounds.y + 0.2, w: bounds.w - 0.36, h: 0.5 }, {
    fontFace: FONT_HEAD,
    fontSize: 22,
    bold: true,
    color: valueColor,
  });
  addText(slide, stat.label.toUpperCase(), { x: bounds.x + 0.18, y: bounds.y + 0.74, w: bounds.w - 0.36, h: 0.18 }, {
    fontFace: FONT_BODY,
    fontSize: 8.5,
    bold: true,
    color: textColor,
    charSpace: 0.6,
  });
  if (stat.desc?.trim()) {
    addText(slide, trimSentence(stat.desc.trim(), 74), { x: bounds.x + 0.18, y: bounds.y + 0.98, w: bounds.w - 0.36, h: bounds.h - 1.12 }, {
      fontFace: FONT_BODY,
      fontSize: 10.5,
      color: textColor,
      valign: "top",
      breakLine: true,
      fit: "shrink",
    });
  }
}

function renderStats(slide: PptxSlide, plan: SlidePlan, bounds: Bounds, pal: RenderPalette): void {
  const stats = (plan.stats ?? []).slice(0, 4);
  if (stats.length === 0) {
    return renderBullets(slide, plan, bounds, pal);
  }
  const gap = 0.16;
  const heroH = stats.length > 2 ? 2.78 : bounds.h;
  const bandH = stats.length > 2 ? 0.44 : 0;
  const lowerY = bounds.y + heroH + gap;
  const lowerH = stats.length > 2 ? bounds.h - heroH - gap - bandH - gap : 0;
  const cardW = (bounds.w - gap) / 2;

  stats.slice(0, 2).forEach((stat, index) => {
    renderMetricTile(
      slide,
      stat,
      { x: bounds.x + index * (cardW + gap), y: bounds.y, w: cardW, h: heroH },
      index === 0 ? pal.header : mixHex(pal.header, pal.card, 0.18),
      mixHex(pal.header, pal.card, 0.18),
      accentColor(pal, index),
      accentColor(pal, index),
      pal.darkMuted,
    );
  });

  stats.slice(2).forEach((stat, index) => {
    renderMetricTile(
      slide,
      stat,
      { x: bounds.x + index * (cardW + gap), y: lowerY, w: cardW, h: lowerH },
      pal.card,
      mixHex(pal.canvas, pal.border, 0.84),
      accentColor(pal, index + 2),
      pal.text,
      pal.lightMuted,
    );
  });

  if (stats.length > 2) {
    addPanel(slide, { x: bounds.x, y: bounds.y + bounds.h - bandH, w: bounds.w, h: bandH }, mixHex(pal.canvas, pal.header, 0.08), mixHex(pal.canvas, pal.border, 0.76), 0.04);
    addText(slide, proposalStatsTakeaway(stats), { x: bounds.x + 0.18, y: bounds.y + bounds.h - bandH + 0.12, w: bounds.w - 0.36, h: bandH - 0.2 }, {
      fontFace: FONT_BODY,
      fontSize: 10.2,
      color: pal.text,
      bold: true,
      fit: "shrink",
    });
  }
}

function pointColumns(points: string[]): [string[], string[]] {
  if (points.length < 4) return [points, []];
  const left: string[] = [];
  const right: string[] = [];
  points.forEach((point, index) => {
    if (index % 2 === 0) left.push(point);
    else right.push(point);
  });
  return [left, right];
}

function renderPointList(slide: PptxSlide, items: string[], bounds: Bounds, pal: RenderPalette, offset = 0): void {
  const rowGap = 0.17;
  const rowH = items.length > 4 ? 0.62 : 0.8;
  items.forEach((item, index) => {
    const top = bounds.y + index * (rowH + rowGap);
    const { title, desc } = splitCardText(item);
    slide.addShape("ellipse", {
      x: bounds.x,
      y: top + 0.18,
      w: 0.1,
      h: 0.1,
      line: { color: accentColor(pal, offset + index), transparency: 100 },
      fill: { color: accentColor(pal, offset + index) },
    });
    addText(slide, title, { x: bounds.x + 0.18, y: top + 0.02, w: bounds.w - 0.18, h: 0.22 }, {
      fontFace: FONT_BODY,
      fontSize: 11.5,
      color: pal.text,
      bold: true,
      valign: "top",
    });
    if (desc) {
      addText(slide, desc, { x: bounds.x + 0.18, y: top + 0.26, w: bounds.w - 0.18, h: rowH - 0.16 }, {
        fontFace: FONT_BODY,
        fontSize: 10,
        color: pal.lightMuted,
        valign: "top",
        breakLine: true,
      });
    }
  });
}

function renderBullets(slide: PptxSlide, plan: SlidePlan, bounds: Bounds, pal: RenderPalette): void {
  const points = (plan.points ?? []).slice(0, 8);
  const stats = (plan.stats ?? []).slice(0, 4);
  let cursorY = bounds.y;
  if (stats.length > 0) {
    const gap = 0.14;
    const cardW = (bounds.w - gap * 3) / 4;
    const cardH = 1.18;
    stats.forEach((stat, index) => {
      renderMetricTile(
        slide,
        stat,
        { x: bounds.x + index * (cardW + gap), y: bounds.y, w: cardW, h: cardH },
        pal.card,
        mixHex(pal.canvas, pal.border, 0.84),
        accentColor(pal, index),
        pal.text,
        pal.lightMuted,
      );
    });
    cursorY += cardH + 0.24;
  }

  const lead = proposalLeadText(plan, points);
  if (lead) {
    addText(slide, lead, { x: bounds.x, y: cursorY, w: bounds.w, h: 0.58 }, {
      fontFace: "Georgia",
      fontSize: 13.2,
      color: pal.lightMuted,
      breakLine: true,
      fit: "shrink",
    });
    cursorY += 0.72;
  }

  addText(slide, proposalSectionLabel(plan), { x: bounds.x, y: cursorY, w: 3.2, h: 0.18 }, {
    fontFace: FONT_BODY,
    fontSize: 8.8,
    bold: true,
    color: pal.lightMuted,
    charSpace: 0.8,
  });
  cursorY += 0.28;

  const [left, right] = pointColumns(points);
  const gap = right.length > 0 ? 0.34 : 0;
  const colW = right.length > 0 ? (bounds.w - gap) / 2 : bounds.w;
  renderPointList(slide, left, { x: bounds.x, y: cursorY, w: colW, h: bounds.h - (cursorY - bounds.y) }, pal, 0);
  if (right.length > 0) {
    renderPointList(slide, right, { x: bounds.x + colW + gap, y: cursorY, w: colW, h: bounds.h - (cursorY - bounds.y) }, pal, left.length);
  }
}

function renderRoadmapRow(slide: PptxSlide, index: number, title: string, meta: string, desc: string, token: string, bounds: Bounds, pal: RenderPalette): void {
  addPanel(slide, bounds, index % 2 === 1 ? mixHex(pal.canvas, pal.card, 0.74) : pal.card, mixHex(pal.canvas, pal.border, 0.84), 0.08);
  addPanel(slide, { x: bounds.x, y: bounds.y, w: 0.06, h: bounds.h }, accentColor(pal, index), accentColor(pal, index), 0.03);
  addIcon(slide, token, bounds.x + 0.18, bounds.y + 0.17, 0.36, {
    ...pal,
    accent: accentColor(pal, index),
    accent2: mixHex(accentColor(pal, index), pal.card, 0.4),
    text: "FFFFFF",
  });
  addText(slide, meta.toUpperCase(), { x: bounds.x + 0.7, y: bounds.y + 0.16, w: 1.4, h: 0.18 }, {
    fontFace: FONT_BODY,
    fontSize: 8.5,
    bold: true,
    color: pal.lightMuted,
    charSpace: 0.7,
  });
  addText(slide, title, { x: bounds.x + 0.7, y: bounds.y + 0.38, w: bounds.w - 2.4, h: 0.24 }, {
    fontFace: FONT_HEAD,
    fontSize: 13.5,
    bold: true,
    color: pal.text,
  });
  if (desc) {
    addText(slide, desc, { x: bounds.x + 0.7, y: bounds.y + 0.68, w: bounds.w - 2.7, h: bounds.h - 0.82 }, {
      fontFace: FONT_BODY,
      fontSize: 10.2,
      color: pal.lightMuted,
      valign: "top",
      breakLine: true,
    });
  }
  addPanel(slide, { x: bounds.x + bounds.w - 1.1, y: bounds.y + 0.18, w: 0.88, h: 0.22 }, mixHex(pal.canvas, accentColor(pal, index), 0.14), mixHex(pal.canvas, pal.border, 0.8), 0.08);
  addText(slide, `${index + 1}`.padStart(2, "0"), { x: bounds.x + bounds.w - 0.96, y: bounds.y + 0.22, w: 0.58, h: 0.12 }, {
    fontFace: FONT_BODY,
    fontSize: 8.8,
    bold: true,
    color: pal.text,
    align: "center",
  });
}

function renderSteps(slide: PptxSlide, plan: SlidePlan, bounds: Bounds, pal: RenderPalette): void {
  const steps = (plan.steps ?? plan.points ?? []).slice(0, 6);
  const rowGap = 0.18;
  const rowH = (bounds.h - rowGap * Math.max(steps.length - 1, 0)) / Math.max(steps.length, 1);
  steps.forEach((step, index) => {
    const { title, desc } = splitCardText(step);
    renderRoadmapRow(slide, index, title || step, `Step ${index + 1}`, desc, inferIconToken(title || step) || "strategy", {
      x: bounds.x,
      y: bounds.y + index * (rowH + rowGap),
      w: bounds.w,
      h: rowH,
    }, pal);
  });
}

function renderCards(slide: PptxSlide, plan: SlidePlan, bounds: Bounds, pal: RenderPalette): void {
  const cards = (plan.cards ?? []).slice(0, 6);
  if (cards.length === 0) {
    return renderBullets(slide, plan, bounds, pal);
  }
  const startY = addTopSummaryStrip(slide, plan, bounds, pal, slideIconToken(plan));
  const cols = cards.length <= 2 ? 2 : 3;
  const rows = Math.ceil(cards.length / cols);
  const gapX = 0.22;
  const gapY = 0.22;
  const cardW = (bounds.w - gapX * (cols - 1)) / cols;
  const cardH = (bounds.h - (startY - bounds.y) - gapY * (rows - 1)) / rows;
  cards.forEach((card, index) => {
    const col = index % cols;
    const row = Math.floor(index / cols);
    const x = bounds.x + col * (cardW + gapX);
    const y = startY + row * (cardH + gapY);
    addPanel(slide, { x, y, w: cardW, h: cardH }, row % 2 === 0 ? pal.card : mixHex(pal.canvas, pal.card, 0.74), mixHex(pal.canvas, pal.border, 0.84), 0.08);
    addIcon(slide, card.icon || card.title, x + 0.18, y + 0.18, 0.42, {
      ...pal,
      accent: accentColor(pal, index),
      accent2: mixHex(accentColor(pal, index), pal.card, 0.4),
      text: "FFFFFF",
    });
    addText(slide, card.title, { x: x + 0.72, y: y + 0.2, w: cardW - 0.9, h: 0.24 }, {
      fontFace: FONT_HEAD,
      fontSize: 12.5,
      bold: true,
      color: pal.text,
    });
    addText(slide, card.desc?.trim() || card.title, { x: x + 0.18, y: y + 0.72, w: cardW - 0.36, h: cardH - 0.88 }, {
      fontFace: FONT_BODY,
      fontSize: 10.2,
      color: pal.lightMuted,
      valign: "top",
      breakLine: true,
    });
  });
}

function chartTypeName(kind: string): "bar" | "line" | "pie" | "doughnut" {
  switch (kind.trim().toLowerCase()) {
    case "line":
      return "line";
    case "pie":
      return "pie";
    case "donut":
    case "doughnut":
      return "doughnut";
    default:
      return "bar";
  }
}

function chartInsight(plan: SlidePlan): string {
  const series = plan.chart_series ?? [];
  if (series.length === 0) return buildLead(plan);
  const first = series[0];
  const values = first.values ?? [];
  if (values.length >= 2) {
    const delta = values[values.length - 1] - values[0];
    const direction = delta >= 0 ? "up" : "down";
    return `${first.name} trends ${direction} by ${Math.abs(delta).toFixed(0)} points across the visible range.`;
  }
  return buildLead(plan);
}

function renderChart(slide: PptxSlide, plan: SlidePlan, bounds: Bounds, pal: RenderPalette): void {
  const categories = plan.chart_categories ?? [];
  const series = (plan.chart_series ?? []).filter((entry) => Array.isArray(entry.values) && entry.values.length > 0);
  if (categories.length === 0 || series.length === 0) {
    return renderBullets(slide, plan, bounds, pal);
  }

  const startY = addTopSummaryStrip(slide, plan, bounds, pal, "chart");
  const usableH = bounds.h - (startY - bounds.y);
  const mainW = 8.55;
  const sideW = bounds.w - mainW - 0.28;
  addPanel(slide, { x: bounds.x, y: startY, w: mainW, h: usableH }, pal.card, mixHex(pal.canvas, pal.border, 0.84), 0.06);
  addPanel(slide, { x: bounds.x + mainW + 0.28, y: startY, w: sideW, h: usableH }, mixHex(pal.canvas, pal.card, 0.74), mixHex(pal.canvas, pal.border, 0.84), 0.06);
  addPanel(slide, { x: bounds.x, y: startY, w: mainW, h: 0.04 }, pal.accent, pal.accent, 0);
  const chartData = series.map((entry) => ({
    name: entry.name,
    labels: categories,
    values: entry.values,
  }));
  const type = chartTypeName(plan.chart_type ?? "");
  slide.addChart(type, chartData, {
    x: bounds.x + 0.18,
    y: startY + 0.24,
    w: mainW - 0.44,
    h: usableH - 0.42,
    showLegend: series.length > 1,
    legendPos: "b",
    chartColors: series.map((entry, index) => normalizeHex(entry.color || accentColor(pal, index))),
    chartArea: { fill: { color: pal.card }, border: { color: pal.card, pt: 0 } },
    plotArea: { fill: { color: pal.card }, border: { color: pal.card, pt: 0 } },
    catAxisLabelColor: pal.lightMuted,
    catAxisLabelFontFace: FONT_BODY,
    catAxisLabelFontSize: 10,
    catAxisLineColor: mixHex(pal.canvas, pal.border, 0.52),
    catAxisLineShow: true,
    catAxisMajorTickMark: "none",
    catAxisLabelRotate: categories.some((label) => label.length > 7) ? -35 : 0,
    valAxisLabelColor: pal.lightMuted,
    valAxisLabelFontFace: FONT_BODY,
    valAxisLabelFontSize: 10,
    valAxisLineShow: false,
    valGridLine: { color: mixHex(pal.canvas, pal.border, 0.4), size: 1, style: "solid" },
    valAxisTitle: plan.y_label || "",
    valAxisTitleColor: pal.lightMuted,
    valAxisTitleFontFace: FONT_BODY,
    valAxisTitleFontSize: 10,
    showValAxisTitle: Boolean(plan.y_label),
    showTitle: false,
    barDir: type === "bar" ? "col" : undefined,
    showValue: false,
    showLabel: false,
    dataBorder: { color: "FFFFFF", pt: 0 },
    layout: { x: 0.12, y: 0.05, w: 0.84, h: 0.8 },
  });
  addText(slide, "Key read", { x: bounds.x + mainW + 0.48, y: startY + 0.24, w: sideW - 0.4, h: 0.22 }, {
    fontFace: FONT_BODY,
    fontSize: 9.2,
    bold: true,
    color: pal.lightMuted,
    charSpace: 0.7,
  });
  addText(slide, chartInsight(plan), { x: bounds.x + mainW + 0.46, y: startY + 0.56, w: sideW - 0.56, h: 0.86 }, {
    fontFace: FONT_HEAD,
    fontSize: 15,
    bold: true,
    color: pal.text,
    breakLine: true,
    valign: "top",
  });
  series.slice(0, 4).forEach((entry, index) => {
    const rowY = startY + 1.62 + index * 0.88;
    addPanel(slide, { x: bounds.x + mainW + 0.42, y: rowY, w: sideW - 0.52, h: 0.72 }, pal.card, mixHex(pal.canvas, pal.border, 0.84), 0.05);
    slide.addShape("rect", {
      x: bounds.x + mainW + 0.56,
      y: rowY + 0.28,
      w: 0.14,
      h: 0.14,
      line: { color: accentColor(pal, index), transparency: 100 },
      fill: { color: accentColor(pal, index) },
    });
    addText(slide, entry.name, { x: bounds.x + mainW + 0.78, y: rowY + 0.18, w: sideW - 1.2, h: 0.18 }, {
      fontFace: FONT_BODY,
      fontSize: 10.8,
      bold: true,
      color: pal.text,
    });
    addText(slide, `${entry.values.at(-1) ?? 0}`, { x: bounds.x + mainW + sideW - 0.86, y: rowY + 0.16, w: 0.4, h: 0.2 }, {
      fontFace: FONT_HEAD,
      fontSize: 14,
      bold: true,
      color: accentColor(pal, index),
      align: "right",
    });
  });
}

function renderTimeline(slide: PptxSlide, plan: SlidePlan, bounds: Bounds, pal: RenderPalette): void {
  const items = (plan.timeline ?? []).slice(0, 6);
  const startY = addTopSummaryStrip(slide, plan, bounds, pal, "strategy");
  const rowGap = 0.18;
  const rowH = (bounds.h - (startY - bounds.y) - rowGap * Math.max(items.length - 1, 0)) / Math.max(items.length, 1);
  items.forEach((item, index) => {
    renderRoadmapRow(slide, index, item.title, item.date || `Phase ${index + 1}`, item.desc?.trim() || item.title, inferIconToken(item.title) || slideIconToken(plan), {
      x: bounds.x,
      y: startY + index * (rowH + rowGap),
      w: bounds.w,
      h: rowH,
    }, pal);
  });
}

function renderCompare(slide: PptxSlide, plan: SlidePlan, bounds: Bounds, pal: RenderPalette): void {
  const left = plan.left_column ?? { heading: "Current State", points: [] };
  const right = plan.right_column ?? { heading: "Future State", points: [] };
  const startY = addTopSummaryStrip(slide, plan, bounds, pal, "integration");
  const gap = 0.28;
  const colW = (bounds.w - gap) / 2;
  [
    { data: left, x: bounds.x, accent: accentColor(pal, 0), offset: 0 },
    { data: right, x: bounds.x + colW + gap, accent: accentColor(pal, 1), offset: left.points.length },
  ].forEach((column) => {
    addPanel(slide, { x: column.x, y: startY, w: colW, h: bounds.h - (startY - bounds.y) }, column.data === left ? mixHex(pal.canvas, pal.card, 0.74) : pal.card, mixHex(pal.canvas, pal.border, 0.84), 0.08);
    addPanel(slide, { x: column.x, y: startY, w: colW, h: 0.06 }, column.accent, column.accent, 0);
    addText(slide, column.data.heading, { x: column.x + 0.18, y: startY + 0.18, w: colW - 0.36, h: 0.24 }, {
      fontFace: FONT_HEAD,
      fontSize: 15,
      bold: true,
      color: pal.text,
    });
    renderPointList(slide, column.data.points.slice(0, 6), { x: column.x + 0.18, y: startY + 0.62, w: colW - 0.36, h: bounds.h - (startY - bounds.y) - 0.8 }, pal, column.offset);
  });
}

function renderTable(slide: PptxSlide, plan: SlidePlan, bounds: Bounds, pal: RenderPalette): void {
  const table = plan.table;
  if (!table || table.headers.length === 0) {
    return renderBullets(slide, plan, bounds, pal);
  }
  const startY = addTopSummaryStrip(slide, plan, bounds, pal, "shield");
  addPanel(slide, { x: bounds.x, y: startY, w: bounds.w, h: bounds.h - (startY - bounds.y) }, pal.card, mixHex(pal.canvas, pal.border, 0.84), 0.04);
  addPanel(slide, { x: bounds.x, y: startY, w: bounds.w, h: 0.04 }, pal.accent, pal.accent, 0);
  const rows: TableRow[] = [
    table.headers.map((header) => ({
      text: header,
      options: {
        bold: true,
        color: pal.text,
        fill: { color: mixHex(pal.canvas, pal.border, 0.12) },
        fontFace: FONT_BODY,
        fontSize: 10.5,
        align: "left",
        margin: 0.08,
      },
    })),
    ...table.rows.map((row) =>
      row.map((cell) => ({
        text: cell,
        options: {
          fontFace: FONT_BODY,
          fontSize: 10.2,
          color: pal.text,
          margin: 0.08,
          valign: "mid",
          fill: { color: pal.card },
        },
      })),
    ),
  ];
  const colW = table.headers.map(() => Number((bounds.w / table.headers.length).toFixed(2)));
  slide.addTable(rows, {
    x: bounds.x + 0.16,
    y: startY + 0.18,
    w: bounds.w - 0.32,
    h: bounds.h - (startY - bounds.y) - 0.36,
    colW,
    border: { pt: 1, color: mixHex(pal.canvas, pal.border, 0.84) },
    margin: 0.06,
    fontFace: FONT_BODY,
    fontSize: 10.2,
    color: pal.text,
    fill: { color: pal.card },
    autoPage: false,
  });
}

function renderSection(slide: PptxSlide, plan: SlidePlan, bounds: Bounds, family: RenderFamily, pal: RenderPalette): void {
  const fill = family === "proposal" ? pal.header : mixHex(pal.header, pal.accent, 0.12);
  addPanel(slide, { x: bounds.x + 0.1, y: bounds.y + 0.42, w: bounds.w - 0.2, h: 4.35 }, fill, mixHex(fill, pal.card, 0.14), 0.08);
  addText(slide, (plan.purpose?.trim() || "Section transition").toUpperCase(), { x: bounds.x + 0.42, y: bounds.y + 0.82, w: 3.2, h: 0.2 }, {
    fontFace: FONT_BODY,
    fontSize: 9,
    bold: true,
    color: pal.darkMuted,
    charSpace: 0.9,
  });
  addText(slide, plan.heading, { x: bounds.x + 0.38, y: bounds.y + 1.22, w: bounds.w - 0.76, h: 1.05 }, {
    fontFace: FONT_HEAD,
    fontSize: 30,
    bold: true,
    color: family === "proposal" ? "FFFFFF" : pal.text,
    breakLine: true,
  });
  addText(slide, buildLead(plan), { x: bounds.x + 0.42, y: bounds.y + 2.68, w: bounds.w - 0.84, h: 0.8 }, {
    fontFace: FONT_BODY,
    fontSize: 12,
    color: family === "proposal" ? pal.darkMuted : pal.lightMuted,
    breakLine: true,
  });
}

function renderSlideBody(slide: PptxSlide, plan: SlidePlan, bounds: Bounds, family: RenderFamily, pal: RenderPalette): void {
  switch (plan.layout) {
    case "stats":
      renderStats(slide, plan, bounds, pal);
      return;
    case "steps":
      renderSteps(slide, plan, bounds, pal);
      return;
    case "cards":
      renderCards(slide, plan, bounds, pal);
      return;
    case "chart":
      renderChart(slide, plan, bounds, pal);
      return;
    case "timeline":
      renderTimeline(slide, plan, bounds, pal);
      return;
    case "compare":
      renderCompare(slide, plan, bounds, pal);
      return;
    case "table":
      renderTable(slide, plan, bounds, pal);
      return;
    case "title":
    case "blank":
      renderSection(slide, plan, bounds, family, pal);
      return;
    default:
      renderBullets(slide, plan, bounds, pal);
  }
}

function validateManifest(input: unknown): Manifest {
  if (!input || typeof input !== "object") {
    throw new Error("manifest payload must be an object");
  }
  const manifest = input as Partial<Manifest>;
  if (!Array.isArray(manifest.slides) || manifest.slides.length === 0) {
    throw new Error("manifest.slides must contain at least one slide");
  }
  return manifest as Manifest;
}

async function readStdin(): Promise<string> {
  return await new Promise((resolve, reject) => {
    let data = "";
    process.stdin.setEncoding("utf8");
    process.stdin.on("data", (chunk) => {
      data += chunk;
    });
    process.stdin.on("end", () => resolve(data));
    process.stdin.on("error", reject);
  });
}

async function buildPresentation(manifest: Manifest, outputPath: string): Promise<void> {
  const family = previewFamily(manifest);
  const pal = buildPalette(manifest, family);
  const PPTXRuntime = PptxGenJS as unknown as new () => any;
  const pptx = new PPTXRuntime();
  pptx.layout = "LAYOUT_WIDE";
  pptx.author = "YASSERRMD";
  pptx.company = "Barq Cowork";
  pptx.subject = manifest.deck_plan.subject || manifest.title;
  pptx.title = manifest.title;
  pptx.lang = "en-US";
  pptx.theme = {
    headFontFace: FONT_HEAD,
    bodyFontFace: FONT_BODY,
  };

  const cover = pptx.addSlide();
  renderCover(cover, manifest, family, pal);
  cover.addNotes(manifest.narrative || manifest.deck_plan.narrative_arc || manifest.title);

  const totalSlides = manifest.slides.length + 1;
  for (const plan of manifest.slides) {
    const slide = pptx.addSlide();
    const bounds = renderSlideChrome(slide, plan, totalSlides, family, pal);
    renderSlideBody(slide, plan, bounds, family, pal);
    if (plan.speaker_notes?.trim()) {
      slide.addNotes(plan.speaker_notes.trim());
    }
  }

  fs.mkdirSync(path.dirname(outputPath), { recursive: true });
  await pptx.writeFile({ fileName: outputPath, compression: true });
}

async function main(): Promise<void> {
  const args = process.argv.slice(2);
  const outIndex = args.indexOf("--output");
  if (outIndex === -1 || outIndex === args.length - 1) {
    throw new Error("missing --output <path>");
  }
  const outputPath = path.resolve(args[outIndex + 1]);
  const payload = await readStdin();
  if (!payload.trim()) {
    throw new Error("missing manifest payload on stdin");
  }
  const manifest = validateManifest(JSON.parse(payload));
  await buildPresentation(manifest, outputPath);
  process.stdout.write(JSON.stringify({ status: "ok", output: outputPath }) + "\n");
}

main().catch((err) => {
  process.stderr.write(String(err instanceof Error ? err.stack || err.message : err) + "\n");
  process.exitCode = 1;
});
