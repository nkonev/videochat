CREATE TABLE chat_participant (
    id BIGSERIAL PRIMARY KEY, -- only for future migrations, actually not need
    chat_id bigint NOT NULL REFERENCES chat(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL,
    admin BOOLEAN NOT NULL
)