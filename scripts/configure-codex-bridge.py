#!/usr/bin/env python3
"""Prepare opt-in bridge credentials without starting services or exposing secrets."""
import json
import os
import secrets

from codex_bridge_env import ROOT, read_parameters

try:
    content, parameters = read_parameters()
    defaults = {
        'PORTWAY_CODEX_BRIDGE_KEY': secrets.token_urlsafe(32),
        'PORTWAY_CODEX_API_URL': 'http://127.0.0.1:8080',
        'PORTWAY_CODEX_WORKSPACE': str(ROOT),
        'PORTWAY_CODEX_BINARY': '',
        'PORTWAY_CLAUDE_BINARY': '',
        'PORTWAY_BRIDGE_DATA_DIR': '',
    }
    for name, value in defaults.items():
        if parameters.get(name):
            continue
        replacement = name + '=' + (json.dumps(value) if ' ' in value else value)
        if name in parameters:
            lines = content.splitlines()
            content = '\n'.join(replacement if line.partition('=')[0].strip() == name else line
                                for line in lines) + '\n'
        else:
            content = content.rstrip('\n') + '\n' + replacement + '\n'
    descriptor = os.open(ROOT / '.env', os.O_WRONLY | os.O_TRUNC | os.O_NOFOLLOW)
    with os.fdopen(descriptor, 'w') as output:
        os.fchmod(output.fileno(), 0o600)
        output.write(content)
    print('Prepared native bridge parameters in ignored .env (0600); existing values preserved, secrets omitted.')
    print('No services started. Review workspace, then deploy the tested API and run scripts/start-codex-bridge.py.')
except (OSError, ValueError):
    raise SystemExit('Cannot prepare native bridge parameters; check local .env permissions and syntax (values omitted).')
