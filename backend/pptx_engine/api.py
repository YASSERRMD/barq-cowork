"""
FastAPI application for the Barq PPTX Engine.

Endpoints:
  POST /deck/create      - Create a new deck from natural language + optional template
  POST /deck/plan        - Generate a DeckPlan from a natural language prompt
  POST /deck/render      - Render an existing Deck JSON to PPTX bytes
  POST /deck/edit        - Surgically edit specific slides
  POST /deck/validate    - Validate a Deck JSON
  GET  /deck/{id}        - Get current Deck JSON + version info
  GET  /deck/{id}/export - Download current PPTX file
"""

from __future__ import annotations

import io
import os
import tempfile
import uuid
from datetime import datetime
from typing import Optional

from fastapi import FastAPI, File, HTTPException, Response, UploadFile
from fastapi.responses import JSONResponse, StreamingResponse
from pydantic import BaseModel

from .schema import (
    ChangeEntry,
    ConfirmationRequired,
    CreateDeckRequest,
    Deck,
    DeckMeta,
    DeckPlan,
    DeckTheme,
    EditRequest,
    PlanRequest,
    RenderRequest,
    ValidationResult,
)
from .deck_store import DeckStore
from .edit_engine import EditEngine
from .planner import create_plan
from .renderer import PresentationRenderer
from .slide_registry import SlideTypeRegistry
from .template_parser import TemplateParser
from .validator import DeckValidator

# ── App + singletons ───────────────────────────────────────────────────────────

app = FastAPI(
    title="Barq PPTX Engine",
    version="1.0.0",
    description="AI-powered presentation engine for Barq Cowork",
)

_registry = SlideTypeRegistry.build_default()
_renderer = PresentationRenderer(_registry)
_edit_engine = EditEngine(_renderer)
_validator = DeckValidator()
_store = DeckStore(db_path=os.environ.get("DECK_DB_PATH", "decks.db"))
_template_parser = TemplateParser()


# ── Response helpers ───────────────────────────────────────────────────────────

def _pptx_response(pptx_bytes: bytes, filename: str = "presentation.pptx") -> Response:
    return Response(
        content=pptx_bytes,
        media_type="application/vnd.openxmlformats-officedocument.presentationml.presentation",
        headers={"Content-Disposition": f'attachment; filename="{filename}"'},
    )


# ── Endpoints ──────────────────────────────────────────────────────────────────

@app.post("/deck/plan")
async def plan_deck(request: PlanRequest):
    """
    Generate a DeckPlan from a natural language prompt.
    Uses LLM if API key available, otherwise rule-based.
    """
    plan = create_plan(request)
    return plan.model_dump()


@app.post("/deck/render")
async def render_deck(request: RenderRequest):
    """
    Render a Deck JSON to a PPTX file download.
    """
    try:
        pptx_bytes = _renderer.render_to_bytes(request.deck)
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Render error: {e}")

    filename = f"{request.deck.meta.title.replace(' ', '_')}.pptx"
    return _pptx_response(pptx_bytes, filename)


@app.post("/deck/create")
async def create_deck(
    request: Optional[str] = None,
    template: Optional[UploadFile] = File(default=None),
):
    """
    Create a new deck from JSON body + optional PPTX template upload.

    Accepts multipart/form-data with:
      - request: JSON string of CreateDeckRequest
      - template: optional PPTX file upload

    Or plain JSON body for CreateDeckRequest.
    """
    # Parse the request body
    import json as _json

    if request is not None:
        try:
            req_data = _json.loads(request)
            create_req = CreateDeckRequest.model_validate(req_data)
        except Exception as e:
            raise HTTPException(status_code=422, detail=f"Invalid request JSON: {e}")
    else:
        raise HTTPException(status_code=422, detail="'request' field is required")

    # Handle template upload
    template_path: Optional[str] = None
    brand_rules = []
    if template is not None:
        tmp = tempfile.NamedTemporaryFile(delete=False, suffix=".pptx")
        try:
            tmp.write(await template.read())
            tmp.flush()
            template_path = tmp.name
            try:
                parsed = _template_parser.parse(template_path)
                brand_rules = _template_parser.generate_brand_rules(parsed)
            except Exception:
                pass
        finally:
            tmp.close()

    # Build plan
    plan_req = PlanRequest(
        nl_prompt=create_req.nl_prompt or create_req.title,
        title=create_req.title,
        slide_count=create_req.slide_count,
        template_id=create_req.template_id,
    )
    plan = create_plan(plan_req)

    # Assemble deck
    meta = DeckMeta(
        id=str(uuid.uuid4()),
        title=create_req.title,
        description=create_req.description,
        template_id=create_req.template_id,
        author=create_req.author,
        created_at=datetime.utcnow(),
        updated_at=datetime.utcnow(),
        version=1,
        change_log=[
            ChangeEntry(
                version=1,
                author=create_req.author,
                action="create",
                slide_ids_affected=[s.id for s in plan.slides],
                description=f"Initial creation: {create_req.title}",
            )
        ],
    )
    theme = create_req.theme or DeckTheme()
    deck = Deck(meta=meta, theme=theme, plan=plan)

    # Render
    try:
        pptx_bytes = _renderer.render_to_bytes(deck)
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Render error: {e}")

    # Validate
    validator = DeckValidator(brand_rules=brand_rules)
    validation = validator.validate(deck, pptx_bytes)

    # Store
    try:
        _store.create(deck, pptx_bytes)
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Storage error: {e}")

    # Clean up temp file
    if template_path:
        try:
            os.unlink(template_path)
        except Exception:
            pass

    return JSONResponse(
        content={
            "deck": deck.model_dump(mode="json"),
            "validation": validation.model_dump(mode="json"),
            "download_url": f"/deck/{deck.meta.id}/export",
            "slide_count": len(deck.plan.slides),
        }
    )


