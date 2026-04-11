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


# ── Keyword-based accent colour picker (mirrors write_pptx.go logic) ──────────
def _pick_accent(title: str, subtitle: str = "") -> str:
    c = (title + " " + subtitle).lower()

    def has(*words: str) -> bool:
        for w in words:
            # simple word-boundary check
            idx = c.find(w)
            while idx != -1:
                before = c[idx - 1] if idx > 0 else " "
                after  = c[idx + len(w)] if idx + len(w) < len(c) else " "
                if not before.isalpha() and not after.isalpha():
                    return True
                idx = c.find(w, idx + 1)
        return False

    # Multi-word tech phrases first
    if any(p in c for p in ("machine learning", "deep learning", "neural network",
                             "artificial intelligence", "large language model")):
        return "6366F1"  # indigo

    if has("health", "healthcare", "medical", "clinical", "hospital", "pharma"):
        return "06B6D4"  # cyan
    if has("education", "school", "learning", "kids", "student", "university"):
        return "F59E0B"  # amber
    if has("environment", "green", "sustainable", "climate", "ecology", "solar"):
        return "10B981"  # emerald
    if has("finance", "investment", "banking", "revenue", "budget", "wealth"):
        return "8B5CF6"  # violet
    if has("creative", "design", "art", "brand", "marketing", "media"):
        return "EC4899"  # pink
    if has("security", "cyber", "privacy", "compliance", "risk", "trust"):
        return "EF4444"  # red
    if has("data", "analytics", "insight", "dashboard", "bi", "intelligence"):
        return "14B8A6"  # teal
    if has("tech", "software", "platform", "cloud", "api", "developer", "engineering"):
        return "6366F1"  # indigo (default tech)
    return "6366F1"  # fallback indigo


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
        kw["chart_title"] = str(raw.get("y_label", raw.get("chart_title", "")))

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
    accent   = str(data.get("accent",  _pick_accent(title, subtitle))).lstrip("#")
    author   = str(data.get("author",  "Barq Cowork"))

    # Ensure 6-char hex
    if len(accent) == 3:
        accent = "".join(c * 2 for c in accent)
    accent = accent.upper()

    theme = DeckTheme(
        colors=ThemeColors(
            accent=f"#{accent}",
            background="#0F172A",
            surface="#1E293B",
            accent2="#A5B4FC",
            text="#F8FAFC",
            text_muted="#94A3B8",
            border="#2D3F55",
        ),
        fonts=ThemeFonts(
            heading="Calibri Light",
            body="Calibri",
            size_heading=31,
            size_body=15,
        ),
    )

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
