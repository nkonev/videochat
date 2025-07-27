create index idx_view_asc on metadata_cache(chat_id, file_item_uuid, create_date_time);
create index idx_view_desc on metadata_cache(chat_id, file_item_uuid, create_date_time desc);
