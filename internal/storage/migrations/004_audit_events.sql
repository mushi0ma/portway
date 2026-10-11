-- Source service owns audit + outbox. Audit projection has no cross-service FK.
CREATE TABLE portway.outbox (
 sequence bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 event_id uuid NOT NULL UNIQUE,
 installation_id uuid NOT NULL,
 actor_id uuid,
 action text NOT NULL,
 object_type text NOT NULL,
 object_id uuid,
 outcome text NOT NULL,
 occurred_at timestamptz NOT NULL,
 published_at timestamptz
);
CREATE INDEX outbox_pending ON portway.outbox(installation_id,sequence) WHERE published_at IS NULL;

CREATE FUNCTION portway.event_status(target uuid) RETURNS TABLE(pending bigint,oldest_at timestamptz)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = '' AS $$
 SELECT count(*),min(occurred_at) FROM portway.outbox WHERE installation_id=target AND published_at IS NULL;
$$;
REVOKE ALL ON FUNCTION portway.event_status(uuid) FROM PUBLIC;

CREATE FUNCTION portway.enqueue_audit() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path = '' AS $$
BEGIN
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtext('portway.enqueue'), pg_catalog.hashtext(NEW.installation_id::text));
 -- Security revocation must still commit even when configuration admission closes.
 IF NEW.action <> 'device.revoked' AND (
   (SELECT count(*) FROM portway.outbox WHERE installation_id=NEW.installation_id AND published_at IS NULL) >= 10000
   OR EXISTS(SELECT 1 FROM portway.outbox WHERE installation_id=NEW.installation_id AND published_at IS NULL AND occurred_at < now()-interval '24 hours')
 ) THEN
   RAISE EXCEPTION 'event backlog unavailable' USING ERRCODE='PW001';
 END IF;
 INSERT INTO portway.outbox(event_id,installation_id,actor_id,action,object_type,object_id,outcome,occurred_at)
 VALUES(NEW.id,NEW.installation_id,NEW.actor_id,NEW.action,NEW.object_type,NEW.object_id,NEW.outcome,NEW.server_time);
 RETURN NEW;
END;
$$;
REVOKE ALL ON FUNCTION portway.enqueue_audit() FROM PUBLIC;
CREATE TRIGGER audit_outbox AFTER INSERT ON portway.audit_entries
 FOR EACH ROW EXECUTE FUNCTION portway.enqueue_audit();

CREATE SCHEMA portway_audit;
REVOKE ALL ON SCHEMA portway_audit FROM PUBLIC;
CREATE TABLE portway_audit.inbox (
 installation_id uuid NOT NULL,
 event_id uuid NOT NULL,
 content_hash text NOT NULL,
 received_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(installation_id,event_id)
);
CREATE TABLE portway_audit.entries (
 installation_id uuid NOT NULL,
 event_id uuid NOT NULL,
 actor_id uuid,
 action text NOT NULL,
 object_type text NOT NULL,
 object_id uuid,
 outcome text NOT NULL,
 occurred_at timestamptz NOT NULL,
 received_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(installation_id,event_id)
);
CREATE INDEX audit_projection_order ON portway_audit.entries(installation_id,occurred_at DESC,event_id DESC);
REVOKE ALL ON ALL TABLES IN SCHEMA portway_audit FROM PUBLIC;

-- Bounded maintenance; receipts live longer than the allowed delivery age.
CREATE FUNCTION portway.prune_outbox(target uuid) RETURNS void
LANGUAGE sql SECURITY DEFINER SET search_path = '' AS $$
 DELETE FROM portway.outbox WHERE sequence IN (
   SELECT sequence FROM portway.outbox WHERE installation_id=target
   AND published_at < now()-interval '7 days' ORDER BY sequence LIMIT 1000
 );
$$;
CREATE FUNCTION portway_audit.prune(target uuid) RETURNS void
LANGUAGE plpgsql SECURITY DEFINER SET search_path = '' AS $$
BEGIN
 DELETE FROM portway_audit.entries WHERE (installation_id,event_id) IN (
   SELECT installation_id,event_id FROM portway_audit.entries WHERE installation_id=target
   AND received_at < now()-interval '7 days' ORDER BY received_at LIMIT 1000
 );
 DELETE FROM portway_audit.inbox WHERE (installation_id,event_id) IN (
   SELECT installation_id,event_id FROM portway_audit.inbox WHERE installation_id=target
   AND received_at < now()-interval '7 days' ORDER BY received_at LIMIT 1000
 );
END;
$$;
REVOKE ALL ON FUNCTION portway.prune_outbox(uuid) FROM PUBLIC;
REVOKE ALL ON FUNCTION portway_audit.prune(uuid) FROM PUBLIC;

DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='portway_app') THEN
  GRANT EXECUTE ON FUNCTION portway.event_status(uuid) TO portway_app;
 END IF;
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='portway_relay') THEN
  GRANT USAGE ON SCHEMA portway TO portway_relay;
  GRANT SELECT ON portway.outbox TO portway_relay;
  GRANT UPDATE(published_at) ON portway.outbox TO portway_relay;
  GRANT EXECUTE ON FUNCTION portway.prune_outbox(uuid) TO portway_relay;
 END IF;
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='portway_audit') THEN
  GRANT USAGE ON SCHEMA portway_audit TO portway_audit;
  GRANT SELECT,INSERT ON portway_audit.inbox,portway_audit.entries TO portway_audit;
  GRANT EXECUTE ON FUNCTION portway_audit.prune(uuid) TO portway_audit;
 END IF;
END $$;
