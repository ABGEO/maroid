BEGIN;

CREATE TABLE checkin_questions
(
    id          UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    user_id     UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.user_id', true), '')::uuid
        REFERENCES public.users (id),
    checkin_id  UUID        NOT NULL,
    question_id UUID        NOT NULL,
    position    SMALLINT    NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT checkin_questions_position_key UNIQUE (user_id, checkin_id, position),
    CONSTRAINT checkin_questions_question_key UNIQUE (user_id, checkin_id, question_id),
    CONSTRAINT checkin_questions_checkin_fkey FOREIGN KEY (user_id, checkin_id)
        REFERENCES checkins (user_id, id) ON DELETE CASCADE,
    CONSTRAINT checkin_questions_question_fkey FOREIGN KEY (user_id, question_id)
        REFERENCES questions (user_id, id) ON DELETE CASCADE
);

COMMENT ON TABLE checkin_questions IS
    'Scoped to a user: the order of the questions of the check-ins of one person, in every workspace.';

CREATE TRIGGER set_updated_at
    BEFORE UPDATE ON checkin_questions
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

ALTER TABLE checkin_questions
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE checkin_questions
    FORCE ROW LEVEL SECURITY;

CREATE POLICY checkin_questions_user_isolation ON checkin_questions
    USING (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid)
    WITH CHECK (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid);

COMMIT;
