-- Run from any directory: lua config/noctalia/plugins/status/test.lua
local root = arg[0]:match("^(.*)/[^/]+$") or "."
local plugin = root .. "/"
local now, calls, streamCallback, streams = 0, {}, nil, 0
local recordingCallback, recordingStreams = nil, 0
local state, watchers = {}, {}
local decoded = {
    usage = { text = "18.8M", tooltip = "OpenAI: 18.8M" },
    waiting = { text = "Codex: 1", tooltip = "Action required", class = "action-required" },
    idle = { text = "", tooltip = "No action required", class = "idle" },
    malformed = { text = 123 },
    tailscale = { text = "tailscale", tooltip = "Tailscale: test tailnet", class = "tailscale" },
    netbird = { text = "netbird", tooltip = "NetBird connected", class = "netbird" },
    disconnected = { text = "off", tooltip = "No VPN active", class = "disconnected" },
    reviews = { text = "\u{f296} 2", tooltip = "group/project!1  Fix", class = "pending" },
    noreviews = { text = "", tooltip = "No open MRs awaiting review", class = "none" },
}
local host = {
    setUpdateInterval = function(ms) assert(ms == 1000 or ms == 500) end,
    expandPath = function(path) return path end,
    runAsync = function(argv, callback)
        table.insert(calls, { argv = argv, callback = callback })
        return true
    end,
    runStream = function(command, callback)
        if command:find("hypr-screencapture status -subscribe", 1, true) then
            assert(command:find("^while :; do"), "the subscription must restart when it exits")
            recordingStreams = recordingStreams + 1
            recordingCallback = callback
            return true
        end
        assert(command == "~/.dotfiles/noctalia-workspaces-zig/zig-out/bin/noctalia-workspaces")
        streams = streams + 1
        streamCallback = callback
        return true
    end,
    json = { decode = function(raw) return decoded[raw] end },
    state = {
        get = function(key) return state[key] end,
        set = function(key, value)
            state[key] = value
            for _, callback in ipairs(watchers[key] or {}) do callback(value) end
        end,
        watch = function(key, callback)
            watchers[key] = watchers[key] or {}
            table.insert(watchers[key], callback)
        end,
    },
}

local function loadEntry(name, widget)
    local env = setmetatable({
        noctalia = host,
        os = { time = function() return now end },
        barWidget = widget,
        ui = {},
    }, { __index = _G })
    for _, kind in ipairs({ "row", "column", "box", "label", "glyph" }) do
        env.ui[kind] = function(props, children)
            return { kind = kind, props = props, children = children }
        end
    end
    env.require = function(path)
        assert(path:match("^%./.*%.luau$"), "Noctalia requires an explicit relative module path")
        return assert(loadfile(plugin .. path, "t", env))()
    end
    assert(loadfile(plugin .. name, "t", env))()
    return env
end

local function widget(name, output)
    local view = {}
    local initializing = true
    local env = loadEntry(name .. ".luau", {
        setVisible = function(value) view.visible = value end,
        render = function(tree) view.tree = tree end,
        setTooltip = function(value) view.tooltip = value end,
        outputName = function() return initializing and (output or "DP-2") or nil end,
    })
    initializing = false
    view.update = env.update
    return view
end

local function complete(index, raw, code)
    calls[index].callback({ exitCode = code or 0, stdout = raw })
end

local service = loadEntry("service.luau")
local usage, codex, claude = widget("ef-usage"), widget("codex"), widget("claude")
local vpn, glab = widget("vpn"), widget("glab")
local secondMonitor = widget("codex")
assert(not usage.visible and not codex.visible and not claude.visible and not glab.visible)

