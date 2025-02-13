INSERT INTO message(id, text, owner_id, chat_id) VALUES
(1, 'text 1', 1, 1);


INSERT INTO message(id, text, owner_id, chat_id)
	SELECT
	    (i + 2),
		'generated_message' || i || ' Lorem Ipsum - это текст-"рыба", часто используемый в печати и вэб-дизайне. Lorem Ipsum является стандартной "рыбой" для текстов на латинице с начала XVI века. В то время некий безымянный печатник создал большую коллекцию размеров и форм шрифтов, используя Lorem Ipsum для распечатки образцов. Lorem Ipsum не только успешно пережил без заметных изменений пять веков, но и перешагнул в электронный дизайн. Его популяризации в новое время послужили публикация листов Letraset с образцами Lorem Ipsum в 60-х годах и, в более недавнее время, программы электронной вёрстки типа Aldus PageMaker, в шаблонах которых используется Lorem Ipsum.',
		1,
		1
	FROM generate_series(0, 500) AS i;

UPDATE chat set last_generated_message_id = (select max(id) from message where chat_id = 1) where id = 1;
