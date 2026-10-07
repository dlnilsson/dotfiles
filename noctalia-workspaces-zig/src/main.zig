const std = @import("std");
const c = @cImport({
    @cDefine("PCRE2_CODE_UNIT_WIDTH", "8");
    @cInclude("pcre2.h");
    @cInclude("sys/socket.h");
    @cInclude("sys/un.h");
    @cInclude("sys/inotify.h");
    @cInclude("poll.h");
    @cInclude("unistd.h");
    @cInclude("fcntl.h");
    @cInclude("errno.h");
    @cInclude("time.h");
});
const Allocator = std.mem.Allocator;
const heap = std.heap.page_allocator;
const Rule = struct {
    code: *c.pcre2_code_8,
    match: *c.pcre2_match_data_8,
    replacement: []const u8,
};
const RuleSet = struct {
    arena: std.heap.ArenaAllocator,
    items: []const Rule,

    fn load(path: [:0]const u8) !RuleSet {
        var arena = std.heap.ArenaAllocator.init(heap);
        errdefer arena.deinit();
        const items = try loadRules(arena.allocator(), path);
        return .{ .arena = arena, .items = items };
    }

    fn deinit(self: *RuleSet) void {
        for (self.items) |rule| {
            c.pcre2_match_data_free_8(rule.match);
            c.pcre2_code_free_8(rule.code);
        }
        self.arena.deinit();
    }
};

const RuleWatcher = struct {
    fd: c_int,
    filename: []const u8,

    fn init(alloc: Allocator, path: []const u8) !RuleWatcher {
        const directory = try alloc.dupeZ(u8, std.fs.path.dirname(path) orelse ".");
        const fd = c.inotify_init1(c.IN_NONBLOCK | c.IN_CLOEXEC);
        if (fd < 0) return error.RulesWatchFailed;
        errdefer _ = c.close(fd);
        // Watch the directory so editor saves that replace the file are visible.
        const mask = c.IN_CLOSE_WRITE | c.IN_MOVED_TO | c.IN_MOVED_FROM | c.IN_CREATE | c.IN_DELETE | c.IN_ATTRIB;
        if (c.inotify_add_watch(fd, directory.ptr, mask) < 0) return error.RulesWatchFailed;
        return .{ .fd = fd, .filename = std.fs.path.basename(path) };
    }

    fn changed(self: RuleWatcher) !bool {
        var changed_file = false;
        var buffer: [16384]u8 align(@alignOf(c.inotify_event)) = undefined;
        while (true) {
            const n = c.read(self.fd, &buffer, buffer.len);
            if (n < 0) {
                if (c.__errno_location().* == c.EINTR) continue;
                if (c.__errno_location().* == c.EAGAIN) return changed_file;
                return error.RulesWatchFailed;
            }
            if (n == 0) return error.RulesWatchFailed;
            var offset: usize = 0;
            while (offset < @as(usize, @intCast(n))) {
                const event: *const c.inotify_event = @ptrCast(@alignCast(buffer[offset..].ptr));
                const start = offset + @sizeOf(c.inotify_event);
                const name = std.mem.sliceTo(buffer[start .. start + event.len], 0);
                if (event.mask & c.IN_Q_OVERFLOW != 0 or std.mem.eql(u8, name, self.filename)) changed_file = true;
                offset = start + event.len;
            }
        }
    }
};
const Monitor = struct {
    name: []const u8,
    activeWorkspace: struct { id: i64 },
    specialWorkspace: struct { id: i64 = 0 } = .{},
};
const Workspace = struct {
    id: i64,
    windows: i64 = 0,
    lastwindowtitle: []const u8 = "",
};
const empty = "{\"active\":{},\"occupied\":[],\"titles\":{}}";
const events = [_][]const u8{
    "workspacev2",    "focusedmonv2",   "openwindow",     "closewindow",
    "movewindowv2",   "monitoraddedv2", "monitorremoved", "moveworkspacev2",
    "activewindowv2", "windowtitlev2",  "activespecial",
};

pub fn main(init: std.process.Init.Minimal) void {
    run(init) catch |err| {
        report("workspace status", err);
        std.process.exit(1);
    };
}

