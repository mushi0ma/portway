#!/usr/bin/env python3
"""Create local configuration without exposing or overwriting existing secrets."""
import base64
import os
from pathlib import Path
import secrets
import uuid

root = Path(__file__).resolve().parent.parent
target = root / '.env'
content = (root / '.env.example').read_text()
values = {
    'PORTWAY_INSTALLATION_ID': str(uuid.uuid4()),
    'PORTWAY_MASTER_KEY': base64.b64encode(secrets.token_bytes(32)).decode(),
    'PORTWAY_SETUP_TOKEN': secrets.token_urlsafe(32),
}
for key, value in values.items():
    content = content.replace(key + '=\n', key + '=' + value + '\n')
try:
    fd = os.open(target, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
except FileExistsError:
    fd = os.open(target, os.O_RDWR | os.O_NOFOLLOW)
    if os.read(fd, 1):
        os.close(fd)
        raise SystemExit('.env already contains data; preserved without changes.')
    os.fchmod(fd, 0o600)
with os.fdopen(fd, 'w') as output:
    output.write(content)
print('Created .env (0600). Local installation ID/master key/setup proof generated; values not printed.')
print('Fill Supabase Auth URL, publishable key, migration DSN and app-role DSN; then set the public HTTPS URL.')
