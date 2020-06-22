INSERT INTO chat(
    title,
    owner_id
)
VALUES
('first', 1),
('second', 1),
('Тест кириллицы', 1),
('lorem', 1),
('ipsum', 1),
('dolor', 1),
('sit', 1),
('amet', 1),
('With collegues', 1);

INSERT INTO chat (
    title,
    owner_id
)
	SELECT
		'generated_chat' || i,
		1
	FROM generate_series(2, 100) AS i;
