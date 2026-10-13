-- Agent Bridge v2: provider/model selection, encrypted bridge snapshot and account commands.
-- Additive only: existing Codex rows default to provider 'codex' and keep their meaning.
ALTER TABLE portway.codex_runs
 ADD COLUMN provider text NOT NULL DEFAULT 'codex' CHECK(provider IN ('codex','claude')),
 ADD COLUMN model text CHECK(char_length(model) BETWEEN 1 AND 100),
 ADD COLUMN effort text CHECK(effort IN ('low','medium','high','xhigh','max')),
 ADD COLUMN resolved_model text CHECK(char_length(resolved_model) BETWEEN 1 AND 100),
 ADD COLUMN account_label text CHECK(char_length(account_label) BETWEEN 1 AND 60),
 ADD COLUMN input_tokens bigint CHECK(input_tokens >= 0),
 ADD COLUMN output_tokens bigint CHECK(output_tokens >= 0);
-- The laptop snapshot contains account emails, so it is stored only as AES-GCM ciphertext.
ALTER TABLE portway.codex_bridge
 ADD COLUMN state_ciphertext bytea CHECK(octet_length(state_ciphertext) <= 65552),
 ADD COLUMN state_nonce bytea CHECK(octet_length(state_nonce) = 12),
 ADD COLUMN state_updated_at timestamptz,
 ADD CONSTRAINT codex_bridge_state_pair CHECK((state_ciphertext IS NULL) = (state_nonce IS NULL));
CREATE TABLE portway.codex_commands (
 id uuid PRIMARY KEY,
 installation_id uuid NOT NULL REFERENCES portway.installations(id),
 owner_id uuid NOT NULL REFERENCES portway.owners(id),
 device_id uuid NOT NULL REFERENCES portway.device_sessions(id),
 kind text NOT NULL CHECK(kind IN ('login_start','login_code','login_cancel','account_activate','account_remove','usage_refresh')),
 state text NOT NULL CHECK(state IN ('queued','delivered','done','failed')),
 provider text CHECK(provider IN ('codex','claude')),
 method text CHECK(method IN ('phone','computer')),
 login_id uuid,
 -- Payload (login code included) is encrypted and wiped as soon as the bridge receives it.
 payload_ciphertext bytea CHECK(octet_length(payload_ciphertext) <= 4096),
 payload_nonce bytea CHECK(octet_length(payload_nonce) = 12),
 error_code text CHECK(error_code ~ '^[A-Z][A-Z0-9_]{1,63}$'),
 created_at timestamptz NOT NULL DEFAULT now(),
 delivered_at timestamptz,
 updated_at timestamptz NOT NULL DEFAULT now(),
 CHECK((payload_ciphertext IS NULL) = (payload_nonce IS NULL)),
 CHECK(state = 'queued' OR payload_ciphertext IS NULL)
);
CREATE INDEX codex_commands_queue ON portway.codex_commands(installation_id,created_at) WHERE state IN ('queued','delivered');
CREATE INDEX codex_commands_recent ON portway.codex_commands(installation_id,created_at DESC);
CREATE INDEX codex_commands_login ON portway.codex_commands(installation_id,login_id) WHERE login_id IS NOT NULL;
REVOKE ALL ON portway.codex_commands FROM PUBLIC;
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='portway_app') THEN
  GRANT SELECT,INSERT,UPDATE,DELETE ON portway.codex_commands TO portway_app;
 END IF;
END $$;
