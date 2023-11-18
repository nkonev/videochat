-- insert test data
INSERT INTO user_account(username, password, avatar, email, confirmed) VALUES
	('nikita', '$2a$10$e3pEnL2d3RB7jBrlEA3B9eUhayb/bmEG1V35h.4EhdReUAMzlAWxS', '/654853-user-men-2-512.png', 'nikita@example.com', true), -- bcrypt('password', 10)
	('alice', '$2a$10$e3pEnL2d3RB7jBrlEA3B9eUhayb/bmEG1V35h.4EhdReUAMzlAWxS', '/girl-512.png', 'alice@example.com', true),
	('bob', '$2a$10$e3pEnL2d3RB7jBrlEA3B9eUhayb/bmEG1V35h.4EhdReUAMzlAWxS', NULL, 'bob@example.com', true),
	('John Smith', '$2a$10$e3pEnL2d3RB7jBrlEA3B9eUhayb/bmEG1V35h.4EhdReUAMzlAWxS', NULL, 'jsmith@example.com', true)
;
-- insert many test users
INSERT INTO user_account (username, password, avatar, email, confirmed)
	SELECT
    'generated_user_' || i,
    '$2a$10$0nGRZ4Quy0hW2W.prjc.AOyUkNqgFulVckZQ.gFsOly5ESntrW7E.', -- bcrypt('generated_user_password', 10)
    CASE
      WHEN i % 2 = 0 THEN '/Avatar_Alien-512.png'
      ELSE NULL
    END,
		'generated' || i || '@example.com',
         true
	FROM generate_series(0, 1000) AS i;


UPDATE user_account SET role = 'ROLE_ADMIN' WHERE id = (SELECT id FROM user_account WHERE username = 'admin');

-- insert additional users and roles
INSERT INTO user_account(username, password, avatar, email, confirmed) VALUES
	('forgot-password-user', '$2a$10$e3pEnL2d3RB7jBrlEA3B9eUhayb/bmEG1V35h.4EhdReUAMzlAWxS', NULL, 'forgot-password-user@example.com', true);