service.update()
assert(#calls == 5)
assert(calls[1].argv[1] == "~/.dotfiles/bin/waybar-ef-usage")
assert(calls[2].argv[1] == "~/.dotfiles/bin/codex-status")
assert(calls[3].argv[1] == "~/.dotfiles/bin/claude-status")
assert(calls[4].argv[1] == "~/.dotfiles/bin/vpn-status")
assert(calls[5].argv[1] == "~/.dotfiles/bin/glab-my-reviews")
now = 10
service.update()
assert(#calls == 5, "polls must not overlap while a script is running")
complete(1, "usage")
complete(2, "waiting")
complete(3, "idle")
complete(4, "tailscale")
complete(5, "reviews")
assert(glab.visible and glab.tree.children[1].props.text == "\u{f296} 2")
assert(glab.tree.children[1].props.color == "#d08770")
assert(glab.tooltip == "group/project!1  Fix")
glab.tree.props.onClick()
assert(calls[#calls].argv[1] == "xdg-open")
table.remove(calls)
assert(usage.visible and usage.tree.children[2].props.text == "18.8M")
assert(usage.tooltip == "OpenAI: 18.8M")
assert(codex.visible and secondMonitor.visible)
assert(codex.tree.props.fill == "#ebcb8b")
assert(codex.tree.children[1].props.color == "#2e3440")
assert(codex.tree.children[1].props.fontWeight == "bold")
assert(not claude.visible)
assert(vpn.visible and vpn.tree.children[1].props.text == "tailscale")
assert(vpn.tree.children[1].props.color == "#88c0d0")
assert(vpn.tooltip == "Tailscale: test tailnet")

now = 20
service.update()
assert(#calls == 8, "agent and VPN polling must not rerun the 60-second usage poll")
complete(6, "idle")
complete(7, "waiting")
complete(8, "netbird")
assert(not codex.visible and not secondMonitor.visible)
assert(claude.visible and claude.tree.props.fill == "#ebcb8b")
assert(vpn.tree.children[1].props.color == "#a3be8c")

now = 30
service.update()
assert(#calls == 10, "VPN must wait for its 15-second interval")
complete(9, "idle")
complete(10, "idle")
now = 35
service.update()
assert(#calls == 11 and calls[11].argv[1] == "~/.dotfiles/bin/vpn-status")
complete(11, "disconnected")
assert(vpn.visible and vpn.tree.children[1].props.text == "off")
assert(vpn.tree.children[1].props.color == "#4c566a")

now = 60
service.update()
assert(#calls == 15)
complete(12, "malformed")
complete(13, "invalid JSON")
complete(14, "", 1)
complete(15, "", 1)
assert(usage.visible and usage.tree.children[2].props.text == "?")
assert(usage.tooltip:find("unavailable"))
assert(not codex.visible and not claude.visible)
assert(vpn.visible and vpn.tree.children[1].props.text == "?")

now = 120
service.update()
complete(16, "usage")
complete(17, "waiting")
complete(18, "idle")
complete(19, "tailscale")
assert(calls[20].argv[1] == "~/.dotfiles/bin/glab-my-reviews", "GitLab must wait for its 120-second interval")
complete(20, "noreviews")
assert(usage.tree.children[2].props.text == "18.8M" and codex.visible)
assert(not glab.visible, "GitLab must hide when there are no reviews")
print("PASS: shared polls, intervals, agent styling, VPN states, GitLab reviews, invalid output, and recovery")

local feed = loadEntry("workspace-service.luau")
local workspaces, laptop = widget("workspaces"), widget("workspaces", "eDP-1")
local window, laptopWindow = widget("active-window"), widget("active-window", "eDP-1")
assert(not window.visible and not laptopWindow.visible)
feed.update()
feed.update()
assert(streams == 1, "there must be one event subscription for every monitor")
decoded.workspaces = {
    active = { ["DP-2"] = 6, ["eDP-1"] = 2 }, occupied = { 1, 2, 6 },
    titles = {
        ["DP-2"] = { text = " Example", tooltip = "Example — Zen Browser" },
        ["eDP-1"] = { text = " general", tooltip = "general - Slack" },
    },
}
streamCallback("workspaces")
assert(#workspaces.tree.children == 9 and #laptop.tree.children == 9)
for id, node in ipairs(workspaces.tree.children) do
    assert(node.children[2].props.text == tostring(id))
    local underline = node.children[3].props
    assert(underline.height == 3)
    assert(underline.fill == (id == 6 and "#d08770" or "#00000000"))
    assert(node.props.fill == nil, "workspace labels must not have a filled background")
end
assert(laptop.tree.children[2].children[3].props.fill == "#d08770")
    for id = 1, 9 do
    workspaces.tree.children[id].props.onClick()
    local argv = calls[#calls].argv
    assert(argv[1] == "hyprctl" and argv[2] == "dispatch")
    assert(argv[3] == 'hl.dsp.focus({ workspace = "' .. id .. '" })',
        "click must activate the chosen workspace using Hyprland's Lua API")
end
decoded.workspaces.active["DP-2"] = 9
streamCallback("workspaces")
assert(workspaces.tree.children[6].children[3].props.fill == "#00000000")
assert(workspaces.tree.children[9].children[3].props.fill == "#d08770")
print("PASS: all nine labels, monitor-local active underline, event updates, and workspace clicks")
assert(window.visible and window.tree.props.text == " Example")
assert(window.tree.props.maxWidth == 400 and window.tree.props.maxLines == 1)
assert(window.tooltip == "Example — Zen Browser")
assert(laptopWindow.visible and laptopWindow.tree.props.text == " general")
decoded.workspaces.titles["DP-2"] = { text = "New title", tooltip = "New title" }
streamCallback("workspaces")
assert(window.tree.props.text == "New title")
assert(laptopWindow.tree.props.text == " general")
decoded.workspaces.titles = {}
streamCallback("workspaces")
assert(not window.visible and window.tooltip == "" and not laptopWindow.visible)
print("PASS: monitor-local window titles, original tooltips, title changes, and empty workspaces")

local recordingFeed = loadEntry("recording-service.luau")
local recording, laptopRecording = widget("recording"), widget("recording", "eDP-1")
assert(not recording.visible and not laptopRecording.visible)
recordingFeed.update()
recordingFeed.update()
assert(recordingStreams == 1, "there must be one recording subscription for every monitor")
recordingCallback("●")
assert(recording.visible and laptopRecording.visible)
assert(recording.tree.children[1].props.text == "●")
assert(recording.tree.children[1].props.color == "#ff0000")
recording.update()
assert(recording.tree.children[1].props.color == "#00000000", "the dot must blink off")
recording.update()
assert(recording.tree.children[1].props.color == "#ff0000", "the dot must blink on")
recording.tree.props.onClick()
assert(calls[#calls].argv[1] == "~/.dotfiles/bin/screen-recorder")
recordingCallback("")
assert(not recording.visible and not laptopRecording.visible)
print("PASS: shared recording subscription, blinking dot, click action, and hide when off")
