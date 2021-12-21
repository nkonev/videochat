CREATE OR REPLACE FUNCTION CREATE_CHAT(IN chat_name TEXT) RETURNS BIGINT AS $$
    DECLARE
            chat_id BIGINT;
            query1 text;
    BEGIN
        INSERT INTO chat(title) VALUES(chat_name) RETURNING id INTO chat_id;
        query1 := format('CREATE TABLE %s() INHERITS (message)', 'message_chat_' || chat_id);
        EXECUTE query1;
        query1 := format('ALTER TABLE %s ADD PRIMARY KEY(id);', 'message_chat_' || chat_id);
        EXECUTE query1;
        query1 := format('ALTER TABLE %s ADD FOREIGN KEY (chat_id) REFERENCES chat(id);', 'message_chat_' || chat_id);
        EXECUTE query1;
        RETURN chat_id;
    END
$$ LANGUAGE plpgsql;

DO
$do$
    DECLARE
        chat_names  text[];
        chat_name text;
    BEGIN
        chat_names := ARRAY ['first', 'second', 'Тест кириллицы', 'lorem', 'ipsum', 'dolor', 'sit', 'amet', 'With collegues'];
        FOREACH chat_name IN ARRAY chat_names LOOP
        PERFORM CREATE_CHAT(chat_name);
        END LOOP;
    END
$do$;

DO
$do$
    DECLARE
        chat_name text;
    BEGIN
        FOR chat_name IN SELECT 'generated_chat' || i FROM generate_series(2, 100) AS i LOOP
            PERFORM CREATE_CHAT(chat_name);
        END LOOP;
    END
$do$;