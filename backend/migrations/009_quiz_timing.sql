ALTER TABLE lessons ADD COLUMN quiz_time_limit_seconds integer, ADD CONSTRAINT lesson_quiz_time_limit CHECK(quiz_time_limit_seconds IS NULL OR quiz_time_limit_seconds BETWEEN 60 AND 43200);
ALTER TABLE quiz_attempts ADD COLUMN time_limit_seconds integer, ADD COLUMN expires_at timestamptz, ADD COLUMN closes_at timestamptz;
ALTER TABLE quiz_attempts ADD CONSTRAINT quiz_attempt_time_limit CHECK(time_limit_seconds IS NULL OR time_limit_seconds BETWEEN 60 AND 43200);
ALTER TABLE quiz_attempts ADD CONSTRAINT quiz_attempt_expiry_requires_limit CHECK(expires_at IS NULL OR time_limit_seconds IS NOT NULL);
CREATE INDEX quiz_attempts_open_deadline ON quiz_attempts(expires_at) WHERE status='in_progress' AND expires_at IS NOT NULL;
-- Per-student time exception for a published quiz: only widens limits, never creates them.
CREATE TABLE quiz_extensions (
 lesson_id bigint NOT NULL REFERENCES lessons(id), user_id bigint NOT NULL REFERENCES users(id),
 version integer NOT NULL CHECK(version>0), due_at timestamptz, closes_at timestamptz,
 extra_seconds integer NOT NULL DEFAULT 0 CHECK(extra_seconds BETWEEN 0 AND 14400),
 reason text NOT NULL CHECK(char_length(reason) BETWEEN 1 AND 1000),
 PRIMARY KEY(lesson_id,user_id), CHECK(due_at IS NULL OR closes_at IS NULL OR due_at<=closes_at)
);
CREATE TABLE quiz_extension_revisions (
 id bigserial PRIMARY KEY, lesson_id bigint NOT NULL, user_id bigint NOT NULL,
 actor_id bigint NOT NULL REFERENCES users(id), version integer NOT NULL,
 due_at timestamptz, closes_at timestamptz, extra_seconds integer NOT NULL,
 reason text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(),
 FOREIGN KEY(lesson_id,user_id) REFERENCES quiz_extensions(lesson_id,user_id), UNIQUE(lesson_id,user_id,version)
);
