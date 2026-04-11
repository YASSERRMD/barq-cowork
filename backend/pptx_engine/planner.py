"""
Deck planner: NL prompt → DeckPlan.

If OPENAI_API_KEY or ANTHROPIC_API_KEY is set, delegates to the respective LLM.
Otherwise uses a rule-based keyword planner.
"""

from __future__ import annotations

import json
import os
import re
from typing import Optional

from .schema import (
    CardItem,
    ChartSeries,
    ChartType,
    CompareColumn,
    Deck,
    DeckMeta,
    DeckPlan,
    DeckTheme,
    PlanRequest,
    Slide,
    SlideContent,
    SlideID,
    SlideType,
    StatItem,
    TimelineItem,
)


# ── Keyword → SlideType inference rules ──────────────────────────────────────

_KEYWORD_RULES: list[tuple[list[str], SlideType]] = [
    (["intro", "welcome", "agenda", "overview", "about"], SlideType.title),
    (["metric", "metrics", "data", "number", "numbers", "kpi", "stat", "statistics", "performance", "result", "results"], SlideType.stats),
    (["process", "flow", "step", "steps", "how to", "how-to", "workflow", "procedure", "method"], SlideType.steps),
    (["feature", "features", "benefit", "benefits", "offer", "offering", "capability", "capabilities", "service", "services"], SlideType.cards),
    (["compare", "comparison", "versus", "vs.", " vs ", "before", "after", "pros", "cons", "tradeoff"], SlideType.compare),
    (["timeline", "roadmap", "milestone", "milestones", "phase", "phases", "history", "schedule", "plan"], SlideType.timeline),
    (["table", "matrix", "grid", "list", "breakdown", "structure"], SlideType.table),
    (["chart", "graph", "trend", "growth", "revenue", "sales", "forecast"], SlideType.chart),
]


def _infer_slide_type(text: str) -> SlideType:
    """Infer a slide type from free-form text using keyword matching."""
    lower = text.lower()
    for keywords, slide_type in _KEYWORD_RULES:
        for kw in keywords:
            if kw in lower:
                return slide_type
    return SlideType.bullets


# ── Rule-based planner ────────────────────────────────────────────────────────

_SLIDE_TEMPLATE_FRAGMENTS: list[str] = [
    "Introduction / Agenda",
    "Problem Statement",
    "Our Solution",
    "Key Features",
    "How It Works",
    "Performance Metrics",
    "Market Opportunity",
    "Competitive Comparison",
    "Roadmap",
    "Team",
    "Next Steps",
    "Call to Action",
]


def _rule_based_plan(request: PlanRequest) -> DeckPlan:
    """
    Build a DeckPlan from the prompt using keyword inference.
    Generates *slide_count* slides distributed across inferred types.
    """
    prompt = request.nl_prompt or request.title
    n = request.slide_count

    # Split prompt into segment words for slide title generation
    words = [w.strip(".,;:!?") for w in prompt.split() if w.strip()]

    slides: list[Slide] = []

    # First slide is always a title slide
    slides.append(
        Slide(
            id=SlideID.generate(),
            type=SlideType.title,
            heading=request.title,
            content=SlideContent(
                subtitle=_truncate(prompt, 120),
            ),
        )
    )

    remaining = n - 1
    if remaining <= 0:
        return DeckPlan(slides=slides)

    # Generate topic headings from the prompt or fall back to templates
    topics = _extract_topics(prompt, remaining)

    for i, topic in enumerate(topics[:remaining]):
        slide_type = _infer_slide_type(topic)
        content = _default_content_for_type(slide_type, topic)
        slides.append(
            Slide(
                id=SlideID.generate(),
                type=slide_type,
                heading=_truncate(topic.title(), 60),
                content=content,
                speaker_notes=f"Speaker notes for: {topic}",
            )
        )

    return DeckPlan(slides=slides)


def _extract_topics(prompt: str, count: int) -> list[str]:
    """Extract topic hints from prompt; pad with generic headings if needed."""
    # Split on common delimiters
    raw = re.split(r"[,;/\n]+", prompt)
    topics = [t.strip() for t in raw if len(t.strip()) > 3]

    # If not enough from splitting, generate generic ones
    while len(topics) < count:
        idx = len(topics)
        frag = _SLIDE_TEMPLATE_FRAGMENTS[idx % len(_SLIDE_TEMPLATE_FRAGMENTS)]
        topics.append(frag)

    return topics[:count]


