-- Base tables
CREATE TABLE folder (
    id INTEGER PRIMARY KEY,
    parent_id INTEGER REFERENCES folder(id) ON DELETE CASCADE,
    title TEXT NOT NULL CHECK(length(title) <= 30),
    FOREIGN KEY (parent_id) REFERENCES folder(id) ON DELETE CASCADE
);

CREATE TABLE tag (
    id INTEGER PRIMARY KEY,
    title TEXT NOT NULL UNIQUE CHECK(length(title) <= 30)
);

CREATE TABLE command (
    id INTEGER PRIMARY KEY,
    creation_datetime TEXT NOT NULL DEFAULT (datetime('now')),
    modification_datetime TEXT NOT NULL DEFAULT (datetime('now')),
    title TEXT NOT NULL CHECK(length(title) <= 50),
    description TEXT,
    script TEXT NOT NULL,
    lint_issues TEXT,
    lint_status TEXT NOT NULL CHECK(lint_status IN ('NOT_AVAILABLE', 'OK', 'WARNING', 'ERROR', 'SHELLCHECK_FAILED')),
    elapsed INTEGER,
    status TEXT NOT NULL CHECK(status IN ('IMPORTED', 'SAVED', 'DELETED', 'OBSOLETE', 'BOOKMARKED')),
    folder_id INTEGER,
    FOREIGN KEY (folder_id) REFERENCES folder(id) ON DELETE CASCADE
);

CREATE TABLE command_has_tag (
    command_id INTEGER NOT NULL,
    tag_id INTEGER NOT NULL,
    PRIMARY KEY (command_id, tag_id),
    FOREIGN KEY (command_id) REFERENCES command(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tag(id) ON DELETE CASCADE
);

-- Indexes
CREATE INDEX idx_folder_parent_id ON folder(parent_id);
CREATE INDEX idx_command_folder ON command(folder_id);
CREATE INDEX idx_command_status ON command(status);
CREATE INDEX idx_command_script ON command(script);
CREATE INDEX idx_command_lint_status ON command(lint_status);
CREATE INDEX idx_command_creation ON command(creation_datetime);
CREATE INDEX idx_command_modification ON command(modification_datetime);

-- FTS5 virtual table for full-text search
CREATE VIRTUAL TABLE command_fts USING fts5(
    title,
    description,
    script,
    content='command',  -- Reference to external content table
    content_rowid='id'  -- Primary key of external content table
);

-- Triggers to keep FTS table in sync with base table
CREATE TRIGGER command_ai AFTER INSERT ON command BEGIN
    INSERT INTO command_fts(rowid, title, description, script)
    VALUES (new.id, new.title, new.description, new.script);
END;

CREATE TRIGGER command_ad AFTER DELETE ON command BEGIN
    INSERT INTO command_fts(command_fts, rowid, title, description, script)
    VALUES('delete', old.id, old.title, old.description, old.script);
END;

CREATE TRIGGER command_au AFTER UPDATE ON command BEGIN
    INSERT INTO command_fts(command_fts, rowid, title, description, script)
    VALUES('delete', old.id, old.title, old.description, old.script);
    INSERT INTO command_fts(rowid, title, description, script)
    VALUES (new.id, new.title, new.description, new.script);
END;
