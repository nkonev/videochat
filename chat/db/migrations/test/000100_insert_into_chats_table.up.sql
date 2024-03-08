DO
$do$
    DECLARE
        chat_name text;
    BEGIN
        FOR chat_name IN SELECT 'generated_chat' || i FROM generate_series(1, 1000) AS i LOOP
            PERFORM CREATE_CHAT(chat_name);
        END LOOP;
    END
$do$;
