"""
Tests for DeckValidator.

Verifies that:
- heading > 60 chars produces a warning
- > 6 bullets produces a warning
- duplicate slide IDs produce an error
- a valid deck passes all checks
"""

from __future__ import annotations

import os
import sys

import pytest

sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", ".."))

from pptx_engine.validator import DeckValidator
from pptx_engine.schema import (
    ChangeEntry,
    Deck,
    DeckMeta,
    DeckPlan,
    DeckTheme,
    Slide,
    SlideContent,
    SlideID,
    SlideType,
    StatItem,
    TableData,
    ValidationSeverity,
)


# ── Fixtures ──────────────────────────────────────────────────────────────────

@pytest.fixture
def validator():
    return DeckValidator()


def _deck(slides: list[Slide]) -> Deck:
    meta = DeckMeta(
        title="Validator Test Deck",
        author="test",
        change_log=[
            ChangeEntry(version=1, author="test", action="create",
                        slide_ids_affected=[], description="test")
        ],
    )
    return Deck(meta=meta, theme=DeckTheme(), plan=DeckPlan(slides=slides))


def _valid_slide(**kwargs) -> Slide:
    defaults = {
        "id": SlideID.generate(),
        "type": SlideType.bullets,
        "heading": "A Valid Heading",
        "content": SlideContent(points=["Point 1", "Point 2"]),
    }
    defaults.update(kwargs)
    return Slide(**defaults)


# ── Tests ─────────────────────────────────────────────────────────────────────

class TestValidDeck:
    def test_valid_deck_passes(self, validator):
        """A well-formed deck with valid content passes validation."""
        deck = _deck([
            Slide(id=SlideID.generate(), type=SlideType.title,
                  heading="My Presentation", content=SlideContent(subtitle="Welcome")),
            _valid_slide(heading="Overview"),
            _valid_slide(heading="Details"),
        ])
        result = validator.validate(deck)
        assert result.passed, f"Expected pass but got issues: {result.issues}"
        assert result.error_count == 0

    def test_empty_issues_on_valid_deck(self, validator):
        """A valid deck produces zero issues of severity 'error'."""
        deck = _deck([_valid_slide()])
        result = validator.validate(deck)
        errors = [i for i in result.issues if i.severity == ValidationSeverity.error]
        assert len(errors) == 0


class TestHeadingLength:
    def test_heading_over_60_chars_produces_warning(self, validator):
        """A heading longer than 60 characters generates a WARNING."""
        long_heading = "A" * 61
        deck = _deck([_valid_slide(heading=long_heading)])
        result = validator.validate(deck)

        heading_warnings = [
            i for i in result.issues
            if i.code == "HEADING_TOO_LONG" and i.severity == ValidationSeverity.warning
        ]
        assert len(heading_warnings) >= 1, (
            f"Expected HEADING_TOO_LONG warning, got: {result.issues}"
        )

    def test_heading_exactly_60_chars_no_warning(self, validator):
        """A heading of exactly 60 characters does NOT generate a warning."""
        heading = "A" * 60
        deck = _deck([_valid_slide(heading=heading)])
        result = validator.validate(deck)

        heading_warnings = [
            i for i in result.issues if i.code == "HEADING_TOO_LONG"
        ]
        assert len(heading_warnings) == 0

    def test_heading_59_chars_no_warning(self, validator):
        """A heading of 59 characters does NOT generate a warning."""
        heading = "A" * 59
        deck = _deck([_valid_slide(heading=heading)])
        result = validator.validate(deck)

        heading_warnings = [
            i for i in result.issues if i.code == "HEADING_TOO_LONG"
        ]
        assert len(heading_warnings) == 0


class TestBulletCount:
    def test_more_than_6_bullets_produces_warning(self, validator):
        """A bullet slide with > 6 points generates a TOO_MANY_BULLETS warning."""
        deck = _deck([
            _valid_slide(content=SlideContent(
                points=["Bullet 1", "Bullet 2", "Bullet 3", "Bullet 4",
                        "Bullet 5", "Bullet 6", "Bullet 7"]
            ))
        ])
        result = validator.validate(deck)

        bullet_warnings = [
            i for i in result.issues
            if i.code == "TOO_MANY_BULLETS" and i.severity == ValidationSeverity.warning
        ]
        assert len(bullet_warnings) >= 1, (
            f"Expected TOO_MANY_BULLETS warning, got: {result.issues}"
        )

    def test_exactly_6_bullets_no_warning(self, validator):
        """A bullet slide with exactly 6 points does NOT generate a warning."""
        deck = _deck([
            _valid_slide(content=SlideContent(
                points=["B1", "B2", "B3", "B4", "B5", "B6"]
            ))
        ])
        result = validator.validate(deck)

        bullet_warnings = [i for i in result.issues if i.code == "TOO_MANY_BULLETS"]
        assert len(bullet_warnings) == 0

    def test_3_bullets_no_warning(self, validator):
        """Three bullet points do not trigger any bullet warning."""
        deck = _deck([_valid_slide(content=SlideContent(points=["A", "B", "C"]))])
        result = validator.validate(deck)

        bullet_warnings = [i for i in result.issues if i.code == "TOO_MANY_BULLETS"]
        assert len(bullet_warnings) == 0


