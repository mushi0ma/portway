#!/usr/bin/env python3
"""Provision missing Portway roles; preserve existing passwords; print no secrets.
Requires operator DSN, Docker Compose, psql, and verified TLS.
New passwords are saved before DB writes, so interrupted runs can be retried.
"""
import json
import os
from pathlib import Path
import secrets
import subprocess
import urllib.parse

root = Path(__file__).resolve().parent.parent
os.chdir(root)
result = subprocess.run(['docker', 'compose', '--profile', 'execution', 'config', '--format', 'json'], capture_output=True)
if result.returncode or b'variable is not set' in result.stderr:
    raise SystemExit('Invalid Compose configuration or unquoted dollar sign in .env; details withheld.')
services = json.loads(result.stdout)['services']
operator = urllib.parse.urlsplit(services['migrate']['environment']['PORTWAY_MIGRATION_DATABASE_URL'])
params = urllib.parse.parse_qs(operator.query)
if params.get('sslmode') != ['verify-full']:
    raise SystemExit('Verified TLS is required.')
cert = params.get('sslrootcert', ['/etc/ssl/certs/ca-certificates.crt'])[0]
if cert.startswith('/etc/portway/certs/'):
    cert = str(root / 'deploy/certs' / Path(cert).name)
env = dict(os.environ, PGHOST=operator.hostname, PGPORT=str(operator.port or 5432),
           PGUSER=urllib.parse.unquote(operator.username), PGPASSWORD=urllib.parse.unquote(operator.password),
           PGDATABASE=operator.path.lstrip('/'), PGSSLMODE='verify-full', PGSSLROOTCERT=cert, PGCONNECT_TIMEOUT='10')

def sql(statement):
    p = subprocess.run(['psql', '-X', '-w', '-qAt', '-v', 'ON_ERROR_STOP=1'], input=statement,
                       env=env, capture_output=True, text=True, timeout=30)
    if p.returncode:
        categories = [s for s in ('password authentication failed', 'certificate verify failed',
                                 'permission denied', 'timeout expired') if s in p.stderr.lower()]
        raise SystemExit('Database operation failed: ' + ', '.join(categories or ['details withheld']))
    return p.stdout.strip()

if sql('SELECT 1;') != '1':
    raise SystemExit('Operator connection not verified.')
print('Operator TLS connection verified.')
path = root / '.env'
text = path.read_text()
updates = {}
roles = [('portway_app', 'PORTWAY_DATABASE_URL', services['api']['environment']['PORTWAY_DATABASE_URL']),
         ('portway_relay', 'PORTWAY_RELAY_DATABASE_URL', services['outbox-relay']['environment']['PORTWAY_DATABASE_URL']),
         ('portway_audit', 'PORTWAY_AUDIT_DATABASE_URL', services['audit']['environment']['PORTWAY_DATABASE_URL']),
         ('portway_execution', 'PORTWAY_EXECUTION_DATABASE_URL', services['execution']['environment']['PORTWAY_DATABASE_URL'])]
if not services['api']['environment'].get('PORTWAY_EXECUTION_SERVICE_KEY'):
    updates['PORTWAY_EXECUTION_SERVICE_KEY'] = secrets.token_urlsafe(32)
configured = []
for role, key, dsn in roles:
    exists = sql("SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='" + role + "');") == 't'
    if not dsn:
        if exists:
            raise SystemExit(role + ' already exists; supply its DSN. Existing passwords are not changed.')
        suffix = '.' + operator.username.split('.', 1)[1] if '.' in operator.username else ''
        password = secrets.token_urlsafe(32)
        netloc = role + suffix + ':' + password + '@' + operator.netloc.rsplit('@', 1)[1]
        dsn = urllib.parse.urlunsplit(operator._replace(netloc=netloc))
        updates[key] = dsn
    u = urllib.parse.urlsplit(dsn)
    user = urllib.parse.unquote(u.username or '').split('.', 1)[0]
    if user != role or u.hostname != operator.hostname or u.path != operator.path or not u.password:
        raise SystemExit('Role DSN must use the same project and intended role: ' + role)
    configured.append((role, urllib.parse.unquote(u.password), exists))
if updates:
    lines = text.splitlines()
    for i, line in enumerate(lines):
        key = line.split('=', 1)[0]
        if key in updates:
            lines[i] = key + "='" + updates.pop(key) + "'"
    lines.extend(key + "='" + value + "'" for key, value in updates.items())
    if path.read_text() != text:
        raise SystemExit('.env changed during provisioning; retry with the new contents.')
    path.write_text('\n'.join(lines) + '\n')
    path.chmod(0o600)
for role, password, exists in configured:
    if not exists:
        escaped = password.replace("'", "''")
        sql('CREATE ROLE ' + role + " LOGIN PASSWORD '" + escaped + "' NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOBYPASSRLS;")
        print('Created role: ' + role)
    else:
        print('Preserved existing role: ' + role)
print('Role provisioning complete. Apply migrations for service-specific grants.')