fn run(init: std.process.Init.Minimal) !void {
    // stdout is a stream owned by Noctalia; a closed reader should end the process.
    var startup = std.heap.ArenaAllocator.init(heap);
    defer startup.deinit();
    const alloc = startup.allocator();
    var args = init.args.iterate();
    _ = args.next();
    var once = false;
    var rules_path: ?[:0]const u8 = null;
    while (args.next()) |arg| {
        if (std.mem.eql(u8, arg, "--once")) {
            once = true;
        } else if (std.mem.eql(u8, arg, "--rules")) {
            rules_path = args.next() orelse return error.MissingRulesPath;
        } else if (std.mem.eql(u8, arg, "--help")) {
            try writeAll(1, "Usage: noctalia-workspaces [--once] [--rules PATH]\n");
            return;
        } else return error.UnknownArgument;
    }
    const home = init.environ.getPosix("HOME") orelse return error.MissingHome;
    const path = rules_path orelse try std.fmt.allocPrintSentinel(alloc, "{s}/.dotfiles/config/noctalia/plugins/status/window-rewrites.toml", .{home}, 0);
    const watcher = if (once) null else try RuleWatcher.init(alloc, path);
    defer if (watcher) |watch| {
        _ = c.close(watch.fd);
    };
    var rules = try RuleSet.load(path);
    defer rules.deinit();
    const runtime = init.environ.getPosix("XDG_RUNTIME_DIR") orelse return error.MissingRuntimeDir;
    const signature = init.environ.getPosix("HYPRLAND_INSTANCE_SIGNATURE") orelse return error.MissingHyprlandSignature;
    const directory = try std.fmt.allocPrint(alloc, "{s}/hypr/{s}", .{ runtime, signature });
    const command_path = try std.fmt.allocPrint(alloc, "{s}/.socket.sock", .{directory});
    const event_path = try std.fmt.allocPrint(alloc, "{s}/.socket2.sock", .{directory});
    if (once) {
        var arena = std.heap.ArenaAllocator.init(heap);
        defer arena.deinit();
        try emit(try snapshot(arena.allocator(), command_path, rules.items));
        return;
    }
    var last: ?[]const u8 = null;
    defer if (last) |value| heap.free(value);
    while (true) {
        session(command_path, event_path, path, &rules, watcher.?, &last) catch |err| {
            report("workspace status", err);
            try emit(empty);
            if (last) |value| heap.free(value);
            last = null;
        };
        _ = c.sleep(1);
    }
}

fn report(context: []const u8, err: anyerror) void {
    var buffer: [256]u8 = undefined;
    const line = std.fmt.bufPrint(&buffer, "Hyprland {s}: {s}\n", .{ context, @errorName(err) }) catch return;
    writeAll(2, line) catch {};
}

fn writeAll(fd: c_int, bytes: []const u8) !void {
    var offset: usize = 0;
    while (offset < bytes.len) {
        const n = c.write(fd, bytes.ptr + offset, bytes.len - offset);
        if (n < 0) {
            if (c.__errno_location().* == c.EINTR) continue;
            return error.WriteFailed;
        }
        if (n == 0) return error.WriteFailed;
        offset += @intCast(n);
    }
}

fn emit(value: []const u8) !void {
    try writeAll(1, value);
    try writeAll(1, "\n");
}

fn readAll(alloc: Allocator, fd: c_int) ![]const u8 {
    var bytes: std.ArrayList(u8) = .empty;
    var buffer: [8192]u8 = undefined;
    while (true) {
        const n = c.read(fd, &buffer, buffer.len);
        if (n < 0) {
            if (c.__errno_location().* == c.EINTR) continue;
            return error.ReadFailed;
        }
        if (n == 0) break;
        if (bytes.items.len + @as(usize, @intCast(n)) > 16 * 1024 * 1024) return error.ResponseTooLarge;
        try bytes.appendSlice(alloc, buffer[0..@intCast(n)]);
    }
    return bytes.items;
}

// The existing configuration uses a [rewrite] table with single-line TOML
// literal strings. Reject other syntax rather than silently interpreting it.
fn literal(input: *[]const u8) ![]const u8 {
    input.* = std.mem.trimStart(u8, input.*, " \t");
    if (input.*.len == 0 or input.*[0] != '\'') return error.ExpectedTomlLiteralString;
    const end = std.mem.indexOfScalarPos(u8, input.*, 1, '\'') orelse return error.UnterminatedTomlString;
    const result = input.*[1..end];
    input.* = input.*[end + 1 ..];
    return result;
}

