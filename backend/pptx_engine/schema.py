"""
Pydantic v2 schema for the Barq PPTX Engine.
All models used throughout the system are defined here as the single source of truth.
"""

from __future__ import annotations

import secrets
import uuid
from datetime import datetime
from enum import Enum
from typing import Any, Optional

from pydantic import BaseModel, Field, field_validator, model_validator


# ── Stable Slide ID ────────────────────────────────────────────────────────────

class SlideID:
    """Stable slide identifier: 's-' + 8 hex chars."""

    PREFIX = "s-"
    HEX_LEN = 8

    @classmethod
    def generate(cls) -> str:
        return f"{cls.PREFIX}{secrets.token_hex(cls.HEX_LEN // 2)}"

    @classmethod
    def is_valid(cls, value: str) -> bool:
        if not isinstance(value, str):
            return False
        if not value.startswith(cls.PREFIX):
            return False
        hex_part = value[len(cls.PREFIX):]
        if len(hex_part) != cls.HEX_LEN:
            return False
        try:
            int(hex_part, 16)
            return True
        except ValueError:
            return False


# ── Enums ──────────────────────────────────────────────────────────────────────

class SlideType(str, Enum):
    title = "title"
    bullets = "bullets"
    stats = "stats"
    steps = "steps"
    cards = "cards"
    chart = "chart"
    timeline = "timeline"
    compare = "compare"
    table = "table"
    blank = "blank"


class ChartType(str, Enum):
    column = "column"
    bar = "bar"
    line = "line"
    pie = "pie"
    doughnut = "doughnut"
    area = "area"
    scatter = "scatter"


class ValidationSeverity(str, Enum):
    error = "error"
    warning = "warning"
    info = "info"


# ── Theme Models ───────────────────────────────────────────────────────────────

class ThemeColors(BaseModel):
    background: str = Field(default="#0F172A", description="Slide background hex color")
    surface: str = Field(default="#1E293B", description="Card/surface hex color")
    accent: str = Field(default="#6366F1", description="Primary accent hex color")
    accent2: str = Field(default="#A78BFA", description="Secondary accent hex color")
    text: str = Field(default="#F8FAFC", description="Primary text hex color")
    text_muted: str = Field(default="#94A3B8", description="Muted text hex color")
    border: str = Field(default="#2D3F55", description="Border/divider hex color")
    success: str = Field(default="#10B981", description="Success indicator hex color")
    warning: str = Field(default="#F59E0B", description="Warning indicator hex color")
    danger: str = Field(default="#EF4444", description="Danger/error indicator hex color")

    @field_validator("background", "surface", "accent", "accent2", "text",
                     "text_muted", "border", "success", "warning", "danger",
                     mode="before")
    @classmethod
    def validate_hex_color(cls, v: str) -> str:
        if not isinstance(v, str):
            raise ValueError("Color must be a string")
        v = v.strip()
        if not v.startswith("#"):
            v = f"#{v}"
        hex_part = v[1:]
        if len(hex_part) not in (3, 6):
            raise ValueError(f"Invalid hex color: {v}")
        try:
            int(hex_part, 16)
        except ValueError:
            raise ValueError(f"Invalid hex color: {v}")
        return v.upper()


class ThemeFonts(BaseModel):
    heading: str = Field(default="Calibri Light", description="Heading font family")
    body: str = Field(default="Calibri", description="Body text font family")
    mono: str = Field(default="Courier New", description="Monospace font family")
    size_heading: int = Field(default=31, description="Heading font size in points")
    size_body: int = Field(default=15, description="Body text font size in points")
    size_caption: int = Field(default=11, description="Caption font size in points")


class DeckTheme(BaseModel):
    colors: ThemeColors = Field(default_factory=ThemeColors)
    fonts: ThemeFonts = Field(default_factory=ThemeFonts)


# ── Change Log ─────────────────────────────────────────────────────────────────

class ChangeEntry(BaseModel):
    version: int = Field(description="Version number this change creates")
    timestamp: datetime = Field(default_factory=datetime.utcnow)
    author: str = Field(default="system")
    action: str = Field(description="Action type: create|edit|reorder|theme|delete_slide")
    slide_ids_affected: list[str] = Field(default_factory=list)
    description: str = Field(description="Human-readable description of the change")


# ── Deck Meta ──────────────────────────────────────────────────────────────────

class DeckMeta(BaseModel):
    id: str = Field(default_factory=lambda: str(uuid.uuid4()))
    title: str = Field(description="Presentation title")
    description: str = Field(default="", description="Short description of the deck")
    template_id: Optional[str] = Field(default=None, description="Template identifier")
    author: str = Field(default="system", description="Deck author")
    created_at: datetime = Field(default_factory=datetime.utcnow)
    updated_at: datetime = Field(default_factory=datetime.utcnow)
    version: int = Field(default=1, ge=1, description="Current deck version")
    change_log: list[ChangeEntry] = Field(default_factory=list)


