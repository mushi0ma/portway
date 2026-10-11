CREATE SCHEMA IF NOT EXISTS portway;
REVOKE ALL ON SCHEMA portway FROM PUBLIC;

CREATE TABLE portway.installations (
 id uuid PRIMARY KEY,
 name text NOT NULL,
 setup_hash text,
 status text NOT NULL DEFAULT 'unclaimed' CHECK (status IN ('unclaimed','pending_confirmation','claimed')),
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE portway.registration_intents (
 id uuid PRIMARY KEY,
 installation_id uuid NOT NULL UNIQUE REFERENCES portway.installations(id),
 auth_user_id uuid,
 email_hash text NOT NULL,
 display_name text NOT NULL,
 state text NOT NULL CHECK (state IN ('pending','sent','uncertain','claimed')),
 key_hash text NOT NULL,
 request_hash text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE portway.owners (
 id uuid PRIMARY KEY,
 installation_id uuid NOT NULL UNIQUE REFERENCES portway.installations(id),
 auth_user_id uuid NOT NULL,
 display_name text NOT NULL,
 email text NOT NULL,
 UNIQUE(installation_id,id), UNIQUE(installation_id,auth_user_id)
);
CREATE TABLE portway.device_sessions (
 id uuid PRIMARY KEY,
 installation_id uuid NOT NULL,
 owner_id uuid NOT NULL,
 proof_hash text NOT NULL UNIQUE,
 auth_session_id uuid NOT NULL,
 label text NOT NULL,
 revoked_at timestamptz,
 last_seen_at timestamptz NOT NULL DEFAULT now(),
 created_at timestamptz NOT NULL DEFAULT now(),
 FOREIGN KEY(installation_id,owner_id) REFERENCES portway.owners(installation_id,id),
 UNIQUE(installation_id,id)
);
CREATE TABLE portway.audit_entries (
 id uuid PRIMARY KEY,
 installation_id uuid NOT NULL REFERENCES portway.installations(id),
 actor_type text NOT NULL,
 actor_id uuid,
 action text NOT NULL,
 object_type text NOT NULL,
 object_id uuid,
 outcome text NOT NULL,
 reason text NOT NULL DEFAULT '',
 safe_metadata jsonb NOT NULL DEFAULT '{}',
 server_time timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX audit_order ON portway.audit_entries(installation_id,server_time DESC,id DESC);
CREATE TABLE portway.request_keys (
 installation_id uuid NOT NULL,
 owner_id uuid NOT NULL,
 operation text NOT NULL,
 key_hash text NOT NULL,
 request_hash text NOT NULL,
 response_status integer NOT NULL,
 response_body_safe jsonb NOT NULL,
 expires_at timestamptz NOT NULL DEFAULT now()+interval '7 days',
 PRIMARY KEY(installation_id,owner_id,operation,key_hash),
 FOREIGN KEY(installation_id,owner_id) REFERENCES portway.owners(installation_id,id)
);
