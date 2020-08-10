INSERT INTO chat(title)
VALUES
('first'),
('second'),
('Тест кириллицы'),
('lorem'),
('ipsum'),
('dolor'),
('sit'),
('amet'),
('With collegues');

INSERT INTO chat (title)
	SELECT 'generated_chat' || i
	FROM generate_series(2, 100) AS i;
