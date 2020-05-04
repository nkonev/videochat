ALTER TABLE posts.post ALTER COLUMN title_img DROP NOT NULL;
UPDATE posts.post SET title_img = NULL WHERE title_img = '';