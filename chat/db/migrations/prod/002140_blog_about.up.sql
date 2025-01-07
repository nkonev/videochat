ALTER TABLE chat ADD COLUMN blog_about BOOLEAN NOT NULL DEFAULT FALSE;

DROP FUNCTION IF EXISTS CREATE_CHAT(IN chat_name TEXT, IN tet_a_tet BOOLEAN, IN can_resend BOOLEAN, IN available_to_search BOOLEAN, IN blog BOOLEAN, IN regular_participant_can_publish_message BOOLEAN, IN regular_participant_can_pin_message BOOLEAN);
