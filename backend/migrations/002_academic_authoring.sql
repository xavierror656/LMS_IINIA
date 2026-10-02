CREATE TABLE course_staff (
  course_id bigint NOT NULL REFERENCES courses(id),
  user_id bigint NOT NULL REFERENCES users(id),
  PRIMARY KEY(course_id,user_id)
);
CREATE INDEX course_staff_user ON course_staff(user_id,course_id);
ALTER TABLE lessons DROP CONSTRAINT lessons_type_check;
ALTER TABLE lessons ADD CONSTRAINT lessons_type_check CHECK(type IN ('reading','code','h5p','assignment'));
CREATE TABLE authored_activities (
  id bigserial PRIMARY KEY,
  module_id bigint NOT NULL REFERENCES modules(id),
  lesson_id bigint UNIQUE REFERENCES lessons(id),
  title text NOT NULL CHECK(char_length(title) BETWEEN 1 AND 160),
  description text NOT NULL CHECK(char_length(description)<=1000),
  type text NOT NULL CHECK(type IN ('reading','assignment')),
  body text NOT NULL CHECK(char_length(body) BETWEEN 1 AND 12000),
  version integer NOT NULL DEFAULT 1 CHECK(version>0),
  published_version integer NOT NULL DEFAULT 0 CHECK(published_version>=0 AND published_version<=version),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX authored_module ON authored_activities(module_id,id);
CREATE TABLE activity_publications (
  id bigserial PRIMARY KEY,
  activity_id bigint NOT NULL REFERENCES authored_activities(id),
  lesson_id bigint NOT NULL REFERENCES lessons(id),
  version integer NOT NULL CHECK(version>0),
  title text NOT NULL,
  body text NOT NULL,
  published_by bigint NOT NULL REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(activity_id,version),
  UNIQUE(id,lesson_id)
);
CREATE TABLE submissions (
  id bigserial PRIMARY KEY,
  lesson_id bigint NOT NULL REFERENCES lessons(id),
  user_id bigint NOT NULL REFERENCES users(id),
  publication_id bigint NOT NULL,
  body text NOT NULL CHECK(char_length(body) BETWEEN 1 AND 12000),
  version integer NOT NULL DEFAULT 1 CHECK(version>0),
  status text NOT NULL DEFAULT 'draft' CHECK(status IN ('draft','submitted')),
  submitted_at timestamptz,
  UNIQUE(lesson_id,user_id),
  FOREIGN KEY(publication_id,lesson_id) REFERENCES activity_publications(id,lesson_id),
  CHECK((status='draft' AND submitted_at IS NULL) OR (status='submitted' AND submitted_at IS NOT NULL))
);
CREATE INDEX submissions_queue ON submissions(lesson_id,status,id);
CREATE TABLE submission_grades (
  submission_id bigint PRIMARY KEY REFERENCES submissions(id),
  score integer NOT NULL CHECK(score BETWEEN 0 AND 100),
  feedback text NOT NULL CHECK(char_length(feedback)<=4000),
  version integer NOT NULL DEFAULT 1 CHECK(version>0),
  status text NOT NULL DEFAULT 'draft' CHECK(status IN ('draft','published'))
);
CREATE TABLE grade_revisions (
  id bigserial PRIMARY KEY,
  submission_id bigint NOT NULL REFERENCES submissions(id),
  actor_id bigint NOT NULL REFERENCES users(id),
  score integer NOT NULL CHECK(score BETWEEN 0 AND 100),
  feedback text NOT NULL,
  version integer NOT NULL,
  status text NOT NULL CHECK(status IN ('draft','published')),
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(submission_id,version,status)
);
