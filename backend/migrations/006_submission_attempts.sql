ALTER TABLE authored_activities ADD COLUMN max_attempts integer NOT NULL DEFAULT 1 CHECK(max_attempts BETWEEN 1 AND 10);
ALTER TABLE lessons ADD COLUMN max_attempts integer NOT NULL DEFAULT 1 CHECK(max_attempts BETWEEN 1 AND 10);
ALTER TABLE submissions DROP CONSTRAINT submissions_lesson_id_user_id_key;
ALTER TABLE submissions ADD COLUMN attempt integer NOT NULL DEFAULT 1 CHECK(attempt BETWEEN 1 AND 10);
ALTER TABLE submissions ADD COLUMN previous_submission_id bigint UNIQUE;
ALTER TABLE submissions ADD COLUMN reopened_by bigint REFERENCES users(id);
ALTER TABLE submissions ADD COLUMN reopen_reason text;
ALTER TABLE submissions ADD COLUMN reopened_at timestamptz;
ALTER TABLE submissions ADD UNIQUE(lesson_id,user_id,attempt);
ALTER TABLE submissions ADD UNIQUE(id,lesson_id,user_id);
ALTER TABLE submissions ADD FOREIGN KEY(previous_submission_id,lesson_id,user_id) REFERENCES submissions(id,lesson_id,user_id);
ALTER TABLE submissions ADD CHECK(
 (attempt=1 AND previous_submission_id IS NULL AND reopened_by IS NULL AND reopen_reason IS NULL AND reopened_at IS NULL)
 OR (attempt>1 AND previous_submission_id IS NOT NULL AND reopened_by IS NOT NULL AND reopen_reason IS NOT NULL AND char_length(reopen_reason) BETWEEN 1 AND 1000 AND reopened_at IS NOT NULL)
);
CREATE UNIQUE INDEX submissions_one_draft ON submissions(lesson_id,user_id) WHERE status='draft';
