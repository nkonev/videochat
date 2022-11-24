create table notification(
    id bigserial primary key,
    notification_type text not null,
    description text not null,
    message_id bigint,
    user_id bigint not null,
    chat_id bigint not null,
    unique (notification_type, message_id, user_id, chat_id)
);

CREATE INDEX message_id__user_id__idx ON notification (message_id, user_id);
CREATE INDEX id__user_id__idx ON notification (id, user_id);