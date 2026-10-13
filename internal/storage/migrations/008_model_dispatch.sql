-- Control-side network dispatch journal; execution has no SQL access here.
-- Uncertain attempts are never automatically sent again. Cached replies are
-- encrypted with the installation master key and a step-specific AAD.
CREATE TABLE portway.model_dispatches (
 installation_id uuid NOT NULL REFERENCES portway.installations(id),
 step_id uuid NOT NULL,
 run_id uuid NOT NULL,
 request_hash text NOT NULL CHECK(length(request_hash)=64),
 state text NOT NULL CHECK(state IN ('started','finished')),
 ciphertext bytea,
 nonce bytea,
 expires_at timestamptz NOT NULL DEFAULT now()+interval '7 days',
 PRIMARY KEY(installation_id,step_id),
 CHECK ((state='started' AND ciphertext IS NULL AND nonce IS NULL) OR
        (state='finished' AND ciphertext IS NOT NULL AND octet_length(ciphertext)<=131072 AND octet_length(nonce)=12))
);
CREATE INDEX model_dispatch_expiry ON portway.model_dispatches(installation_id,expires_at);
REVOKE ALL ON portway.model_dispatches FROM PUBLIC;
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='portway_app') THEN
  GRANT SELECT,INSERT,UPDATE,DELETE ON portway.model_dispatches TO portway_app;
 END IF;
END $$;
