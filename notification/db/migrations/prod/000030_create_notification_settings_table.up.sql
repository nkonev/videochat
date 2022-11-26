create table notification_settings(
    user_id bigint primary key,
    mentions_enabled boolean not null default true,
    missed_calls_enabled boolean not null default true
);
