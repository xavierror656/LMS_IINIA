ALTER TABLE authored_activities ADD COLUMN opens_at timestamptz, ADD COLUMN due_at timestamptz, ADD COLUMN closes_at timestamptz;
ALTER TABLE authored_activities ADD CONSTRAINT authored_schedule_order CHECK((opens_at IS NULL OR due_at IS NULL OR opens_at<=due_at) AND (due_at IS NULL OR closes_at IS NULL OR due_at<=closes_at) AND (opens_at IS NULL OR closes_at IS NULL OR opens_at<=closes_at));
ALTER TABLE lessons ADD COLUMN opens_at timestamptz, ADD COLUMN due_at timestamptz, ADD COLUMN closes_at timestamptz;
ALTER TABLE lessons ADD CONSTRAINT lesson_schedule_order CHECK((opens_at IS NULL OR due_at IS NULL OR opens_at<=due_at) AND (due_at IS NULL OR closes_at IS NULL OR due_at<=closes_at) AND (opens_at IS NULL OR closes_at IS NULL OR opens_at<=closes_at));
CREATE TABLE assignment_extensions (
 lesson_id bigint NOT NULL REFERENCES lessons(id), user_id bigint NOT NULL REFERENCES users(id),
 version integer NOT NULL CHECK(version>0), due_at timestamptz, closes_at timestamptz,
 reason text NOT NULL CHECK(char_length(reason) BETWEEN 1 AND 1000),
 PRIMARY KEY(lesson_id,user_id), CHECK(due_at IS NULL OR closes_at IS NULL OR due_at<=closes_at)
);
CREATE TABLE extension_revisions (
 id bigserial PRIMARY KEY, lesson_id bigint NOT NULL, user_id bigint NOT NULL,
 actor_id bigint NOT NULL REFERENCES users(id), version integer NOT NULL, due_at timestamptz, closes_at timestamptz,
 reason text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(),
 FOREIGN KEY(lesson_id,user_id) REFERENCES assignment_extensions(lesson_id,user_id), UNIQUE(lesson_id,user_id,version)
);
ALTER TABLE submissions ADD COLUMN effective_due_at timestamptz, ADD COLUMN late boolean NOT NULL DEFAULT false;
