-- A null is no zone, apart from a person who chose UTC. A web view falls back to the
-- zone of the browser for the first one alone.
ALTER TABLE public.users
    ADD COLUMN time_zone TEXT CHECK (time_zone <> '');
