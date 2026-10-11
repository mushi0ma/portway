-- This fixed role is created by the operator, never by the API.
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='portway_app') THEN
  GRANT USAGE ON SCHEMA portway TO portway_app;
  GRANT SELECT ON ALL TABLES IN SCHEMA portway TO portway_app;
  GRANT INSERT,UPDATE ON portway.installations,portway.registration_intents,portway.device_sessions,portway.providers TO portway_app;
  GRANT INSERT ON portway.owners,portway.secrets,portway.agent_versions,portway.audit_entries TO portway_app;
  GRANT INSERT,UPDATE ON portway.agents TO portway_app;
  GRANT INSERT,DELETE ON portway.request_keys TO portway_app;
 END IF;
END $$;
REVOKE ALL ON ALL TABLES IN SCHEMA portway FROM PUBLIC;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA portway FROM PUBLIC;
