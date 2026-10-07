#!/usr/bin/env python3
"""Compare both executables through Unix sockets and measure daemon RSS/PSS."""

import argparse
import contextlib
import copy
import json
import os
from pathlib import Path
import selectors
import socket
import statistics
import subprocess
import sys
import tempfile
import threading
import time

ROOT = Path(__file__).resolve().parent
SOURCE = ROOT.parent / "config/noctalia/plugins/status/workspaces.py"
RULES = SOURCE.with_name("window-rewrites.toml")
BINARY = ROOT / "zig-out/bin/noctalia-workspaces"
EMPTY = {"active": {}, "occupied": [], "titles": {}}
EVENTS = [
    "workspacev2", "focusedmonv2", "openwindow", "closewindow",
    "movewindowv2", "monitoraddedv2", "monitorremoved", "moveworkspacev2",
    "activewindowv2", "windowtitlev2", "activespecial",
]


class MockHyprland:
    def __init__(self, directory):
        self.directory = directory
        directory.mkdir(parents=True)
        self.state = {"monitors": [], "workspaces": []}
        self.clients = []
        self.lock = threading.Lock()
        self.queries = 0
        self.malformed = False
        self.delay = 0
        self.stopped = threading.Event()
        self.sockets = []
        self.threads = []
        for filename, handler in [(".socket.sock", self.command), (".socket2.sock", self.events)]:
            server = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
            server.bind(str(directory / filename))
            server.listen()
            server.settimeout(0.1)
            self.sockets.append(server)
            thread = threading.Thread(target=self.accept, args=(server, handler), daemon=True)
            thread.start()
            self.threads.append(thread)

    def accept(self, server, handler):
        while not self.stopped.is_set():
            try:
                client, _ = server.accept()
            except socket.timeout:
                continue
            except OSError:
                return
            threading.Thread(target=handler, args=(client,), daemon=True).start()

    def command(self, client):
        with client:
            request = client.recv(128).decode()
            assert request in ("j/monitors", "j/workspaces"), request
            with self.lock:
                data = b"invalid JSON" if self.malformed else json.dumps(self.state[request[2:]]).encode()
                delay = self.delay
                self.queries += 1
            if delay:
                self.stopped.wait(delay)
            try:
                # Exercise partial reads, including replies larger than the Zig buffer.
                for offset in range(0, len(data), 4093):
                    client.sendall(data[offset:offset + 4093])
            except (BrokenPipeError, ConnectionResetError):
                pass

    def events(self, client):
        with self.lock:
            self.clients.append(client)

    def set(self, state):
        with self.lock:
            self.state = copy.deepcopy(state)

    def broadcast(self, data):
        with self.lock:
            for client in self.clients[:]:
                try:
                    client.sendall(data)
                except (BrokenPipeError, ConnectionResetError):
                    client.close()
                    self.clients.remove(client)

    def disconnect(self):
        with self.lock:
            for client in self.clients:
                client.shutdown(socket.SHUT_RDWR)
                client.close()
            self.clients.clear()

    def close(self):
        self.stopped.set()
        self.disconnect()
        for server in self.sockets:
            server.close()
        for thread in self.threads:
            thread.join()


class Stream:
    def __init__(self, command, env):
        self.errors = tempfile.TemporaryFile()
        self.process = subprocess.Popen(command, env=env, stdout=subprocess.PIPE, stderr=self.errors)
        self.selector = selectors.DefaultSelector()
        self.selector.register(self.process.stdout, selectors.EVENT_READ)
        self.pending = b""

    def line(self, timeout=5):
        deadline = time.monotonic() + timeout
        while b"\n" not in self.pending:
            if not self.selector.select(max(0, deadline - time.monotonic())):
                raise TimeoutError("No JSON line received")
            data = os.read(self.process.stdout.fileno(), 65536)
            if not data:
                self.errors.seek(0)
                raise AssertionError(f"Stream exited: {self.errors.read().decode()}")
            self.pending += data
        line, self.pending = self.pending.split(b"\n", 1)
        return json.loads(line)

    def silent(self):
        assert not self.pending and not self.selector.select(0.15), "Unexpected publication"

    def close(self):
        self.process.terminate()
        try:
            self.process.wait(timeout=5)
        except subprocess.TimeoutExpired:
            self.process.kill()
            self.process.wait()
        self.selector.close()
        self.process.stdout.close()
        self.errors.close()


@contextlib.contextmanager
def streams(commands, env):
    running = []
    try:
        for command in commands:
            running.append(Stream(command, env))
        yield running
    finally:
        for stream in running:
            stream.close()


def once(command, env):
    result = subprocess.run(command + ["--once"], env=env, capture_output=True, text=True, timeout=7)
    assert result.returncode == 0, result.stderr
    return json.loads(result.stdout)


def memory(pid):
    values = {}
    for line in Path(f"/proc/{pid}/smaps_rollup").read_text().splitlines():
        fields = line.split()
        if fields[0] in ("Rss:", "Pss:"):
            values[fields[0][:-1]] = int(fields[1])
    return values


