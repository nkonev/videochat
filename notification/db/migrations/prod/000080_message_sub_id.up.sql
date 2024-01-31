alter table notification add column message_sub_id text;

alter table notification drop constraint notification_notification_type_message_id_user_id_chat_id_key;

alter table notification add unique (user_id, chat_id, message_id, notification_type, message_sub_id);