def _default_content_for_type(slide_type: SlideType, topic: str) -> SlideContent:
    """Return representative placeholder content for a slide type."""
    if slide_type == SlideType.bullets:
        return SlideContent(
            points=[
                f"Key point 1 about {topic[:30]}",
                f"Key point 2 about {topic[:30]}",
                f"Supporting detail or evidence",
                f"Impact and benefits",
            ]
        )
    if slide_type == SlideType.stats:
        return SlideContent(
            stats=[
                StatItem(value="98%", label="Satisfaction", desc="User satisfaction rate"),
                StatItem(value="3.2M", label="Users", desc="Active monthly users"),
                StatItem(value="45%", label="Growth", desc="Year-over-year growth"),
                StatItem(value="$12M", label="ARR", desc="Annual recurring revenue"),
            ]
        )
    if slide_type == SlideType.steps:
        return SlideContent(
            steps=[
                "Define the problem and gather requirements",
                "Design the solution architecture",
                "Implement and iterate",
                "Test and validate",
                "Deploy and monitor",
            ]
        )
    if slide_type == SlideType.cards:
        return SlideContent(
            cards=[
                CardItem(icon="⚡", title="Speed", desc="Blazing fast performance"),
                CardItem(icon="🔒", title="Security", desc="Enterprise-grade security"),
                CardItem(icon="🔌", title="Integration", desc="Works with your tools"),
                CardItem(icon="📊", title="Analytics", desc="Real-time insights"),
            ]
        )
    if slide_type == SlideType.chart:
        return SlideContent(
            chart_type=ChartType.column,
            chart_categories=["Q1 2024", "Q2 2024", "Q3 2024", "Q4 2024"],
            chart_series=[
                ChartSeries(name="Revenue", values=[1.2, 1.8, 2.4, 3.1]),
                ChartSeries(name="Target", values=[1.0, 1.5, 2.0, 2.5]),
            ],
            chart_title=topic[:40],
        )
    if slide_type == SlideType.timeline:
        return SlideContent(
            timeline=[
                TimelineItem(date="Q1 2024", title="Project Kickoff", desc="Initial planning"),
                TimelineItem(date="Q2 2024", title="Beta Launch", desc="Limited release"),
                TimelineItem(date="Q3 2024", title="GA Release", desc="Full launch"),
                TimelineItem(date="Q4 2024", title="Scale Phase", desc="Expand market"),
            ]
        )
    if slide_type == SlideType.compare:
        return SlideContent(
            left_column=CompareColumn(
                heading="Current State",
                points=["Manual processes", "High operational cost", "Limited scalability", "Slow time-to-market"],
            ),
            right_column=CompareColumn(
                heading="With Our Solution",
                points=["Automated workflows", "60% cost reduction", "Infinite scalability", "10x faster delivery"],
            ),
        )
    if slide_type == SlideType.table:
        return SlideContent(
            table=_default_table(),
        )
    # Default bullets
    return SlideContent(
        points=[
            f"Key insight about {topic[:30]}",
            "Supporting data point",
            "Strategic implication",
        ]
    )


def _default_table():
    from .schema import TableData
    return TableData(
        headers=["Feature", "Basic", "Pro", "Enterprise"],
        rows=[
            ["Core Features", "✓", "✓", "✓"],
            ["Advanced Analytics", "—", "✓", "✓"],
            ["SLA", "99.9%", "99.95%", "99.99%"],
            ["Support", "Email", "Priority", "Dedicated"],
        ],
    )


def _truncate(text: str, max_len: int) -> str:
    if len(text) <= max_len:
        return text
    return text[: max_len - 1] + "…"


# ── LLM planner (OpenAI) ──────────────────────────────────────────────────────

