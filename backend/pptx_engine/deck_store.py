"""
SQLite-backed versioned deck storage with WAL mode.

Schema:
  decks(id, meta_json, pptx_blob, current_version, created_at, updated_at)
  deck_versions(id, deck_id, version, meta_json, pptx_blob, created_at)
"""

from __future__ import annotations

import json
import sqlite3
import uuid
from datetime import datetime
from typing import Optional

from .schema import (
    ChangeEntry,
    ConfirmationRequired,
    Deck,
    DeckMeta,
    DeckPlan,
    DeckTheme,
    UpdateResult,
    VersionInfo,
)


def _now_iso() -> str:
    return datetime.utcnow().isoformat()


def _deck_from_row(meta_json: str) -> Deck:
    """Deserialise a Deck from its JSON representation."""
    return Deck.model_validate_json(meta_json)


class DeckStore:
    """Versioned SQLite storage for decks and their PPTX blobs."""

    def __init__(self, db_path: str = "decks.db"):
        self._db_path = db_path
        self._init_db()

    # ── DB setup ───────────────────────────────────────────────────────────────

    def _init_db(self) -> None:
        with self._connect() as conn:
            conn.execute("""
                CREATE TABLE IF NOT EXISTS decks (
                    id              TEXT PRIMARY KEY,
                    meta_json       TEXT NOT NULL,
                    pptx_blob       BLOB NOT NULL,
                    current_version INTEGER NOT NULL DEFAULT 1,
                    created_at      TEXT NOT NULL,
                    updated_at      TEXT NOT NULL
                )
            """)
            conn.execute("""
                CREATE TABLE IF NOT EXISTS deck_versions (
                    id          TEXT PRIMARY KEY,
                    deck_id     TEXT NOT NULL,
                    version     INTEGER NOT NULL,
                    meta_json   TEXT NOT NULL,
                    pptx_blob   BLOB NOT NULL,
                    created_at  TEXT NOT NULL,
                    UNIQUE(deck_id, version)
                )
            """)
            conn.execute("CREATE INDEX IF NOT EXISTS idx_dv_deck_id ON deck_versions(deck_id)")

    def _connect(self) -> sqlite3.Connection:
        conn = sqlite3.connect(self._db_path, timeout=30)
        conn.execute("PRAGMA journal_mode=WAL")
        conn.execute("PRAGMA foreign_keys=ON")
        conn.row_factory = sqlite3.Row
        return conn

    # ── CRUD ───────────────────────────────────────────────────────────────────

    def create(self, deck: Deck, pptx_bytes: bytes) -> str:
        """Persist a new deck. Returns deck.meta.id."""
        now = _now_iso()
        meta_json = deck.model_dump_json()
        version = deck.meta.version

        with self._connect() as conn:
            conn.execute(
                """
                INSERT INTO decks (id, meta_json, pptx_blob, current_version, created_at, updated_at)
                VALUES (?, ?, ?, ?, ?, ?)
                """,
                (deck.meta.id, meta_json, pptx_bytes, version, now, now),
            )
            # Also snapshot as version 1
            conn.execute(
                """
                INSERT INTO deck_versions (id, deck_id, version, meta_json, pptx_blob, created_at)
                VALUES (?, ?, ?, ?, ?, ?)
                """,
                (str(uuid.uuid4()), deck.meta.id, version, meta_json, pptx_bytes, now),
            )

        return deck.meta.id

    def get(self, deck_id: str) -> Optional[tuple[Deck, bytes]]:
        """Retrieve the current version of a deck. Returns (Deck, pptx_bytes) or None."""
        with self._connect() as conn:
            row = conn.execute(
                "SELECT meta_json, pptx_blob FROM decks WHERE id = ?",
                (deck_id,),
            ).fetchone()

        if row is None:
            return None
        deck = _deck_from_row(row["meta_json"])
        return deck, bytes(row["pptx_blob"])

    def update(
        self,
        deck_id: str,
        deck: Deck,
        pptx_bytes: bytes,
        change: ChangeEntry,
        require_confirmation: bool = True,
    ) -> UpdateResult:
        """
        Overwrite the current deck version and append a snapshot to deck_versions.

        If require_confirmation=True and the change action is destructive
        (theme change, slide deletion, reorder), raise ConfirmationRequired.
        """
        if require_confirmation:
            self._check_destructive_update(change)

        now = _now_iso()
        meta_json = deck.model_dump_json()
        new_version = deck.meta.version

        with self._connect() as conn:
            conn.execute(
                """
                UPDATE decks
                SET meta_json=?, pptx_blob=?, current_version=?, updated_at=?
                WHERE id=?
                """,
                (meta_json, pptx_bytes, new_version, now, deck_id),
            )
            # Snapshot
            conn.execute(
                """
                INSERT OR REPLACE INTO deck_versions
                    (id, deck_id, version, meta_json, pptx_blob, created_at)
                VALUES (?, ?, ?, ?, ?, ?)
                """,
                (str(uuid.uuid4()), deck_id, new_version, meta_json, pptx_bytes, now),
            )

        return UpdateResult(
            deck_id=deck_id,
            new_version=new_version,
            change=change,
        )

    def list_versions(self, deck_id: str) -> list[VersionInfo]:
        """Return all version snapshots for a deck, ordered by version asc."""
        with self._connect() as conn:
            rows = conn.execute(
                """
                SELECT version, meta_json, created_at
                FROM deck_versions
                WHERE deck_id = ?
                ORDER BY version ASC
                """,
                (deck_id,),
            ).fetchall()

        result: list[VersionInfo] = []
        for row in rows:
            try:
                deck = _deck_from_row(row["meta_json"])
                last_change = deck.meta.change_log[-1] if deck.meta.change_log else None
                result.append(
                    VersionInfo(
                        version=row["version"],
                        created_at=datetime.fromisoformat(row["created_at"]),
                        author=last_change.author if last_change else "system",
                        action=last_change.action if last_change else "create",
                        description=last_change.description if last_change else "",
                        slide_count=len(deck.plan.slides),
                    )
                )
            except Exception:
                continue
        return result

    def get_version(
        self, deck_id: str, version: int
    ) -> Optional[tuple[Deck, bytes]]:
        """Retrieve a specific historical version."""
        with self._connect() as conn:
            row = conn.execute(
                """
                SELECT meta_json, pptx_blob
                FROM deck_versions
                WHERE deck_id = ? AND version = ?
                """,
                (deck_id, version),
            ).fetchone()

        if row is None:
            return None
        deck = _deck_from_row(row["meta_json"])
        return deck, bytes(row["pptx_blob"])

    def rollback(self, deck_id: str, to_version: int) -> Deck:
        """
        Roll back a deck to a historical version.
        Creates a new version entry that is a copy of the target version.
        """
        result = self.get_version(deck_id, to_version)
        if result is None:
            raise ValueError(f"Version {to_version} not found for deck {deck_id}")

        target_deck, pptx_bytes = result

        # Get current version to increment
        current = self.get(deck_id)
        if current is None:
            raise ValueError(f"Deck {deck_id} not found")
        current_deck, _ = current
        new_version = current_deck.meta.version + 1

        rollback_change = ChangeEntry(
            version=new_version,
            timestamp=datetime.utcnow(),
            author="system",
            action="rollback",
            slide_ids_affected=[],
            description=f"Rolled back to version {to_version}",
        )

        new_log = list(target_deck.meta.change_log) + [rollback_change]
        new_meta = target_deck.meta.model_copy(update={
            "version": new_version,
            "updated_at": datetime.utcnow(),
            "change_log": new_log,
        })
        rolled_deck = target_deck.model_copy(update={"meta": new_meta})

        self.update(
            deck_id,
            rolled_deck,
            pptx_bytes,
            rollback_change,
            require_confirmation=False,
        )
        return rolled_deck

    def delete(self, deck_id: str, confirm: bool = False) -> None:
        """
        Delete a deck and all its versions.
        Requires confirm=True to prevent accidental deletion.
        """
        if not confirm:
            raise ConfirmationRequired(
                "Deck deletion requires confirm=True.",
                {"deck_id": deck_id, "action": "delete"},
            )

        with self._connect() as conn:
            conn.execute("DELETE FROM deck_versions WHERE deck_id = ?", (deck_id,))
            conn.execute("DELETE FROM decks WHERE id = ?", (deck_id,))

    def get_change_log(self, deck_id: str) -> list[ChangeEntry]:
        """Return the full change log for a deck."""
        result = self.get(deck_id)
        if result is None:
            return []
        deck, _ = result
        return deck.meta.change_log

    def list_decks(self) -> list[dict]:
        """List all deck IDs and titles (for browsing)."""
        with self._connect() as conn:
            rows = conn.execute(
                "SELECT id, current_version, created_at, updated_at FROM decks ORDER BY updated_at DESC"
            ).fetchall()

        result = []
        for row in rows:
            result.append({
                "id": row["id"],
                "current_version": row["current_version"],
                "created_at": row["created_at"],
                "updated_at": row["updated_at"],
            })
        return result

    # ── Internal ───────────────────────────────────────────────────────────────

    def _check_destructive_update(self, change: ChangeEntry) -> None:
        """Raise ConfirmationRequired for destructive actions."""
        destructive_actions = {"theme", "delete_slide", "reorder"}
        if change.action in destructive_actions:
            raise ConfirmationRequired(
                f"Action '{change.action}' is destructive and requires explicit confirmation. "
                "Pass require_confirmation=False or set confirm=True in the request.",
                {
                    "action": change.action,
                    "affected_slide_ids": change.slide_ids_affected,
                },
            )
