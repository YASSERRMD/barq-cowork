# Barq PPTX Engine — Architecture Document

## 1. System Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          Barq PPTX Engine                               │
│                                                                         │
│  ┌─────────────┐    ┌──────────────┐    ┌──────────────────────────┐   │
│  │   FastAPI   │───▶│   Planner    │───▶│      DeckPlan (JSON)     │   │
│  │   api.py    │    │  planner.py  │    │  list[Slide] + metadata  │   │
│  └──────┬──────┘    └──────────────┘    └────────────┬─────────────┘   │
│         │                                             │                 │
│         │           ┌──────────────────────────────────────────────┐   │
│         │           │          Presentation Renderer               │   │
│         ├──────────▶│              renderer.py                     │   │
│         │           │  ┌─────────────────────────────────────────┐ │   │
│         │           │  │           SlideTypeRegistry             │ │   │
│         │           │  │           slide_registry.py             │ │   │
│         │           │  │  TitleRenderer  BulletsRenderer  ...    │ │   │
│         │           │  └─────────────────────────────────────────┘ │   │
│         │           └────────────────────────┬─────────────────────┘   │
│         │                                    │ PPTX bytes               │
│         │           ┌────────────────────────▼─────────────────────┐   │
│         ├──────────▶│            Edit Engine                       │   │
│         │           │            edit_engine.py                    │   │
│         │           │  stable_id lookup → surgical patch           │   │
│         │           └────────────────────────┬─────────────────────┘   │
│         │                                    │                         │
│         │           ┌────────────────────────▼─────────────────────┐   │
│         ├──────────▶│            Deck Validator                    │   │
│         │           │            validator.py                      │   │
│         │           │  schema | content length | overflow | brand  │   │
│         │           └──────────────────────────────────────────────┘   │
│         │                                                               │
│         │           ┌──────────────────────────────────────────────┐   │
│         └──────────▶│            Deck Store                        │   │
│                     │            deck_store.py                     │   │
│                     │  SQLite WAL  decks + deck_versions tables    │   │
│                     └──────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘

External input paths:
  NL prompt ──▶ Planner (LLM or rule-based) ──▶ DeckPlan
  PPTX template ──▶ TemplateParser ──▶ ThemeColors/Fonts/BrandRules
  Deck JSON ──▶ Renderer ──▶ .pptx file
```

---

## 2. Component Responsibilities

| Component | File | Responsibility |
|---|---|---|
| Schema | `schema.py` | All Pydantic v2 models — single source of truth for every data structure |
| Planner | `planner.py` | Convert NL prompt → DeckPlan; LLM or rule-based |
| Template Parser | `template_parser.py` | Parse existing .pptx to extract theme, fonts, layouts, brand rules |
| Slide Registry | `slide_registry.py` | One SlideRenderer per SlideType; all rendering logic lives here |
| Renderer | `renderer.py` | Orchestrate prs creation, dispatch to registry, embed stable IDs |
| Edit Engine | `edit_engine.py` | Locate slides by stable ID, clear + re-render targeted slides only |
| Validator | `validator.py` | Schema, content-length, overflow, and brand compliance checks |
| Deck Store | `deck_store.py` | SQLite WAL storage, versioning, rollback, change log |
| API | `api.py` | FastAPI HTTP endpoints wiring all components together |

---

## 3. Data Flow

### 3.1 New deck creation

```
Client
  │
  │  POST /deck/create/json  { title, nl_prompt, slide_count, theme? }
  ▼
api.py
  │  create_plan(PlanRequest)
  ▼
planner.py
  │  ANTHROPIC_API_KEY? → Anthropic Messages API
  │  OPENAI_API_KEY?    → OpenAI Chat API
  │  else               → keyword rule-based planner
  │
  │  returns DeckPlan { slides: [Slide, ...] }
  ▼
api.py
  │  assemble Deck { meta, theme, plan }
  │  PresentationRenderer.render_to_bytes(deck)
  ▼
renderer.py
  │  for each slide in deck.plan.slides:
  │    add_slide(blank_layout)
  │    clear inherited shapes
  │    inject content._heading_override = slide.heading
  │    registry.get(slide.type).render(pptx_slide, content, theme)
  │    embed stable_id in XML extLst
  │    set speaker notes
  │  prs.save() → bytes
  ▼
validator.py
  │  validate(deck, pptx_bytes) → ValidationResult
  ▼
