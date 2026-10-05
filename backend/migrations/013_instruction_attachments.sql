-- PDF joins the allowed formats after real structural analysis.
ALTER TABLE submission_attachments DROP CONSTRAINT submission_attachments_content_type_check;
ALTER TABLE submission_attachments ADD CONSTRAINT submission_attachments_content_type_check CHECK(content_type IN ('text/plain; charset=utf-8','image/png','image/jpeg','application/pdf'));
-- Instruction files: the teacher uploads them on the draft activity and publishing
-- freezes a copy in the publication, exactly like the instructions text.
CREATE TABLE activity_attachments (
 id text PRIMARY KEY CHECK(id ~ '^[a-f0-9]{64}$'),
 activity_id bigint NOT NULL REFERENCES authored_activities(id),
 slot smallint NOT NULL CHECK(slot BETWEEN 1 AND 5),
 name text NOT NULL CHECK(char_length(name) BETWEEN 1 AND 120),
 content_type text NOT NULL CHECK(content_type IN ('text/plain; charset=utf-8','image/png','image/jpeg','application/pdf')),
 size integer NOT NULL CHECK(size BETWEEN 1 AND 2097152),
 content bytea NOT NULL,
 uploaded_by bigint NOT NULL REFERENCES users(id),
 created_at timestamptz NOT NULL DEFAULT now(),
 CHECK(octet_length(content)=size), UNIQUE(activity_id,slot)
);
CREATE INDEX activity_attachments_uploader ON activity_attachments(uploaded_by);
CREATE TABLE publication_attachments (
 id text PRIMARY KEY CHECK(id ~ '^[a-f0-9]{64}$'),
 publication_id bigint NOT NULL REFERENCES activity_publications(id),
 slot smallint NOT NULL CHECK(slot BETWEEN 1 AND 5),
 name text NOT NULL CHECK(char_length(name) BETWEEN 1 AND 120),
 content_type text NOT NULL CHECK(content_type IN ('text/plain; charset=utf-8','image/png','image/jpeg','application/pdf')),
 size integer NOT NULL CHECK(size BETWEEN 1 AND 2097152),
 content bytea NOT NULL,
 uploaded_by bigint NOT NULL REFERENCES users(id),
 created_at timestamptz NOT NULL DEFAULT now(),
 CHECK(octet_length(content)=size), UNIQUE(publication_id,slot)
);
CREATE INDEX publication_attachments_uploader ON publication_attachments(uploaded_by);
-- Retention needs to know when a draft was last touched: submissions carried no
-- timestamp of their own, only the submission date.
ALTER TABLE submissions ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now();
CREATE INDEX submissions_drafts ON submissions(updated_at) WHERE status='draft';
