alter table user_account
    add column override_add_permissions varchar[]
    ,add column override_remove_permissions varchar[]
;