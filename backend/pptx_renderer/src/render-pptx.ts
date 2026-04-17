import fs from "node:fs";
import os from "node:os";
import path from "node:path";

import PptxGenJS from "pptxgenjs";
import bootstrapIconsSprite from "bootstrap-icons/bootstrap-icons.svg";
import bootstrapCSS from "bootstrap/dist/css/bootstrap.min.css";
import bootstrapIconsCSS from "bootstrap-icons/font/bootstrap-icons.min.css";
import bootstrapIconsWoff2 from "bootstrap-icons/font/fonts/bootstrap-icons.woff2";

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
  archetype?: string;
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
  theme_css?: string;
  cover_html?: string;
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
  html?: string;
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

type RenderFamily = "proposal" | "studio" | "playful" | "civic" | "tech";

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

type CompositionParams = {
  // Layout geometry
  splitRatio: number;      // 0 = no split; 0.35–0.55 = left panel width ratio
  heroFirst: boolean;      // hero/lead block appears before content (top or left)
  columns: number;         // content columns: 1, 2, 3
  horizontal: boolean;     // content flows left-right (not top-bottom)
  // Surface
  panelFilled: boolean;    // lead/hero panel has solid fill
  panelOutline: boolean;   // lead panel has outline only
  panelGlass: boolean;     // lead panel has semi-transparent fill
  // Accent placement
  accentRail: boolean;     // left vertical accent bar
  accentBand: boolean;     // top horizontal accent bar
  accentChip: boolean;     // small badge accent
  accentGlow: boolean;     // ambient glow effect
  // Density
  density: number;         // 0.7=airy, 1.0=balanced, 1.3=dense (scales font/spacing)
};

function parseComposition(design?: SlideDesign): CompositionParams {
  const ls = (design?.layout_style ?? "").toLowerCase();
  const ps = (design?.panel_style ?? "soft").toLowerCase();
  const am = (design?.accent_mode ?? "rail").toLowerCase();
  const dn = (design?.density ?? "balanced").toLowerCase();

  let splitRatio = 0;
  if (/split|side|panel|aside|left|right/.test(ls)) {
    if (/wide|broad|major|60|65/.test(ls)) splitRatio = 0.52;
    else if (/narrow|minor|30|35/.test(ls)) splitRatio = 0.32;
    else splitRatio = 0.40;
  }

  let columns = 1;
  if (/3.col|three.col|3col|triple/.test(ls)) columns = 3;
  else if (/2.col|two.col|2col|dual|double|split|grid/.test(ls)) columns = 2;

  const heroFirst = /hero|spotlight|focus|stage|lead|featured|highlight|banner|above|top/.test(ls);
  const horizontal = /rail|horizontal|row|h.?flow/.test(ls);

  const panelFilled = /solid|filled|block|dark|bold/.test(ps) || /solid|block/.test(ls);
  const panelOutline = /outline|border|wire|ghost/.test(ps);
  const accentBandVal = /band|top|bar|stripe/.test(am);
  const accentChipVal = /chip|badge|dot|pill/.test(am);
  const accentGlowVal = /glow|ambient|soft/.test(am);
  const accentRail = !accentBandVal && !accentChipVal && !accentGlowVal;

  let density = 1.0;
  if (/airy|sparse|open/.test(dn)) density = 0.75;
  else if (/dense|compact|tight/.test(dn)) density = 1.28;

  return {
    splitRatio, heroFirst, columns, horizontal,
    panelFilled, panelOutline, panelGlass: !panelFilled && !panelOutline,
    accentRail, accentBand: accentBandVal, accentChip: accentChipVal, accentGlow: accentGlowVal,
    density,
  };
}

const SLIDE_W = 13.333;
const SLIDE_H = 7.5;
const FONT_HEAD = "Arial";
const FONT_BODY = "Arial";
const BOOTSTRAP_ICON_FALLBACK = "stars";

type BootstrapIcon = {
  viewBox: string;
  content: string;
};

const bootstrapIcons = parseBootstrapIconsSprite(bootstrapIconsSprite);

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

function parseAttributes(raw: string): Record<string, string> {
  const attributes: Record<string, string> = {};
  for (const match of raw.matchAll(/([\w:-]+)\s*=\s*"([^"]*)"/g)) {
    attributes[match[1].toLowerCase()] = match[2];
  }
  return attributes;
}

