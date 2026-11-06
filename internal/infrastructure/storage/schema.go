package storage

import "webvigil/internal/application"

var errNotFound = application.ErrNotFound

// migrations are applied in order; the database's user_version is the number applied. A step is
// never edited once released: a change is a new step.
var migrations = []string{schemaV1}

// schemaV1 is the first schema. Change events refer to snapshots by id without a foreign key, so
// retention can decide which snapshots to keep without a cascade deciding for it.
const schemaV1 = `
CREATE TABLE sites (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    name              TEXT    NOT NULL,
    url               TEXT    NOT NULL,
    enabled           INTEGER NOT NULL,
    interval_seconds  INTEGER NOT NULL,
    timeout_seconds   INTEGER NOT NULL,
    detection_mode    TEXT    NOT NULL,
    extraction_json   TEXT    NOT NULL,
    ignore_json       TEXT    NOT NULL,
    notify_sound      INTEGER NOT NULL,
    notify_visual     INTEGER NOT NULL,
    require_ack       INTEGER NOT NULL,
    failure_threshold INTEGER NOT NULL
);
CREATE TABLE site_state (
    site_id              INTEGER PRIMARY KEY REFERENCES sites(id) ON DELETE CASCADE,
    last_outcome         TEXT    NOT NULL DEFAULT '',
    last_category        TEXT    NOT NULL DEFAULT '',
    last_detail          TEXT    NOT NULL DEFAULT '',
    last_http_status     INTEGER NOT NULL DEFAULT 0,
    last_checked_at      INTEGER NOT NULL DEFAULT 0,
    last_success_at      INTEGER NOT NULL DEFAULT 0,
    last_change_at       INTEGER NOT NULL DEFAULT 0,
    baseline_snapshot_id INTEGER NOT NULL DEFAULT 0,
    baseline_hash        TEXT    NOT NULL DEFAULT '',
    pending_change_id    INTEGER NOT NULL DEFAULT 0,
    consecutive_failures INTEGER NOT NULL DEFAULT 0,
    failure_warned       INTEGER NOT NULL DEFAULT 0,
    etag                 TEXT    NOT NULL DEFAULT '',
    last_modified        TEXT    NOT NULL DEFAULT '',
    next_due_at          INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE check_history (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id      INTEGER NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    at           INTEGER NOT NULL,
    request_url  TEXT    NOT NULL,
    final_url    TEXT    NOT NULL,
    http_status  INTEGER NOT NULL,
    duration_ms  INTEGER NOT NULL,
    content_type TEXT    NOT NULL,
    bytes        INTEGER NOT NULL,
    outcome      TEXT    NOT NULL,
    category     TEXT    NOT NULL,
    detail       TEXT    NOT NULL,
    hash         TEXT    NOT NULL,
    change_kind  TEXT    NOT NULL,
    manual       INTEGER NOT NULL
);
CREATE INDEX idx_check_history_site ON check_history(site_id, id);
CREATE TABLE snapshots (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id     INTEGER NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    hash        TEXT    NOT NULL,
    text        TEXT    NOT NULL,
    fields_json TEXT    NOT NULL,
    captured_at INTEGER NOT NULL
);
CREATE INDEX idx_snapshots_site ON snapshots(site_id, id);
CREATE TABLE change_events (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id          INTEGER NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    prev_snapshot_id INTEGER NOT NULL,
    new_snapshot_id  INTEGER NOT NULL,
    prev_hash        TEXT    NOT NULL,
    new_hash         TEXT    NOT NULL,
    summary          TEXT    NOT NULL,
    detected_at      INTEGER NOT NULL,
    acknowledged_at  INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_change_events_site ON change_events(site_id, id);
CREATE TABLE application_settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
`
