BEGIN;

CREATE TABLE public.auth_flows
(
    id            UUID        NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    state         TEXT        NOT NULL UNIQUE,
    intent        TEXT        NOT NULL
        CONSTRAINT auth_flows_intent_check CHECK (intent IN ('sign_in', 'attach', 'redeem')),
    user_id       UUID REFERENCES public.users (id) ON DELETE CASCADE,
    invitation_id UUID REFERENCES public.invitations (id) ON DELETE CASCADE,
    binding_hash  BYTEA       NOT NULL,
    nonce         TEXT        NOT NULL,
    verifier      TEXT        NOT NULL,
    redirect      TEXT        NOT NULL,
    provider      TEXT,
    expires_at    TIMESTAMPTZ NOT NULL,
    consumed_at   TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT auth_flows_intent_target_check CHECK (
        (intent = 'sign_in' AND user_id IS NULL AND invitation_id IS NULL) OR
        (intent = 'attach' AND user_id IS NOT NULL AND invitation_id IS NULL) OR
        (intent = 'redeem' AND user_id IS NULL AND invitation_id IS NOT NULL)
        )
);

COMMIT;
