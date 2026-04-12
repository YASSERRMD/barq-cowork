#!/usr/bin/env python3
"""
pptx_bridge.py — Bridge between the barq-cowork Go backend and pptx_engine.

Accepts the Go tool's extended JSON format on stdin, translates it into the
full pptx_engine Deck schema, renders with the engine's slide registry
(10 professional layouts), and writes PPTX bytes to stdout.

Input format (stdin JSON):
  {
    "title":    "Deck Title",
    "subtitle": "Optional subtitle",
    "accent":   "6366F1",           // hex without #
    "author":   "Optional author",
    "slides": [
      {
        "heading": "Slide heading",
        "type":    "bullets|stats|steps|cards|chart|timeline|compare|table|title|blank",
        // --- bullets ---
        "points":  ["..."],
        // --- stats ---
        "stats":   [{"value":"42%","label":"Growth","desc":"YoY"}],
        // --- steps ---
        "steps":   ["Step 1","Step 2"],   // or use "points" for compat
        // --- cards ---
        "cards":   [{"icon":"⚡","title":"Speed","desc":"Fast"}],
        // --- chart ---
        "chart_type":       "column|bar|line|pie|doughnut|area|scatter",
        "chart_categories": ["Q1","Q2"],
        "chart_series":     [{"name":"Revenue","values":[1.2,1.8],"color":"6366F1"}],
        "y_label":          "Revenue ($M)",
        // --- timeline ---
        "timeline": [{"date":"Q1 2026","title":"Launch","desc":"Details"}],
        // --- compare ---
        "left_column":  {"heading":"Before","points":["..."]},
        "right_column": {"heading":"After","points":["..."]},
        // legacy compare keys still accepted:
        "left_title":"...", "right_title":"...",
        "left_points":["..."], "right_points":["..."],
        // --- table ---
        "table": {"headers":["A","B"],"rows":[["a","b"]]},
        // --- title ---
        "subtitle": "...",
        // --- all types ---
        "speaker_notes": "Optional speaker notes"
      }
    ]
  }
"""

from __future__ import annotations

import json
import os
import sys

# Add the backend directory to sys.path so we can import pptx_engine
_script_dir = os.path.dirname(os.path.abspath(__file__))
_backend_dir = os.path.dirname(_script_dir)
if _backend_dir not in sys.path:
    sys.path.insert(0, _backend_dir)

from pptx_engine.schema import (
    CardItem,
    ChangeEntry,
    ChartSeries,
    ChartType,
    CompareColumn,
    Deck,
    DeckMeta,
    DeckPlan,
    DeckTheme,
    SlideContent,
    SlideID,
    SlideType,
    StatItem,
    TableData,
    ThemeColors,
    ThemeFonts,
    TimelineItem,
    Slide,
)
from pptx_engine.renderer import PresentationRenderer
from pptx_engine.slide_registry import SlideTypeRegistry

# ── Type alias map (legacy layout names → SlideType) ──────────────────────────
_LAYOUT_MAP: dict[str, SlideType] = {
    "bullets":  SlideType.bullets,
    "stats":    SlideType.stats,
    "steps":    SlideType.steps,
    "cards":    SlideType.cards,
    "chart":    SlideType.chart,
    "timeline": SlideType.timeline,
    "compare":  SlideType.compare,
    "table":    SlideType.table,
    "title":    SlideType.title,
    "blank":    SlideType.blank,
    # keep old gen_pptx.py layout names
    "bullets_slide": SlideType.bullets,
    "stats_slide":   SlideType.stats,
}

# Default icons for cards when only a string is given
_CARD_ICONS = ["⚡", "🔒", "🔌", "📊", "🚀", "🎯", "🌐", "🧠", "📱", "🔑"]


# ── Full visual theme builder (replaces _pick_accent) ─────────────────────────

