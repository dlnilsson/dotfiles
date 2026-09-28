#!/usr/bin/env python3

import json
import math
import os
import subprocess
import threading
from datetime import datetime
from pathlib import Path

os.environ["GDK_BACKEND"] = "wayland"

import gi

gi.require_version("Gtk", "3.0")
gi.require_version("Gdk", "3.0")
gi.require_version("GtkLayerShell", "0.1")
from gi.repository import Gdk, GLib, Gtk, GtkLayerShell


def limit_label(limit):
    percent = limit["percent"]
    if isinstance(percent, bool) or not isinstance(percent, (int, float)) or not math.isfinite(percent):
        raise ValueError("Invalid usage percentage")
    label = f'{limit["label"]}: {percent:.0%} used'
    reset = limit.get("resetsAt")
    if reset:
        reset = datetime.fromisoformat(reset.replace("Z", "+00:00")).astimezone()
        label += f' · resets {reset:%a %d %b, %H:%M %Z}'
    return label


def read_limits(agent):
    try:
        result = subprocess.run(
            [str(Path.home() / ".dotfiles/bin" / f"agent-usage-{agent}"), "--limits-only"],
            capture_output=True,
            text=True,
            check=True,
            timeout=30,
        )
        record = json.loads(result.stdout)
        labels = [limit_label(limit) for limit in record.get("limits", [])]
        status = record.get("usageStatusText")
        if status:
            labels.append(status)
        return labels or ["Limits unavailable"]
    except subprocess.TimeoutExpired:
        return ["Usage request timed out"]
    except (OSError, subprocess.CalledProcessError, ValueError, KeyError, TypeError, AttributeError):
        return ["Limits unavailable"]


def main():
    window = Gtk.Window(title="Agent usage limits")
    window.set_name("agent-usage-menu")
    window.set_decorated(False)
    window.set_resizable(False)
    GtkLayerShell.init_for_window(window)
    GtkLayerShell.set_namespace(window, "agent-usage-menu")
    GtkLayerShell.set_layer(window, GtkLayerShell.Layer.OVERLAY)
    GtkLayerShell.set_keyboard_mode(window, GtkLayerShell.KeyboardMode.EXCLUSIVE)
    GtkLayerShell.set_anchor(window, GtkLayerShell.Edge.TOP, True)
    GtkLayerShell.set_anchor(window, GtkLayerShell.Edge.RIGHT, True)
    GtkLayerShell.set_margin(window, GtkLayerShell.Edge.TOP, 40)
    GtkLayerShell.set_margin(window, GtkLayerShell.Edge.RIGHT, 16)
    try:
        pointer = json.loads(subprocess.check_output(["hyprctl", "-j", "cursorpos"], timeout=2))
        monitor = Gdk.Display.get_default().get_monitor_at_point(pointer["x"], pointer["y"])
        GtkLayerShell.set_monitor(window, monitor)
    except (OSError, subprocess.SubprocessError, ValueError, KeyError):
        pass

    content = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=10)
    content.set_border_width(12)
    window.add(content)
    css = Gtk.CssProvider()
    css.load_from_data(b"""
        #agent-usage-menu { background: #2e3440; color: #d8dee9; border: 1px solid #4c566a; }
        #agent-usage-menu .heading { color: #88c0d0; font-weight: bold; }
        #agent-usage-menu button { background: #3b4252; color: #d8dee9; padding: 6px 12px; }
        #agent-usage-menu button:hover { background: #434c5e; }
    """)
    Gtk.StyleContext.add_provider_for_screen(
        Gdk.Screen.get_default(), css, Gtk.STYLE_PROVIDER_PRIORITY_APPLICATION
    )

    def update(label, lines):
        label.set_text("\n".join(lines))
        return GLib.SOURCE_REMOVE

    def fetch(agent, placeholder):
        GLib.idle_add(update, placeholder, read_limits(agent))

    for agent, name in (("claude", "Claude"), ("codex", "Codex")):
        if content.get_children():
            content.pack_start(Gtk.Separator(), False, False, 0)
        heading = Gtk.Label(label=name, xalign=0)
        heading.get_style_context().add_class("heading")
        content.pack_start(heading, False, False, 0)
        placeholder = Gtk.Label(label="Loading usage limits…", xalign=0)
        content.pack_start(placeholder, False, False, 0)
        threading.Thread(target=fetch, args=(agent, placeholder), daemon=True).start()

    def close(*_):
        window.destroy()

    def key_press(_, event):
        if event.keyval == Gdk.KEY_Escape:
            close()
            return True
        return False

    close_button = Gtk.Button(label="Close")
    close_button.connect("clicked", close)
    content.pack_start(close_button, False, False, 0)
    window.connect("key-press-event", key_press)
    window.connect("destroy", lambda *_: Gtk.main_quit())
    window.show_all()
    Gtk.main()


if __name__ == "__main__":
    main()
