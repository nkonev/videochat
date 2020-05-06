INSERT INTO auth.users(username, password, avatar, email) VALUES
-- bcrypt('admin', 10)
	('admin', '$2a$10$HsyFGy9IO//nJZxYc2xjDeV/kF7koiPrgIDzPOfgmngKVe9cOyOS2', 'https://cdn3.iconfinder.com/data/icons/rcons-user-action/32/boy-512.png', 'admin@example.com') ON CONFLICT DO NOTHING;
