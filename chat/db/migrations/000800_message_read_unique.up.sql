alter table message_read drop column id;
alter table message_read add primary key (message_id, user_id);