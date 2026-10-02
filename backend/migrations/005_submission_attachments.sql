ALTER TABLE submissions DROP CONSTRAINT submissions_body_check;
ALTER TABLE submissions ADD CONSTRAINT submissions_body_check CHECK(char_length(body)<=12000);
CREATE TABLE attachment_accounts (
 user_id bigint PRIMARY KEY REFERENCES users(id),
 bytes_used bigint NOT NULL DEFAULT 0 CHECK(bytes_used BETWEEN 0 AND 52428800)
);
CREATE TABLE submission_attachments (
 id text PRIMARY KEY CHECK(id ~ '^[a-f0-9]{64}$'),
 submission_id bigint NOT NULL REFERENCES submissions(id),
 slot smallint NOT NULL CHECK(slot BETWEEN 1 AND 5),
 name text NOT NULL CHECK(char_length(name) BETWEEN 1 AND 120),
 content_type text NOT NULL CHECK(content_type IN ('text/plain; charset=utf-8','image/png','image/jpeg')),
 size integer NOT NULL CHECK(size BETWEEN 1 AND 2097152),
 content bytea NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 CHECK(octet_length(content)=size), UNIQUE(submission_id,slot)
);
