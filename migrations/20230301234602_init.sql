-- +goose Up
-- +goose StatementBegin
CREATE TABLE hiring_story (
    id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    hn_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    time INTEGER NOT NULL
);
CREATE TABLE hiring_job (
    id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    hn_id INTEGER NOT NULL,
    hiring_story_hn_id INTEGER NOT NULL,
    text TEXT NOT NULL,
    time INTEGER NOT NULL,
    status INTEGER,
    FOREIGN KEY(hiring_story_hn_id) REFERENCES hiring_story(hn_id) ON DELETE CASCADE
);
CREATE INDEX hs_hn_id_index ON hiring_job (hiring_story_hn_id);
CREATE INDEX time_index ON hiring_job (time);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE hiring_job;
DROP TABLE hiring_story;
-- +goose StatementEnd
