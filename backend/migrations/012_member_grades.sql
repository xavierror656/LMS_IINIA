-- A grade stops being one per delivery and becomes one per student: a group
-- delivery carries one grade for each member.
ALTER TABLE submission_grades ADD COLUMN student_id bigint REFERENCES users(id);
UPDATE submission_grades g SET student_id=s.user_id FROM submissions s WHERE s.id=g.submission_id;
ALTER TABLE submission_grades ALTER COLUMN student_id SET NOT NULL;
ALTER TABLE submission_grades DROP CONSTRAINT submission_grades_pkey;
ALTER TABLE submission_grades ADD PRIMARY KEY (submission_id,student_id);
ALTER TABLE grade_revisions ADD COLUMN student_id bigint REFERENCES users(id);
UPDATE grade_revisions r SET student_id=s.user_id FROM submissions s WHERE s.id=r.submission_id;
ALTER TABLE grade_revisions ALTER COLUMN student_id SET NOT NULL;
-- The audit uniqueness was per delivery; it becomes per member so two members of
-- the same team can each have their own revision history.
ALTER TABLE grade_revisions DROP CONSTRAINT grade_revisions_submission_id_version_status_key;
ALTER TABLE grade_revisions ADD CONSTRAINT grade_revisions_member_revision_key UNIQUE(submission_id,student_id,version,status);
-- Every member of a group sees the group's latest submitted attempt with their own
-- grade; an individual delivery keeps its previous rule.
CREATE OR REPLACE VIEW gradebook_entries AS
 SELECT s.lesson_id,s.user_id,s.id submission_id,NULL::bigint quiz_attempt_id,g.score,g.status
 FROM submissions s LEFT JOIN submission_grades g ON g.submission_id=s.id AND g.student_id=s.user_id
 WHERE s.status='submitted' AND s.group_id IS NULL AND NOT EXISTS(SELECT 1 FROM submissions newer WHERE newer.lesson_id=s.lesson_id AND newer.user_id=s.user_id AND newer.status='submitted' AND newer.attempt>s.attempt)
 UNION ALL
 SELECT s.lesson_id,gm.user_id,s.id,NULL::bigint,g.score,g.status
 FROM submissions s JOIN group_members gm ON gm.group_id=s.group_id LEFT JOIN submission_grades g ON g.submission_id=s.id AND g.student_id=gm.user_id
 WHERE s.status='submitted' AND NOT EXISTS(SELECT 1 FROM submissions newer WHERE newer.lesson_id=s.lesson_id AND newer.group_id=s.group_id AND newer.status='submitted' AND newer.attempt>s.attempt)
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
