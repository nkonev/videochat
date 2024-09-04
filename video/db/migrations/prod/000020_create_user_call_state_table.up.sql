create table user_call_state(
    token_id varchar(36) not null,
    user_id bigint not null,

    chat_id bigint not null,

    token_taken boolean not null,

    owner_token_id varchar(36),
    owner_user_id bigint,

    status varchar(32) not null,
    chat_tet_a_tet boolean not null default false,
    owner_avatar text,

    marked_for_remove_at timestamp,
    marked_for_orphan_remove_attempt smallint not null default 0,

    create_date_time timestamp not null default utc_now(),

    primary key (user_id, token_id)
);
