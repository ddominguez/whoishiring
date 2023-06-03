-- +goose Up
-- +goose StatementBegin
ALTER TABLE hiring_job ADD COLUMN seen INTEGER DEFAULT 0;
ALTER TABLE hiring_job ADD COLUMN saved INTEGER DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE hiring_job DROP COLUMN seen;
ALTER TABLE hiring_job DROP COLUMN saved;
-- +goose StatementEnd
