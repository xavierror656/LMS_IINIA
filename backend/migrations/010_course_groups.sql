CREATE TABLE course_groups (
 id bigserial PRIMARY KEY,
 course_id bigint NOT NULL REFERENCES courses(id),
 name text NOT NULL CHECK(char_length(name) BETWEEN 1 AND 100),
 version integer NOT NULL DEFAULT 1 CHECK(version>0),
 created_by bigint NOT NULL REFERENCES users(id),
 created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(course_id,name), UNIQUE(course_id,id)
);
-- One group per student and course; the denormalized course keeps the rule in the
-- database instead of trusting the caller.
CREATE TABLE group_members (
 group_id bigint NOT NULL,
 course_id bigint NOT NULL,
 user_id bigint NOT NULL REFERENCES users(id),
 added_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(group_id,user_id),
 UNIQUE(course_id,user_id),
 FOREIGN KEY(group_id,course_id) REFERENCES course_groups(id,course_id)
);
CREATE INDEX group_members_roster ON group_members(course_id,user_id);
