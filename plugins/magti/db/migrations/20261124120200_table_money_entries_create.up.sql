BEGIN;

CREATE TABLE money_entries
(
    id                UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    workspace_id      UUID        NOT NULL
        DEFAULT NULLIF(current_setting('app.workspace_id', true), '')::uuid
        REFERENCES public.workspaces (id) ON DELETE CASCADE,
    subscriber_number TEXT        NOT NULL CHECK (char_length(subscriber_number) BETWEEN 1 AND 64),
    occurred_at       TIMESTAMPTZ NOT NULL,
    direction         TEXT        NOT NULL CHECK (direction IN ('in', 'out')),
    kind              TEXT        NOT NULL,
    amount            NUMERIC     NOT NULL CHECK (amount >= 0),
    collected_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- The portal answers no identifier of a money entry. Two entries of one second
    -- differ by the kind, the direction, or the amount.
    CONSTRAINT money_entries_identity_key
        UNIQUE (workspace_id, subscriber_number, occurred_at, kind, direction, amount)
);

CREATE INDEX money_entries_list_idx
    ON money_entries (workspace_id, subscriber_number, occurred_at DESC, id DESC);

COMMENT ON TABLE money_entries IS
    'Scoped to a workspace: the top ups and the charges of a mobile subscriber. An entry outlives its assignment.';

ALTER TABLE money_entries
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE money_entries
    FORCE ROW LEVEL SECURITY;

-- MAGTIACC-INV-002: A money entry of one workspace serves no other workspace.
CREATE POLICY money_entries_workspace_isolation ON money_entries
    USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);

COMMIT;