deck_store.py
  │  INSERT decks + deck_versions (snapshot v1)
  ▼
Client ← { deck JSON, validation, download_url }
```

### 3.2 Surgical edit

```
Client
  │  POST /deck/edit  { deck_id, slide_ids, updated_slides }
  ▼
api.py
  │  store.get(deck_id) → current Deck + pptx_bytes
  ▼
edit_engine.apply_edit(deck, edit_request, pptx_bytes)
  │
  │  1. Presentation(BytesIO(pptx_bytes))
  │  2. build stable_id → idx map (read extLst from each slide)
  │  3. For each target_id in edit_request.slide_ids:
  │       slide = prs.slides[id_to_idx[target_id]]
  │       _clear_slide_shapes(slide)           ← erase only shapes
  │       renderer.render(slide, new_content, theme)
  │       embed_slide_id(slide, target_id)     ← re-stamp ID
  │  4. prs.save() → result_bytes
  │  5. Verify untouched slides (hash comparison, warning on mismatch)
  │  6. Bump version, append ChangeEntry
  ▼
deck_store.update(deck_id, updated_deck, result_bytes, change)
  ▼
Client ← { deck JSON, changed_slide_ids, version, download_url }
```

---

## 4. Deck JSON as Source of Truth

The canonical representation of a presentation is `Deck`, serialised as JSON:

```
Deck
├── meta: DeckMeta        id, title, author, version, change_log[]
├── theme: DeckTheme      colors: ThemeColors, fonts: ThemeFonts
└── plan: DeckPlan
    └── slides: [Slide]
        ├── id            stable string "s-{8hex}" — survives edits
        ├── type          SlideType enum
        ├── heading       ≤60 chars
        ├── content       SlideContent (all fields Optional)
        ├── speaker_notes
        └── locked        bool — locked slides skip edit engine
```

The PPTX binary is a derived artefact generated from the Deck JSON.
Edits always go through the JSON model first, then re-render.
The store keeps both the JSON and the PPTX bytes at every version.

---

## 5. Slide Type Registry Pattern

```python
class SlideTypeRegistry:
    _renderers: dict[SlideType, SlideRenderer]

    def register(renderer: SlideRenderer)
    def get(slide_type: SlideType) -> SlideRenderer
    def build_default() -> SlideTypeRegistry   # registers all built-ins
```

`SlideRenderer` is a structural Protocol (not ABC):

```python
class SlideRenderer(Protocol):
    type: SlideType
    def render(self, slide: pptx.slide.Slide,
               content: SlideContent, theme: DeckTheme) -> None: ...
```

Adding a new slide type requires:
1. Create a class with `type = SlideType.my_type` and a `render()` method.
2. Call `registry.register(MyTypeRenderer())`.

No base class inheritance required; duck typing via Protocol.

---

## 6. Versioning and Change Log Strategy

Each mutation to a deck creates a new version snapshot:

```
decks table:        current state (mutable)
deck_versions table: append-only snapshots (immutable)

Deck.meta.version   monotonically increasing integer
Deck.meta.change_log  list[ChangeEntry] embedded in JSON
```

`ChangeEntry` records:
- `version`          the version this change creates
- `timestamp`        UTC ISO string
- `author`           user/system identifier
- `action`           create | edit | reorder | theme | delete_slide | rollback
- `slide_ids_affected` list of stable slide IDs
- `description`      human-readable summary

Rollback creates a NEW version (does not delete history):

```
v1 (create) → v2 (edit s-abc) → v3 (rollback to v1)
                                     ↑ deck at v3 has same content as v1
```

Destructive actions (theme change, slide deletion, reorder) require
`require_confirmation=False` or `confirm=True` in the request, otherwise
`ConfirmationRequired` is raised with a summary dict.

---

## 7. Template Parsing Approach

`TemplateParser.parse(pptx_path)` uses `python-pptx` + lxml XPath:

1. Load PPTX with `Presentation(path)`.
2. Walk `slide_masters[0].element` subtree looking for `a:clrScheme`.
3. Map theme color slot names (dk1, lt1, accent1 … accent6) to `ThemeColors` fields.
4. Walk `a:fontScheme` → `a:majorFont/a:latin[@typeface]` for heading font,
   `a:minorFont/a:latin[@typeface]` for body font.
5. Iterate `master.slide_layouts` collecting placeholder positions/sizes.
6. Return `ParsedTemplate` with all extracted data.

`generate_brand_rules(parsed)` converts the extracted values into
`BrandRule` objects that the validator can enforce.

---

## 8. Edit Engine: Stable Slide IDs and Surgical Patching

### 8.1 Stable ID Storage

Stable IDs are stored inside the slide XML using a custom extension element:

```xml
<p:sld>
  <p:cSld>
    <p:extLst>
      <p:ext uri="{barq.io/slideId}">
        <barq:slideId xmlns:barq="http://barq.io/pptx/ext/2024"
                      id="s-3f8a1b2c"/>
      </p:ext>
    </p:extLst>
    ...
  </p:cSld>
