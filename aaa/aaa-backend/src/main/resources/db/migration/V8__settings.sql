CREATE SCHEMA IF NOT EXISTS settings;

CREATE TABLE settings.runtime_settings(
    KEY TEXT PRIMARY KEY,
    VALUE TEXT
);

insert into settings.runtime_settings values
('image.background', NULL),
('header', 'Блог Конева Никиты'),
('title.template', $$%s | nkonev's blog$$)
;

create table images.settings_image (
	id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
	img BYTEA,
	content_type VARCHAR(64),
	create_date_time timestamp NOT NULL DEFAULT (now() at time zone 'utc')
);