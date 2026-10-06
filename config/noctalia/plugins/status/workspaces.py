#!/usr/bin/env python3
"""Publish workspaces and monitor-local window titles on Hyprland events."""

import json
import os
import re
import socket
import subprocess
import sys
import time
import tomllib
from pathlib import Path


def load_rules():
    path = Path(__file__).with_name("window-rewrites.toml")
    with path.open("rb") as config:
        rewrites = tomllib.load(config)["rewrite"]
    return [(re.compile(pattern), replacement) for pattern, replacement in rewrites.items()]


def rewrite_title(title, rules):
    title = " ".join(title.splitlines())
    for pattern, replacement in rules:
        match = pattern.fullmatch(title)
        if match:
            return re.sub(r"\$(\d+)", lambda ref: match.group(int(ref[1])) or "", replacement)
    return title


def window_titles(monitors, workspaces, rules):
    by_id = {workspace["id"]: workspace for workspace in workspaces}
    titles = {}
    for monitor in monitors:
        special = monitor.get("specialWorkspace", {}).get("id", 0)
        workspace_id = special or monitor["activeWorkspace"]["id"]
        workspace = by_id.get(workspace_id, {})
        title = workspace.get("lastwindowtitle", "") if workspace.get("windows", 0) > 0 else ""
        titles[monitor["name"]] = {"text": rewrite_title(title, rules), "tooltip": title}
    return titles


def snapshot(rules):
    def query(name):
        return json.loads(subprocess.check_output(
            ["hyprctl", "-j", name], timeout=3, text=True
        ))

    monitors = query("monitors")
    workspaces = query("workspaces")
    return {
        "active": {m["name"]: m["activeWorkspace"]["id"] for m in monitors},
        "occupied": sorted(w["id"] for w in workspaces if w["windows"] > 0),
        "titles": window_titles(monitors, workspaces, rules),
    }


def subscribe(rules):
    path = Path(os.environ["XDG_RUNTIME_DIR"]) / "hypr" / os.environ["HYPRLAND_INSTANCE_SIGNATURE"] / ".socket2.sock"
    events = {
        "workspacev2", "focusedmonv2", "openwindow", "closewindow",
        "movewindowv2", "monitoraddedv2", "monitorremoved", "moveworkspacev2",
        "activewindowv2", "windowtitlev2", "activespecial",
    }
    last = None

    def publish():
        nonlocal last
        value = json.dumps(snapshot(rules))
        if value != last:
            print(value, flush=True)
            last = value

    while True:
        try:
            with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as connection:
                connection.connect(str(path))
                publish()
                with connection.makefile("r") as stream:
                    for line in stream:
                        if line.split(">>", 1)[0] in events:
                            publish()
        except (OSError, ValueError, KeyError, subprocess.SubprocessError) as error:
            print(f"Hyprland workspace status: {error}", file=sys.stderr)
            print(json.dumps({"active": {}, "occupied": [], "titles": {}}), flush=True)
            last = None
        time.sleep(1)


if __name__ == "__main__":
    rules = load_rules()
    if sys.argv[1:] == ["--once"]:
        print(json.dumps(snapshot(rules)))
    else:
        subscribe(rules)
