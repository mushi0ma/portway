#!/usr/bin/env python3
"""Start the laptop agent bridge (Codex CLI + Claude Code) with only its own parameters, never the full .env."""
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys

from codex_bridge_env import ROOT, read_parameters

# OS basics for the CLIs; display variables are used only to open a "computer" login in the laptop browser.
PASSTHROUGH = ('HOME', 'PATH', 'USER', 'SHELL', 'LANG', 'SSL_CERT_FILE', 'SSL_CERT_DIR',
               'CODEX_HOME', 'CLAUDE_CONFIG_DIR', 'XDG_CONFIG_HOME', 'XDG_DATA_HOME', 'XDG_CACHE_HOME',
               'XDG_STATE_HOME', 'DISPLAY', 'WAYLAND_DISPLAY', 'XDG_RUNTIME_DIR', 'DBUS_SESSION_BUS_ADDRESS',
               'XAUTHORITY', 'XDG_CURRENT_DESKTOP', 'XDG_SESSION_TYPE')


def binary(parameters, name, command):
    value = parameters.get(name, '')
    if value:
        if not os.path.isabs(value):
            raise ValueError(name + ' must be absolute')
        return value if os.access(value, os.X_OK) else ''
    return shutil.which(command) or ''


def cli_env(environment):
    return {name: environment[name] for name in PASSTHROUGH if name in environment}


def codex_state(path, environment):
    result = subprocess.run([path, 'login', 'status'], env=cli_env(environment), capture_output=True, timeout=15)
    return 'ChatGPT login OK' if result.returncode == 0 and b'Logged in using ChatGPT' in result.stdout + result.stderr \
        else 'not logged in with ChatGPT (connect from the phone or run codex login)'


def claude_state(path, environment):
    result = subprocess.run([path, 'auth', 'status', '--json'], env=cli_env(environment), capture_output=True, timeout=20)
    try:
        status = json.loads(result.stdout)
    except ValueError:
        return 'status unavailable'
    # Only subscription logins are used; the email is never printed.
    return 'Claude subscription login OK' if status.get('loggedIn') and status.get('authMethod') == 'claude.ai' \
        else 'not logged in with a Claude subscription (connect from the phone or run claude auth login)'


try:
    _, parameters = read_parameters()
    environment = {name: os.environ[name] for name in PASSTHROUGH if name in os.environ}
    environment.update({name: value for name, value in parameters.items() if value})
    key = parameters.get('PORTWAY_CODEX_BRIDGE_KEY', '')
    workspace = Path(parameters.get('PORTWAY_CODEX_WORKSPACE', ''))
    codex = binary(parameters, 'PORTWAY_CODEX_BINARY', 'codex')
    claude = binary(parameters, 'PORTWAY_CLAUDE_BINARY', 'claude')
    if len(key) < 32 or not workspace.is_absolute() or not workspace.is_dir() or not (codex or claude):
        raise ValueError('Invalid native bridge parameters')
    for name, path in (('PORTWAY_CODEX_BINARY', codex), ('PORTWAY_CLAUDE_BINARY', claude)):
        environment.pop(name, None)
        if path:
            environment[name] = path
    portway = ROOT / 'bin/portway'
    if not portway.is_file():
        raise ValueError('Build bin/portway first')
    if sys.argv[1:] == ['--check']:
        # Logged-out CLIs are reported, not fatal: accounts can be connected from the phone.
        print('Codex CLI:', codex_state(codex, environment) if codex else 'not installed')
        print('Claude Code:', claude_state(claude, environment) if claude else 'not installed')
        print('Native bridge parameters valid; secrets and emails omitted; services not started.')
    elif not sys.argv[1:]:
        os.execve(portway, [str(portway), 'codex-bridge'], environment)
    else:
        raise ValueError('usage: start-codex-bridge.py [--check]')
except (OSError, ValueError, subprocess.TimeoutExpired):
    raise SystemExit('Native bridge unavailable: build bin/portway, prepare .env bridge parameters, verify workspace '
                     'and install codex and/or claude (details omitted).')
