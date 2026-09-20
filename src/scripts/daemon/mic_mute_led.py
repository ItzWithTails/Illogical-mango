#!/usr/bin/env python3

"""Keep the kernel microphone-mute LED in sync with PipeWire's mute state.

Some laptops expose the LED through the ALSA ``audio-micmute`` trigger, while
the active digital microphone is muted only in PipeWire.  In that situation an
unused ALSA Capture Switch can remain off and leave the LED permanently lit.
Synchronising the capture switches gives the kernel trigger the state it
expects without requiring write access to /sys.
"""

import argparse
import ctypes
import ctypes.util
import glob
import re
from pathlib import Path


MIC_MUTE_LED = Path("/sys/class/leds/platform::micmute")


def _active_audio_micmute_trigger() -> bool:
    try:
        return "[audio-micmute]" in (MIC_MUTE_LED / "trigger").read_text()
    except OSError:
        return False


def _load_alsa() -> ctypes.CDLL:
    library_path = ctypes.util.find_library("asound") or "libasound.so.2"
    alsa = ctypes.CDLL(library_path)

    handle_ptr = ctypes.POINTER(ctypes.c_void_p)
    alsa.snd_mixer_open.argtypes = [handle_ptr, ctypes.c_int]
    alsa.snd_mixer_open.restype = ctypes.c_int
    alsa.snd_mixer_attach.argtypes = [ctypes.c_void_p, ctypes.c_char_p]
    alsa.snd_mixer_attach.restype = ctypes.c_int
    alsa.snd_mixer_selem_register.argtypes = [ctypes.c_void_p, ctypes.c_void_p, ctypes.c_void_p]
    alsa.snd_mixer_selem_register.restype = ctypes.c_int
    alsa.snd_mixer_load.argtypes = [ctypes.c_void_p]
    alsa.snd_mixer_load.restype = ctypes.c_int
    alsa.snd_mixer_first_elem.argtypes = [ctypes.c_void_p]
    alsa.snd_mixer_first_elem.restype = ctypes.c_void_p
    alsa.snd_mixer_elem_next.argtypes = [ctypes.c_void_p]
    alsa.snd_mixer_elem_next.restype = ctypes.c_void_p
    alsa.snd_mixer_selem_is_active.argtypes = [ctypes.c_void_p]
    alsa.snd_mixer_selem_is_active.restype = ctypes.c_int
    alsa.snd_mixer_selem_has_capture_switch.argtypes = [ctypes.c_void_p]
    alsa.snd_mixer_selem_has_capture_switch.restype = ctypes.c_int
    alsa.snd_mixer_selem_set_capture_switch_all.argtypes = [ctypes.c_void_p, ctypes.c_int]
    alsa.snd_mixer_selem_set_capture_switch_all.restype = ctypes.c_int
    alsa.snd_mixer_close.argtypes = [ctypes.c_void_p]
    alsa.snd_mixer_close.restype = ctypes.c_int
    return alsa


def _card_numbers() -> list[int]:
    numbers = []
    for path in glob.glob("/dev/snd/controlC*"):
        match = re.search(r"controlC(\d+)$", path)
        if match:
            numbers.append(int(match.group(1)))
    return sorted(set(numbers))


def _set_card_capture_switches(alsa: ctypes.CDLL, card_number: int, enabled: bool) -> int:
    mixer = ctypes.c_void_p()
    if alsa.snd_mixer_open(ctypes.byref(mixer), 0) < 0:
        return 0

    changed = 0
    try:
        if alsa.snd_mixer_attach(mixer, f"hw:{card_number}".encode()) < 0:
            return 0
        if alsa.snd_mixer_selem_register(mixer, None, None) < 0:
            return 0
        if alsa.snd_mixer_load(mixer) < 0:
            return 0

        element = alsa.snd_mixer_first_elem(mixer)
        while element:
            if (
                alsa.snd_mixer_selem_is_active(element)
                and alsa.snd_mixer_selem_has_capture_switch(element)
                and alsa.snd_mixer_selem_set_capture_switch_all(element, int(enabled)) >= 0
            ):
                changed += 1
            element = alsa.snd_mixer_elem_next(element)
    finally:
        alsa.snd_mixer_close(mixer)
    return changed


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("state", choices=("muted", "unmuted"))
    args = parser.parse_args()

    if not _active_audio_micmute_trigger():
        return 0

    alsa = _load_alsa()
    capture_enabled = args.state == "unmuted"
    for card_number in _card_numbers():
        _set_card_capture_switches(alsa, card_number, capture_enabled)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