def _openai_plan(request: PlanRequest, api_key: str) -> DeckPlan:
    """Call OpenAI Chat API to generate a structured DeckPlan."""
    try:
        import httpx

        system_prompt = (
            "You are a professional presentation designer. "
            "Given a topic/description, create a structured presentation plan as JSON. "
            "Output a JSON object with a 'slides' array. "
            "Each slide must have: type (one of: title, bullets, stats, steps, cards, "
            "chart, timeline, compare, table, blank), heading (string ≤60 chars), "
            "and content (object with relevant fields). "
            "Do not include any explanation — only the raw JSON."
        )

        user_prompt = (
            f"Create a {request.slide_count}-slide presentation plan for: "
            f"'{request.title}'\n\nDescription: {request.nl_prompt}"
        )

        response = httpx.post(
            "https://api.openai.com/v1/chat/completions",
            headers={"Authorization": f"Bearer {api_key}", "Content-Type": "application/json"},
            json={
                "model": "gpt-4o-mini",
                "messages": [
                    {"role": "system", "content": system_prompt},
                    {"role": "user", "content": user_prompt},
                ],
                "temperature": 0.7,
                "max_tokens": 3000,
            },
            timeout=30,
        )
        response.raise_for_status()
        data = response.json()
        content = data["choices"][0]["message"]["content"]

        # Strip markdown code fences if present
        content = re.sub(r"^```(?:json)?\n?", "", content.strip())
        content = re.sub(r"\n?```$", "", content.strip())

        return _parse_llm_response(content, request)

    except Exception:
        return _rule_based_plan(request)


def _anthropic_plan(request: PlanRequest, api_key: str) -> DeckPlan:
    """Call Anthropic Messages API to generate a structured DeckPlan."""
    try:
        import httpx

        user_prompt = (
            f"Create a {request.slide_count}-slide presentation plan for: "
            f"'{request.title}'\n\nDescription: {request.nl_prompt}\n\n"
            "Output ONLY a JSON object with a 'slides' array. "
            "Each slide: type (title|bullets|stats|steps|cards|chart|timeline|compare|table|blank), "
            "heading (≤60 chars), content object. No extra text."
        )

        response = httpx.post(
            "https://api.anthropic.com/v1/messages",
            headers={
                "x-api-key": api_key,
                "anthropic-version": "2023-06-01",
                "Content-Type": "application/json",
            },
            json={
                "model": "claude-3-5-haiku-20241022",
                "max_tokens": 3000,
                "messages": [{"role": "user", "content": user_prompt}],
            },
            timeout=30,
        )
        response.raise_for_status()
        data = response.json()
        content = data["content"][0]["text"]

        content = re.sub(r"^```(?:json)?\n?", "", content.strip())
        content = re.sub(r"\n?```$", "", content.strip())

        return _parse_llm_response(content, request)

    except Exception:
        return _rule_based_plan(request)


def _parse_llm_response(json_str: str, request: PlanRequest) -> DeckPlan:
    """
    Parse the LLM JSON response into a DeckPlan.
    Falls back to rule-based if parsing fails.
    """
    try:
        data = json.loads(json_str)
        slides_raw = data.get("slides", [])
        slides: list[Slide] = []

        for raw in slides_raw:
            slide_type_str = raw.get("type", "bullets")
            try:
                slide_type = SlideType(slide_type_str)
            except ValueError:
                slide_type = SlideType.bullets

            heading = str(raw.get("heading", ""))[:60]
            raw_content = raw.get("content", {})

            content = _parse_content(raw_content, slide_type)

            slides.append(
                Slide(
                    id=SlideID.generate(),
                    type=slide_type,
                    heading=heading,
                    content=content,
                )
            )

        if not slides:
            return _rule_based_plan(request)

        return DeckPlan(slides=slides)

    except (json.JSONDecodeError, KeyError, TypeError):
        return _rule_based_plan(request)


def _parse_content(raw: dict, slide_type: SlideType) -> SlideContent:
    """Convert a raw dict from LLM output to a SlideContent model."""
    try:
        # Try direct Pydantic parse
        return SlideContent.model_validate(raw)
    except Exception:
        # Fall back to default content for the type
        return _default_content_for_type(slide_type, "")


# ── Public entry point ─────────────────────────────────────────────────────────

def create_plan(request: PlanRequest) -> DeckPlan:
    """
    Generate a DeckPlan from a PlanRequest.

    Uses LLM if API keys are available, otherwise falls back to rule-based.
    """
    anthropic_key = os.environ.get("ANTHROPIC_API_KEY", "")
    openai_key = os.environ.get("OPENAI_API_KEY", "")

    if anthropic_key:
        return _anthropic_plan(request, anthropic_key)
    if openai_key:
        return _openai_plan(request, openai_key)

    return _rule_based_plan(request)
