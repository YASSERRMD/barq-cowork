-- Migration 004: store API keys directly in provider_profiles
-- Keys are user-owned data in a local SQLite DB; previously only env var names were saved.
ALTER TABLE provider_profiles ADD COLUMN api_key TEXT NOT NULL DEFAULT '';
