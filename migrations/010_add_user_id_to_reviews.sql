-- 010_add_user_id_to_reviews.sql

ALTER TABLE reviews ADD COLUMN user_id UUID REFERENCES users(id);

-- If there are existing reviews, you might want to link them to a default user or handle them accordingly.
-- For now, we'll just add the column.
