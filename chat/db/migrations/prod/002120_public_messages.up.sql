alter table message add column published boolean not null default false;
alter table chat add column regular_participant_can_publish_message boolean not null default false;
