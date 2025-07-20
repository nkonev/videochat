create unlogged table metadata_cache(
   -- S3 key; see ./services.files.go::GetKey()
   chat_id bigint not null,
   file_item_uuid varchar(36) not null,
   filename varchar(255) not null, -- see ./utils/utils.go::GeneralMaxFilenameLength

   -- S3 metadata (unchangeable)
   owner_user_id bigint not null,
   correlation_id varchar(36),

    -- S3 tags (changeable)
   published boolean not null,

   file_size bigint not null,

   create_date_time timestamp not null default utc_now(),
   edit_date_time timestamp not null default utc_now(),

   primary key (chat_id, file_item_uuid, filename)
);

SELECT create_distributed_table('metadata_cache', 'chat_id');

create index idx_belongs on metadata_cache(chat_id, file_item_uuid, owner_user_id);