@app.post("/deck/create/json")
async def create_deck_json(create_req: CreateDeckRequest):
    """
    Create a new deck from a JSON body (no template upload).
    """
    plan_req = PlanRequest(
        nl_prompt=create_req.nl_prompt or create_req.title,
        title=create_req.title,
        slide_count=create_req.slide_count,
        template_id=create_req.template_id,
    )
    plan = create_plan(plan_req)

    meta = DeckMeta(
        id=str(uuid.uuid4()),
        title=create_req.title,
        description=create_req.description,
        template_id=create_req.template_id,
        author=create_req.author,
        created_at=datetime.utcnow(),
        updated_at=datetime.utcnow(),
        version=1,
        change_log=[
            ChangeEntry(
                version=1,
                author=create_req.author,
                action="create",
                slide_ids_affected=[s.id for s in plan.slides],
                description=f"Initial creation: {create_req.title}",
            )
        ],
    )
    theme = create_req.theme or DeckTheme()
    deck = Deck(meta=meta, theme=theme, plan=plan)

    try:
        pptx_bytes = _renderer.render_to_bytes(deck)
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Render error: {e}")

    validation = _validator.validate(deck, pptx_bytes)

    try:
        _store.create(deck, pptx_bytes)
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Storage error: {e}")

    return JSONResponse(
        content={
            "deck": deck.model_dump(mode="json"),
            "validation": validation.model_dump(mode="json"),
            "download_url": f"/deck/{deck.meta.id}/export",
            "slide_count": len(deck.plan.slides),
        }
    )


@app.post("/deck/edit")
async def edit_deck(edit_req: EditRequest):
    """
    Surgically edit specific slides in an existing deck.
    Only targeted slides are re-rendered; all others are preserved.
    """
    result = _store.get(edit_req.deck_id)
    if result is None:
        raise HTTPException(status_code=404, detail=f"Deck {edit_req.deck_id} not found")

    current_deck, pptx_bytes = result

    try:
        edit_result = _edit_engine.apply_edit(current_deck, edit_req, pptx_bytes)
    except ConfirmationRequired as e:
        raise HTTPException(
            status_code=409,
            detail={"message": str(e), "summary": e.summary, "requires_confirm": True},
        )
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Edit error: {e}")

    # Persist updated deck
    change = edit_result.deck.meta.change_log[-1] if edit_result.deck.meta.change_log else ChangeEntry(
        version=edit_result.version,
        author=edit_req.author,
        action="edit",
        slide_ids_affected=edit_result.changed_slide_ids,
        description=edit_req.description,
    )

    try:
        _store.update(
            edit_req.deck_id,
            edit_result.deck,
            edit_result.pptx_bytes,
            change,
            require_confirmation=False,
        )
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Storage error: {e}")

    return JSONResponse(
        content={
            "deck": edit_result.deck.model_dump(mode="json"),
            "changed_slide_ids": edit_result.changed_slide_ids,
            "version": edit_result.version,
            "download_url": f"/deck/{edit_req.deck_id}/export",
        }
    )


@app.post("/deck/validate")
async def validate_deck(deck: Deck):
    """
    Validate a Deck JSON and return a ValidationResult.
    Does not require a stored deck; validates the provided JSON.
    """
    validation = _validator.validate(deck)
    return validation.model_dump(mode="json")


@app.get("/deck/{deck_id}")
async def get_deck(deck_id: str):
    """
    Return the current Deck JSON plus version history summary.
    """
    result = _store.get(deck_id)
    if result is None:
        raise HTTPException(status_code=404, detail=f"Deck {deck_id} not found")

    deck, _ = result
    versions = _store.list_versions(deck_id)

    return JSONResponse(
        content={
            "deck": deck.model_dump(mode="json"),
            "current_version": deck.meta.version,
            "versions": [v.model_dump(mode="json") for v in versions],
        }
    )


@app.get("/deck/{deck_id}/export")
async def export_deck(deck_id: str, version: Optional[int] = None):
    """
    Download the PPTX file for a deck.
    Optionally specify ?version=N for a historical version.
    """
    if version is not None:
        result = _store.get_version(deck_id, version)
        if result is None:
            raise HTTPException(
                status_code=404,
                detail=f"Version {version} not found for deck {deck_id}",
            )
        deck, pptx_bytes = result
    else:
        result = _store.get(deck_id)
        if result is None:
            raise HTTPException(status_code=404, detail=f"Deck {deck_id} not found")
        deck, pptx_bytes = result

    filename = f"{deck.meta.title.replace(' ', '_')}_v{deck.meta.version}.pptx"
    return _pptx_response(pptx_bytes, filename)


@app.get("/deck/{deck_id}/changelog")
async def get_changelog(deck_id: str):
    """Return the full change log for a deck."""
    log = _store.get_change_log(deck_id)
    return [entry.model_dump(mode="json") for entry in log]


@app.post("/deck/{deck_id}/rollback")
async def rollback_deck(deck_id: str, to_version: int):
    """Roll back a deck to a specific historical version."""
    try:
        rolled_deck = _store.rollback(deck_id, to_version)
    except ValueError as e:
        raise HTTPException(status_code=404, detail=str(e))
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Rollback error: {e}")

    return JSONResponse(
        content={
            "deck": rolled_deck.model_dump(mode="json"),
            "rolled_back_to": to_version,
            "new_version": rolled_deck.meta.version,
        }
    )


@app.get("/health")
async def health():
    return {"status": "ok", "service": "barq-pptx-engine", "version": "1.0.0"}


# ── Entry point ────────────────────────────────────────────────────────────────

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8001, reload=True)
