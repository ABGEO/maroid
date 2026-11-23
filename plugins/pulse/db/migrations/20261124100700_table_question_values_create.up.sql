BEGIN;

CREATE TABLE question_values
(
    id            UUID             NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    user_id       UUID             NOT NULL
        DEFAULT NULLIF(current_setting('app.user_id', true), '')::uuid
        REFERENCES public.users (id),
    question_id   UUID             NOT NULL,
    type          TEXT             NOT NULL,
    occurred_at   TIMESTAMPTZ      NOT NULL,
    answered_at   TIMESTAMPTZ      NOT NULL,
    source        TEXT             NOT NULL CHECK (source ~ '^[a-z][a-z0-9_]{0,31}$'),
    prompt_id     UUID,
    number_value  NUMERIC,
    boolean_value BOOLEAN,
    choice_id     UUID,
    text_value    TEXT CHECK (char_length(text_value) BETWEEN 1 AND 1000),
    latitude      DOUBLE PRECISION CHECK (latitude BETWEEN -90 AND 90),
    longitude     DOUBLE PRECISION CHECK (longitude BETWEEN -180 AND 180),
    created_at    TIMESTAMPTZ      NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ      NOT NULL DEFAULT now(),
    -- A question with a value is never deleted. Archive it instead.
    CONSTRAINT question_values_question_fkey FOREIGN KEY (user_id, question_id)
        REFERENCES questions (user_id, id) ON DELETE RESTRICT,
    -- A choice that a value records is never removed, so the series keeps its labels.
    CONSTRAINT question_values_choice_fkey FOREIGN KEY (user_id, choice_id)
        REFERENCES choices (user_id, id) ON DELETE RESTRICT,
    -- PULSE-INV-002: One question of one round holds at most one value.
    CONSTRAINT question_values_prompt_key UNIQUE (user_id, prompt_id),
    CONSTRAINT question_values_prompt_fkey FOREIGN KEY (user_id, prompt_id)
        REFERENCES prompts (user_id, id) ON DELETE SET NULL (prompt_id),
    -- The type chooses the one column that holds the value.
    CONSTRAINT question_values_shape_check CHECK (
        (type IN ('scale', 'number') AND number_value IS NOT NULL
            AND num_nulls(boolean_value, choice_id, text_value, latitude, longitude) = 5)
        OR (type = 'boolean' AND boolean_value IS NOT NULL
            AND num_nulls(number_value, choice_id, text_value, latitude, longitude) = 5)
        OR (type = 'choice' AND choice_id IS NOT NULL
            AND num_nulls(number_value, boolean_value, text_value, latitude, longitude) = 5)
        OR (type = 'text' AND text_value IS NOT NULL
            AND num_nulls(number_value, boolean_value, choice_id, latitude, longitude) = 5)
        OR (type = 'location' AND num_nonnulls(latitude, longitude) = 2
            AND num_nulls(number_value, boolean_value, choice_id, text_value) = 4))
);

CREATE INDEX question_values_series_idx
    ON question_values (user_id, question_id, occurred_at DESC, id DESC);
CREATE INDEX question_values_choice_idx ON question_values (user_id, choice_id)
    WHERE choice_id IS NOT NULL;

CREATE FUNCTION pulse_refuse_change() RETURNS TRIGGER AS $$
BEGIN
    IF NEW.question_id <> OLD.question_id OR NEW.type <> OLD.type THEN
        RAISE EXCEPTION 'a value keeps its question and its type'
            USING ERRCODE = 'check_violation';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- PULSE-INV-004: A value keeps its question and its type.
CREATE TRIGGER question_values_frozen
    BEFORE UPDATE OF question_id, type ON question_values
    FOR EACH ROW
EXECUTE FUNCTION pulse_refuse_change();

COMMENT ON TABLE question_values IS
    'Scoped to a user: the answers of one person, in every workspace.';

CREATE TRIGGER set_updated_at
    BEFORE UPDATE ON question_values
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

ALTER TABLE question_values
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE question_values
    FORCE ROW LEVEL SECURITY;

CREATE POLICY question_values_user_isolation ON question_values
    USING (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid)
    WITH CHECK (user_id = NULLIF(current_setting('app.user_id', true), '')::uuid);

COMMIT;
