#!/usr/bin/env python3
"""Read-only checks of the configured deployment; never print credentials/bodies.

Uses the same resolved Compose environment as deployment. Does not register an
owner, send email, create test business records, or change database settings.
"""
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import urllib.error
import urllib.parse
import urllib.request

root = Path(__file__).resolve().parent.parent
os.chdir(root)
failures = []


def report(label, ok, detail=''):
    print(('PASS' if ok else 'FAIL') + ' ' + label + (': ' + detail if detail else ''), flush=True)
    if not ok:
        failures.append(label)


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


opener = urllib.request.build_opener(NoRedirect())


def http(url, headers=None):
    try:
        with opener.open(urllib.request.Request(url, headers=headers or {}), timeout=15) as response:
            return response.status, json.loads(response.read(1 << 20))
    except urllib.error.HTTPError as error:
        return error.code, {}
    except Exception:
        return 0, {}


def query(dsn, statement):
    try:
        u = urllib.parse.urlsplit(dsn)
        params = urllib.parse.parse_qs(u.query)
        if params.get('sslmode') != ['verify-full']:
            return None
        cert = params.get('sslrootcert', ['/etc/ssl/certs/ca-certificates.crt'])[0]
        if cert.startswith('/etc/portway/certs/'):
            cert = str(root / 'deploy/certs' / Path(cert).name)
        env = dict(os.environ, PGHOST=u.hostname, PGPORT=str(u.port or 5432),
                   PGUSER=urllib.parse.unquote(u.username), PGPASSWORD=urllib.parse.unquote(u.password),
                   PGDATABASE=u.path.lstrip('/'), PGSSLMODE='verify-full', PGSSLROOTCERT=cert,
                   PGCONNECT_TIMEOUT='10', PGOPTIONS='-c statement_timeout=10000 -c default_transaction_read_only=on')
        result = subprocess.run(['psql', '-X', '-w', '-qAt', '-v', 'ON_ERROR_STOP=1'],
                                input=statement, env=env, capture_output=True, text=True, timeout=25)
        return result.stdout.strip() if result.returncode == 0 else None
    except Exception:
        return None


result = subprocess.run(['docker', 'compose', '--profile', 'execution', 'config', '--format', 'json'], capture_output=True)
if result.returncode or b'variable is not set' in result.stderr:
    raise SystemExit('FAIL Compose configuration; details withheld.')
services = json.loads(result.stdout)['services']
api = services['api']['environment']
admin = services['migrate']['environment']['PORTWAY_MIGRATION_DATABASE_URL']
roles = [('api', 'portway_app', 'portway'), ('outbox-relay', 'portway_relay', 'portway'), ('audit', 'portway_audit', 'portway_audit')]
if api.get('PORTWAY_EXECUTION_URL'):
    roles.append(('execution', 'portway_execution', 'portway_execution'))
for service, role, schema in roles:
    dsn = services[service]['environment']['PORTWAY_DATABASE_URL']
    sql = "SELECT current_user,rolsuper,rolbypassrls,has_schema_privilege(current_user,'" + schema + "','CREATE') FROM pg_roles WHERE rolname=current_user;"
    report(service + ' verified TLS and restricted DB role', query(dsn, sql) == role + '|f|f|f')

expected = []
for path in sorted((root / 'internal/storage/migrations').glob('*.sql')):
    expected.append(str(int(path.name.split('_')[0])) + '|' + hashlib.sha256(path.read_bytes()).hexdigest())
actual = query(admin, 'SELECT version,checksum FROM portway.schema_migrations ORDER BY version;')
report('migration checksums', actual == '\n'.join(expected), str(len(expected)) + ' expected migrations')
anonymous = query(admin, "SELECT count(*)=6 AND bool_and(NOT has_schema_privilege(r.oid,n.oid,'USAGE')) FROM pg_roles r CROSS JOIN pg_namespace n WHERE r.rolname IN ('anon','authenticated') AND n.nspname IN ('portway','portway_audit','portway_execution');")
report('business schemas closed to Supabase client roles', anonymous == 't')

code, body = http(api['PORTWAY_AUTH_URL'] + '/settings', {'apikey': api['PORTWAY_AUTH_PUBLISHABLE_KEY']})
report('Supabase Auth publishable key and email confirmation', code == 200 and body.get('external', {}).get('email') is True and body.get('mailer_autoconfirm') is False, 'HTTP ' + str(code))
port = services['api']['ports'][0]['published']
endpoints = [] if '--private-only' in sys.argv else [('local API', 'http://127.0.0.1:' + str(port)), ('public API', api['PORTWAY_PUBLIC_URL'].rstrip('/'))]
for label, base in endpoints:
    for path, expected_status in [('/healthz', 200), ('/v1/client-config', 200), ('/v1/agents', 401), ('/v1/runs', 401), ('/v1/runs/30000000-0000-4000-8000-000000000001/events', 401), ('/auth/confirm', 400), ('/readyz', 503)]:
        code, body = http(base + path)
        ok = code == expected_status
        if path == '/v1/client-config':
            ok = ok and body.get('data', {}).get('installation_id') == api['PORTWAY_INSTALLATION_ID']
        report(label + ' ' + path, ok, 'HTTP ' + str(code))
        if code == 0:
            break
for service in ('outbox-relay', 'audit'):
    p = subprocess.run(['docker', 'compose', 'exec', '-T', service, '/events', 'probe'], capture_output=True)
    report(service + ' process readiness (DB + Kafka)', p.returncode == 0)
if api.get('PORTWAY_EXECUTION_URL'):
    p = subprocess.run(['docker', 'compose', 'exec', '-T', 'execution', '/execution', 'probe'], capture_output=True)
    report('execution process readiness (DB + Kafka; run admission closed)', p.returncode == 0)
raise SystemExit(1 if failures else 0)
