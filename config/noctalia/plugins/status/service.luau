local sources = {
    { id = "ef-usage", script = "waybar-ef-usage", interval = 60 },
    { id = "codex", script = "codex-status", interval = 10 },
    { id = "claude", script = "claude-status", interval = 10 },
    { id = "vpn", script = "vpn-status", interval = 15 },
    { id = "glab", script = "glab-my-reviews", interval = 120 },
}
local pending = {}
local nextPoll = {}

noctalia.setUpdateInterval(1000)

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
    for _, source in ipairs(sources) do
        if now >= (nextPoll[source.id] or 0) and not pending[source.id] then
            nextPoll[source.id] = now + source.interval
            poll(source)
        end
    end
end
