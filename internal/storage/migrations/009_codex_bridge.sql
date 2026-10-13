-- Separate, opt-in laptop demo profile. Never grants execution access to control SQL.
CREATE TABLE portway.codex_bridge (
 installation_id uuid PRIMARY KEY REFERENCES portway.installations(id),
 instance_id uuid NOT NULL,
 workspace text NOT NULL CHECK(workspace='portway'),
 last_seen_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE portway.codex_runs (
 id uuid PRIMARY KEY,
 installation_id uuid NOT NULL REFERENCES portway.installations(id),
 owner_id uuid NOT NULL REFERENCES portway.owners(id),
 device_id uuid NOT NULL REFERENCES portway.device_sessions(id),
 state text NOT NULL CHECK(state IN ('queued','running','completed','failed','stopped')),
 prompt_ciphertext bytea NOT NULL CHECK(octet_length(prompt_ciphertext)<=16400),
 prompt_nonce bytea NOT NULL CHECK(octet_length(prompt_nonce)=12),
 result_ciphertext bytea,
 result_nonce bytea,
 error_code text,
 lease_hash text,
 lease_until timestamptz,
 instance_id uuid,
 created_at timestamptz NOT NULL DEFAULT now(),
 started_at timestamptz,
 finished_at timestamptz,
 expires_at timestamptz NOT NULL DEFAULT now()+interval '7 days',
 CHECK ((result_ciphertext IS NULL AND result_nonce IS NULL) OR
        (octet_length(result_ciphertext)<=65552 AND octet_length(result_nonce)=12))
);
CREATE UNIQUE INDEX codex_one_active ON portway.codex_runs(installation_id) WHERE state IN ('queued','running');
CREATE INDEX codex_history ON portway.codex_runs(installation_id,created_at DESC);
CREATE TABLE portway.codex_events (
 run_id uuid NOT NULL REFERENCES portway.codex_runs(id) ON DELETE CASCADE,
 sequence integer NOT NULL CHECK(sequence BETWEEN 1 AND 256),
 request_hash text NOT NULL,
 ciphertext bytea NOT NULL CHECK(octet_length(ciphertext)<=70000),
 nonce bytea NOT NULL CHECK(octet_length(nonce)=12),
 PRIMARY KEY(run_id,sequence)
);
REVOKE ALL ON portway.codex_bridge,portway.codex_runs,portway.codex_events FROM PUBLIC;
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='portway_app') THEN
  GRANT SELECT,INSERT,UPDATE,DELETE ON portway.codex_bridge,portway.codex_runs,portway.codex_events TO portway_app;
 END IF;
END $$;
