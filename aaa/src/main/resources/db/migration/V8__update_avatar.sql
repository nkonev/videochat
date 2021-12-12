UPDATE users SET avatar = replace(avatar, '/storage/public/avatar', '/storage/public/user/avatar');
UPDATE users SET avatar_big = replace(avatar_big, 'old_text', 'new_text');