# Each theme tuple: (background, surface, accent, accent2, border)
_THEME_PRESETS = {
    "tech":        ("#0F172A", "#1E293B", "#6366F1", "#A5B4FC", "#2D3F55"),
    "healthcare":  ("#061B2E", "#0E2D45", "#06B6D4", "#67E8F9", "#1A4060"),
    "education":   ("#1A1100", "#2D1E00", "#F59E0B", "#FCD34D", "#4A3500"),
    "environment": ("#071A10", "#0F2B1C", "#10B981", "#6EE7B7", "#1A4A30"),
    "finance":     ("#061A0E", "#102B1A", "#22C55E", "#86EFAC", "#1A4A28"),
    "creative":    ("#140A2A", "#221545", "#8B5CF6", "#C4B5FD", "#3A2060"),
    "security":    ("#1A0808", "#2E1010", "#EF4444", "#FCA5A5", "#4A1818"),
    "data":        ("#061A1E", "#0E2B30", "#14B8A6", "#5EEAD4", "#1A4048"),
    "logistics":   ("#0A1020", "#162035", "#3B82F6", "#93C5FD", "#203050"),
    "retail":      ("#1A0C00", "#2E1800", "#F97316", "#FDBA74", "#4A2800"),
    "hr":          ("#1A0A18", "#2E1530", "#EC4899", "#F9A8D4", "#4A1840"),
}


def _build_full_theme(title: str, subtitle: str = "") -> DeckTheme:
    """Pick a complete visual theme based on topic keywords."""
    c = (title + " " + subtitle).lower()

    def has(*words):
        return any(w in c for w in words)

    # Detect category
    if has("machine learning", "deep learning", "artificial intelligence",
           "neural", "llm", "gpt", "claude", "openai"):
        preset_key = "tech"
    elif has("health", "healthcare", "medical", "clinical", "hospital", "pharma",
             "doctor", "patient"):
        preset_key = "healthcare"
    elif has("education", "school", "university", "learning", "course", "student",
             "teach", "curriculum", "kids"):
        preset_key = "education"
    elif has("environment", "climate", "sustainable", "green", "carbon",
             "renewable", "eco", "solar"):
        preset_key = "environment"
    elif has("finance", "banking", "investment", "revenue", "budget", "portfolio",
             "fund", "wealth", "stock"):
        preset_key = "finance"
    elif has("design", "creative", "art", "brand", "marketing", "media",
             "visual", "photo", "film", "fashion"):
        preset_key = "creative"
    elif has("security", "cyber", "threat", "hack", "ransomware", "firewall",
             "privacy", "compliance", "risk"):
        preset_key = "security"
    elif has("data", "analytics", "intelligence", "bi ", "databrick", "snowflake",
             "warehouse", "insight"):
        preset_key = "data"
    elif has("logistics", "supply chain", "shipping", "warehouse", "delivery",
             "transport", "fleet"):
        preset_key = "logistics"
    elif has("retail", "ecommerce", "shop", "consumer", "product", "merchandise",
             "store", "fashion"):
        preset_key = "retail"
    elif has("hr", "human resource", "talent", "recruit", "employee", "workforce",
             "people ops"):
        preset_key = "hr"
    elif has("tech", "software", "platform", "cloud", "api", "developer",
             "engineering", "saas", "devops"):
        preset_key = "tech"
    else:
        # Pick based on hash of title for variety even on generic topics
        keys = list(_THEME_PRESETS.keys())
        preset_key = keys[hash(title.lower()) % len(keys)]

    bg, surface, accent, accent2, border = _THEME_PRESETS[preset_key]
    return DeckTheme(
        colors=ThemeColors(
            background=bg,
            surface=surface,
            accent=accent,
            accent2=accent2,
            text="#F8FAFC",
            text_muted="#94A3B8",
            border=border,
        ),
        fonts=ThemeFonts(heading="Calibri Light", body="Calibri", size_heading=31, size_body=15),
    )


# ── Slide translation ─────────────────────────────────────────────────────────

def _resolve_type(raw: dict) -> SlideType:
    """Determine SlideType from the 'type' or legacy 'layout' key."""
    for key in ("type", "layout"):
        val = raw.get(key, "")
        if val:
            mapped = _LAYOUT_MAP.get(str(val).lower())
            if mapped:
                return mapped
    return SlideType.bullets


