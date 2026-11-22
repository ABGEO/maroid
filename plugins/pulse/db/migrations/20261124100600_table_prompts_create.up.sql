BEGIN;

CREATE TABLE prompts
(
    id          UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    user_id     UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.user_id', true), '')::uuid
        REFERENCES public.users (id),
    round_id    UUID        NOT NULL,
    question_id UUID        NOT NULL,
    position    SMALLINT    NOT NULL,
    state       TEXT        NOT NULL DEFAULT 'pending'
        CHECK (state IN ('pending', 'waiting', 'answered', 'skipped', 'expired')),
    message_id  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT prompts_user_id_key UNIQUE (user_id, id),
    CONSTRAINT prompts_position_key UNIQUE (user_id, round_id, position),
    CONSTRAINT prompts_round_fkey FOREIGN KEY (user_id, round_id)
        REFERENCES rounds (user_id, id) ON DELETE CASCADE,
    CONSTRAINT prompts_question_fkey FOREIGN KEY (user_id, question_id)
        REFERENCES questions (user_id, id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX prompts_message_idx ON prompts (user_id, message_id);

COMMENT ON TABLE prompts IS
    'Scoped to a user: the questions of the rounds of one person, in every workspace.';

CREATE TRIGGER set_updated_at
    BEFORE UPDATE ON prompts
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

ALTER TABLE prompts
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE prompts
    FORCE ROW LEVEL SECURITY;

CREATE POLICY prompts_user_isolation ON prompts
    USING (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid)
    WITH CHECK (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid);

COMMIT;
