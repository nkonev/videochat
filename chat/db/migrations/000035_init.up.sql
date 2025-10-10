-- if not exists becasue we use this table to preserve techical info (need to fast-worward, need to skip old db migration and so on)
create unlogged table if not exists technical(
    the_key varchar(256) primary key,
    the_value text
);
