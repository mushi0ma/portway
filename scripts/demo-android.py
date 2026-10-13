#!/usr/bin/env python3
"""Start the existing Portway stack/bridge and install the demo APK on its AVD.

No account creation, email sending, credential injection, or model request.
The Android SDK image and AVD must already be installed (see the client README).
"""
import argparse
import os
from pathlib import Path
import shutil
import subprocess
import sys
import time

ROOT = Path(__file__).resolve().parent.parent
AVD = "Portway_Demo_API36"
STATE = ROOT / ".tmp"
APK = ROOT / "output/android/portway-demo-0.2.0-3.apk"


def run(args, **kwargs):
    return subprocess.run(args, cwd=ROOT, check=True, **kwargs)


def background(args, name, env=None):
    STATE.mkdir(exist_ok=True)
    fd = os.open(STATE / (name + ".log"), os.O_WRONLY | os.O_CREAT | os.O_APPEND, 0o600)
    with os.fdopen(fd, "a") as log:
        process = subprocess.Popen(args, cwd=ROOT, stdin=subprocess.DEVNULL,
                                   stdout=log, stderr=subprocess.STDOUT, start_new_session=True, env=env)
    path = STATE / (name + ".pid")
    path.write_text(str(process.pid))
    path.chmod(0o600)
    return process


def bridge_running():
    try:
        pid = int((STATE / "codex-bridge.pid").read_text())
        command = Path(f"/proc/{pid}/cmdline").read_bytes().split(b"\0")
        return command[:2] == [str(ROOT / "bin/portway").encode(), b"codex-bridge"]
    except (ValueError, OSError):
        return False


def device(adb):
    listing = run([adb, "devices"], capture_output=True, text=True, timeout=10).stdout
    for line in listing.splitlines():
        fields = line.split()
        if len(fields) == 2 and fields[0].startswith("emulator-") and fields[1] == "device":
            result = subprocess.run([adb, "-s", fields[0], "emu", "avd", "name"],
                                    capture_output=True, text=True, timeout=10)
            if result.returncode == 0 and result.stdout.splitlines()[0] == AVD:
                return fields[0]
    return None


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true", help="check files and Codex/Claude CLI login state only")
    parser.add_argument("--skip-stack", action="store_true", help="use already running backend and bridge")
    args = parser.parse_args()
    sdk = Path(os.environ.get("ANDROID_HOME") or os.environ.get("ANDROID_SDK_ROOT") or Path.home() / "Android/Sdk")
    emulator = sdk / "emulator/emulator"
    adb = shutil.which("adb") or str(sdk / "platform-tools/adb")
    if not APK.is_file() or not emulator.is_file():
        raise RuntimeError("Нужны собранный APK и Android SDK; см. apps/portway/README.md")
    avds = run([str(emulator), "-list-avds"], capture_output=True, text=True).stdout.splitlines()
    if AVD not in avds:
        raise RuntimeError(f"Создайте AVD {AVD} по apps/portway/README.md")
    run([sys.executable, "scripts/start-codex-bridge.py", "--check"])
    if args.check:
        print(f"Готовы SDK, AVD {AVD} и APK; состояние входа Codex/Claude показано выше (аккаунты можно подключить с телефона). Реальный вход Portway проверяется в приложении.")
        return
    if not args.skip_stack:
        run(["docker", "compose", "up", "-d", "--no-build"])
        if not bridge_running():
            background([sys.executable, "scripts/start-codex-bridge.py"], "codex-bridge")
    run([adb, "start-server"])
    serial = device(adb)
    if not serial:
        # The SDK bundles xcb, but not a Qt Wayland plugin. Use XWayland on
        # Linux desktops that export QT_QPA_PLATFORM=wayland globally.
        environment = dict(os.environ, QT_QPA_PLATFORM="xcb")
        background([str(emulator), "-avd", AVD, "-memory", "2560", "-cores", "2",
                    "-no-boot-anim", "-no-snapshot", "-no-audio"], "portway-emulator", environment)
    deadline = time.monotonic() + 240
    print("Ожидаю загрузку Android…", flush=True)
    while time.monotonic() < deadline:
        serial = device(adb)
        if serial:
            boot = run([adb, "-s", serial, "shell", "getprop", "sys.boot_completed"],
                       capture_output=True, text=True, timeout=10)
            if boot.stdout.strip() == "1":
                break
        time.sleep(2)
    else:
        raise RuntimeError("Android не загрузился за 4 минуты; журнал .tmp/portway-emulator.log")
    run([adb, "-s", serial, "install", "-r", str(APK)], timeout=120)
    run([adb, "-s", serial, "shell", "am", "start", "-n", "app.portway.portway/.MainActivity"], timeout=15)
    print(f"Portway открыт в {AVD} ({serial}). Войдите в свой аккаунт в окне эмулятора.")


if __name__ == "__main__":
    try:
        main()
    except (RuntimeError, OSError, subprocess.SubprocessError) as error:
        # Commands never include credentials; captured output is deliberately omitted.
        print(f"Не удалось подготовить показ: {error if isinstance(error, RuntimeError) else type(error).__name__}", file=sys.stderr)
        sys.exit(1)
