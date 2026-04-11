-- Skills registry
CREATE TABLE IF NOT EXISTS skills (
    id               TEXT PRIMARY KEY,
    name             TEXT NOT NULL,
    kind             TEXT NOT NULL DEFAULT 'text',
    description      TEXT NOT NULL DEFAULT '',
    output_mime_type TEXT NOT NULL DEFAULT '',
    output_file_ext  TEXT NOT NULL DEFAULT '',
    prompt_template  TEXT NOT NULL DEFAULT '',
    built_in         INTEGER NOT NULL DEFAULT 0,
    enabled          INTEGER NOT NULL DEFAULT 1,
    tags             TEXT NOT NULL DEFAULT '',
    input_mime_types TEXT NOT NULL DEFAULT '',
    created_at       TEXT NOT NULL,
    updated_at       TEXT NOT NULL
);
