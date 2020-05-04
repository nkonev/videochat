-- insert test data
INSERT INTO auth.users(username, password, avatar, email) VALUES
	('nikita', '$2a$10$e3pEnL2d3RB7jBrlEA3B9eUhayb/bmEG1V35h.4EhdReUAMzlAWxS', '/654853-user-men-2-512.png', 'nikita@example.com'), -- bcrypt('password', 10)
	('alice', '$2a$10$e3pEnL2d3RB7jBrlEA3B9eUhayb/bmEG1V35h.4EhdReUAMzlAWxS', '/girl-512.png', 'alice@example.com'),
	('bob', '$2a$10$e3pEnL2d3RB7jBrlEA3B9eUhayb/bmEG1V35h.4EhdReUAMzlAWxS', NULL, 'bob@example.com'),
	('John Smith', '$2a$10$e3pEnL2d3RB7jBrlEA3B9eUhayb/bmEG1V35h.4EhdReUAMzlAWxS', NULL, 'jsmith@example.com')
;
-- insert many test users
INSERT INTO auth.users (username, password, avatar, email)
	SELECT
    'generated_user_' || i,
    '$2a$10$0nGRZ4Quy0hW2W.prjc.AOyUkNqgFulVckZQ.gFsOly5ESntrW7E.', -- bcrypt('generated_user_password', 10)
    CASE
      WHEN i % 2 = 0 THEN '/Avatar_Alien-512.png'
      ELSE NULL
    END,
		'generated' || i || '@example.com'
	FROM generate_series(0, 1000) AS i;


UPDATE auth.users SET role = 'ROLE_ADMIN' WHERE id = (SELECT id FROM auth.users WHERE username = 'admin');

-- insert additional users and roles
INSERT INTO auth.users(username, password, avatar, email) VALUES
	('forgot-password-user', '$2a$10$e3pEnL2d3RB7jBrlEA3B9eUhayb/bmEG1V35h.4EhdReUAMzlAWxS', NULL, 'forgot-password-user@example.com');


INSERT INTO posts.post (title, text, title_img, owner_id)
	SELECT
		'generated_post_' || i,
		'Lorem Ipsum - это текст-"рыба", часто используемый в печати и вэб-дизайне. Lorem Ipsum является стандартной "рыбой" для текстов на латинице с начала XVI века. В то время некий безымянный печатник создал большую коллекцию размеров и форм шрифтов, используя Lorem Ipsum для распечатки образцов. Lorem Ipsum не только успешно пережил без заметных изменений пять веков, но и перешагнул в электронный дизайн. Его популяризации в новое время послужили публикация листов Letraset с образцами Lorem Ipsum в 60-х годах и, в более недавнее время, программы электронной вёрстки типа Aldus PageMaker, в шаблонах которых используется Lorem Ipsum.',
		'/logo_mono.png',
		(SELECT id FROM auth.users WHERE username = 'nikita')
	FROM generate_series(0, 100) AS i;

INSERT INTO posts.comment (text, post_id, owner_id)
	SELECT
		'generated_comment' || i || ' Lorem Ipsum - это текст-"рыба", часто используемый в печати и вэб-дизайне. Lorem Ipsum является стандартной "рыбой" для текстов на латинице с начала XVI века. В то время некий безымянный печатник создал большую коллекцию размеров и форм шрифтов, используя Lorem Ipsum для распечатки образцов. Lorem Ipsum не только успешно пережил без заметных изменений пять веков, но и перешагнул в электронный дизайн. Его популяризации в новое время послужили публикация листов Letraset с образцами Lorem Ipsum в 60-х годах и, в более недавнее время, программы электронной вёрстки типа Aldus PageMaker, в шаблонах которых используется Lorem Ipsum.',
		(SELECT id from posts.post ORDER BY id DESC LIMIT 1), -- get last id
    CASE
      WHEN i%2 = 0 THEN (SELECT id FROM auth.users WHERE username = 'nikita')
      ELSE (SELECT id FROM auth.users WHERE username = 'alice')
    END
	FROM generate_series(0, 500) AS i;

INSERT INTO posts.post (title, text, owner_id) VALUES
('Hi from kafka', $$
Consumer has failed with exception: org.apache.kafka.clients.consumer.CommitFailedException: Commit cannot be completed due to group rebalance
class com.messagehub.consumer.Consumer is shutting down.
org.apache.kafka.clients.consumer.CommitFailedException: Commit cannot be completed due to group rebalance
at org.apache.kafka.clients.consumer.internals.ConsumerCoordinator$OffsetCommitResponseHandler.handle(ConsumerCoordinator.java:546)
at org.apache.kafka.clients.consumer.internals.ConsumerCoordinator$OffsetCommitResponseHandler.handle(ConsumerCoordinator.java:487)
at org.apache.kafka.clients.consumer.internals.AbstractCoordinator$CoordinatorResponseHandler.onSuccess(AbstractCoordinator.java:681)
at org.apache.kafka.clients.consumer.internals.AbstractCoordinator$CoordinatorResponseHandler.onSuccess(AbstractCoordinator.java:654)
at org.apache.kafka.clients.consumer.internals.RequestFuture$1.onSuccess(RequestFuture.java:167)
at org.apache.kafka.clients.consumer.internals.RequestFuture.fireSuccess(RequestFuture.java:133)
at org.apache.kafka.clients.consumer.internals.RequestFuture.complete(RequestFuture.java:107)
at org.apache.kafka.clients.consumer.internals.ConsumerNetworkClient$RequestFutureCompletionHandler.onComplete(ConsumerNetworkClient.java:350)
at org.apache.kafka.clients.NetworkClient.poll(NetworkClient.java:288)
at org.apache.kafka.clients.consumer.internals.ConsumerNetworkClient.clientPoll(ConsumerNetworkClient.java:303)
at org.apache.kafka.clients.consumer.internals.ConsumerNetworkClient.poll(ConsumerNetworkClient.java:197)
at org.apache.kafka.clients.consumer.internals.ConsumerNetworkClient.poll(ConsumerNetworkClient.java:187)
at org.apache.kafka.clients.consumer.internals.ConsumerNetworkClient.poll(ConsumerNetworkClient.java:157)
at org.apache.kafka.clients.consumer.internals.ConsumerCoordinator.commitOffsetsSync(ConsumerCoordinator.java:352)
at org.apache.kafka.clients.consumer.KafkaConsumer.commitSync(KafkaConsumer.java:936)
at org.apache.kafka.clients.consumer.KafkaConsumer.commitSync(KafkaConsumer.java:905)
$$, (SELECT id FROM auth.users WHERE username = 'John Smith'));


-- insert additional post with comment and images for delete
INSERT INTO posts.post (title, text, title_img, owner_id) VALUES
	('for delete with comments', 'text. This post will be deleted.', '/logo_mono.png', (SELECT id FROM auth.users WHERE username = 'nikita'));
INSERT INTO posts.comment (text, post_id, owner_id) VALUES
	('commment', (SELECT id from posts.post ORDER BY id DESC LIMIT 1), (SELECT id FROM auth.users WHERE username = 'alice'));

UPDATE posts.post SET draft = TRUE WHERE id = 84;