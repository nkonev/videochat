create table notification_settings_chat
(
    user_id              bigint not null,
    chat_id              bigint not null,
    mentions_enabled     boolean,
    missed_calls_enabled boolean,
    answers_enabled      boolean,
    reactions_enabled    boolean,
    primary key (user_id, chat_id)
);
