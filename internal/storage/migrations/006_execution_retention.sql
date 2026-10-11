-- Watermarks advance atomically with bounded prefix deletion in execution.
ALTER TABLE portway_execution.runs ADD COLUMN events_pruned_through bigint NOT NULL DEFAULT 0 CHECK(events_pruned_through>=0);
CREATE INDEX execution_request_expiry ON portway_execution.request_keys(installation_id,expires_at);
CREATE TABLE portway_execution.maintenance_log (
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 installation_id uuid NOT NULL,
 run_id uuid,
 pruned_through bigint NOT NULL,
 events_deleted integer NOT NULL,
 keys_deleted integer NOT NULL,
 logs_deleted integer NOT NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX execution_maintenance_retention ON portway_execution.maintenance_log(installation_id,created_at);
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='portway_execution') THEN
  GRANT UPDATE(events_pruned_through) ON portway_execution.runs TO portway_execution;
  GRANT DELETE ON portway_execution.events,portway_execution.outbox TO portway_execution;
  GRANT SELECT,INSERT,DELETE ON portway_execution.maintenance_log TO portway_execution;
  GRANT USAGE ON SEQUENCE portway_execution.maintenance_log_id_seq TO portway_execution;
 END IF;
END $$;
