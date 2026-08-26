import os
from pathlib import Path
import shutil
import subprocess
import time

def resolve_launcher():
    env_launcher = os.environ.get("ILMANGO_LAUNCHER_PATH")
    script_launcher = Path(__file__).resolve().parents[1] / "ilmango"
    xdg_bin_launcher = Path(os.environ.get("XDG_BIN_HOME", "~/.local/bin")).expanduser() / "ilmango"
    path_launcher = shutil.which("ilmango")

    for candidate in (env_launcher, str(script_launcher), str(xdg_bin_launcher), path_launcher):
        if not candidate:
            continue
        if Path(candidate).is_file():
            return candidate
    return None

def main():
    try:
        launcher = resolve_launcher()
        if launcher:
            subprocess.Popen([launcher, "lock", "activate"])
        else:
            print("Failed to lock: could not resolve ilmango launcher")
    except Exception as e:
        print(f"Failed to lock: {e}")

    time.sleep(1)

    try:
        subprocess.run(["systemctl", "suspend"], check=True)
    except Exception as e:
        print(f"Failed to suspend: {e}")

if __name__ == "__main__":
    main()
