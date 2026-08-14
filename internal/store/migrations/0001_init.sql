CREATE TABLE repos (
  id                INTEGER PRIMARY KEY,
  source            TEXT NOT NULL,
  source_id         TEXT NOT NULL,
  owner             TEXT NOT NULL,
  name              TEXT NOT NULL,
  full_name         TEXT NOT NULL,
  description       TEXT,
  url               TEXT NOT NULL,
  homepage          TEXT,
  primary_language  TEXT,
  topics_text       TEXT NOT NULL DEFAULT '',
  stars             INTEGER NOT NULL DEFAULT 0,
  forks             INTEGER NOT NULL DEFAULT 0,
  is_archived       INTEGER NOT NULL DEFAULT 0,
  is_fork           INTEGER NOT NULL DEFAULT 0,
  license           TEXT,
  default_branch    TEXT,
  pushed_at         TIMESTAMP,
  repo_created_at   TIMESTAMP,
  repo_updated_at   TIMESTAMP,
  readme            TEXT,
  readme_path       TEXT,
  first_seen        TIMESTAMP NOT NULL,
  last_synced       TIMESTAMP NOT NULL,
  UNIQUE(source, owner, name)
);

CREATE TABLE topics (
  repo_id INTEGER NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
  topic   TEXT NOT NULL,
  PRIMARY KEY (repo_id, topic)
);
CREATE INDEX idx_topics_topic ON topics(topic);
CREATE INDEX idx_repos_language ON repos(primary_language);
CREATE INDEX idx_repos_pushed_at ON repos(pushed_at);
CREATE INDEX idx_repos_source ON repos(source);

CREATE VIRTUAL TABLE repos_fts USING fts5(
  full_name, description, readme, topics_text,
  content='repos', content_rowid='id',
  tokenize='porter unicode61'
);

CREATE TRIGGER repos_ai AFTER INSERT ON repos BEGIN
  INSERT INTO repos_fts(rowid, full_name, description, readme, topics_text)
  VALUES (new.id, new.full_name, new.description, new.readme, new.topics_text);
END;

CREATE TRIGGER repos_ad AFTER DELETE ON repos BEGIN
  INSERT INTO repos_fts(repos_fts, rowid, full_name, description, readme, topics_text)
  VALUES ('delete', old.id, old.full_name, old.description, old.readme, old.topics_text);
END;

CREATE TRIGGER repos_au AFTER UPDATE ON repos BEGIN
  INSERT INTO repos_fts(repos_fts, rowid, full_name, description, readme, topics_text)
  VALUES ('delete', old.id, old.full_name, old.description, old.readme, old.topics_text);
  INSERT INTO repos_fts(rowid, full_name, description, readme, topics_text)
  VALUES (new.id, new.full_name, new.description, new.readme, new.topics_text);
END;
