BEGIN;

CREATE TABLE choices
(
    id          UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    user_id     UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.user_id', true), '')::uuid
        REFERENCES public.users (id),
    question_id UUID        NOT NULL,
    position    SMALLINT    NOT NULL,
    label       TEXT        NOT NULL CHECK (char_length(label) BETWEEN 1 AND 32),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT choices_user_id_key UNIQUE (user_id, id),
    -- A reorder swaps two positions inside one statement.
    CONSTRAINT choices_position_key UNIQUE (user_id, question_id, position)
        DEFERRABLE INITIALLY DEFERRED,
    CONSTRAINT choices_question_fkey FOREIGN KEY (user_id, question_id)
        REFERENCES questions (user_id, id) ON DELETE CASCADE
);

COMMENT ON TABLE choices IS
    'Scoped to a user: the choices of the questions of one person, in every workspace.';

CREATE TRIGGER set_updated_at
    BEFORE UPDATE ON choices
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

ALTER TABLE choices
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE choices
    FORCE ROW LEVEL SECURITY;

CREATE POLICY choices_user_isolation ON choices
    USING (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid)
    WITH CHECK (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid);

COMMIT;