fn loadRules(alloc: Allocator, path: [:0]const u8) ![]const Rule {
    const fd = c.open(path.ptr, c.O_RDONLY | c.O_CLOEXEC);
    if (fd < 0) return error.OpenRulesFailed;
    defer _ = c.close(fd);
    const source = try readAll(alloc, fd);
    var rules: std.ArrayList(Rule) = .empty;
    errdefer for (rules.items) |rule| {
        c.pcre2_match_data_free_8(rule.match);
        c.pcre2_code_free_8(rule.code);
    };
    var in_table = false;
    var lines = std.mem.splitScalar(u8, source, '\n');
    while (lines.next()) |raw| {
        var line = std.mem.trim(u8, raw, " \t\r");
        if (line.len == 0 or line[0] == '#') continue;
        if (std.mem.eql(u8, line, "[rewrite]")) {
            if (in_table) return error.DuplicateRewriteTable;
            in_table = true;
            continue;
        }
        if (!in_table) return error.ExpectedRewriteTable;
        const pattern = try literal(&line);
        line = std.mem.trimStart(u8, line, " \t");
        if (line.len == 0 or line[0] != '=') return error.ExpectedEquals;
        line = line[1..];
        const replacement = try literal(&line);
        line = std.mem.trim(u8, line, " \t");
        if (line.len != 0 and line[0] != '#') return error.UnsupportedTomlSyntax;
        var code_error: c_int = 0;
        var error_offset: usize = 0;
        const code = c.pcre2_compile_8(pattern.ptr, pattern.len, c.PCRE2_UTF | c.PCRE2_UCP | c.PCRE2_ANCHORED | c.PCRE2_ENDANCHORED, &code_error, &error_offset, null) orelse return error.InvalidRegex;
        errdefer c.pcre2_code_free_8(code);
        const match = c.pcre2_match_data_create_from_pattern_8(code, null) orelse return error.OutOfMemory;
        errdefer c.pcre2_match_data_free_8(match);
        try validateReplacement(replacement, c.pcre2_get_ovector_count_8(match));
        try rules.append(alloc, .{ .code = code, .match = match, .replacement = replacement });
    }
    if (!in_table) return error.ExpectedRewriteTable;
    return rules.items;
}

fn validateReplacement(replacement: []const u8, count: usize) !void {
    var index: usize = 0;
    while (index < replacement.len) : (index += 1) {
        if (replacement[index] != '$' or index + 1 == replacement.len or !std.ascii.isDigit(replacement[index + 1])) continue;
        var end = index + 1;
        while (end < replacement.len and std.ascii.isDigit(replacement[end])) : (end += 1) {}
        const group = try std.fmt.parseInt(usize, replacement[index + 1 .. end], 10);
        if (group >= count) return error.InvalidCaptureGroup;
        index = end - 1;
    }
}

fn reloadRules(path: [:0]const u8, rules: *RuleSet) bool {
    const replacement = RuleSet.load(path) catch |err| {
        report("workspace rules reload", err);
        return false;
    };
    rules.deinit();
    rules.* = replacement;
    return true;
}

fn flatten(alloc: Allocator, title: []const u8) ![]const u8 {
    var bytes: std.ArrayList(u8) = .empty;
    var iterator = (try std.unicode.Utf8View.init(title)).iterator();
    while (iterator.nextCodepointSlice()) |slice| {
        const codepoint = try std.unicode.utf8Decode(slice);
        switch (codepoint) {
            '\r', '\n', 0x0b, 0x0c, 0x1c...0x1e, 0x85, 0x2028, 0x2029 => {
                if (codepoint == '\r' and iterator.i < title.len and title[iterator.i] == '\n') iterator.i += 1;
                // splitlines() drops a final line break.
                if (iterator.i < title.len) try bytes.append(alloc, ' ');
            },
            else => try bytes.appendSlice(alloc, slice),
        }
    }
    return bytes.items;
}

fn rewrite(alloc: Allocator, title: []const u8, rules: []const Rule) ![]const u8 {
    const text = try flatten(alloc, title);
    for (rules) |rule| {
        const n = c.pcre2_match_8(rule.code, text.ptr, text.len, 0, 0, rule.match, null);
        if (n == c.PCRE2_ERROR_NOMATCH) continue;
        if (n < 0) return error.RegexMatchFailed;
        const groups = c.pcre2_get_ovector_pointer_8(rule.match);
        const count = c.pcre2_get_ovector_count_8(rule.match);
        var result: std.ArrayList(u8) = .empty;
        var index: usize = 0;
        while (index < rule.replacement.len) {
            if (rule.replacement[index] == '$' and index + 1 < rule.replacement.len and std.ascii.isDigit(rule.replacement[index + 1])) {
                var end = index + 1;
                while (end < rule.replacement.len and std.ascii.isDigit(rule.replacement[end])) : (end += 1) {}
                const group = try std.fmt.parseInt(usize, rule.replacement[index + 1 .. end], 10);
                if (group >= count) return error.InvalidCaptureGroup;
                // PCRE2_UNSET is SIZE_MAX; Zig 0.16 cannot translate its C macro.
                if (groups[group * 2] != std.math.maxInt(usize)) try result.appendSlice(alloc, text[groups[group * 2]..groups[group * 2 + 1]]);
                index = end;
            } else {
                try result.append(alloc, rule.replacement[index]);
                index += 1;
            }
        }
        return result.items;
    }
    return text;
}

