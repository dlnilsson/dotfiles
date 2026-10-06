local pending = false
local refreshPending = false
local togglePending = false
local nextPoll = 0

local function unavailable()
    noctalia.state.set("powermode", {
        text = "?",
        tooltip = "power_saver: status unavailable",
        class = "unavailable",
    })
end

local function poll()
    if pending then
        refreshPending = true
        return
    end
    pending = true
    local launched = noctalia.runAsync({
        noctalia.expandPath("~/.dotfiles/bin/power_saver"),
        "--status",
    }, function(result)
        pending = false
        if result.exitCode ~= 0 or result.timedOut or result.stdoutTruncated then
            unavailable()
        else
            local data = noctalia.json.decode(result.stdout)
            if type(data) ~= "table" or type(data.text) ~= "string"
                or type(data.tooltip) ~= "string" or type(data.class) ~= "string" then
                unavailable()
            else
                noctalia.state.set("powermode", data)
            end
        end
        if refreshPending then
            refreshPending = false
            poll()
        end
    end)
    if not launched then
        pending = false
        refreshPending = false
        unavailable()
    end
end

noctalia.state.watch("powermode-toggle", function()
    if togglePending then
        return
    end
    togglePending = true
    local launched = noctalia.runAsync({
        noctalia.expandPath("~/.dotfiles/bin/power_saver"),
        "--toggle",
    }, function(result)
        togglePending = false
        if result.exitCode ~= 0 or result.timedOut then
            unavailable()
            return
        end
        poll()
    end)
    if not launched then
        togglePending = false
        unavailable()
    end
end)

noctalia.setUpdateInterval(1000)

function update()
    local now = os.time()
    if now >= nextPoll and not pending then
        nextPoll = now + 120
        poll()
    end
end
