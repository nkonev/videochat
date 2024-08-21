alter table user_account rename column role to roles;
alter table user_account alter column roles drop default;
alter table user_account alter column roles type user_role[] using array[roles];
alter table user_account alter column roles set default array['ROLE_USER'::user_role];
