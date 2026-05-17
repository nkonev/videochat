CREATE EXTENSION pg_trgm;

CREATE OR REPLACE FUNCTION strip_tags(TEXT) RETURNS TEXT AS $$
SELECT regexp_replace($1, '<[^>]*>', '', 'g')
$$ LANGUAGE SQL;

CREATE OR REPLACE FUNCTION cyrillic_transliterate(p_string text) RETURNS character varying AS
$BODY$
SELECT replace(replace(replace(replace(replace(replace(replace(replace(translate(lower($1),'абвгдеёзийклмнопрстуфхцэы','abvgdeezijklmnoprstufхcey'), 'ж', 'zh'), 'ч', 'ch'), 'ш', 'sh'), 'щ', 'sch'), 'ъ', ''), 'ю', 'yu'), 'я', 'ya'), 'ь', '');
$BODY$
LANGUAGE SQL IMMUTABLE COST 100;

create sequence chat_id_sequence;

create unlogged table thread( -- aka chat_common; there is  thread with parent_chat_id=0 in case root chat
    id bigint bigint not null,
    parent_chat_id bigint not null default 0, -- 0 is for root chat itself
    last_generated_message_id bigint not null default 0,
    create_date_time timestamp not null,
    title varchar(512) not null,
    fts_title tsvector generated always as (to_tsvector('russian', title)) stored,
    avatar text,
    avatar_big text,
    last_message_id bigint,
    last_message_content text,
    last_message_owner_id bigint,
    primary key(parent_chat_id, id)
);

create unlogged table chat( -- aka parent_chat, aka chat_settings
    id bigint primary key,
    tet_a_tet boolean not null,
    available_to_search boolean not null,
    can_resend boolean not null,
    can_react BOOLEAN NOT NULL,
    regular_participant_can_publish_message boolean not null,
    regular_participant_can_pin_message BOOLEAN NOT NULL,
    regular_participant_can_write_message BOOLEAN NOT NULL,
    regular_participant_can_add_participant BOOLEAN NOT NULL,
    participants_count bigint not null default 0,
    last_n_participant_ids bigint[] not null default array[]::bigint[], -- last N
    can_create_thread boolean not null,
    create_date_time timestamp not null,
    last_generated_thread_id bigint not null default 0
);

create unlogged table chat_participant(
    user_id bigint,
    chat_id bigint,
    create_date_time timestamp not null,
    chat_admin boolean not null default false,
    cp_last_read_message_id bigint not null default 0,
    cp_last_read_message_date_time timestamp,
    primary key(user_id, chat_id)
);
SELECT create_distributed_table('chat_participant', 'chat_id');

create unlogged table message(
    id bigint,
    parent_chat_id bigint, -- linked with thread(parent_chat_id), 0 in case root chat
    thread_id bigint,  -- linked with thread(id)
    owner_id bigint not null,
    content text not null,
    blog_post boolean not null default false,
    embed jsonb,
    file_item_uuid varchar(36),
    published boolean not null default false, -- just a denormalized copy
    pinned boolean not null default false, -- just a denormalized copy
    create_date_time timestamp not null,
    update_date_time timestamp,
    fts_all_content tsvector generated always as (to_tsvector('russian', strip_tags(coalesce(content, '')) || ' ' || strip_tags(coalesce(embed ->> 'embedMessageContent', '')))) stored,
    primary key (parent_chat_id, thread_id, id)
);
SELECT create_distributed_table('message', 'chat_id');

CREATE unlogged TABLE message_reaction(
    parent_chat_id BIGINT,
    thread_id bigint,
    user_id BIGINT,
    reaction VARCHAR(4),
    message_id BIGINT,
    create_date_time timestamp not null,
    PRIMARY KEY (parent_chat_id, thread_id, message_id, user_id, reaction),
    FOREIGN KEY (message_id, parent_chat_id, thread_id) REFERENCES message(id, parent_chat_id, thread_id) ON DELETE CASCADE
);

-- https://docs.citusdata.com/en/v11.1/develop/api_udf.html#example
SELECT create_distributed_table('message_reaction', 'chat_id', colocate_with => 'message');

CREATE unlogged TABLE message_pinned(
    message_id BIGINT,
    parent_chat_id BIGINT,
    thread_id bigint,
    owner_id bigint not null,
    create_date_time timestamp not null,
    update_date_time timestamp,
    preview text not null,
    promoted boolean not null,
    PRIMARY KEY (parent_chat_id, thread_id, message_id),
    FOREIGN KEY (message_id, parent_chat_id, thread_id) REFERENCES message(id, parent_chat_id, thread_id) ON DELETE CASCADE
);

SELECT create_distributed_table('message_pinned', 'chat_id', colocate_with => 'message');

CREATE unlogged TABLE message_published(
    message_id BIGINT,
    parent_chat_id BIGINT,
    thread_id bigint,
    owner_id bigint not null,
    create_date_time timestamp not null,
    update_date_time timestamp,
    preview text not null,
    PRIMARY KEY (parent_chat_id, thread_id, message_id),
    FOREIGN KEY (message_id, parent_chat_id, thread_id) REFERENCES message(id, parent_chat_id, thread_id) ON DELETE CASCADE
);

SELECT create_distributed_table('message_published', 'chat_id', colocate_with => 'message');

create unlogged table chat_user_view(
    id bigint,
    parent_chat_id bigint,
    pinned boolean not null default false,
    user_id bigint,
    update_date_time timestamp not null,
    consider_messages_as_unread BOOLEAN not null default true,
    unread_messages bigint not null default 0,
    cuv_last_read_message_id bigint not null default 0,
    primary key (user_id, parent_chat_id, id)
);
SELECT create_distributed_table('chat_user_view', 'user_id');

create unlogged table has_unread_messages(user_id bigint primary key, has boolean not null default false);
SELECT create_distributed_table('has_unread_messages', 'user_id');

create unlogged table blog(
    id int primary key,
    blog_about boolean not null,
    owner_id bigint,
    message_id bigint,
    title varchar(256) not null,
    image_url text,
    post text,
    preview varchar(512),
    create_date_time timestamp not null,
    update_date_time timestamp,
    file_item_uuid varchar(36),
    fts_all_content tsvector generated always as (to_tsvector('russian', strip_tags(coalesce(title, '')) || ' ' || strip_tags(coalesce(post, '')))) stored
);
