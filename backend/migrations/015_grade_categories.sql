-- Grade categories group graded activities under weighted buckets. Flat: one
-- level per course. Weight is relative inside the course total; version guards
-- edits. Names are unique per course ignoring case and surrounding blanks.
CREATE TABLE grade_categories (
  id bigserial PRIMARY KEY,
  course_id bigint NOT NULL REFERENCES courses(id),
  name text NOT NULL CHECK(char_length(btrim(name)) BETWEEN 1 AND 160),
  weight integer NOT NULL DEFAULT 1 CHECK(weight BETWEEN 1 AND 1000),
  position integer NOT NULL CHECK(position > 0),
  version integer NOT NULL DEFAULT 1 CHECK(version > 0),
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX grade_categories_course_name ON grade_categories(course_id, lower(btrim(name)));
CREATE INDEX grade_categories_course_position ON grade_categories(course_id, position, id);

ALTER TABLE courses ADD COLUMN missing_policy text NOT NULL DEFAULT 'exclude' CHECK(missing_policy IN ('exclude','zero'));

ALTER TABLE authored_activities ADD COLUMN grade_category_id bigint REFERENCES grade_categories(id);
ALTER TABLE activity_publications ADD COLUMN grade_category_id bigint REFERENCES grade_categories(id);
ALTER TABLE lessons ADD COLUMN grade_category_id bigint REFERENCES grade_categories(id);
CREATE INDEX lessons_grade_category ON lessons(grade_category_id);

-- Backfill: every graded lesson gets its course's General category; drafts and
-- publications follow the lesson so existing numbers keep the same weight.
INSERT INTO grade_categories(course_id,name,weight,position)
SELECT DISTINCT m.course_id,'General',1,1 FROM lessons l JOIN modules m ON m.id=l.module_id
WHERE l.type IN ('assignment','quiz');
UPDATE lessons l SET grade_category_id=c.id FROM grade_categories c JOIN modules m ON m.course_id=c.course_id
WHERE m.id=l.module_id AND c.name='General' AND l.type IN ('assignment','quiz');
UPDATE authored_activities a SET grade_category_id=l.grade_category_id FROM lessons l
WHERE l.id=a.lesson_id AND l.grade_category_id IS NOT NULL;
UPDATE activity_publications p SET grade_category_id=l.grade_category_id FROM lessons l
WHERE l.id=p.lesson_id AND l.grade_category_id IS NOT NULL;

-- After the backfill a graded lesson can no longer exist without a category.
ALTER TABLE lessons ADD CONSTRAINT lessons_graded_category CHECK(type NOT IN ('assignment','quiz') OR grade_category_id IS NOT NULL);