fn milliseconds() !i64 {
    var now: c.timespec = undefined;
    if (c.clock_gettime(c.CLOCK_MONOTONIC, &now) != 0) return error.ClockFailed;
    return now.tv_sec * 1000 + @divTrunc(now.tv_nsec, 1000000);
}

fn waitReady(fd: c_int, wanted: c_short, deadline: i64) !void {
    while (true) {
        const remaining = deadline - try milliseconds();
        if (remaining <= 0) return error.QueryTimeout;
        var pollfd = c.pollfd{ .fd = fd, .events = wanted, .revents = 0 };
        const n = c.poll(&pollfd, 1, @intCast(remaining));
        if (n < 0) {
            if (c.__errno_location().* == c.EINTR) continue;
            return error.PollFailed;
        }
        if (n == 0) return error.QueryTimeout;
        return;
    }
}

fn connect(path: []const u8, deadline: i64) !c_int {
    var address: c.sockaddr_un = std.mem.zeroes(c.sockaddr_un);
    address.sun_family = c.AF_UNIX;
    if (path.len >= address.sun_path.len) return error.SocketPathTooLong;
    @memcpy(@as([*]u8, @ptrCast(&address.sun_path))[0..path.len], path);
    const fd = c.socket(c.AF_UNIX, c.SOCK_STREAM | c.SOCK_CLOEXEC | c.SOCK_NONBLOCK, 0);
    if (fd < 0) return error.SocketFailed;
    errdefer _ = c.close(fd);
    if (c.connect(fd, @ptrCast(&address), @sizeOf(c.sockaddr_un)) != 0) {
        const errno = c.__errno_location().*;
        if (errno != c.EINPROGRESS and errno != c.EAGAIN) return error.ConnectFailed;
        try waitReady(fd, c.POLLOUT, deadline);
        var socket_error: c_int = 0;
        var length: c.socklen_t = @sizeOf(c_int);
        if (c.getsockopt(fd, c.SOL_SOCKET, c.SO_ERROR, &socket_error, &length) != 0 or socket_error != 0) return error.ConnectFailed;
    }
    return fd;
}

fn query(alloc: Allocator, path: []const u8, request: []const u8) ![]const u8 {
    const deadline = try milliseconds() + 3000;
    const fd = try connect(path, deadline);
    defer _ = c.close(fd);
    var offset: usize = 0;
    while (offset < request.len) {
        try waitReady(fd, c.POLLOUT, deadline);
        const n = c.send(fd, request.ptr + offset, request.len - offset, c.MSG_NOSIGNAL);
        if (n < 0) {
            if (c.__errno_location().* == c.EAGAIN or c.__errno_location().* == c.EINTR) continue;
            return error.WriteFailed;
        }
        if (n == 0) return error.WriteFailed;
        offset += @intCast(n);
    }
    var bytes: std.ArrayList(u8) = .empty;
    var buffer: [8192]u8 = undefined;
    while (true) {
        try waitReady(fd, c.POLLIN, deadline);
        const n = c.read(fd, &buffer, buffer.len);
        if (n < 0) {
            if (c.__errno_location().* == c.EAGAIN or c.__errno_location().* == c.EINTR) continue;
            return error.ReadFailed;
        }
        if (n == 0) return bytes.items;
        if (bytes.items.len + @as(usize, @intCast(n)) > 16 * 1024 * 1024) return error.ResponseTooLarge;
        try bytes.appendSlice(alloc, buffer[0..@intCast(n)]);
    }
}

