delete from notification;
alter table notification
    add column chat_title text not null
;
