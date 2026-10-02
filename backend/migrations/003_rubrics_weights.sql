ALTER TABLE authored_activities ADD COLUMN rubric jsonb;
ALTER TABLE authored_activities ADD COLUMN weight integer NOT NULL DEFAULT 1 CHECK(weight BETWEEN 1 AND 1000);
ALTER TABLE authored_activities ADD CONSTRAINT activity_rubric_object CHECK(rubric IS NULL OR jsonb_typeof(rubric)='object');
ALTER TABLE activity_publications ADD COLUMN rubric jsonb;
ALTER TABLE lessons ADD COLUMN grade_weight integer NOT NULL DEFAULT 1 CHECK(grade_weight BETWEEN 1 AND 1000);
ALTER TABLE submission_grades ADD COLUMN assessment jsonb;
ALTER TABLE grade_revisions ADD COLUMN assessment jsonb;
