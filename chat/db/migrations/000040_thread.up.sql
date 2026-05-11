alter table chat_common
    add column parent_id bigint not null default 0,
    add column can_create_thread boolean not null default true;

create table child_chat_id (
    parent_id bigint primary key,
    last_generated_child_chat_id bigint not null default 0
);

alter table message add column child_chat_id bigint;