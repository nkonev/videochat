ALTER TABLE message ADD COLUMN pinned BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE message ADD COLUMN pin_promoted BOOLEAN NOT NULL DEFAULT FALSE;