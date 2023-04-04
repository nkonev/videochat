delete from notification;
alter table notification
    add column by_user_id bigint not null,
    add column by_login text not null
;
