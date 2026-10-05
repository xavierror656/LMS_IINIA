ALTER TABLE authored_activities ADD COLUMN group_submission boolean NOT NULL DEFAULT false;
ALTER TABLE lessons ADD COLUMN group_submission boolean NOT NULL DEFAULT false;
ALTER TABLE activity_publications ADD COLUMN group_submission boolean NOT NULL DEFAULT false;
-- Existing submissions stay individual: group_id NULL.
ALTER TABLE submissions ADD COLUMN group_id bigint REFERENCES course_groups(id);
CREATE UNIQUE INDEX submissions_group_attempt ON submissions(lesson_id,group_id,attempt) WHERE group_id IS NOT NULL;
CREATE UNIQUE INDEX submissions_one_group_draft ON submissions(lesson_id,group_id) WHERE group_id IS NOT NULL AND status='draft';
-- Shared attachments charge the uploader, not the submission author.
ALTER TABLE submission_attachments ADD COLUMN uploaded_by bigint REFERENCES users(id);
UPDATE submission_attachments SET uploaded_by=s.user_id FROM submissions s WHERE s.id=submission_attachments.submission_id;
ALTER TABLE submission_attachments ALTER COLUMN uploaded_by SET NOT NULL;
CREATE INDEX submission_attachments_uploader ON submission_attachments(uploaded_by);
-- Every member of a group sees the group's latest submitted attempt; individual
-- submissions keep their previous rule.
CREATE OR REPLACE VIEW gradebook_entries AS
 SELECT s.lesson_id,s.user_id,s.id submission_id,NULL::bigint quiz_attempt_id,g.score,g.status
 FROM submissions s LEFT JOIN submission_grades g ON g.submission_id=s.id
 WHERE s.status='submitted' AND s.group_id IS NULL AND NOT EXISTS(SELECT 1 FROM submissions newer WHERE newer.lesson_id=s.lesson_id AND newer.user_id=s.user_id AND newer.status='submitted' AND newer.attempt>s.attempt)
 UNION ALL
 SELECT s.lesson_id,gm.user_id,s.id,NULL::bigint,g.score,g.status
 FROM submissions s JOIN group_members gm ON gm.group_id=s.group_id LEFT JOIN submission_grades g ON g.submission_id=s.id
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
