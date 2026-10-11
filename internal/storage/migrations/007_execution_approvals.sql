-- Approval content belongs to the locked chain aggregate. This private index
-- supports lookup and expiry without scanning other services' state.
CREATE TABLE portway_execution.approvals (
 installation_id uuid NOT NULL,
 id uuid NOT NULL,
 run_id uuid NOT NULL,
 state text NOT NULL CHECK(state IN ('pending','approved','rejected','expired','cancelled')),
 expires_at timestamptz NOT NULL,
 PRIMARY KEY(installation_id,id),
 FOREIGN KEY(installation_id,run_id) REFERENCES portway_execution.runs(installation_id,id)
);
CREATE UNIQUE INDEX execution_one_pending_approval ON portway_execution.approvals(installation_id,run_id) WHERE state='pending';
CREATE INDEX execution_approval_expiry ON portway_execution.approvals(installation_id,expires_at) WHERE state='pending';
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='portway_execution') THEN
  GRANT SELECT,INSERT ON portway_execution.approvals TO portway_execution;
  GRANT UPDATE(state) ON portway_execution.approvals TO portway_execution;
 END IF;
END $$;
