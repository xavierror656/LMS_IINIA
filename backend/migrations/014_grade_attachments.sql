-- Feedback files belong to one grade, which itself belongs to one student of a
-- delivery. The composite foreign key makes an orphan file impossible.
CREATE TABLE grade_attachments (
 id text PRIMARY KEY CHECK(id ~ '^[a-f0-9]{64}$'),
 submission_id bigint NOT NULL,
 student_id bigint NOT NULL,
 slot smallint NOT NULL CHECK(slot BETWEEN 1 AND 5),
 name text NOT NULL CHECK(char_length(name) BETWEEN 1 AND 120),
 content_type text NOT NULL CHECK(content_type IN ('text/plain; charset=utf-8','image/png','image/jpeg','application/pdf')),
 size integer NOT NULL CHECK(size BETWEEN 1 AND 2097152),
 content bytea NOT NULL,
 uploaded_by bigint NOT NULL REFERENCES users(id),
 created_at timestamptz NOT NULL DEFAULT now(),
 CHECK(octet_length(content)=size),
 UNIQUE(submission_id,student_id,slot),
 FOREIGN KEY(submission_id,student_id) REFERENCES submission_grades(submission_id,student_id)
);
CREATE INDEX grade_attachments_uploader ON grade_attachments(uploaded_by);