fn snapshot(alloc: Allocator, path: []const u8, rules: []const Rule) ![]const u8 {
    const monitors = try std.json.parseFromSlice([]Monitor, alloc, try query(alloc, path, "j/monitors"), .{ .ignore_unknown_fields = true });
    const workspaces = try std.json.parseFromSlice([]Workspace, alloc, try query(alloc, path, "j/workspaces"), .{ .ignore_unknown_fields = true });
    var active: std.json.ObjectMap = .{};
    var titles: std.json.ObjectMap = .{};
    var occupied: std.ArrayList(i64) = .empty;
    var by_id: std.AutoHashMap(i64, Workspace) = .init(alloc);
    for (workspaces.value) |workspace| {
        try by_id.put(workspace.id, workspace);
        if (workspace.windows > 0) try occupied.append(alloc, workspace.id);
    }
    std.mem.sort(i64, occupied.items, {}, std.sort.asc(i64));
    for (monitors.value) |monitor| {
        try active.put(alloc, monitor.name, .{ .integer = monitor.activeWorkspace.id });
        const id = if (monitor.specialWorkspace.id != 0) monitor.specialWorkspace.id else monitor.activeWorkspace.id;
        const workspace = by_id.get(id);
        const title = if (workspace) |w| (if (w.windows > 0) w.lastwindowtitle else "") else "";
        var entry: std.json.ObjectMap = .{};
        try entry.put(alloc, "text", .{ .string = try rewrite(alloc, title, rules) });
        try entry.put(alloc, "tooltip", .{ .string = title });
        try titles.put(alloc, monitor.name, .{ .object = entry });
    }
    return std.json.Stringify.valueAlloc(alloc, .{ .active = std.json.Value{ .object = active }, .occupied = occupied.items, .titles = std.json.Value{ .object = titles } }, .{});
}

fn publish(path: []const u8, rules: []const Rule, last: *?[]const u8) !void {
    var arena = std.heap.ArenaAllocator.init(heap);
    defer arena.deinit();
    const value = try snapshot(arena.allocator(), path, rules);
    if (last.*) |previous| {
        if (std.mem.eql(u8, previous, value)) return;
    }
    const copy = try heap.dupe(u8, value);
    errdefer heap.free(copy);
    try emit(value);
    if (last.*) |previous| heap.free(previous);
    last.* = copy;
}

fn session(command_path: []const u8, event_path: []const u8, rules_path: [:0]const u8, rules: *RuleSet, watcher: RuleWatcher, last: *?[]const u8) !void {
    const fd = try connect(event_path, try milliseconds() + 3000);
    defer _ = c.close(fd);
    if (try watcher.changed()) _ = reloadRules(rules_path, rules);
    try publish(command_path, rules.items, last);
    var pending: std.ArrayList(u8) = .empty;
    defer pending.deinit(heap);
    var buffer: [8192]u8 = undefined;
    var reload_at: ?i64 = null;
    while (true) {
        var pollfds = [_]c.pollfd{
            .{ .fd = fd, .events = c.POLLIN, .revents = 0 },
            .{ .fd = watcher.fd, .events = c.POLLIN, .revents = 0 },
        };
        const timeout: c_int = if (reload_at) |deadline| @intCast(@max(0, deadline - try milliseconds())) else -1;
        if (c.poll(&pollfds, pollfds.len, timeout) < 0) {
            if (c.__errno_location().* == c.EINTR) continue;
            return error.PollFailed;
        }
        if (pollfds[1].revents != 0 and try watcher.changed()) reload_at = try milliseconds() + 50;
        if (reload_at) |deadline| {
            if (try milliseconds() >= deadline) {
                reload_at = null;
                if (reloadRules(rules_path, rules)) try publish(command_path, rules.items, last);
            }
        }
        if (pollfds[0].revents == 0) continue;
        const n = c.read(fd, &buffer, buffer.len);
        if (n < 0) {
            if (c.__errno_location().* == c.EAGAIN or c.__errno_location().* == c.EINTR) continue;
            return error.ReadFailed;
        }
        if (n == 0) return;
        try pending.appendSlice(heap, buffer[0..@intCast(n)]);
        var start: usize = 0;
        while (std.mem.indexOfScalarPos(u8, pending.items, start, '\n')) |end| {
            const line = pending.items[start..end];
            const name = line[0 .. std.mem.indexOf(u8, line, ">>") orelse line.len];
            for (events) |event| {
                if (std.mem.eql(u8, name, event)) {
                    try publish(command_path, rules.items, last);
                    break;
                }
            }
            start = end + 1;
        }
        const remaining = pending.items.len - start;
        std.mem.copyForwards(u8, pending.items[0..remaining], pending.items[start..]);
        pending.shrinkRetainingCapacity(remaining);
        if (remaining > 1024 * 1024) return error.EventTooLarge;
    }
}
