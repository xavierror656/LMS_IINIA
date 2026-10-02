ALTER TABLE lessons DROP CONSTRAINT lessons_type_check;
ALTER TABLE lessons ADD CHECK(type IN ('reading','code','h5p','assignment','quiz'));
ALTER TABLE authored_activities DROP CONSTRAINT authored_activities_type_check;
ALTER TABLE authored_activities ADD CHECK(type IN ('reading','assignment','quiz'));
ALTER TABLE authored_activities ADD COLUMN quiz_config jsonb CHECK(quiz_config IS NULL OR jsonb_typeof(quiz_config)='object');
ALTER TABLE activity_publications ADD COLUMN quiz_config jsonb CHECK(quiz_config IS NULL OR jsonb_typeof(quiz_config)='object');
ALTER TABLE lessons ADD COLUMN quiz_grade_policy text NOT NULL DEFAULT 'last' CHECK(quiz_grade_policy IN ('first','last','highest','average'));
CREATE TABLE quiz_publication_items (
 publication_id bigint NOT NULL REFERENCES activity_publications(id),
 position integer NOT NULL CHECK(position BETWEEN 1 AND 20),
 question_id bigint NOT NULL,
 question_version integer NOT NULL,
 weight integer NOT NULL CHECK(weight BETWEEN 1 AND 1000),
 PRIMARY KEY(publication_id,position),
 UNIQUE(publication_id,question_id),
 FOREIGN KEY(question_id,question_version) REFERENCES question_versions(question_id,version)
);
CREATE TABLE quiz_attempts (
 id bigserial PRIMARY KEY,
 lesson_id bigint NOT NULL REFERENCES lessons(id),
 user_id bigint NOT NULL REFERENCES users(id),
 publication_id bigint NOT NULL,
 attempt integer NOT NULL CHECK(attempt BETWEEN 1 AND 10),
 version integer NOT NULL DEFAULT 1 CHECK(version>0),
 status text NOT NULL DEFAULT 'in_progress' CHECK(status IN ('in_progress','finished')),
 answers jsonb NOT NULL CHECK(jsonb_typeof(answers)='array' AND jsonb_array_length(answers) BETWEEN 1 AND 20),
 score integer CHECK(score BETWEEN 0 AND 100),
 started_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 finished_at timestamptz,
 UNIQUE(lesson_id,user_id,attempt),
 FOREIGN KEY(publication_id,lesson_id) REFERENCES activity_publications(id,lesson_id),
 CHECK((status='in_progress' AND score IS NULL AND finished_at IS NULL) OR (status='finished' AND score IS NOT NULL AND finished_at IS NOT NULL))
);
CREATE UNIQUE INDEX quiz_one_active_attempt ON quiz_attempts(lesson_id,user_id) WHERE status='in_progress';
CREATE INDEX quiz_teacher_results ON quiz_attempts(lesson_id,status,id);

-- One canonical entry per student/lesson; private unfinished attempts never participate.
CREATE VIEW gradebook_entries AS
 SELECT s.lesson_id,s.user_id,s.id submission_id,NULL::bigint quiz_attempt_id,g.score,g.status
 FROM submissions s LEFT JOIN submission_grades g ON g.submission_id=s.id
 WHERE s.status='submitted' AND NOT EXISTS(SELECT 1 FROM submissions newer WHERE newer.lesson_id=s.lesson_id AND newer.user_id=s.user_id AND newer.status='submitted' AND newer.attempt>s.attempt)
 UNION ALL
 SELECT a.lesson_id,a.user_id,NULL::bigint submission_id,
 CASE l.quiz_grade_policy
 WHEN 'first' THEN (array_agg(a.id ORDER BY a.attempt))[1]
 WHEN 'highest' THEN (array_agg(a.id ORDER BY a.score DESC,a.attempt))[1]
 ELSE (array_agg(a.id ORDER BY a.attempt DESC))[1] END quiz_attempt_id,
 CASE l.quiz_grade_policy
 WHEN 'first' THEN (array_agg(a.score ORDER BY a.attempt))[1]
 WHEN 'last' THEN (array_agg(a.score ORDER BY a.attempt DESC))[1]
 WHEN 'highest' THEN max(a.score)
 ELSE round(avg(a.score))::integer END score,
 'published'::text status
 FROM quiz_attempts a JOIN lessons l ON l.id=a.lesson_id WHERE a.status='finished'
 GROUP BY a.lesson_id,a.user_id,l.quiz_grade_policy;
