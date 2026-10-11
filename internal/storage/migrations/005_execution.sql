-- Execution owns its aggregates and durable history; no cross-service FK.
CREATE SCHEMA portway_execution;
REVOKE ALL ON SCHEMA portway_execution FROM PUBLIC;
CREATE TABLE portway_execution.chains (
 installation_id uuid NOT NULL,
 owner_id uuid NOT NULL,
 id uuid NOT NULL,
 state text NOT NULL CHECK(state IN ('active','stopping','completed','failed','cancelled')),
 body jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(installation_id,id)
);
CREATE INDEX execution_active ON portway_execution.chains(installation_id,state);
CREATE TABLE portway_execution.runs (
 installation_id uuid NOT NULL,
 id uuid NOT NULL,
 chain_id uuid NOT NULL,
 PRIMARY KEY(installation_id,id),
 FOREIGN KEY(installation_id,chain_id) REFERENCES portway_execution.chains(installation_id,id)
);
CREATE TABLE portway_execution.events (
 installation_id uuid NOT NULL,
 id uuid NOT NULL,
 run_id uuid NOT NULL,
 seq bigint NOT NULL CHECK(seq>0),
 body jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(installation_id,id),
 UNIQUE(installation_id,run_id,seq),
 FOREIGN KEY(installation_id,run_id) REFERENCES portway_execution.runs(installation_id,id)
);
CREATE TABLE portway_execution.request_keys (
 installation_id uuid NOT NULL,
 owner_id uuid NOT NULL,
 operation text NOT NULL,
 key_hash text NOT NULL,
 request_hash text NOT NULL,
 response jsonb NOT NULL,
 expires_at timestamptz NOT NULL DEFAULT (clock_timestamp()+interval '7 days'),
 PRIMARY KEY(installation_id,owner_id,operation,key_hash)
);
-- Only immutable safe lifecycle facts go to Kafka, never pinned tasks/configs.
CREATE TABLE portway_execution.outbox (
 installation_id uuid NOT NULL,
 id uuid NOT NULL,
 sequence bigint GENERATED ALWAYS AS IDENTITY,
 body jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 published_at timestamptz,
 PRIMARY KEY(installation_id,id),
 FOREIGN KEY(installation_id,id) REFERENCES portway_execution.events(installation_id,id)
);
CREATE INDEX execution_outbox_pending ON portway_execution.outbox(installation_id,sequence) WHERE published_at IS NULL;
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='portway_execution') THEN
  GRANT USAGE ON SCHEMA portway_execution TO portway_execution;
  GRANT SELECT,INSERT ON portway_execution.chains,portway_execution.runs,portway_execution.events,portway_execution.outbox TO portway_execution;
  GRANT UPDATE(state,body) ON portway_execution.chains TO portway_execution;
  GRANT UPDATE(published_at) ON portway_execution.outbox TO portway_execution;
  GRANT SELECT,INSERT,DELETE ON portway_execution.request_keys TO portway_execution;
  GRANT USAGE ON ALL SEQUENCES IN SCHEMA portway_execution TO portway_execution;
 END IF;
END $$;