# ── Content Models ─────────────────────────────────────────────────────────────

class StatItem(BaseModel):
    value: str = Field(description="Displayed metric value, e.g. '98%' or '1.2M'")
    label: str = Field(default="", description="Short label below the value")
    desc: str = Field(default="", description="Optional longer description")


class ChartSeries(BaseModel):
    name: str = Field(description="Series name shown in legend")
    values: list[float] = Field(description="Data values")
    color: Optional[str] = Field(default=None, description="Optional hex color override")


class TableData(BaseModel):
    headers: list[str] = Field(description="Column header labels")
    rows: list[list[str]] = Field(description="Table data rows")
    style: Optional[str] = Field(default="default", description="Table style preset")


class TimelineItem(BaseModel):
    date: str = Field(description="Date or period label")
    title: str = Field(description="Milestone title")
    desc: str = Field(default="", description="Optional description")


class CardItem(BaseModel):
    icon: str = Field(default="★", description="Icon character or emoji")
    title: str = Field(description="Card title")
    desc: str = Field(default="", description="Card description text")


class CompareColumn(BaseModel):
    heading: str = Field(description="Column heading (e.g. 'Before' / 'After')")
    points: list[str] = Field(default_factory=list)
    color: Optional[str] = Field(default=None, description="Accent color override")


class SlideContent(BaseModel):
    """Universal content container — all fields are optional.
    The renderer picks fields relevant to the slide type."""

    subtitle: Optional[str] = Field(default=None, description="Subtitle or tagline (title slides)")
    points: Optional[list[str]] = Field(default=None, description="Bullet points")
    stats: Optional[list[StatItem]] = Field(default=None, description="KPI stat items")
    chart_type: Optional[ChartType] = Field(default=None, description="Chart type")
    chart_categories: Optional[list[str]] = Field(default=None, description="Category axis labels")
    chart_series: Optional[list[ChartSeries]] = Field(default=None, description="Chart data series")
    chart_title: Optional[str] = Field(default=None, description="Chart title")
    steps: Optional[list[str]] = Field(default=None, description="Process steps")
    cards: Optional[list[CardItem]] = Field(default=None, description="Feature cards")
    timeline: Optional[list[TimelineItem]] = Field(default=None, description="Timeline items")
    left_column: Optional[CompareColumn] = Field(default=None, description="Compare left column")
    right_column: Optional[CompareColumn] = Field(default=None, description="Compare right column")
    table: Optional[TableData] = Field(default=None, description="Table data")
    image_url: Optional[str] = Field(default=None, description="Optional background/inset image URL")
    body_text: Optional[str] = Field(default=None, description="Free-form body text for blank slides")


# ── Slide ──────────────────────────────────────────────────────────────────────

class Slide(BaseModel):
    id: str = Field(default_factory=SlideID.generate, description="Stable slide ID")
    type: SlideType = Field(description="Slide layout type")
    heading: str = Field(default="", description="Slide heading / title")
    content: SlideContent = Field(default_factory=SlideContent)
    speaker_notes: str = Field(default="", description="Speaker notes for this slide")
    locked: bool = Field(default=False, description="When True, edit engine will not modify this slide")
    metadata: dict[str, Any] = Field(default_factory=dict, description="Arbitrary metadata")

    @field_validator("id", mode="before")
    @classmethod
    def validate_slide_id(cls, v: Any) -> str:
        if v is None:
            return SlideID.generate()
        if isinstance(v, str) and SlideID.is_valid(v):
            return v
        # Accept any non-empty string for flexibility but warn
        if isinstance(v, str) and v:
            return v
        return SlideID.generate()


# ── Deck Plan ──────────────────────────────────────────────────────────────────

class DeckPlan(BaseModel):
    slides: list[Slide] = Field(default_factory=list)
    total_slides: int = Field(default=0, description="Total number of slides")
    estimated_duration_min: float = Field(default=0.0, description="Estimated presentation duration in minutes")

    @model_validator(mode="after")
    def sync_total_slides(self) -> "DeckPlan":
        self.total_slides = len(self.slides)
        if self.estimated_duration_min == 0.0 and self.slides:
            self.estimated_duration_min = len(self.slides) * 1.5
        return self


# ── Top-Level Deck ─────────────────────────────────────────────────────────────

class Deck(BaseModel):
    meta: DeckMeta = Field(description="Deck metadata and change log")
    theme: DeckTheme = Field(default_factory=DeckTheme, description="Visual theme")
    plan: DeckPlan = Field(description="The slides plan")


# ── Brand Rules ────────────────────────────────────────────────────────────────