function parseBootstrapIconsSprite(sprite: string): Map<string, BootstrapIcon> {
  const icons = new Map<string, BootstrapIcon>();
  const symbolPattern = /<symbol\b([^>]*)>([\s\S]*?)<\/symbol>/g;
  for (const match of sprite.matchAll(symbolPattern)) {
    const attributes = parseAttributes(match[1] ?? "");
    const id = (attributes.id ?? "").trim().toLowerCase();
    const content = (match[2] ?? "").trim();
    if (!id || !content) continue;
    icons.set(id, {
      viewBox: attributes.viewbox || "0 0 16 16",
      content,
    });
  }
  return icons;
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

function bootstrapIconNameForToken(raw: string): string {
  const explicit = raw
    .trim()
    .toLowerCase()
    .replace(/^bi-/, "")
    .replace(/_/g, "-")
    .replace(/[^a-z0-9-]/g, "");
  if (explicit && bootstrapIcons.has(explicit)) {
    return explicit;
  }

  switch (inferIconToken(raw) || "spark") {
    case "shield":
      return "shield-check";
    case "chart":
      return "bar-chart-line";
    case "growth":
      return "graph-up-arrow";
    case "automation":
      return "lightning-charge";
    case "integration":
      return "diagram-3";
    case "people":
      return "people";
    case "strategy":
      return "signpost-split";
    case "learning":
      return "book";
    case "health":
      return "heart-pulse";
    case "leaf":
      return "leaf";
    case "logistics":
      return "truck";
    default:
      return BOOTSTRAP_ICON_FALLBACK;
  }
}

function iconSvg(token: string, palette: { text: string }): string {
  const iconName = bootstrapIconNameForToken(token);
  const icon = bootstrapIcons.get(iconName) ?? bootstrapIcons.get(BOOTSTRAP_ICON_FALLBACK) ?? bootstrapIcons.values().next().value;
  if (!icon) {
    return "";
  }
  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="${icon.viewBox}" fill="${cssHex(palette.text)}">${icon.content}</svg>`;
}

function svgDataUri(svg: string): string {
  return `data:image/svg+xml;base64,${Buffer.from(svg).toString("base64")}`;
}

function familyFromArchetype(archetype: string): RenderFamily | "" {
  const text = archetype.trim().toLowerCase();
  if (!text) return "";
  if (containsAny(text, "playful", "storybook", "classroom", "kids")) return "playful";
  if (containsAny(text, "poster", "studio", "campaign", "showcase")) return "studio";
  if (containsAny(text, "policy", "civic", "government", "public sector", "national", "regulatory", "institutional")) return "civic";
  if (containsAny(text, "technology", "product narrative", "innovation narrative", "platform narrative", "technology showcase", "future narrative")) return "tech";
  if (containsAny(text, "proposal", "operating plan", "cost proposal", "business case", "executive brief", "board brief", "implementation plan")) return "proposal";
  return "";
}

function dominantFamilyHint(text: string): RenderFamily | "" {
  if (containsAny(text, "proposal", "cost", "pricing", "budget", "business case", "implementation plan", "delivery roadmap", "rollout", "scope", "rate card")) {
    return "proposal";
  }
  if (containsAny(text, "policy", "government", "public sector", "national", "regulatory", "uae", "ministry", "authority", "civic")) {
    return "civic";
  }
  if (containsAny(text, "technology", "platform", "software", "product", "innovation", "future", "digital", "ai")) {
    return "tech";
  }
  if (containsAny(text, "poster", "campaign", "showcase", "gallery", "bold studio")) {
    return "studio";
  }
  if (containsAny(text, "playful", "storybook", "collage", "cartoon", "fun", "kids", "children", "classroom")) {
    return "playful";
  }
  return "";
}

function previewFamily(manifest: Manifest): RenderFamily {
  const text = [
    manifest.title,
    manifest.subtitle ?? "",
    manifest.theme,
    manifest.deck_plan.archetype ?? "",
    manifest.deck_plan.subject,
    manifest.deck_plan.audience,
    manifest.deck_plan.narrative_arc,
    manifest.deck_plan.visual_direction,
    manifest.deck_plan.color_story ?? "",
    manifest.deck_plan.dominant_need,
    manifest.deck_plan.cover_style ?? "",
  ].join(" ");
  const dominant = dominantFamilyHint(text);
  const archetypeFamily = familyFromArchetype(manifest.deck_plan.archetype ?? "");

  if (dominant === "proposal") return "proposal";
  if (dominant === "civic" && archetypeFamily !== "proposal") return "civic";
  if (dominant === "tech" && archetypeFamily !== "proposal" && archetypeFamily !== "civic") return "tech";
  if (archetypeFamily) return archetypeFamily;
  if (dominant) return dominant;
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
    case "civic":
      return {
        ...base,
        bg: mixHex(base.bg, "F6F1E8", 0.72),
        card: "FFFDFC",
        canvas: mixHex(base.bg, "FFFCF8", 0.84),
        header: mixHex("12202E", base.text, 0.18),
        footer: mixHex(base.text, "F5F0E7", 0.18),
        darkMuted: mixHex("12202E", "FFFFFF", 0.66),
        lightMuted: mixHex("FFFCF8", base.text, 0.56),
        border: mixHex(base.border, "D8CFC2", 0.46),
      };
    case "tech":
      return {
        ...base,
        canvas: mixHex(base.bg, base.card, 0.86),
        header: mixHex(base.bg, "000000", 0.18),
        footer: mixHex(base.bg, "000000", 0.14),
        darkMuted: mixHex(base.text, "FFFFFF", 0.62),
        lightMuted: mixHex(base.bg, base.text, 0.58),
      };
    default:
      return base;
  }
}

function coverSectionLabels(manifest: Manifest, count: number): string[] {
  return manifest.slides
    .map((slide) => shortTitle(slide.heading || "").toUpperCase())
    .filter((label) => label.length > 0)
    .slice(0, count);
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
    fontSize: 14,
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

  slide.addShape("ellipse", {
    x,
    y,
    w: size,
    h: size,
    line: { color: accent, transparency: 100 },
    fill: { color: accent },
  });

  const svg = iconSvg(token, { text: "FFFFFF" });
  if (svg) {
    const inner = size * 0.54;
    slide.addImage({
      data: svgDataUri(svg),
      x: x + size * 0.23,
      y: y + size * 0.23,
      w: inner,
      h: inner,
    });
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

function looksLikePlanningText(value: string): boolean {
  const text = value.trim().toLowerCase();
  return text.startsWith("close with ") ||
    text.startsWith("open with ") ||
    text.startsWith("show the ") ||
    text.startsWith("show how ") ||
    text.startsWith("frame the ") ||
    text.startsWith("explain how ") ||
    text.startsWith("explain why ") ||
    text.startsWith("sequence the ") ||
    text.startsWith("give a structured ") ||
    text.startsWith("prove the ") ||
    text.startsWith("use this slide ") ||
    text.startsWith("lead with ");
}

function cleanVisibleText(value: string): string {
  const text = value.trim();
  if (!text || looksLikePlanningText(text)) return "";
  return text;
}

function contentLead(slide: SlidePlan): string {
  const points = (slide.points ?? []).map(cleanVisibleText).filter(Boolean);
  if (points.length > 0) {
    const candidate = points.find((point) => point.length >= 56) ?? points[0];
    return trimSentence(candidate, 160);
  }

  const stats = slide.stats ?? [];
  if (stats.length > 0) {
    return proposalStatsTakeaway(stats);
  }

  const steps = (slide.steps ?? []).map(cleanVisibleText).filter(Boolean);
  if (steps.length >= 2) {
    const first = splitCardText(steps[0]).title;
    const last = splitCardText(steps[steps.length - 1]).title;
    return `${first} through ${last} across ${steps.length} staged actions.`;
  }

  const timeline = slide.timeline ?? [];
  if (timeline.length >= 2) {
    return `${timeline[0].title} through ${timeline[timeline.length - 1].title} across ${timeline.length} rollout phases.`;
  }

  const cards = slide.cards ?? [];
  if (cards.length >= 2) {
    return `${cards[0].title}, ${cards[1].title}, and the capabilities needed to scale responsibly.`;
  }

  if (slide.left_column && slide.right_column) {
    return `${slide.left_column.heading} to ${slide.right_column.heading} across the decisions that matter most.`;
  }

  if (slide.table?.rows?.length) {
    return trimSentence(slide.table.rows[0].join(" | "), 140);
  }

  return "";
}

function buildLead(slide: SlidePlan): string {
  return contentLead(slide);
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
  const cleanPoints = points.map(cleanVisibleText).filter(Boolean);
  if (plan.layout === "bullets" && cleanPoints.length >= 3) {
    const titles = cleanPoints
      .slice(0, 3)
      .map((point) => trimSentence(splitCardText(point).title, 34).replace(/[.]+$/g, ""))
      .filter(Boolean);
    if (titles.length === 3) {
      return `The case for action is defined by ${titles[0]}, ${titles[1]}, and ${titles[2]}.`;
    }
  }
  const lead = contentLead(plan);
  if (lead) return lead;
  if (cleanPoints.length >= 2) {
    const left = splitCardText(cleanPoints[0]).title;
    const right = splitCardText(cleanPoints[1]).title;
    return `Focus areas include ${left} and ${right}.`;
  }
  return "";
}

function proposalSectionLabel(plan: SlidePlan): string {
  const text = normalizeText(plan.heading || "");
  if (containsAny(text, "deliverable", "scope", "phase", "build", "implementation")) return "KEY DELIVERABLES";
  if (containsAny(text, "assumption", "constraint", "dependency")) return "KEY ASSUMPTIONS";
  if (containsAny(text, "risk", "issue")) return "KEY RISKS";
  if (containsAny(text, "team", "role")) return "TEAM FOCUS";
  return "KEY POINTS";
}

function slideChipLabel(plan: SlidePlan): string {
  switch (plan.layout) {
    case "stats":
      return "DECISION SIGNALS";
    case "chart":
      return "TREND READOUT";
    case "steps":
      return "IMPLEMENTATION PATH";
    case "timeline":
      return "ROADMAP";
    case "compare":
      return "CURRENT VS TARGET";
    case "table":
      return "DECISION MATRIX";
    case "cards":
      return "CAPABILITIES";
    case "title":
    case "blank":
      return "SECTION";
    default:
      return proposalSectionLabel(plan);
  }
}

function coverWordmark(title: string): string {
  const words = title.trim().split(/\s+/).filter(Boolean);
  if (words.length === 0) return "";
  const first = words[0].replace(/[^A-Za-z0-9]/g, "");
  if (!first) return "";
  if (["AI", "IT", "HR", "DR", "CRM", "ERP", "API", "BI"].includes(first.toUpperCase())) return "";
  if (first === first.toUpperCase() && first.length <= 8) return first;
  if (first.length <= 4) return first.toUpperCase();
  return "";
}

function sectionKicker(plan: SlidePlan): string {
  switch (plan.layout) {
    case "stats":
      return "EXECUTIVE READOUT";
    case "chart":
      return "TREND SIGNAL";
    case "steps":
      return "DELIVERY SEQUENCE";
    case "timeline":
      return "MILESTONE PLAN";
    case "compare":
      return "DECISION SHIFT";
    case "table":
      return "STRUCTURED VIEW";
    case "cards":
      return "CAPABILITY SYSTEM";
    case "title":
    case "blank":
      return "SECTION TRANSITION";
    default: {
      const h = normalizeText(plan.heading || "");
      if (containsAny(h, "overview", "introduction", "background", "about", "what")) return "OVERVIEW";
      if (containsAny(h, "challenge", "problem", "issue", "pain", "risk")) return "KEY CHALLENGES";
      if (containsAny(h, "solution", "approach", "strategy", "plan", "recommendation")) return "OUR APPROACH";
      if (containsAny(h, "result", "outcome", "impact", "benefit", "value", "roi")) return "IMPACT";
      if (containsAny(h, "next", "action", "step")) return "NEXT STEPS";
      if (containsAny(h, "team", "people", "talent", "resource")) return "THE TEAM";
      if (containsAny(h, "data", "insight", "finding", "analysis", "research")) return "DATA & INSIGHTS";
      if (containsAny(h, "market", "industry", "landscape", "competitor", "trend")) return "MARKET VIEW";
      if (containsAny(h, "product", "feature", "capability", "service")) return "CAPABILITIES";
      if (containsAny(h, "finance", "revenue", "cost", "budget", "investment")) return "FINANCIALS";
      return "KEY CONTEXT";
    }
  }
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
    fontSize: 11,
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
    fontSize: 10,
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
      fontSize: 12,
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
    fontSize: 10,
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
  const stripH = 0.64;
  addPanel(slide, { x: bounds.x, y: bounds.y, w: bounds.w, h: stripH }, pal.card, mixHex(pal.canvas, pal.border, 0.84), 0.06);
  if (token) {
    addIcon(slide, token, bounds.x + 0.18, bounds.y + 0.18, 0.38, {
      ...pal,
      accent: accentColor(pal, 0),
      accent2: mixHex(accentColor(pal, 0), pal.card, 0.42),
      text: "FFFFFF",
    });
  }
  addText(slide, sectionKicker(plan), { x: bounds.x + (token ? 0.66 : 0.18), y: bounds.y + 0.1, w: 3.1, h: 0.2 }, {
    fontFace: FONT_BODY,
    fontSize: 12,
    bold: true,
    color: pal.lightMuted,
    charSpace: 0.8,
  });
  addText(slide, buildLead(plan), { x: bounds.x + (token ? 0.66 : 0.18), y: bounds.y + 0.32, w: bounds.w - (token ? 0.86 : 0.36), h: 0.24 }, {
    fontFace: FONT_BODY,
    fontSize: 14,
    bold: true,
    color: pal.text,
    fit: "shrink",
  });
  return bounds.y + stripH + 0.16;
}

function renderCover(slide: PptxSlide, manifest: Manifest, family: RenderFamily, pal: RenderPalette): void {
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

  if (family === "civic") {
    const modules = coverSectionLabels(manifest, 3);
    addFullRect(slide, pal.canvas);
    addPanel(slide, { x: 0, y: 0, w: SLIDE_W, h: 0.16 }, pal.accent, pal.accent, 0);
    addLine(slide, 8.86, 1.02, 0, 5.22, mixHex(pal.border, pal.accent, 0.28), 1.1);
    addPanel(slide, { x: 9.14, y: 1.1, w: 2.84, h: 0.4 }, mixHex(pal.canvas, pal.card, 0.42), mixHex(pal.border, pal.accent, 0.18), 0.05);
    addText(slide, "POLICY FRAME", { x: 9.34, y: 1.2, w: 1.24, h: 0.14 }, {
      fontFace: FONT_BODY,
      fontSize: 10,
      color: pal.muted,
      bold: true,
      charSpace: 0.7,
    });
    addPanel(slide, { x: 10.68, y: 1.19, w: 1.1, h: 0.08 }, mixHex(pal.accent2, pal.card, 0.14), mixHex(pal.accent2, pal.card, 0.14), 0);
    addText(slide, kicker.toUpperCase(), { x: 1.02, y: 0.98, w: 4.8, h: 0.22 }, {
      fontFace: FONT_BODY,
      fontSize: 11,
      color: pal.muted,
      charSpace: 1.2,
      bold: true,
    });
    addText(slide, title, { x: 0.98, y: 1.62, w: 7.2, h: 1.08 }, {
      fontFace: FONT_HEAD,
      fontSize: 27.5,
      bold: true,
      color: pal.text,
      breakLine: true,
      fit: "shrink",
      valign: "top",
    });
    addLine(slide, 1.04, 2.96, 2.42, 0, "C79A35", 2.4);
    if (subtitle) {
      addText(slide, subtitle, { x: 1.0, y: 3.6, w: 7.6, h: 0.58 }, {
        fontFace: FONT_BODY,
        fontSize: 16.2,
        color: pal.text,
        breakLine: true,
      });
    }
    if (support) {
      addText(slide, support, { x: 1.02, y: 4.35, w: 7.2, h: 0.24 }, {
        fontFace: FONT_BODY,
        fontSize: 13,
        color: pal.muted,
      });
    }
    if (meta) {
      addText(slide, meta, { x: 1.02, y: 6.82, w: 8.2, h: 0.22 }, {
        fontFace: FONT_BODY,
        fontSize: 12,
        color: pal.muted,
      });
    }
    modules.forEach((label, index) => {
      const boxY = 2.02 + index * 1.08;
      const accent = accentColor(pal, index);
      addPanel(slide, { x: 9.14, y: boxY, w: 2.84, h: 0.78 }, mixHex(pal.canvas, pal.card, 0.46), mixHex(pal.border, accent, 0.16), 0.05);
      addPanel(slide, { x: 9.34, y: boxY + 0.2, w: 0.08, h: 0.38 }, accent, accent, 0.02);
      addText(slide, label, { x: 9.56, y: boxY + 0.2, w: 1.82, h: 0.16 }, {
        fontFace: FONT_BODY,
        fontSize: 11,
        color: pal.text,
        bold: true,
        fit: "shrink",
      });
      addText(slide, index === 0 ? "context" : index === 1 ? "controls" : "execution", { x: 9.56, y: boxY + 0.46, w: 1.14, h: 0.12 }, {
        fontFace: FONT_BODY,
        fontSize: 10,
        color: pal.muted,
        charSpace: 0.5,
      });
    });
    addIcon(slide, motif, 10.84, 5.56, 0.74, {
      ...pal,
      accent: mixHex(pal.accent, pal.card, 0.14),
      accent2: mixHex(pal.accent2, pal.card, 0.16),
      text: "FFFFFF",
    });
    return;
  }

  if (family === "tech") {
    const modules = coverSectionLabels(manifest, 3);
    addFullRect(slide, pal.header);
    addPanel(slide, { x: 0, y: 0, w: SLIDE_W, h: 0.16 }, pal.accent, pal.accent, 0);
    addLine(slide, 8.66, 1.04, 0, 5.18, mixHex(pal.header, pal.accent, 0.22), 1.2);
    addPanel(slide, { x: 9.1, y: 1.14, w: 2.62, h: 0.38 }, mixHex(pal.header, pal.card, 0.08), mixHex(pal.header, pal.accent2, 0.18), 0.06);
    addText(slide, (manifest.deck_plan.subject || manifest.title).toUpperCase(), { x: 9.28, y: 1.23, w: 2.26, h: 0.14 }, {
      fontFace: FONT_BODY,
      fontSize: 7.9,
      color: pal.darkMuted,
      bold: true,
      charSpace: 0.5,
      fit: "shrink",
      align: "center",
    });
    addText(slide, kicker.toUpperCase(), { x: 1.0, y: 1.02, w: 4.8, h: 0.22 }, {
      fontFace: FONT_BODY,
      fontSize: 11,
      color: pal.darkMuted,
      charSpace: 1.3,
      bold: true,
    });
    addText(slide, title, { x: 0.96, y: 1.68, w: 7.24, h: 1.08 }, {
      fontFace: FONT_HEAD,
      fontSize: 27.5,
      bold: true,
      color: pal.text,
      breakLine: true,
      fit: "shrink",
      valign: "top",
    });
    addLine(slide, 1.02, 2.98, 2.7, 0, mixHex(pal.accent2, "FFFFFF", 0.18), 2.4);
    if (subtitle) {
      addText(slide, subtitle, { x: 1.0, y: 3.62, w: 7.4, h: 0.58 }, {
        fontFace: FONT_BODY,
        fontSize: 16,
        color: pal.text,
        breakLine: true,
      });
    }
    if (support) {
      addText(slide, support, { x: 1.0, y: 4.38, w: 7.1, h: 0.24 }, {
        fontFace: FONT_BODY,
        fontSize: 13,
        color: pal.darkMuted,
      });
    }
    if (meta) {
      addText(slide, meta, { x: 1.0, y: 6.82, w: 8.2, h: 0.22 }, {
        fontFace: FONT_BODY,
        fontSize: 12,
        color: pal.darkMuted,
      });
    }
    modules.forEach((label, index) => {
      const boxY = 2.08 + index * 1.1;
      const accent = accentColor(pal, index);
      addPanel(slide, { x: 8.98, y: boxY, w: 3.1, h: 0.84 }, mixHex(pal.header, pal.card, 0.1), mixHex(pal.header, accent, 0.16), 0.08);
      addPanel(slide, { x: 9.16, y: boxY + 0.18, w: 0.08, h: 0.48 }, accent, accent, 0.02);
      addText(slide, `${String(index + 1).padStart(2, "0")}  ${label}`, { x: 9.42, y: boxY + 0.22, w: 2.22, h: 0.18 }, {
        fontFace: FONT_BODY,
        fontSize: 11,
        color: pal.text,
        bold: true,
        charSpace: 0.3,
        fit: "shrink",
      });
    });
    addIcon(slide, motif, 11.18, 5.88, 0.72, {
      ...pal,
      accent: mixHex(pal.accent, pal.card, 0.12),
      accent2: mixHex(pal.accent2, pal.card, 0.16),
      text: "FFFFFF",
    });
    return;
  }

  addFullRect(slide, family === "playful" ? mixHex(pal.bg, pal.card, 0.72) : pal.header);
  addPanel(slide, { x: 0, y: 0, w: SLIDE_W, h: 0.18 }, pal.accent, pal.accent, 0);

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

  addText(slide, kicker.toUpperCase(), { x: 1.08, y: 1.12, w: 4.3, h: 0.25 }, {
    fontFace: FONT_BODY,
    fontSize: 12,
    color: family === "playful" ? pal.header : pal.darkMuted,
    charSpace: 1.4,
    bold: true,
    valign: "mid",
  });

  if (family === "proposal" && wordmark) {
    addText(slide, wordmark, { x: 1.02, y: 1.68, w: 4.2, h: 0.68 }, {
      fontFace: FONT_HEAD,
      fontSize: 36,
      bold: true,
      color: pal.accent,
      fit: "shrink",
      valign: "top",
    });
  }

  const titleY = family === "proposal" ? (wordmark ? 2.54 : 1.92) : 1.54;
  const titleH = family === "proposal" ? 1.04 : 1.78;
  const titleColor = family === "proposal" ? "FFFFFF" : (family === "studio" ? "FFFFFF" : pal.header);
  addText(slide, family === "proposal" ? (titleRemainder || title) : title, { x: 1.0, y: titleY, w: family === "proposal" ? 6.7 : 7.6, h: titleH }, {
    fontFace: FONT_HEAD,
    fontSize: family === "proposal" ? 27.5 : (family === "playful" ? 25 : 30),
    bold: true,
    color: titleColor,
    breakLine: true,
    fit: "shrink",
    valign: "top",
  });

  if (family === "proposal") {
    addLine(slide, 1.06, titleY + titleH + 0.1, 2.7, 0, "D6A33E", 2.5);
  }

  if (subtitle) {
    addText(slide, subtitle, { x: 1.02, y: family === "proposal" ? 4.02 : 3.18, w: family === "proposal" ? 6.55 : 7.4, h: 0.66 }, {
      fontFace: family === "proposal" ? "Georgia" : FONT_BODY,
      fontSize: family === "proposal" ? 16.2 : 17,
      italic: family === "proposal",
      color: family === "playful" ? mixHex(pal.header, "FFFFFF", 0.22) : "FFFFFF",
      breakLine: true,
    });
  }
  if (support) {
    addText(slide, support, { x: 1.04, y: family === "proposal" ? 4.86 : 4.02, w: family === "proposal" ? 6.4 : 6.8, h: 0.28 }, {
      fontFace: FONT_BODY,
      fontSize: family === "proposal" ? 10.5 : 11,
      color: family === "playful" ? mixHex(pal.header, "FFFFFF", 0.42) : pal.darkMuted,
    });
  }
  if (family === "proposal") {
    const coverSlides = manifest.slides.slice(0, 3);
    addMetaPill(slide, "STRUCTURED OPERATING PLAN", 8.88, 1.08, 3.08, pal);
    coverSlides.forEach((entry, index) => {
      addCoverSideCard(
        slide,
        slideChipLabel(entry),
        entry.heading || `Section ${index + 1}`,
        proposalLeadText(entry, entry.points ?? []),
        { x: 8.88, y: 1.62 + index * 1.38, w: 3.08, h: 1.18 },
        slideIconToken(entry),
        pal,
      );
    });
    const proposalMetaLabel = manifest.deck_plan.audience?.trim() || manifest.deck_plan.subject?.trim() || manifest.title?.trim() || "";
    addMetaPill(slide, proposalMetaLabel, 8.88, 5.86, 3.08, pal);
    const footer = manifest.deck_plan.subject.trim() || manifest.title;
    addText(slide, footer, { x: 1.08, y: 7.02, w: 7.6, h: 0.22 }, {
      fontFace: FONT_BODY,
      fontSize: 12,
      color: pal.darkMuted,
    });
  } else if (meta) {
    addText(slide, meta, { x: 1.04, y: 5.02, w: 7.1, h: 0.32 }, {
      fontFace: "Georgia",
      fontSize: 12,
      italic: true,
      color: family === "playful" ? mixHex(pal.header, "FFFFFF", 0.48) : pal.darkMuted,
    });
  }
}

function renderSlideChrome(slide: PptxSlide, content: SlidePlan, _totalSlides: number, family: RenderFamily, pal: RenderPalette): Bounds {
  if (family === "civic") {
    addFullRect(slide, pal.canvas);
    addPanel(slide, { x: 0, y: 0, w: SLIDE_W, h: 0.12 }, pal.accent, pal.accent, 0);
    addIcon(slide, slideIconToken(content), 0.42, 0.26, 0.24, {
      ...pal,
      accent: pal.accent,
      accent2: mixHex(pal.accent2, pal.card, 0.2),
      text: "FFFFFF",
    });
    addText(slide, shortTitle(content.heading || "Untitled Slide"), { x: 0.78, y: 0.16, w: 8.6, h: 0.32 }, {
      fontFace: FONT_BODY,
      fontSize: 18,
      color: pal.text,
      bold: true,
      charSpace: 0.2,
    });
    addPanel(slide, { x: 10.78, y: 0.2, w: 1.9, h: 0.28 }, mixHex(pal.canvas, pal.card, 0.18), mixHex(pal.border, pal.accent, 0.18), 0.05);
    addText(slide, slideChipLabel(content), { x: 10.9, y: 0.245, w: 1.64, h: 0.14 }, {
      fontFace: FONT_BODY,
      fontSize: 10,
      color: pal.text,
      bold: true,
      align: "center",
      fit: "shrink",
      charSpace: 0.4,
    });
    return { x: 0.52, y: 0.92, w: 12.25, h: 6.08 };
  }

  if (family === "tech") {
    addFullRect(slide, pal.canvas);
    addPanel(slide, { x: 0, y: 0, w: SLIDE_W, h: 0.48 }, pal.header, pal.header, 0);
    addPanel(slide, { x: 0, y: 0.48, w: SLIDE_W, h: 0.04 }, mixHex(pal.header, pal.accent, 0.28), mixHex(pal.header, pal.accent, 0.28), 0);
    addIcon(slide, slideIconToken(content), 0.38, 0.1, 0.28, {
      ...pal,
      accent: mixHex(pal.header, pal.accent, 0.18),
      accent2: mixHex(pal.header, pal.accent2, 0.18),
      text: "FFFFFF",
    });
    addText(slide, shortTitle(content.heading || "Untitled Slide"), { x: 0.78, y: 0.1, w: 8.5, h: 0.3 }, {
      fontFace: FONT_BODY,
      fontSize: 17,
      color: pal.text,
      bold: true,
      charSpace: 0.3,
    });
    addPanel(slide, { x: 10.78, y: 0.08, w: 1.9, h: 0.28 }, mixHex(pal.header, pal.card, 0.1), mixHex(pal.header, pal.accent2, 0.14), 0.08);
    addText(slide, slideChipLabel(content), { x: 10.9, y: 0.125, w: 1.64, h: 0.14 }, {
      fontFace: FONT_BODY,
      fontSize: 10,
      color: pal.darkMuted,
      bold: true,
      align: "center",
      fit: "shrink",
      charSpace: 0.4,
    });
    return { x: 0.5, y: 0.74, w: 12.3, h: 6.28 };
  }

  addFullRect(slide, family === "proposal" ? pal.canvas : pal.bg);
  const headFill = family === "playful" ? mixHex(pal.header, "FFFFFF", 0.08) : pal.header;
  addPanel(slide, { x: 0, y: 0, w: SLIDE_W, h: 0.54 }, headFill, headFill, 0);
  if (family !== "playful") {
    addPanel(slide, { x: 0, y: 0.54, w: SLIDE_W, h: 0.04 }, mixHex(pal.canvas, pal.card, 0.2), mixHex(pal.canvas, pal.card, 0.2), 0);
  }

  addIcon(slide, slideIconToken(content), 0.38, 0.12, 0.28, {
    ...pal,
    accent: mixHex(headFill, pal.accent, 0.44),
    accent2: mixHex(headFill, pal.accent2, 0.44),
    text: "FFFFFF",
  });
  addText(slide, shortTitle(content.heading || "Untitled Slide"), { x: 0.78, y: 0.1, w: 8.4, h: 0.32 }, {
    fontFace: FONT_BODY,
    fontSize: 18,
    color: "FFFFFF",
    bold: true,
    charSpace: 0.3,
  });
  addPanel(slide, { x: 10.78, y: 0.1, w: 1.9, h: 0.28 }, mixHex(headFill, pal.accent, 0.48), mixHex(headFill, pal.card, 0.1), 0.08);
  addText(slide, slideChipLabel(content), { x: 10.9, y: 0.145, w: 1.64, h: 0.14 }, {
    fontFace: FONT_BODY,
    fontSize: 10,
    color: "FFFFFF",
    bold: true,
    align: "center",
    fit: "shrink",
    charSpace: 0.5,
  });
  return { x: 0.48, y: 0.72, w: 12.37, h: 6.36 };
}

function renderMetricTile(slide: PptxSlide, stat: Stat, bounds: Bounds, fill: string, border: string, accent: string, valueColor: string, textColor: string): void {
  addPanel(slide, bounds, fill, border, 0.12);
  addPanel(slide, { x: bounds.x, y: bounds.y, w: bounds.w, h: 0.04 }, accent, accent, 0);
  addText(slide, stat.value, { x: bounds.x + 0.18, y: bounds.y + 0.2, w: bounds.w - 0.36, h: 0.56 }, {
    fontFace: FONT_HEAD,
    fontSize: 28,
    bold: true,
    color: valueColor,
  });
  addText(slide, stat.label.toUpperCase(), { x: bounds.x + 0.18, y: bounds.y + 0.8, w: bounds.w - 0.36, h: 0.22 }, {
    fontFace: FONT_BODY,
    fontSize: 11,
    bold: true,
    color: textColor,
    charSpace: 0.6,
  });
  if (stat.desc?.trim()) {
    addText(slide, trimSentence(stat.desc.trim(), 74), { x: bounds.x + 0.18, y: bounds.y + 1.06, w: bounds.w - 0.36, h: bounds.h - 1.2 }, {
      fontFace: FONT_BODY,
      fontSize: 13,
      color: textColor,
      valign: "top",
      breakLine: true,
      fit: "shrink",
    });
  }
}

function renderStats(slide: PptxSlide, plan: SlidePlan, bounds: Bounds, family: RenderFamily, pal: RenderPalette): void {
  const stats = (plan.stats ?? []).slice(0, 4);
  if (stats.length === 0) return renderBullets(slide, plan, bounds, family, pal);

  const cp = parseComposition(plan.design);

  if (cp.splitRatio > 0 && stats.length >= 2) {
    // SPLIT HERO: big single stat + supporting stats
    const hero = stats[0];
    const rest = stats.slice(1);
    const heroW = bounds.w * cp.splitRatio;
    const rightX = bounds.x + heroW + 0.2;
    const rightW = bounds.w - heroW - 0.2;
    const heroFill = cp.panelFilled ? pal.header : mixHex(pal.canvas, pal.card, 0.86);
    const heroTextColor = cp.panelFilled ? "FFFFFF" : pal.text;
    const heroBorder = cp.panelFilled ? mixHex(pal.header, pal.accent, 0.12) : mixHex(pal.canvas, pal.border, 0.84);
    addPanel(slide, { x: bounds.x, y: bounds.y, w: heroW, h: bounds.h }, heroFill, heroBorder, 0.06);
    addPanel(slide, { x: bounds.x, y: bounds.y, w: heroW, h: 0.06 }, pal.accent, pal.accent, 0);
    const valueFontSize = Math.round(Math.min(72, 580 / Math.max(hero.value.length, 2)));
    addText(slide, hero.value, { x: bounds.x + 0.22, y: bounds.y + 0.5, w: heroW - 0.44, h: 1.6 }, {
      fontFace: FONT_HEAD, fontSize: valueFontSize, bold: true, color: cp.panelFilled ? pal.accent : accentColor(pal, 0), valign: "mid",
    });
    addText(slide, hero.label.toUpperCase(), { x: bounds.x + 0.22, y: bounds.y + 2.2, w: heroW - 0.44, h: 0.28 }, {
      fontFace: FONT_BODY, fontSize: 13, bold: true, color: cp.panelFilled ? pal.darkMuted : pal.lightMuted, charSpace: 0.8,
    });
    if (hero.desc?.trim()) {
      addText(slide, hero.desc.trim(), { x: bounds.x + 0.22, y: bounds.y + 2.58, w: heroW - 0.44, h: bounds.h - 2.82 }, {
        fontFace: FONT_BODY, fontSize: 13, color: cp.panelFilled ? pal.darkMuted : pal.lightMuted, breakLine: true, valign: "top", fit: "shrink",
      });
    }
    const gapY = 0.16;
    const tileH = (bounds.h - gapY * Math.max(rest.length - 1, 0)) / Math.max(rest.length, 1);
    rest.forEach((stat, index) => {
      const accent = accentColor(pal, index + 1);
      renderMetricTile(slide, stat, { x: rightX, y: bounds.y + index * (tileH + gapY), w: rightW, h: tileH }, pal.card, mixHex(pal.canvas, pal.border, 0.84), accent, accent, pal.lightMuted);
    });
    return;
  }

  // DEFAULT GRID
  const startY = addTopSummaryStrip(slide, plan, bounds, pal, "chart");
  const rows = stats.length > 2 ? 2 : 1;
  const cols = stats.length === 1 ? 1 : 2;
  const gapX = 0.18;
  const gapY = 0.18;
  const bandH = stats.length > 1 ? 0.52 : 0;
  const tileAreaH = bounds.h - (startY - bounds.y) - bandH - (bandH > 0 ? 0.16 : 0);
  const tileW = (bounds.w - gapX * Math.max(cols - 1, 0)) / cols;
  const tileH = (tileAreaH - gapY * Math.max(rows - 1, 0)) / rows;
  stats.forEach((stat, index) => {
    const row = rows === 1 ? 0 : Math.floor(index / 2);
    const col = cols === 1 ? 0 : index % 2;
    const accent = accentColor(pal, index);
    const fill = index === 0 && family === "proposal" ? pal.header : (index % 2 === 0 ? pal.card : mixHex(pal.canvas, pal.card, 0.74));
    const border = mixHex(pal.canvas, pal.border, 0.84);
    const valueColor = fill === pal.header ? accent : accent;
    const bodyColor = fill === pal.header ? pal.darkMuted : pal.lightMuted;
    renderMetricTile(slide, stat, { x: bounds.x + col * (tileW + gapX), y: startY + row * (tileH + gapY), w: tileW, h: tileH }, fill, border, accent, valueColor, bodyColor);
  });
  if (bandH > 0) {
    addPanel(slide, { x: bounds.x, y: bounds.y + bounds.h - bandH, w: bounds.w, h: bandH }, mixHex(pal.canvas, pal.header, 0.08), mixHex(pal.canvas, pal.border, 0.76), 0.04);
    addText(slide, proposalStatsTakeaway(stats), { x: bounds.x + 0.18, y: bounds.y + bounds.h - bandH + 0.13, w: bounds.w - 0.36, h: bandH - 0.2 }, {
      fontFace: FONT_BODY, fontSize: 12, color: pal.text, bold: true, fit: "shrink",
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
  const rowGap = items.length > 4 ? 0.12 : 0.14;
  const rowH = items.length > 4 ? 0.54 : 0.68;
  items.forEach((item, index) => {
    const top = bounds.y + index * (rowH + rowGap);
    const { title, desc } = splitCardText(item);
    slide.addShape("ellipse", {
      x: bounds.x,
      y: top + 0.16,
      w: 0.1,
      h: 0.1,
      line: { color: accentColor(pal, offset + index), transparency: 100 },
      fill: { color: accentColor(pal, offset + index) },
    });
    addText(slide, title, { x: bounds.x + 0.18, y: top + 0.01, w: bounds.w - 0.18, h: 0.26 }, {
      fontFace: FONT_BODY,
      fontSize: 16,
      color: pal.text,
      bold: true,
      valign: "top",
    });
    if (desc) {
      addText(slide, desc, { x: bounds.x + 0.18, y: top + 0.3, w: bounds.w - 0.18, h: rowH - 0.18 }, {
        fontFace: FONT_BODY,
        fontSize: 13,
        color: pal.lightMuted,
        valign: "top",
        breakLine: true,
      });
    }
  });
}

function renderBullets(slide: PptxSlide, plan: SlidePlan, bounds: Bounds, family: RenderFamily, pal: RenderPalette): void {
  const cp = parseComposition(plan.design);
  const points = (plan.points ?? []).slice(0, 8);
  const stats = (plan.stats ?? []).slice(0, 3);

  if (cp.splitRatio > 0) {
    // SPLIT COMPOSITION: lead panel | content panel
    const leadW = bounds.w * cp.splitRatio;
    const contentX = bounds.x + leadW + 0.16;
    const contentW = bounds.w - leadW - 0.16;
    const lead = proposalLeadText(plan, points) || buildLead(plan);

    const panelFill = cp.panelFilled ? pal.header : mixHex(pal.canvas, pal.card, 0.82);
    const panelBorder = cp.panelFilled ? mixHex(pal.header, pal.accent, 0.12) : mixHex(pal.canvas, pal.border, 0.84);
    const leadColor = cp.panelFilled ? "FFFFFF" : pal.text;
    const mutedColor = cp.panelFilled ? pal.darkMuted : pal.lightMuted;
    const radius = cp.panelGlass ? 0.08 : 0;

    addPanel(slide, { x: bounds.x, y: bounds.y, w: leadW, h: bounds.h }, panelFill, panelBorder, radius);
    if (cp.accentBand) {
      addPanel(slide, { x: bounds.x, y: bounds.y, w: leadW, h: 0.06 }, pal.accent, pal.accent, 0);
    } else {
      addPanel(slide, { x: bounds.x, y: bounds.y, w: 0.06, h: bounds.h }, pal.accent, pal.accent, 0.03);
    }
    addText(slide, sectionKicker(plan), { x: bounds.x + 0.22, y: bounds.y + 0.22, w: leadW - 0.44, h: 0.22 }, {
      fontFace: FONT_BODY, fontSize: Math.round(11 * cp.density), bold: true, color: mutedColor, charSpace: 0.8,
    });
    addText(slide, lead, { x: bounds.x + 0.22, y: bounds.y + 0.58, w: leadW - 0.44, h: bounds.h - 0.92 }, {
      fontFace: FONT_HEAD, fontSize: Math.round(22 * cp.density), bold: true, color: leadColor,
      breakLine: true, valign: "top", fit: "shrink",
    });

    const listY = bounds.y + 0.1;
    addText(slide, proposalSectionLabel(plan), { x: contentX, y: listY, w: contentW, h: 0.22 }, {
      fontFace: FONT_BODY, fontSize: Math.round(12 * cp.density), bold: true, color: pal.lightMuted, charSpace: 0.8,
    });
    const cols = cp.columns >= 2 ? 2 : 1;
    const [left, right] = cols === 2 ? pointColumns(points) : [points, [] as string[]];
    const gap = right.length > 0 ? 0.24 : 0;
    const colW = right.length > 0 ? (contentW - gap) / 2 : contentW;
    renderPointList(slide, left, { x: contentX, y: listY + 0.28, w: colW, h: bounds.h - (listY + 0.28 - bounds.y) }, pal, 0);
    if (right.length > 0) {
      renderPointList(slide, right, { x: contentX + colW + gap, y: listY + 0.28, w: colW, h: bounds.h - (listY + 0.28 - bounds.y) }, pal, left.length);
    }
    return;
  }

  if (cp.heroFirst && points.length > 0) {
    // HERO-FIRST: big lead block above, columns below
    const heroH = Math.min(bounds.h * (0.28 + cp.density * 0.12), 2.6);
    const bodyY = bounds.y + heroH + 0.18;
    const bodyH = bounds.h - heroH - 0.18;
    const lead = proposalLeadText(plan, points) || buildLead(plan);
    addPanel(slide, { x: bounds.x, y: bounds.y, w: bounds.w, h: heroH }, pal.canvas, mixHex(pal.canvas, pal.border, 0.82), 0.08);
    if (cp.accentBand) {
      addPanel(slide, { x: bounds.x, y: bounds.y, w: bounds.w, h: 0.06 }, pal.accent, pal.accent, 0);
    } else {
      addPanel(slide, { x: bounds.x, y: bounds.y, w: 0.08, h: heroH }, pal.accent, pal.accent, 0.03);
    }
    addText(slide, lead, { x: bounds.x + 0.28, y: bounds.y + 0.18, w: bounds.w - 0.56, h: heroH - 0.36 }, {
      fontFace: FONT_HEAD, fontSize: Math.round(26 * cp.density), bold: true, color: pal.text,
      breakLine: true, valign: "mid", fit: "shrink",
    });
    const cols = cp.columns >= 2 ? 2 : (points.length >= 4 ? 2 : 1);
    const [left, right] = cols === 2 ? pointColumns(points) : [points, [] as string[]];
    const gap = right.length > 0 ? 0.28 : 0;
    const colW = right.length > 0 ? (bounds.w - gap) / 2 : bounds.w;
    renderPointList(slide, left, { x: bounds.x, y: bodyY, w: colW, h: bodyH }, pal, 0);
    if (right.length > 0) {
      renderPointList(slide, right, { x: bounds.x + colW + gap, y: bodyY, w: colW, h: bodyH }, pal, left.length);
    }
    return;
  }

  // DEFAULT STACK
  const startY = addTopSummaryStrip(slide, plan, bounds, pal, slideIconToken(plan));
  let cursorY = startY;
  if (stats.length > 0) {
    const gap = 0.14;
    const cardCount = Math.min(stats.length, 3);
    const cardW = (bounds.w - gap * Math.max(cardCount - 1, 0)) / cardCount;
    const cardH = 0.94;
    stats.slice(0, cardCount).forEach((stat, index) => {
      renderMetricTile(slide, stat, { x: bounds.x + index * (cardW + gap), y: bounds.y, w: cardW, h: cardH }, pal.card, mixHex(pal.canvas, pal.border, 0.84), accentColor(pal, index), pal.text, pal.lightMuted);
    });
    cursorY += cardH + 0.18;
  }
  const lead = proposalLeadText(plan, points);
  if (lead && points.length >= 3) {
    const leadW = bounds.w * 0.30;
    addPanel(slide, { x: bounds.x, y: cursorY, w: leadW, h: bounds.h - (cursorY - bounds.y) }, mixHex(pal.canvas, pal.card, 0.78), mixHex(pal.canvas, pal.border, 0.84), 0.08);
    addText(slide, "WHY IT MATTERS", { x: bounds.x + 0.18, y: cursorY + 0.18, w: leadW - 0.36, h: 0.22 }, {
      fontFace: FONT_BODY, fontSize: 12, color: pal.lightMuted, bold: true, charSpace: 0.7,
    });
    addText(slide, lead, { x: bounds.x + 0.18, y: cursorY + 0.48, w: leadW - 0.36, h: 1.18 }, {
      fontFace: FONT_HEAD, fontSize: 19, color: pal.text, bold: true, breakLine: true, valign: "top", fit: "shrink",
    });
    addText(slide, sectionKicker(plan), { x: bounds.x + 0.18, y: cursorY + bounds.h - (cursorY - bounds.y) - 0.38, w: leadW - 0.36, h: 0.22 }, {
      fontFace: FONT_BODY, fontSize: 12, color: pal.lightMuted, bold: true, charSpace: 0.6,
    });
    const listX = bounds.x + leadW + 0.26;
    const listW = bounds.w - leadW - 0.26;
    addText(slide, proposalSectionLabel(plan), { x: listX, y: cursorY + 0.02, w: 3.2, h: 0.22 }, {
      fontFace: FONT_BODY, fontSize: 12, bold: true, color: pal.lightMuted, charSpace: 0.8,
    });
    const [left, right] = pointColumns(points);
    const gap = right.length > 0 ? 0.24 : 0;
    const colW = right.length > 0 ? (listW - gap) / 2 : listW;
    renderPointList(slide, left, { x: listX, y: cursorY + 0.28, w: colW, h: bounds.h - (cursorY + 0.28 - bounds.y) }, pal, 0);
    if (right.length > 0) {
      renderPointList(slide, right, { x: listX + colW + gap, y: cursorY + 0.28, w: colW, h: bounds.h - (cursorY + 0.28 - bounds.y) }, pal, left.length);
    }
    return;
  }
  if (lead) {
    addText(slide, lead, { x: bounds.x, y: cursorY, w: bounds.w, h: 0.52 }, {
      fontFace: FONT_BODY, fontSize: 15, color: pal.lightMuted, breakLine: true, fit: "shrink",
    });
    cursorY += 0.6;
  }
  addText(slide, proposalSectionLabel(plan), { x: bounds.x, y: cursorY, w: 3.2, h: 0.22 }, {
    fontFace: FONT_BODY, fontSize: 12, bold: true, color: pal.lightMuted, charSpace: 0.8,
  });
  cursorY += 0.22;
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
  addIcon(slide, token, bounds.x + 0.18, bounds.y + 0.14, 0.32, {
    ...pal,
    accent: accentColor(pal, index),
    accent2: mixHex(accentColor(pal, index), pal.card, 0.4),
    text: "FFFFFF",
  });
  addText(slide, meta.toUpperCase(), { x: bounds.x + 0.62, y: bounds.y + 0.1, w: 1.56, h: 0.22 }, {
    fontFace: FONT_BODY,
    fontSize: 12,
    bold: true,
    color: pal.lightMuted,
    charSpace: 0.7,
  });
  addText(slide, title, { x: bounds.x + 0.62, y: bounds.y + 0.34, w: bounds.w - 2.3, h: 0.28 }, {
    fontFace: FONT_HEAD,
    fontSize: 17,
    bold: true,
    color: pal.text,
  });
  if (desc) {
    addText(slide, desc, { x: bounds.x + 0.62, y: bounds.y + 0.68, w: bounds.w - 2.58, h: bounds.h - 0.76 }, {
      fontFace: FONT_BODY,
      fontSize: 13,
      color: pal.lightMuted,
      valign: "top",
      breakLine: true,
    });
  }
  addPanel(slide, { x: bounds.x + bounds.w - 1.04, y: bounds.y + 0.16, w: 0.82, h: 0.2 }, mixHex(pal.canvas, accentColor(pal, index), 0.14), mixHex(pal.canvas, pal.border, 0.8), 0.08);
  addText(slide, `${index + 1}`.padStart(2, "0"), { x: bounds.x + bounds.w - 0.92, y: bounds.y + 0.2, w: 0.54, h: 0.1 }, {
    fontFace: FONT_BODY,
    fontSize: 10,
    bold: true,
    color: pal.text,
    align: "center",
  });
}

function renderSteps(slide: PptxSlide, plan: SlidePlan, bounds: Bounds, pal: RenderPalette): void {
  const steps = (plan.steps ?? plan.points ?? []).slice(0, 6);
  const cp = parseComposition(plan.design);
  const startY = addTopSummaryStrip(slide, plan, bounds, pal, slideIconToken(plan));

  if (cp.horizontal) {
    // HORIZONTAL RAIL
    const n = Math.max(steps.length, 1);
    const nodeW = bounds.w / n;
    const railY = startY + (bounds.h - startY) * 0.28;
    addPanel(slide, { x: bounds.x, y: railY + 0.14, w: bounds.w, h: 0.05 }, mixHex(pal.canvas, pal.border, 0.6), mixHex(pal.canvas, pal.border, 0.6), 0);
    steps.forEach((step, index) => {
      const { title, desc } = splitCardText(step);
      const accent = accentColor(pal, index);
      const nodeX = bounds.x + index * nodeW + nodeW / 2;
      slide.addShape("ellipse", { x: nodeX - 0.18, y: railY, w: 0.36, h: 0.36, fill: { color: accent }, line: { color: accent, pt: 0 } });
      addText(slide, `${index + 1}`, { x: nodeX - 0.18, y: railY + 0.04, w: 0.36, h: 0.22 }, {
        fontFace: FONT_HEAD, fontSize: 14, bold: true, color: "FFFFFF", align: "center",
      });
      addText(slide, (title || step).toUpperCase(), { x: bounds.x + index * nodeW, y: railY - 0.36, w: nodeW, h: 0.22 }, {
        fontFace: FONT_BODY, fontSize: Math.round(10 * cp.density), bold: true, color: accent, align: "center", charSpace: 0.5,
      });
      const cardY = railY + 0.54;
      const cardH = bounds.h - cardY - bounds.y + bounds.y;
      addPanel(slide, { x: bounds.x + index * nodeW + 0.1, y: cardY, w: nodeW - 0.2, h: cardH }, pal.card, mixHex(pal.canvas, pal.border, 0.84), 0.08);
      addPanel(slide, { x: bounds.x + index * nodeW + 0.1, y: cardY, w: nodeW - 0.2, h: 0.04 }, accent, accent, 0);
      addText(slide, title || step, { x: bounds.x + index * nodeW + 0.2, y: cardY + 0.12, w: nodeW - 0.4, h: 0.32 }, {
        fontFace: FONT_HEAD, fontSize: Math.round(14 * cp.density), bold: true, color: pal.text, breakLine: true, valign: "top",
      });
      if (desc) {
        addText(slide, desc, { x: bounds.x + index * nodeW + 0.2, y: cardY + 0.5, w: nodeW - 0.4, h: cardH - 0.6 }, {
          fontFace: FONT_BODY, fontSize: Math.round(12 * cp.density), color: pal.lightMuted, breakLine: true, valign: "top", fit: "shrink",
        });
      }
    });
    return;
  }

  // DEFAULT VERTICAL ROWS
  const rowGap = 0.14;
  const rowH = (bounds.h - (startY - bounds.y) - rowGap * Math.max(steps.length - 1, 0)) / Math.max(steps.length, 1);
  steps.forEach((step, index) => {
    const { title, desc } = splitCardText(step);
    renderRoadmapRow(slide, index, title || step, `Step ${index + 1}`, desc, inferIconToken(title || step) || "strategy", {
      x: bounds.x,
      y: startY + index * (rowH + rowGap),
      w: bounds.w,
      h: rowH,
    }, pal);
  });
}

function renderCards(slide: PptxSlide, plan: SlidePlan, bounds: Bounds, family: RenderFamily, pal: RenderPalette): void {
  const cards = (plan.cards ?? []).slice(0, 6);
  if (cards.length === 0) return renderBullets(slide, plan, bounds, family, pal);

  const cp = parseComposition(plan.design);
  const startY = addTopSummaryStrip(slide, plan, bounds, pal, slideIconToken(plan));

  if (cp.horizontal || cp.splitRatio === 0 && cards.length <= 4 && /matrix|row|list|stack|rail/.test((plan.design?.layout_style ?? "").toLowerCase())) {
    // MATRIX ROWS: full-width horizontal card rows
    const gapY = 0.14;
    const cardH = (bounds.h - (startY - bounds.y) - gapY * Math.max(cards.length - 1, 0)) / Math.max(cards.length, 1);
    const iconStripW = 0.68;
    cards.forEach((card, index) => {
      const y = startY + index * (cardH + gapY);
      const accent = accentColor(pal, index);
      const fill = index % 2 === 0 ? pal.card : mixHex(pal.canvas, pal.card, 0.74);
      addPanel(slide, { x: bounds.x, y, w: bounds.w, h: cardH }, fill, mixHex(pal.canvas, pal.border, 0.84), 0.06);
      addPanel(slide, { x: bounds.x, y, w: 0.06, h: cardH }, accent, accent, 0.03);
      addIcon(slide, card.icon || card.title, bounds.x + 0.14, y + (cardH - 0.36) / 2, Math.min(0.36, cardH * 0.52), { ...pal, accent, accent2: mixHex(accent, pal.card, 0.4), text: "FFFFFF" });
      const titleW = bounds.w * 0.36;
      addText(slide, card.title, { x: bounds.x + iconStripW, y: y + 0.08, w: titleW, h: cardH - 0.16 }, {
        fontFace: FONT_HEAD, fontSize: Math.round(14 * cp.density), bold: true, color: pal.text, valign: "mid",
      });
      addText(slide, card.desc?.trim() || card.title, { x: bounds.x + iconStripW + titleW + 0.12, y: y + 0.08, w: bounds.w - iconStripW - titleW - 0.24, h: cardH - 0.16 }, {
        fontFace: FONT_BODY, fontSize: Math.round(12 * cp.density), color: pal.lightMuted, valign: "mid", breakLine: true, fit: "shrink",
      });
    });
    return;
  }

  // DEFAULT GRID
  const cols = cards.length <= 2 ? 2 : (cp.columns >= 3 ? 3 : (cards.length <= 4 ? 2 : 3));
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
    addIcon(slide, card.icon || card.title, x + 0.18, y + 0.18, 0.42, { ...pal, accent: accentColor(pal, index), accent2: mixHex(accentColor(pal, index), pal.card, 0.4), text: "FFFFFF" });
    addText(slide, card.title, { x: x + 0.72, y: y + 0.18, w: cardW - 0.9, h: 0.3 }, {
      fontFace: FONT_HEAD, fontSize: Math.round(16 * cp.density), bold: true, color: pal.text,
    });
    addText(slide, card.desc?.trim() || card.title, { x: x + 0.18, y: y + 0.76, w: cardW - 0.36, h: cardH - 0.92 }, {
      fontFace: FONT_BODY, fontSize: Math.round(13 * cp.density), color: pal.lightMuted, valign: "top", breakLine: true,
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

function renderChart(slide: PptxSlide, plan: SlidePlan, bounds: Bounds, family: RenderFamily, pal: RenderPalette): void {
  const categories = plan.chart_categories ?? [];
  const series = (plan.chart_series ?? []).filter((entry) => Array.isArray(entry.values) && entry.values.length > 0);
  if (categories.length === 0 || series.length === 0) {
    return renderBullets(slide, plan, bounds, family, pal);
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
    catAxisLabelFontSize: 12,
    catAxisLineColor: mixHex(pal.canvas, pal.border, 0.52),
    catAxisLineShow: true,
    catAxisMajorTickMark: "none",
    catAxisLabelRotate: categories.some((label) => label.length > 7) ? -35 : 0,
    valAxisLabelColor: pal.lightMuted,
    valAxisLabelFontFace: FONT_BODY,
    valAxisLabelFontSize: 12,
    valAxisLineShow: false,
    valGridLine: { color: mixHex(pal.canvas, pal.border, 0.4), size: 1, style: "solid" },
    valAxisTitle: plan.y_label || "",
    valAxisTitleColor: pal.lightMuted,
    valAxisTitleFontFace: FONT_BODY,
    valAxisTitleFontSize: 12,
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
    fontSize: 11,
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
      fontSize: 13,
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
  const startY = addTopSummaryStrip(slide, plan, bounds, pal, slideIconToken(plan));
  const rowGap = 0.14;
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

function renderTable(slide: PptxSlide, plan: SlidePlan, bounds: Bounds, family: RenderFamily, pal: RenderPalette): void {
  const table = plan.table;
  if (!table || table.headers.length === 0) {
    return renderBullets(slide, plan, bounds, family, pal);
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
        fontSize: 13,
        align: "left",
        margin: 0.08,
      },
    })),
    ...table.rows.map((row) =>
      row.map((cell) => ({
        text: cell,
        options: {
          fontFace: FONT_BODY,
          fontSize: 12,
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
    fontSize: 12,
    color: pal.text,
    fill: { color: pal.card },
    autoPage: false,
  });
}

function renderSection(slide: PptxSlide, plan: SlidePlan, bounds: Bounds, family: RenderFamily, pal: RenderPalette): void {
  const fill = family === "proposal" ? pal.header : mixHex(pal.header, pal.accent, 0.12);
  addPanel(slide, { x: bounds.x + 0.1, y: bounds.y + 0.42, w: bounds.w - 0.2, h: 4.35 }, fill, mixHex(fill, pal.card, 0.14), 0.08);
  addText(slide, slideChipLabel(plan), { x: bounds.x + 0.42, y: bounds.y + 0.82, w: 3.2, h: 0.2 }, {
    fontFace: FONT_BODY,
    fontSize: 11,
    bold: true,
    color: pal.darkMuted,
    charSpace: 0.9,
  });
  addText(slide, plan.heading, { x: bounds.x + 0.38, y: bounds.y + 1.22, w: bounds.w - 0.76, h: 1.1 }, {
    fontFace: FONT_HEAD,
    fontSize: 36,
    bold: true,
    color: family === "proposal" ? "FFFFFF" : pal.text,
    breakLine: true,
  });
  addText(slide, buildLead(plan), { x: bounds.x + 0.42, y: bounds.y + 2.68, w: bounds.w - 0.84, h: 0.9 }, {
    fontFace: FONT_BODY,
    fontSize: 16,
    color: family === "proposal" ? pal.darkMuted : pal.lightMuted,
    breakLine: true,
  });
}

// -----------------------------------------------------------------------------
// LLM-HTML rendering path. When the LLM provides slide.html, we render the
// authored content directly via PptxGenJS — no hardcoded section labels,
// no fixed "WHY IT MATTERS" / "KEY POINTS" / "OVERVIEW" chrome. This is
// what the user sees in the main UI; the .pptx must mirror it.
// -----------------------------------------------------------------------------

type ExtractedHTMLBlock =
  | { kind: "heading"; level: number; text: string }
  | { kind: "paragraph"; text: string }
  | { kind: "list"; items: string[] };

function decodeHTMLEntities(s: string): string {
  return s
    .replace(/&nbsp;/gi, " ")
    .replace(/&amp;/gi, "&")
    .replace(/&lt;/gi, "<")
    .replace(/&gt;/gi, ">")
    .replace(/&quot;/gi, '"')
    .replace(/&#39;/gi, "'")
    .replace(/&apos;/gi, "'")
    .replace(/&mdash;/gi, "—")
    .replace(/&ndash;/gi, "–")
    .replace(/&hellip;/gi, "…")
    .replace(/&#(\d+);/g, (_, n) => String.fromCharCode(Number(n)));
}

function stripTagsToText(s: string): string {
  return decodeHTMLEntities(s.replace(/<[^>]+>/g, " ")).replace(/\s+/g, " ").trim();
}

function extractHTMLBlocks(html: string): ExtractedHTMLBlock[] {
  const blocks: ExtractedHTMLBlock[] = [];
  if (!html) return blocks;
  // strip script/style entirely
  let h = html
    .replace(/<script\b[^>]*>[\s\S]*?<\/script>/gi, "")
    .replace(/<style\b[^>]*>[\s\S]*?<\/style>/gi, "");

  // Pull out headings, paragraphs, and lists in document order.
  const tagRe = /<(h[1-6]|p|ul|ol)\b[^>]*>([\s\S]*?)<\/\1>/gi;
  let m: RegExpExecArray | null;
  while ((m = tagRe.exec(h)) !== null) {
    const tag = m[1].toLowerCase();
    const inner = m[2];
    if (tag === "ul" || tag === "ol") {
      const items: string[] = [];
      const liRe = /<li\b[^>]*>([\s\S]*?)<\/li>/gi;
      let li: RegExpExecArray | null;
      while ((li = liRe.exec(inner)) !== null) {
        const t = stripTagsToText(li[1]);
        if (t) items.push(t);
      }
      if (items.length > 0) blocks.push({ kind: "list", items });
    } else if (tag.startsWith("h")) {
      const text = stripTagsToText(inner);
      if (text) blocks.push({ kind: "heading", level: Number(tag[1]), text });
    } else {
      const text = stripTagsToText(inner);
      if (text) blocks.push({ kind: "paragraph", text });
    }
  }

  if (blocks.length === 0) {
    const fallback = stripTagsToText(h);
    if (fallback) blocks.push({ kind: "paragraph", text: fallback });
  }
  return blocks;
}

function llmHTMLHasSubstance(html: string | undefined): boolean {
  if (!html) return false;
  const text = stripTagsToText(html);
  return text.length >= 56;
}

function renderFromLLMHTML(slide: PptxSlide, plan: SlidePlan, bounds: Bounds, _family: RenderFamily, pal: RenderPalette): void {
  const blocks = extractHTMLBlocks(plan.html || "");

  // Extract leading heading (if any) and remaining body blocks.
  let leadingHeading = "";
  const bodyBlocks: ExtractedHTMLBlock[] = [];
  for (const b of blocks) {
    if (!leadingHeading && b.kind === "heading") {
      leadingHeading = b.text;
    } else {
      bodyBlocks.push(b);
    }
  }

  let cursorY = bounds.y;
  const innerX = bounds.x + 0.32;
  const innerW = bounds.w - 0.64;

  if (leadingHeading) {
    addText(slide, leadingHeading, { x: innerX, y: cursorY, w: innerW, h: 0.7 }, {
      fontFace: FONT_HEAD,
      fontSize: 28,
      bold: true,
      color: pal.text,
      breakLine: true,
      valign: "top",
      fit: "shrink",
    });
    cursorY += 0.78;
    // Subtle accent rule under the heading
    addPanel(slide, { x: innerX, y: cursorY - 0.06, w: 1.4, h: 0.04 }, pal.accent, pal.accent, 0);
    cursorY += 0.16;
  }

  const remainingH = Math.max(bounds.h - (cursorY - bounds.y) - 0.1, 0.5);

  // Build a single rich-text block from body blocks so PptxGenJS can flow them.
  const runs: any[] = [];
  for (const b of bodyBlocks) {
    if (b.kind === "heading") {
      runs.push({
        text: b.text,
        options: {
          fontFace: FONT_HEAD,
          fontSize: b.level <= 2 ? 22 : 18,
          bold: true,
          color: pal.text,
          breakLine: true,
          paraSpaceBefore: 6,
          paraSpaceAfter: 4,
        },
      });
    } else if (b.kind === "paragraph") {
      runs.push({
        text: b.text,
        options: {
          fontFace: FONT_BODY,
          fontSize: 16,
          color: pal.text,
          breakLine: true,
          paraSpaceAfter: 8,
        },
      });
    } else {
      // list
      for (const item of b.items) {
        runs.push({
          text: item,
          options: {
            fontFace: FONT_BODY,
            fontSize: 15,
            color: pal.text,
            bullet: { code: "25CF" },
            breakLine: true,
            paraSpaceAfter: 4,
            indentLevel: 0,
          },
        });
      }
    }
  }

  if (runs.length > 0) {
    slide.addText(runs, {
      x: innerX,
      y: cursorY,
      w: innerW,
      h: remainingH,
      margin: 0,
      valign: "top",
      fit: "shrink",
    });
  }
}

function renderSlideBody(slide: PptxSlide, plan: SlidePlan, bounds: Bounds, family: RenderFamily, pal: RenderPalette): void {
  // PRIMARY PATH: when the LLM authored HTML for this slide, render it directly.
  // The .pptx must reflect what the LLM actually produced — not hardcoded chrome.
  if (llmHTMLHasSubstance(plan.html)) {
    renderFromLLMHTML(slide, plan, bounds, family, pal);
    return;
  }

  switch (plan.layout) {
    case "stats":
      renderStats(slide, plan, bounds, family, pal);
      return;
    case "steps":
      renderSteps(slide, plan, bounds, pal);
      return;
    case "cards":
      renderCards(slide, plan, bounds, family, pal);
      return;
    case "chart":
      renderChart(slide, plan, bounds, family, pal);
      return;
    case "timeline":
      renderTimeline(slide, plan, bounds, pal);
      return;
    case "compare":
      renderCompare(slide, plan, bounds, pal);
      return;
    case "table":
      renderTable(slide, plan, bounds, family, pal);
      return;
    case "title":
    case "blank":
      renderSection(slide, plan, bounds, family, pal);
      return;
    default:
      renderBullets(slide, plan, bounds, family, pal);
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

// ---------------------------------------------------------------------------
// Chrome screenshot pipeline — renders each slide's LLM HTML exactly as-is
// using headless Chrome. The PPTX slide images match the UI preview 1:1.
// ---------------------------------------------------------------------------

function systemChromePath(): string {
  const platform = os.platform();
  if (platform === "darwin") {
    return "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome";
  }
  if (platform === "win32") {
    return "C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe";
  }
  // Linux
  for (const p of [
    "/usr/bin/google-chrome",
    "/usr/bin/chromium-browser",
    "/usr/bin/chromium",
    "/snap/bin/chromium",
  ]) {
    if (fs.existsSync(p)) return p;
  }
  return "google-chrome";
}

// Strip @font-face rules from vendor CSS — we replace them with an embedded
// base64 font so Chrome can render icons without any network requests.
function stripFontFaceRules(css: string): string {
  return css.replace(/@font-face\s*\{[^}]*\}/gi, "");
}

// Precomputed CSS strings — strip font-face once, reuse across slides.
const _safeBootstrapCSS = stripFontFaceRules(bootstrapCSS as unknown as string);
const _safeIconsCSS = stripFontFaceRules(bootstrapIconsCSS as unknown as string);
const _iconsFontFace = `@font-face{font-family:"bootstrap-icons";font-display:block;src:url("${bootstrapIconsWoff2}") format("woff2");}`;

function buildSlidePageHTML(slideHTML: string, palette: Palette, themeCss: string, _title: string): string {
  const bg = palette.background || "FFFFFF";
  const textColor = palette.text || "111827";
  // Define CSS custom properties from the palette so seg-* classes and
  // theme_css rules that use var(--accent), var(--bg), etc. resolve correctly.
  const cssVars = `:root{
    --bg:#${bg};
    --card:#${palette.card || "FFFFFF"};
    --accent:#${palette.accent || "4F46E5"};
    --accent2:#${palette.accent2 || "A5B4FC"};
    --text:#${textColor};
    --muted:#${palette.muted || "6B7280"};
    --border:#${palette.border || "E5E7EB"};
  }`;
  // .slide-page has NO height constraint — content renders at natural height.
  // fitSlideToViewport() scales it down if it exceeds 1280×720.
  return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<style>
${_safeBootstrapCSS}
${_iconsFontFace}
${_safeIconsCSS}
${cssVars}
*{box-sizing:border-box;margin:0;padding:0}
html{width:1280px;overflow:hidden}
body{margin:0;background:#${bg};color:#${textColor};font-family:'Inter','Helvetica Neue',Arial,sans-serif;font-size:16px;line-height:1.5}
.slide-page{width:1280px;min-height:720px;display:flex;flex-direction:column;padding:48px 64px}
${themeCss}
</style>
</head>
<body>
<div class="slide-page">${slideHTML}</div>
</body>
</html>`;
}

// After rendering a slide page, scale-to-fit its content into 1280×720 and
// center it within the frame. If content exceeds the 1280×720 viewport, it is
// scaled down uniformly; the remaining space is distributed equally on all
// sides so the slide appears centered (not left-/top-anchored).
const FIT_SCRIPT = `
(function(){
  var el = document.querySelector('.slide-page');
  if (!el) return;
  var h = el.scrollHeight;
  var w = el.scrollWidth;
  var maxH = 720, maxW = 1280;
  if (h <= maxH && w <= maxW) return;
  var scale = Math.min(maxH / h, maxW / w);
  var scaledW = w * scale;
  var scaledH = h * scale;
  var offsetX = Math.round((maxW - scaledW) / 2);
  var offsetY = Math.round((maxH - scaledH) / 2);
  // translate(offsetX, offsetY) then scale — with transform-origin 0 0 this
  // places the scaled element so it is centred inside the 1280×720 frame.
  el.style.transformOrigin = '0 0';
  el.style.transform = 'translate(' + offsetX + 'px,' + offsetY + 'px) scale(' + scale + ')';
  el.style.position = 'absolute';
  // Lock the viewport to exactly 1280×720 so the screenshot clips correctly.
  document.body.style.position = 'relative';
  document.body.style.width = maxW + 'px';
  document.body.style.height = maxH + 'px';
  document.body.style.overflow = 'hidden';
  document.documentElement.style.overflow = 'hidden';
  document.documentElement.style.height = maxH + 'px';
})()
`;

async function screenshotSlides(
  manifest: Manifest
): Promise<Array<string | null>> {
  // Lazily require puppeteer-core — bundle stays external, resolved via NODE_PATH
  // eslint-disable-next-line @typescript-eslint/no-var-requires
  const puppeteer = require("puppeteer-core");

  const browser = await puppeteer.launch({
    executablePath: systemChromePath(),
    headless: "new",
    args: ["--no-sandbox", "--disable-setuid-sandbox", "--disable-dev-shm-usage", "--disable-gpu"],
  });

  try {
    const page = await browser.newPage();
    await page.setViewport({ width: 1280, height: 720, deviceScaleFactor: 1 });

    const themeCss = manifest.deck_plan.theme_css || "";
    const results: Array<string | null> = [];

    async function renderSlide(slideHTML: string, heading: string): Promise<string> {
      const html = buildSlidePageHTML(slideHTML, manifest.palette, themeCss, heading);
      await page.setContent(html, { waitUntil: "domcontentloaded", timeout: 15000 });
      // Scale-to-fit: if content taller than 720px, shrink it uniformly
      await page.evaluate(FIT_SCRIPT);
      return await page.screenshot({ type: "png", encoding: "base64" }) as string;
    }

    // Cover slide
    const coverHTML = manifest.deck_plan.cover_html || "";
    if (coverHTML) {
      results.push(await renderSlide(coverHTML, manifest.title));
    } else {
      results.push(null);
    }

    // Content slides
    for (const plan of manifest.slides) {
      if (plan.html) {
        results.push(await renderSlide(plan.html, plan.heading || manifest.title));
      } else {
        results.push(null);
      }
    }

    return results;
  } finally {
    await browser.close();
  }
}

async function buildPresentationFromStructuredManifest(manifest: Manifest, outputPath: string): Promise<void> {
  const PPTXRuntime = PptxGenJS as unknown as new () => any;
  const pptx = new PPTXRuntime();
  pptx.layout = "LAYOUT_WIDE";
  pptx.author = "YASSERRMD";
  pptx.company = "Barq Cowork";
  pptx.subject = manifest.deck_plan.subject || manifest.title;
  pptx.title = manifest.title;
  pptx.lang = "en-US";

  // Determine if we should use the Chrome screenshot path.
  // Any slide with LLM-authored HTML triggers screenshot mode for the whole deck
  // so the .pptx exactly matches the Reveal.js preview.
  const hasAnyHTML =
    llmHTMLHasSubstance(manifest.deck_plan.cover_html) ||
    manifest.slides.some((s) => llmHTMLHasSubstance(s.html));

  if (hasAnyHTML) {
    // ── Chrome screenshot path ───────────────────────────────────────────────
    // One Chrome instance, one page.setContent() per slide — no template chrome.
    const shots = await screenshotSlides(manifest);
    const family = previewFamily(manifest);
    const pal = buildPalette(manifest, family);

    // shots[0] = cover, shots[1..n] = content slides
    const coverShot = shots[0];
    const cover = pptx.addSlide();
    if (coverShot) {
      cover.addImage({ data: `data:image/png;base64,${coverShot}`, x: 0, y: 0, w: SLIDE_W, h: SLIDE_H });
    } else {
      renderCover(cover, manifest, family, pal);
    }
    cover.addNotes(manifest.narrative || manifest.deck_plan.narrative_arc || manifest.title);

    for (let i = 0; i < manifest.slides.length; i++) {
      const plan = manifest.slides[i];
      const slide = pptx.addSlide();
      const shot = shots[i + 1];
      if (shot) {
        slide.addImage({ data: `data:image/png;base64,${shot}`, x: 0, y: 0, w: SLIDE_W, h: SLIDE_H });
      } else {
        // Fallback for slides that had no HTML: use structured renderer
        const bounds = renderSlideChrome(slide, plan, manifest.slides.length + 1, family, pal);
        renderSlideBody(slide, plan, bounds, family, pal);
      }
      if (plan.speaker_notes?.trim()) {
        slide.addNotes(plan.speaker_notes.trim());
      }
    }
  } else {
    // ── Structured renderer path (no HTML authored by LLM) ──────────────────
    const family = previewFamily(manifest);
    const pal = buildPalette(manifest, family);
    pptx.theme = { headFontFace: FONT_HEAD, bodyFontFace: FONT_BODY };

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
  await buildPresentationFromStructuredManifest(manifest, outputPath);
  process.stdout.write(JSON.stringify({ status: "ok", output: outputPath }) + "\n");
}

main().catch((err) => {
  process.stderr.write(String(err instanceof Error ? err.stack || err.message : err) + "\n");
  process.exitCode = 1;
});
