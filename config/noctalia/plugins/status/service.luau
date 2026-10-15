local sources = {
    { id = "ef-usage", script = "waybar-ef-usage", interval = 60 },
    { id = "codex", script = "codex-status", interval = 10 },
    { id = "claude", script = "claude-status", interval = 10 },
    { id = "vpn", script = "vpn-status", interval = 15 },
    { id = "glab", script = "glab-my-reviews", interval = 120 },
}
local pending = {}
local lastPoll = {}
-- bin/power_saver writes this file. Power saver mode stretches every interval.
local powerSaverState = (noctalia.getenv("XDG_RUNTIME_DIR") or "/tmp") .. "/power_saver_state"
local powerSaverFactor = 6

noctalia.setUpdateInterval(1000)

local function powerSaving()
    local mode = noctalia.readFile(powerSaverState)
    return mode ~= nil and mode:match("^powersave") ~= nil
end

local function unavailable(source)
    noctalia.state.set(source.id, {
        text = (source.id == "ef-usage" or source.id == "vpn") and "?" or "",
        tooltip = source.script .. ": status unavailable",
        class = "unavailable",
    })
end

local function poll(source)
    if pending[source.id] then
        return
    end
    pending[source.id] = true
    local launched = noctalia.runAsync({
        noctalia.expandPath("~/.dotfiles/bin/" .. source.script),
    }, function(result)
        pending[source.id] = false
        if result.exitCode ~= 0 or result.timedOut or result.stdoutTruncated then
            unavailable(source)
            return
        end

        local data = noctalia.json.decode(result.stdout)
        if type(data) ~= "table" or type(data.text) ~= "string"
            or (data.tooltip ~= nil and type(data.tooltip) ~= "string")
            or (data.class ~= nil and type(data.class) ~= "string") then
            unavailable(source)
            return
        end
        noctalia.state.set(source.id, {
            text = data.text,
            tooltip = data.tooltip or "",
            class = data.class or "",
        })
    end)
    if not launched then
        pending[source.id] = false
        unavailable(source)
    end
end

function update()
    local now = os.time()
    local factor = powerSaving() and powerSaverFactor or 1
    for _, source in ipairs(sources) do
        local last = lastPoll[source.id]
        if (last == nil or now >= last + source.interval * factor) and not pending[source.id] then
            lastPoll[source.id] = now
            poll(source)
        end
    end
end