</p:sld>
```

This survives PPTX save/load and is not displayed by PowerPoint.

### 8.2 Surgical Patch Algorithm

```
prs = Presentation(BytesIO(pptx_bytes))
id_to_idx = {read_stable_id(slide): idx for idx, slide in enumerate(prs.slides)}

for target_id in edit_request.slide_ids:
    idx = id_to_idx[target_id]
    slide = prs.slides[idx]           # reference by index, NOT position add
    _clear_slide_shapes(slide)        # remove sp, pic, graphicFrame, etc.
    renderer.render(slide, new_content, theme)
    _embed_slide_id(slide, target_id) # re-stamp ID after re-render

prs.save(BytesIO()) → result_bytes
```

Key property: `prs.slides[idx]` is a direct reference to the existing
slide XML element. We modify it in-place without calling `add_slide`,
so slide order is completely preserved.

### 8.3 No-Touch Verification

After save, the engine loads both original and result PPTX, computes
SHA-256 of each slide's `_element` XML, and logs a warning if any
untouched slide differs. Minor differences due to python-pptx namespace
normalisation are expected and tolerated; shape content differences would
indicate a bug.

---

## 9. Validation Layers

| Layer | What it checks | Severity |
|---|---|---|
| Schema | Required fields, title present, at least 1 slide | error |
| Slide IDs | Uniqueness across all slides in the deck | error |
| Content length | Headings ≤60 chars, ≤6 bullets, 2–4 stats, ≤8 table cols, ≤6 cards/steps | warning |
| Rendered overflow | Heuristic text area vs shape area in rendered PPTX | warning |
| Brand compliance | Font names against allowed list from brand rules | info |

`ValidationResult.passed` is `True` only when `error_count == 0`.
Warnings and info do not block creation/export.

---

## 10. API Contract

### Base URL
`http://localhost:8001`

### Endpoints

#### POST /deck/plan
Request: `PlanRequest { nl_prompt, title, slide_count, template_id? }`
Response: `DeckPlan { slides[], total_slides, estimated_duration_min }`

#### POST /deck/render
Request: `RenderRequest { deck: Deck }`
Response: Binary PPTX (application/vnd.openxmlformats-officedocument.presentationml.presentation)
Header: `Content-Disposition: attachment; filename="...pptx"`

#### POST /deck/create/json
Request: `CreateDeckRequest { title, description, nl_prompt, slide_count, template_id?, author, theme? }`
Response: `{ deck: Deck, validation: ValidationResult, download_url: str, slide_count: int }`

#### POST /deck/create (multipart)
Fields: `request` (JSON string), `template` (optional .pptx file)
Response: Same as /deck/create/json

#### POST /deck/edit
Request: `EditRequest { deck_id, slide_ids[], updated_slides[], author, description, confirm }`
Response: `{ deck: Deck, changed_slide_ids: [], version: int, download_url: str }`
Error 409: `ConfirmationRequired { message, summary, requires_confirm: true }`

#### POST /deck/validate
Request: `Deck`
Response: `ValidationResult { passed, issues[], error_count, warning_count, info_count }`

#### GET /deck/{id}
Response: `{ deck: Deck, current_version: int, versions: VersionInfo[] }`

#### GET /deck/{id}/export?version=N
Response: Binary PPTX download

#### GET /deck/{id}/changelog
Response: `ChangeEntry[]`

#### POST /deck/{id}/rollback?to_version=N
Response: `{ deck: Deck, rolled_back_to: int, new_version: int }`

#### GET /health
Response: `{ status: "ok", service: "barq-pptx-engine", version: "1.0.0" }`

### Error Codes
- `400` Bad request / invalid JSON
- `404` Deck or version not found
- `409` Confirmation required for destructive operation
- `422` Validation error (Pydantic)
- `500` Internal server error (render/storage failure)
