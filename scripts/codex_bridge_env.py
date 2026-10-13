"""Read only native bridge parameters from a local dotenv file, without evaluation."""
import os
from pathlib import Path
import shlex

ROOT = Path(__file__).resolve().parent.parent
NAMES = {
    'PORTWAY_CODEX_BRIDGE_KEY', 'PORTWAY_CODEX_API_URL',
    'PORTWAY_CODEX_WORKSPACE', 'PORTWAY_CODEX_BINARY',
    'PORTWAY_CLAUDE_BINARY', 'PORTWAY_BRIDGE_DATA_DIR',
}


def read_parameters():
    path = ROOT / '.env'
    descriptor = os.open(path, os.O_RDONLY | os.O_NOFOLLOW)
    with os.fdopen(descriptor) as source:
        if os.fstat(source.fileno()).st_mode & 0o007:
            raise ValueError('.env must not be accessible to other users')
        content = source.read(65537)
    if len(content) > 65536:
        raise ValueError('.env exceeds size limit')
    parameters = {}
    for line in content.splitlines():
        name, separator, value = line.partition('=')
        name = name.strip()
        if name not in NAMES or not separator:
            continue
        if name in parameters:
            raise ValueError('Duplicate native bridge parameter')
        words = shlex.split(value, comments=True)
        if len(words) > 1:
            raise ValueError('Quote native bridge parameter values containing spaces')
        parameters[name] = words[0] if words else ''
    return content, parameters
