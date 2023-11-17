alter table user_account add column confirmed boolean;
update user_account set confirmed = enabled;
update user_account set enabled = true;
alter table user_account alter column confirmed set not null;
alter table user_account alter column confirmed set default false;
