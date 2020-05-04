CREATE SCHEMA IF NOT EXISTS locks;

CREATE TABLE locks.task_lock(
  name varchar(64) PRIMARY KEY NOT NULL,
  lock_until timestamp NULL,
  locked_at timestamp NULL,
  locked_by varchar(255) NOT NULL
);