def measure(running):
    samples = [[], []]
    for _ in range(7):
        for index, stream in enumerate(running):
            samples[index].append(memory(stream.process.pid))
        time.sleep(0.03)
    values = [{key: int(statistics.median(s[key] for s in sample)) for key in ("Rss", "Pss")} for sample in samples]
    for name, value in zip(("Python", "Zig"), values):
        print(f"{name}: RSS {value['Rss']} KiB, PSS {value['Pss']} KiB")
    for key in ("Rss", "Pss"):
        print(f"{key} reduction: {(1 - values[1][key] / values[0][key]) * 100:.1f}%")
    return values


def fixture(title="A tab — Zen Browser"):
    return {
        "monitors": [
            {"name": "DP-1", "activeWorkspace": {"id": 2}, "specialWorkspace": {"id": 0}, "ignored": True},
            {"name": "HDMI-A-1", "activeWorkspace": {"id": 1}, "specialWorkspace": {"id": -99}},
            {"name": "eDP-1", "activeWorkspace": {"id": 123}},
        ],
        "workspaces": [
            {"id": 2, "windows": 2, "lastwindowtitle": title},
            {"id": 1, "windows": 0, "lastwindowtitle": "stale title"},
            {"id": -99, "windows": 1, "lastwindowtitle": "special - Slack | Company"},
            {"id": 9, "windows": 3, "lastwindowtitle": "background window"},
        ],
    }


def wait_for(condition):
    deadline = time.monotonic() + 5
    while not condition():
        if time.monotonic() >= deadline:
            raise TimeoutError("Expected reload did not occur")
        time.sleep(0.01)


def reload_check(server, directory, env):
    rules = directory / "live-rewrites.toml"
    reference = directory / "reference"
    reference.mkdir()
    source = reference / "workspaces.py"
    source.write_bytes(SOURCE.read_bytes())
    reference_rules = reference / "window-rewrites.toml"
    state = fixture("idle desktop — Zen Browser")
    server.set(state)

    def save(prefix, atomic=False):
        text = f"[rewrite]\n'(.*) — Zen Browser' = '{prefix} $1'\n"
        reference_rules.write_text(text)
        if atomic:
            temporary = rules.with_suffix(".tmp")
            temporary.write_text(text)
            temporary.replace(rules)
        else:
            rules.write_text(text)
        return once([sys.executable, str(source)], env)

    expected = save("initial")
    with streams([[str(BINARY), "--rules", str(rules)]], env) as running:
        stream = running[0]
        assert stream.line() == expected
        expected = save("in-place")
        assert stream.line() == expected
        # No Hyprland event is sent for either save.
        expected = save("atomic", atomic=True)
        assert stream.line() == expected

        queries = server.queries
        with rules.open("a") as output:
            output.write("# No output change\n")
        wait_for(lambda: server.queries >= queries + 2)
        stream.silent()

        def rejected(text, error):
            previous = os.fstat(stream.errors.fileno()).st_size
            if text is None:
                rules.unlink()
            else:
                rules.write_text(text)
            wait_for(lambda: error.encode() in os.pread(stream.errors.fileno(), 65536, previous))
            stream.silent()
            assert stream.process.poll() is None
            # The last good rules still apply on subsequent Hyprland events.
            state["workspaces"][0]["lastwindowtitle"] += " — Zen Browser"
            server.set(state)
            server.broadcast(b"windowtitlev2>>last-good\n")
            assert stream.line() == once([sys.executable, str(source)], env)

        valid_rule = "[rewrite]\n'(.*) — Zen Browser' = 'discarded $1'\n"
        rejected(valid_rule + "unfinished", "ExpectedTomlLiteralString")
        rejected(valid_rule + "'[' = 'invalid'\n", "InvalidRegex")
        rejected(valid_rule + "'(.*)' = '$2'\n", "InvalidCaptureGroup")
        rejected(None, "OpenRulesFailed")
        expected = save("recreated", atomic=True)
        assert stream.line() == expected

        # Edits queued during a Hyprland disconnect apply on reconnect.
        server.disconnect()
        expected = save("reconnected", atomic=True)
        assert stream.line() == expected
        stream.silent()
        initial = memory(stream.process.pid)
        for index in range(100):
            expected = save(f"reload-{index}", atomic=index % 2 == 0)
            assert stream.line() == expected
        final = memory(stream.process.pid)
        print("PASS: idle rule reloads, in-place and atomic saves, unchanged output, invalid TOML/regex/captures, deletion/recreation, reconnect, and 100 reloads")
        print(f"Zig RSS growth over 100 rule reloads: {final['Rss'] - initial['Rss']} KiB")