class BrandRule(BaseModel):
    rule_id: str = Field(default_factory=lambda: str(uuid.uuid4())[:8])
    name: str = Field(description="Human-readable rule name")
    rule_type: str = Field(description="Type: font|color|layout|content")
    condition: str = Field(description="Description of what the rule enforces")
    allowed_values: list[str] = Field(default_factory=list)
    severity: ValidationSeverity = Field(default=ValidationSeverity.warning)


# ── Validation Models ──────────────────────────────────────────────────────────

class ValidationIssue(BaseModel):
    issue_id: str = Field(default_factory=lambda: str(uuid.uuid4())[:8])
    severity: ValidationSeverity = Field(description="error|warning|info")
    code: str = Field(description="Machine-readable issue code")
    message: str = Field(description="Human-readable issue description")
    slide_id: Optional[str] = Field(default=None)
    field: Optional[str] = Field(default=None)


class ValidationResult(BaseModel):
    passed: bool = Field(default=True, description="True if no error-severity issues")
    issues: list[ValidationIssue] = Field(default_factory=list)
    error_count: int = Field(default=0)
    warning_count: int = Field(default=0)
    info_count: int = Field(default=0)

    @model_validator(mode="after")
    def count_issues(self) -> "ValidationResult":
        self.error_count = sum(1 for i in self.issues if i.severity == ValidationSeverity.error)
        self.warning_count = sum(1 for i in self.issues if i.severity == ValidationSeverity.warning)
        self.info_count = sum(1 for i in self.issues if i.severity == ValidationSeverity.info)
        self.passed = self.error_count == 0
        return self


# ── API Request / Response Models ──────────────────────────────────────────────

class CreateDeckRequest(BaseModel):
    title: str = Field(description="Presentation title")
    description: str = Field(default="")
    nl_prompt: str = Field(default="", description="Natural language description for AI planning")
    slide_count: int = Field(default=8, ge=1, le=50, description="Target number of slides")
    template_id: Optional[str] = Field(default=None, description="Template to use")
    author: str = Field(default="system")
    theme: Optional[DeckTheme] = Field(default=None, description="Theme override")


class PlanRequest(BaseModel):
    nl_prompt: str = Field(description="Natural language description of desired content")
    title: str = Field(default="Untitled Presentation")
    slide_count: int = Field(default=8, ge=1, le=50)
    template_id: Optional[str] = Field(default=None)
    theme: Optional[DeckTheme] = Field(default=None)


class RenderRequest(BaseModel):
    deck: Deck = Field(description="The deck to render")
    output_format: str = Field(default="pptx", description="Output format (only pptx supported)")


class EditRequest(BaseModel):
    deck_id: str = Field(description="ID of the deck to edit")
    slide_ids: list[str] = Field(description="Stable IDs of slides to modify")
    updated_slides: list[Slide] = Field(description="New slide definitions (must match slide_ids)")
    author: str = Field(default="system")
    description: str = Field(default="Edit slides")
    confirm: bool = Field(default=False, description="Set True to confirm destructive operations")

    @model_validator(mode="after")
    def validate_ids_match(self) -> "EditRequest":
        if len(self.slide_ids) != len(self.updated_slides):
            raise ValueError("slide_ids and updated_slides must have the same length")
        return self


class EditResult(BaseModel):
    deck: Deck = Field(description="Updated deck")
    pptx_bytes: bytes = Field(description="Rendered PPTX bytes")
    changed_slide_ids: list[str] = Field(description="IDs of slides that were actually modified")
    version: int = Field(description="New deck version after edit")


class RenderResult(BaseModel):
    deck_id: str
    version: int
    pptx_bytes: bytes
    slide_count: int
    output_path: Optional[str] = None


class VersionInfo(BaseModel):
    version: int
    created_at: datetime
    author: str
    action: str
    description: str
    slide_count: int


class UpdateResult(BaseModel):
    deck_id: str
    new_version: int
    change: ChangeEntry


class ConfirmationRequired(Exception):
    """Raised when a destructive operation requires explicit confirmation."""

    def __init__(self, message: str, summary: dict[str, Any]):
        super().__init__(message)
        self.summary = summary


# ── Parsed Template (from template_parser) ────────────────────────────────────

class PlaceholderInfo(BaseModel):
    idx: int
    placeholder_type: str
    left: int
    top: int
    width: int
    height: int
    name: str = ""


class LayoutInfo(BaseModel):
    name: str
    placeholders: list[PlaceholderInfo] = Field(default_factory=list)
    slide_width: int = 0
    slide_height: int = 0


class MasterInfo(BaseModel):
    name: str
    layout_count: int


class ParsedTemplate(BaseModel):
    theme_colors: ThemeColors = Field(default_factory=ThemeColors)
    theme_fonts: ThemeFonts = Field(default_factory=ThemeFonts)
    layouts: list[LayoutInfo] = Field(default_factory=list)
    masters: list[MasterInfo] = Field(default_factory=list)
    slide_count: int = 0
    slide_width: int = 9144000
    slide_height: int = 6858000
    source_path: str = ""
