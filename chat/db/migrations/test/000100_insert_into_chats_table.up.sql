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