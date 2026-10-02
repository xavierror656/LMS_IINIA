CREATE TABLE questions (
 id bigserial PRIMARY KEY,
 course_id bigint NOT NULL REFERENCES courses(id),
 current_version integer NOT NULL DEFAULT 1 CHECK(current_version>0),
 UNIQUE(id,course_id)
);
CREATE INDEX questions_course ON questions(course_id,id);
CREATE TABLE question_versions (
 question_id bigint NOT NULL REFERENCES questions(id),
 version integer NOT NULL CHECK(version>0),
 content jsonb NOT NULL CHECK(jsonb_typeof(content)='object'),
 archived boolean NOT NULL DEFAULT false,
 created_by bigint NOT NULL REFERENCES users(id),
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(question_id,version),
 CHECK(content ?& ARRAY['name','type','prompt','options','correctChoices','acceptedAnswers','caseSensitive','explanation']),
 CHECK(char_length(content->>'name') BETWEEN 1 AND 160),
 CHECK(char_length(content->>'prompt') BETWEEN 1 AND 4000),
 CHECK(content->>'type' IN ('single_choice','multiple_choice','true_false','short_answer')),
 CHECK(jsonb_typeof(content->'options')='array' AND jsonb_array_length(content->'options')<=8),
 CHECK(jsonb_typeof(content->'correctChoices')='array' AND jsonb_array_length(content->'correctChoices')<=8),
 CHECK(jsonb_typeof(content->'acceptedAnswers')='array' AND jsonb_array_length(content->'acceptedAnswers')<=10)
);
ALTER TABLE questions ADD FOREIGN KEY(id,current_version) REFERENCES question_versions(question_id,version) DEFERRABLE INITIALLY DEFERRED;
