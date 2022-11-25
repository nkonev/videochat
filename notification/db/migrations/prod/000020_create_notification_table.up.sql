create table notification(
    id bigserial primary key,
    notification_type text not null,
    description text,
    message_id bigint,
    user_id bigint not null,
    chat_id bigint not null,
    create_date_time TIMESTAMP NOT NULL DEFAULT utc_now(),
    unique (notification_type, message_id, user_id, chat_id)
);

CREATE INDEX message_id__user_id__idx ON notification (message_id, user_id);
CREATE INDEX user_id__idx ON notification (user_id);