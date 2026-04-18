alter table chat_common
    add column last_generated_thread_id bigint not null default 0,
    add column can_create_thread boolean not null default true;

alter table message add column thread_id bigint;