def _parse_card_items(raw_cards) -> list[CardItem]:
    """Convert list of dicts OR strings to CardItem list."""
    out = []
    for i, item in enumerate(raw_cards or []):
        if isinstance(item, dict):
            out.append(CardItem(
                icon=str(item.get("icon", _CARD_ICONS[i % len(_CARD_ICONS)])),
                title=str(item.get("title", item.get("label", f"Item {i+1}"))),
                desc=str(item.get("desc", item.get("description", ""))),
            ))
        else:
            # Plain string → use as title
            out.append(CardItem(
                icon=_CARD_ICONS[i % len(_CARD_ICONS)],
                title=str(item),
                desc="",
            ))
    return out


def _parse_stat_items(raw_stats) -> list[StatItem]:
    out = []
    for item in raw_stats or []:
        if isinstance(item, dict):
            out.append(StatItem(
                value=str(item.get("value", "")),
                label=str(item.get("label", "")),
                desc=str(item.get("desc", item.get("description", ""))),
            ))
    return out


def _parse_timeline_items(raw_tl) -> list[TimelineItem]:
    out = []
    for item in raw_tl or []:
        if isinstance(item, dict):
            out.append(TimelineItem(
                date=str(item.get("date", item.get("time", ""))),
                title=str(item.get("title", item.get("heading", ""))),
                desc=str(item.get("desc", item.get("description", ""))),
            ))
        else:
            out.append(TimelineItem(date="", title=str(item), desc=""))
    return out


def _parse_chart_series(raw_series) -> list[ChartSeries]:
    out = []
    for s in raw_series or []:
        if isinstance(s, dict):
            out.append(ChartSeries(
                name=str(s.get("name", "Series")),
                values=[float(v) for v in (s.get("values") or [])],
                color=s.get("color"),
            ))
    return out


def _parse_compare_column(heading: str, points: list) -> CompareColumn:
    return CompareColumn(
        heading=heading,
        points=[str(p) for p in (points or [])],
    )


def _parse_table(raw_table) -> TableData | None:
    if not raw_table or not isinstance(raw_table, dict):
        return None
    headers = [str(h) for h in (raw_table.get("headers") or [])]
    rows = []
    for row in (raw_table.get("rows") or []):
        if isinstance(row, list):
            rows.append([str(c) for c in row])
    return TableData(headers=headers, rows=rows) if headers else None


def _translate_slide(raw: dict, idx: int) -> Slide:
    slide_type = _resolve_type(raw)
    heading = str(raw.get("heading", raw.get("title", "")))[:60]
    speaker_notes = str(raw.get("speaker_notes", raw.get("notes", "")))
    locked = bool(raw.get("locked", False))

    # ── Build SlideContent from the raw dict ──────────────────────────────────
    kw: dict = {}

    if slide_type == SlideType.title:
        kw["subtitle"] = str(raw.get("subtitle", raw.get("sub", "")))

    elif slide_type == SlideType.bullets:
        pts = raw.get("points") or raw.get("bullets") or []
        kw["points"] = [str(p) for p in pts]

    elif slide_type == SlideType.stats:
        kw["stats"] = _parse_stat_items(raw.get("stats") or [])

    elif slide_type == SlideType.steps:
        # Accept either "steps" or "points"
        steps_raw = raw.get("steps") or raw.get("points") or []
        kw["steps"] = [str(s) for s in steps_raw]

    elif slide_type == SlideType.cards:
        raw_cards = raw.get("cards")
        if raw_cards:
            kw["cards"] = _parse_card_items(raw_cards)
        else:
            # Fallback: convert plain points to CardItem objects
            pts = raw.get("points") or []
            kw["cards"] = _parse_card_items(pts)

    elif slide_type == SlideType.chart:
        ct_raw = str(raw.get("chart_type", "column")).lower()
        try:
            kw["chart_type"] = ChartType(ct_raw)
        except ValueError:
            kw["chart_type"] = ChartType.column
        kw["chart_categories"] = [str(c) for c in (raw.get("chart_categories") or
                                                     raw.get("categories") or [])]
        kw["chart_series"] = _parse_chart_series(raw.get("chart_series") or
                                                   raw.get("series") or [])
        kw["chart_title"] = str(raw.get("chart_title", ""))
        kw["y_label"] = str(raw.get("y_label", ""))

    elif slide_type == SlideType.timeline:
        raw_tl = raw.get("timeline")
        if raw_tl:
            kw["timeline"] = _parse_timeline_items(raw_tl)
        else:
            # Fallback: convert points to timeline items
            pts = raw.get("points") or []
            kw["timeline"] = _parse_timeline_items(pts)

    elif slide_type == SlideType.compare:
        # Prefer structured left_column/right_column
        lc_raw = raw.get("left_column")
        rc_raw = raw.get("right_column")
        if lc_raw and isinstance(lc_raw, dict):
            kw["left_column"] = _parse_compare_column(
                lc_raw.get("heading", ""),
                lc_raw.get("points", []),
            )
        else:
            kw["left_column"] = _parse_compare_column(
                str(raw.get("left_title", "Before")),
                raw.get("left_points", []),
            )
        if rc_raw and isinstance(rc_raw, dict):
            kw["right_column"] = _parse_compare_column(
                rc_raw.get("heading", ""),
                rc_raw.get("points", []),
            )
        else:
            kw["right_column"] = _parse_compare_column(
                str(raw.get("right_title", "After")),
                raw.get("right_points", []),
            )

    elif slide_type == SlideType.table:
        t = _parse_table(raw.get("table"))
        if t:
            kw["table"] = t

    return Slide(
        id=SlideID.generate(),
        type=slide_type,
        heading=heading,
        content=SlideContent(**kw),
        speaker_notes=speaker_notes,
        locked=locked,
    )


