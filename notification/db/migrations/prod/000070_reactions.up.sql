alter table notification_settings
    add column reactions_enabled boolean not null default false;
