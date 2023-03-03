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
    hiring_story_id INTEGER NOT NULL,
    text TEXT NOT NULL,
    time INTEGER NOT NULL,
    status INTEGER,
    FOREIGN KEY(hiring_story_id) REFERENCES hiring_story(id) ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE hiring_job;
DROP TABLE hiring_story;
-- +goose StatementEnd