def mock_check():
    with tempfile.TemporaryDirectory(prefix="nwork-", dir="/tmp") as temp:
        directory = Path(temp)
        server = MockHyprland(directory / "hypr/test")
        try:
            # The original daemon invokes hyprctl. Send those queries to the same
            # mock command socket used by Zig, preserving the original code path.
            hyprctl = directory / "hyprctl"
            hyprctl.write_text(f"""#!{sys.executable}
import os, socket, sys
s = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
s.connect(os.environ['XDG_RUNTIME_DIR'] + '/hypr/test/.socket.sock')
s.sendall(('j/' + sys.argv[2]).encode())
while data := s.recv(65536):
    sys.stdout.buffer.write(data)
""")
            hyprctl.chmod(0o755)
            env = dict(os.environ, XDG_RUNTIME_DIR=temp, HYPRLAND_INSTANCE_SIGNATURE="test", PATH=f"{temp}:{os.environ['PATH']}")
            commands = [[sys.executable, str(SOURCE)], [str(BINARY), "--rules", str(RULES)]]
            cases = [{"monitors": [], "workspaces": []}, fixture()]
            for title in [
                "File — Code - OSS", "File - Visual Studio Code", "File - Slack | Workspace",
                "File - Discord", "File - Cursor", "File - Chromium", "File - Free Version",
                "plain title", 'quotes " slash \\ tab\t雪🦊', " — Zen Browser",
                "first\r\nsecond\n — Zen Browser", "one\v\ftwo\x1c\x1d\x1e\x85\u2028\u2029",
                "last\n", "\n", "\n\n", "x" * 50000 + " — Zen Browser",
            ]:
                cases.append(fixture(title))
            for state in cases:
                server.set(state)
                assert once(commands[0], env) == once(commands[1], env), state
            # Optional groups, $0, multi-digit captures, first rule wins, comments.
            custom_source = directory / "workspaces.py"
            custom_source.write_bytes(SOURCE.read_bytes())
            custom_rules = directory / "window-rewrites.toml"
            custom_rules.write_text("[rewrite]\n'(a)?(b)(c)(d)(e)(f)(g)(h)(i)(j)' = '$10/$1/$0/$$2' # comment\n'(.*)' = 'fallback $1'\n")
            server.set(fixture("bcdefghij"))
            assert once([sys.executable, str(custom_source)], env) == once([str(BINARY), "--rules", str(custom_rules)], env)
            print(f"PASS: {len(cases) + 1} snapshot and rewrite comparisons")
            server.set(fixture())
            with streams(commands, env) as running:
                expected = once(commands[0], env)
                assert all(stream.line() == expected for stream in running)
                for event in EVENTS:
                    server.set(fixture(event + " — Zen Browser"))
                    server.broadcast((event[:3]).encode())
                    server.broadcast((event[3:] + ">>test\n").encode())
                    first, second = [stream.line() for stream in running]
                    assert first == second and first["titles"]["DP-1"]["text"] == " " + event
                server.broadcast(b"windowtitlev2>>same\nworkspacev2>>same\n")
                for stream in running:
                    stream.silent()
                server.set(fixture("ignored change"))
                server.broadcast(b"unrelated>>ignored\n")
                for stream in running:
                    stream.silent()
                server.broadcast(b"workspacev2>>refresh\n")
                assert running[0].line() == running[1].line()
                server.disconnect()
                time.sleep(1.2)
                # A clean reconnect suppresses an unchanged snapshot.
                for stream in running:
                    stream.silent()
                server.malformed = True
                server.broadcast(b"workspacev2>>bad\n")
                assert all(stream.line() == EMPTY for stream in running)
                server.malformed = False
                server.set(fixture("recovered — Zen Browser"))
                assert running[0].line() == running[1].line()
                print("PASS: all event types, partial lines, deduplication, ignored events, reconnect, malformed JSON recovery")
                server.set(fixture("steady — Zen Browser"))
                server.broadcast(b"workspacev2>>steady\n")
                assert running[0].line() == running[1].line()
                initial = memory(running[1].process.pid)
                for index in range(200):
                    server.set(fixture(f"stress {index} — Zen Browser"))
                    server.broadcast(b"windowtitlev2>>stress\n")
                    assert running[0].line() == running[1].line()
                print("PASS: 200 changing snapshots")
                print("Mock daemon memory, median of 7 samples, excludes short-lived query children:")
                final = measure(running)
                print(f"Zig RSS growth over 200 updates: {final[1]['Rss'] - initial['Rss']} KiB")
            reload_check(server, directory, env)
            server.delay = 3.3
            for command in commands:
                result = subprocess.run(command + ["--once"], env=env, capture_output=True, timeout=6)
                assert result.returncode != 0 and not result.stdout, "Query timeout must fail --once"
            print("PASS: stalled queries time out")
        finally:
            server.close()


def live_check():
    env = dict(os.environ)
    commands = [[sys.executable, str(SOURCE)], [str(BINARY), "--rules", str(RULES)]]
    for _ in range(5):
        python_value, zig_value = [once(command, env) for command in commands]
        if python_value == zig_value:
            break
    else:
        raise AssertionError("Live snapshots differ; retry with a stable desktop")
    print(f"PASS: live Hyprland output matches on {len(zig_value['active'])} monitors")
    with streams(commands, env) as running:
        for stream in running:
            assert stream.line() != EMPTY
        print("Live daemon memory, median of 7 samples, excludes short-lived hyprctl children:")
        measure(running)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--live", action="store_true", help="compare against the running Hyprland session")
    args = parser.parse_args()
    if not BINARY.exists():
        parser.error("build first: zig build -Doptimize=ReleaseSmall")
    live_check() if args.live else mock_check()
