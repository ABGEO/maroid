BEGIN;

CREATE TABLE public.workspaces
(
    id         UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    name       TEXT        NOT NULL
        CONSTRAINT workspaces_name_length CHECK (char_length(name) BETWEEN 1 AND 64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER set_updated_at
    BEFORE UPDATE
    ON public.workspaces
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE public.workspace_members
(
    workspace_id UUID        NOT NULL REFERENCES public.workspaces (id),
    user_id      UUID        NOT NULL REFERENCES public.users (id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (workspace_id, user_id)
);

CREATE INDEX workspace_members_user_id_idx ON public.workspace_members (user_id);

CREATE TRIGGER set_updated_at
    BEFORE UPDATE
    ON public.workspace_members
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

DO
$$
    DECLARE
        person       RECORD;
        workspace_id UUID;
    BEGIN
        FOR person IN SELECT id, first_name FROM public.users ORDER BY id
            LOOP
                INSERT INTO public.workspaces (name)
                VALUES (COALESCE(left(NULLIF(trim(person.first_name), ''), 64), 'Workspace'))
                RETURNING id INTO workspace_id;

                INSERT INTO public.workspace_members (workspace_id, user_id)
                VALUES (workspace_id, person.id);
            END LOOP;
    END
$$;

COMMIT;