# ── Full deck translation ──────────────────────────────────────────────────────

def translate_to_deck(data: dict) -> Deck:
    """Convert the Go tool's extended JSON to a full pptx_engine Deck."""
    title    = str(data.get("title", "Presentation"))
    subtitle = str(data.get("subtitle", ""))
    author   = str(data.get("author",  "Barq Cowork"))

    # If Go passed an explicit theme name, use that preset directly for the
    # full coordinated palette (background, surface, accent, border — all matched).
    # Falls back to keyword detection if no theme name is provided.
    explicit_theme = str(data.get("theme", "")).strip().lower()
    if explicit_theme and explicit_theme in _THEME_PRESETS:
        bg, surface, accent, accent2, border = _THEME_PRESETS[explicit_theme]
        theme = DeckTheme(
            colors=ThemeColors(
                background=bg,
                surface=surface,
                accent=accent,
                accent2=accent2,
                text="#F8FAFC",
                text_muted="#94A3B8",
                border=border,
            ),
            fonts=ThemeFonts(heading="Calibri Light", body="Calibri", size_heading=31, size_body=15),
        )
    else:
        theme = _build_full_theme(title, subtitle)

    # ── Build slides ──────────────────────────────────────────────────────────
    raw_slides = data.get("slides") or []
    slides: list[Slide] = []

    for i, raw in enumerate(raw_slides):
        slide = _translate_slide(raw, i)

        # If first slide has no type specified, make it a title slide
        if i == 0 and slide.type == SlideType.bullets and not raw.get("type") and not raw.get("layout"):
            slide = Slide(
                id=slide.id,
                type=SlideType.title,
                heading=slide.heading,
                content=SlideContent(subtitle=subtitle),
                speaker_notes=slide.speaker_notes,
                locked=slide.locked,
            )

        slides.append(slide)

    meta = DeckMeta(
        title=title,
        description=subtitle,
        author=author,
        version=1,
        change_log=[
            ChangeEntry(
                version=1,
                author=author,
                action="create",
                slide_ids_affected=[s.id for s in slides],
                description=f"Generated '{title}' with {len(slides)} slides",
            )
        ],
    )

    return Deck(
        meta=meta,
        theme=theme,
        plan=DeckPlan(slides=slides),
    )


# ── Entry point ───────────────────────────────────────────────────────────────

def main():
    raw = sys.stdin.buffer.read()
    data = json.loads(raw)

    # Support being called with a pre-formed Deck JSON (meta + plan keys)
    if "meta" in data and "plan" in data:
        deck = Deck.model_validate(data)
    else:
        deck = translate_to_deck(data)

    registry = SlideTypeRegistry.build_default()
    renderer = PresentationRenderer(registry)
    pptx_bytes = renderer.render_to_bytes(deck)
    sys.stdout.buffer.write(pptx_bytes)


if __name__ == "__main__":
    main()
