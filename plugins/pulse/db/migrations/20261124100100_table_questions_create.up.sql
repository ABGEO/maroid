BEGIN;

CREATE TABLE questions
(
    id          UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    user_id     UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.user_id', true), '')::uuid
        REFERENCES public.users (id),
    type        TEXT        NOT NULL
        CHECK (type IN ('scale', 'boolean', 'number', 'choice', 'text', 'location')),
    prompt      TEXT        NOT NULL CHECK (char_length(prompt) BETWEEN 1 AND 200),
    -- The type of the question decodes and validates it.
    config      JSONB       NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(config) = 'object'),
    archived_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT questions_user_id_key UNIQUE (user_id, id)
);

COMMENT ON TABLE questions IS
    'Scoped to a user: the questions that one person asks themselves, in every workspace.';

CREATE TRIGGER set_updated_at
    BEFORE UPDATE ON questions
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

ALTER TABLE questions
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE questions
    FORCE ROW LEVEL SECURITY;

CREATE POLICY questions_user_isolation ON questions
    USING (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid)
    WITH CHECK (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid);

COMMIT;
