INSERT INTO chat (title, blog)
SELECT 'generated_chat' || i, true FROM generate_series(1, 1000) AS i;
