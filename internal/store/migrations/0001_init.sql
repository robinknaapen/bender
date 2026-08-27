CREATE TABLE services (
    id          INTEGER PRIMARY KEY,
    preset      TEXT    NOT NULL DEFAULT '',
    name        TEXT    NOT NULL,
    url         TEXT    NOT NULL,
    profile     TEXT    NOT NULL UNIQUE,
    position    INTEGER NOT NULL,
    enabled     INTEGER NOT NULL DEFAULT 1,
    badge_regex TEXT    NOT NULL DEFAULT ''
);

CREATE TABLE settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
