CREATE TABLE portway.secrets (
 id uuid PRIMARY KEY,
 installation_id uuid NOT NULL REFERENCES portway.installations(id),
 alias text NOT NULL,
 ciphertext bytea NOT NULL,
 nonce bytea NOT NULL CHECK(octet_length(nonce)=12),
 key_version integer NOT NULL DEFAULT 1 CHECK(key_version>0),
 state text NOT NULL DEFAULT 'active' CHECK(state IN ('active','revoked')),
 UNIQUE(installation_id,id), UNIQUE(installation_id,alias), UNIQUE(installation_id,key_version,nonce)
);
CREATE TABLE portway.providers (
 id uuid PRIMARY KEY,
 installation_id uuid NOT NULL REFERENCES portway.installations(id),
 label text NOT NULL,
 endpoint text NOT NULL,
 model_id text NOT NULL,
 secret_ref uuid,
 config_version integer NOT NULL DEFAULT 1 CHECK(config_version>0),
 capabilities jsonb NOT NULL DEFAULT '{}',
 checked_at timestamptz,
 UNIQUE(installation_id,id),
 FOREIGN KEY(installation_id,secret_ref) REFERENCES portway.secrets(installation_id,id)
);
CREATE TABLE portway.agents (
 id uuid PRIMARY KEY,
 installation_id uuid NOT NULL REFERENCES portway.installations(id),
 name text NOT NULL CHECK(length(name) BETWEEN 1 AND 120),
 current_version_id uuid NOT NULL,
 version integer NOT NULL DEFAULT 1 CHECK(version>0),
 created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(installation_id,id)
);
CREATE TABLE portway.agent_versions (
 id uuid PRIMARY KEY,
 installation_id uuid NOT NULL,
 agent_id uuid NOT NULL,
 ordinal integer NOT NULL CHECK(ordinal>0),
 config_json jsonb NOT NULL,
 config_hash text NOT NULL CHECK(length(config_hash)=64),
 created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(installation_id,agent_id,ordinal), UNIQUE(installation_id,agent_id,id),
 FOREIGN KEY(installation_id,agent_id) REFERENCES portway.agents(installation_id,id) ON DELETE RESTRICT
);
ALTER TABLE portway.agents ADD CONSTRAINT current_own_version
 FOREIGN KEY(installation_id,id,current_version_id) REFERENCES portway.agent_versions(installation_id,agent_id,id)
 DEFERRABLE INITIALLY DEFERRED;
CREATE FUNCTION portway.reject_immutable_change() RETURNS trigger LANGUAGE plpgsql AS $$
 BEGIN RAISE EXCEPTION 'immutable record' USING ERRCODE='55000'; END;
$$;
CREATE TRIGGER immutable_version BEFORE UPDATE OR DELETE ON portway.agent_versions
 FOR EACH ROW EXECUTE FUNCTION portway.reject_immutable_change();
CREATE INDEX agent_list ON portway.agents(installation_id,created_at,id);