class TestDuplicateSlideIds:
    def test_duplicate_slide_ids_produce_error(self, validator):
        """Duplicate slide IDs produce a DUPLICATE_SLIDE_ID error."""
        shared_id = SlideID.generate()
        deck = _deck([
            _valid_slide(id=shared_id, heading="Slide A"),
            _valid_slide(id=shared_id, heading="Slide B"),  # same ID!
        ])
        result = validator.validate(deck)

        dup_errors = [
            i for i in result.issues
            if i.code == "DUPLICATE_SLIDE_ID" and i.severity == ValidationSeverity.error
        ]
        assert len(dup_errors) >= 1, (
            f"Expected DUPLICATE_SLIDE_ID error, got: {result.issues}"
        )

    def test_duplicate_slide_ids_fail_validation(self, validator):
        """A deck with duplicate slide IDs does not pass validation."""
        shared_id = SlideID.generate()
        deck = _deck([
            _valid_slide(id=shared_id, heading="Slide A"),
            _valid_slide(id=shared_id, heading="Slide B"),
        ])
        result = validator.validate(deck)
        assert not result.passed

    def test_unique_slide_ids_pass_uniqueness_check(self, validator):
        """A deck with unique slide IDs passes the uniqueness check."""
        deck = _deck([
            _valid_slide(id=SlideID.generate(), heading="Slide A"),
            _valid_slide(id=SlideID.generate(), heading="Slide B"),
        ])
        result = validator.validate(deck)
        dup_errors = [i for i in result.issues if i.code == "DUPLICATE_SLIDE_ID"]
        assert len(dup_errors) == 0


class TestSchemaValidation:
    def test_missing_title_produces_error(self, validator):
        """A deck with an empty title produces a MISSING_TITLE error."""
        # Bypass Pydantic validation by constructing directly
        meta = DeckMeta(title="placeholder", author="test")
        deck = Deck(meta=meta, theme=DeckTheme(), plan=DeckPlan(slides=[_valid_slide()]))
        # Manually clear title after construction
        object.__setattr__(deck.meta, "title", "")
        result = validator.validate(deck)

        errors = [i for i in result.issues if i.code == "MISSING_TITLE"]
        assert len(errors) >= 1

    def test_no_slides_produces_error(self, validator):
        """A deck with no slides produces a NO_SLIDES error."""
        meta = DeckMeta(title="Empty Deck", author="test")
        deck = Deck(meta=meta, theme=DeckTheme(), plan=DeckPlan(slides=[]))
        result = validator.validate(deck)

        errors = [i for i in result.issues if i.code == "NO_SLIDES"]
        assert len(errors) >= 1
        assert not result.passed


class TestContentLengthValidation:
    def test_too_few_stats_produces_warning(self, validator):
        """A stats slide with 1 stat produces a TOO_FEW_STATS warning."""
        deck = _deck([
            Slide(
                id=SlideID.generate(),
                type=SlideType.stats,
                heading="Metrics",
                content=SlideContent(stats=[StatItem(value="99%", label="Only one")]),
            )
        ])
        result = validator.validate(deck)
        warnings = [i for i in result.issues if i.code == "TOO_FEW_STATS"]
        assert len(warnings) >= 1

    def test_too_many_stats_produces_warning(self, validator):
        """A stats slide with 5 stats produces a TOO_MANY_STATS warning."""
        deck = _deck([
            Slide(
                id=SlideID.generate(),
                type=SlideType.stats,
                heading="Metrics",
                content=SlideContent(stats=[
                    StatItem(value=str(i) + "%", label=f"M{i}") for i in range(5)
                ]),
            )
        ])
        result = validator.validate(deck)
        warnings = [i for i in result.issues if i.code == "TOO_MANY_STATS"]
        assert len(warnings) >= 1

    def test_too_many_table_columns_produces_warning(self, validator):
        """A table with 9 columns produces a TOO_MANY_TABLE_COLS warning."""
        headers = [f"Col{i}" for i in range(9)]
        deck = _deck([
            Slide(
                id=SlideID.generate(),
                type=SlideType.table,
                heading="Wide Table",
                content=SlideContent(table=TableData(
                    headers=headers,
                    rows=[["val"] * 9],
                )),
            )
        ])
        result = validator.validate(deck)
        warnings = [i for i in result.issues if i.code == "TOO_MANY_TABLE_COLS"]
        assert len(warnings) >= 1


class TestValidationResultCounts:
    def test_counts_are_accurate(self, validator):
        """ValidationResult counts match the actual issues list."""
        # Cause one error (duplicate) and one warning (long heading)
        shared_id = SlideID.generate()
        deck = _deck([
            _valid_slide(id=shared_id, heading="A" * 61),  # warning
            _valid_slide(id=shared_id, heading="Slide B"),  # error
        ])
        result = validator.validate(deck)
        assert result.error_count == sum(
            1 for i in result.issues if i.severity == ValidationSeverity.error
        )
        assert result.warning_count == sum(
            1 for i in result.issues if i.severity == ValidationSeverity.warning
        )
