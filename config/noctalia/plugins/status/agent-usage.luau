local agents = {
    { id = "claude", name = "Claude" },
    { id = "codex", name = "Codex" },
}
local generation = 0
local open = false

local function unavailable()
    return { "Limits unavailable" }
end

local function utcOffset(timestamp)
    local localTime = os.date("*t", timestamp)
    local utcTime = os.date("!*t", timestamp)
    return os.difftime(os.time(localTime), os.time(utcTime))
end

local function parseTimestamp(value)
    if type(value) ~= "string" or value == "" then
        return nil
    end

    local year, month, day, hour, minute, second, zone = value:match(
        "^(%d%d%d%d)%-(%d%d)%-(%d%d)T(%d%d):(%d%d):(%d%d)[%d%.]*(.*)$"
    )
    if not year then
        return nil
    end

    local timestamp = os.time({
        year = tonumber(year),
        month = tonumber(month),
        day = tonumber(day),
        hour = tonumber(hour),
        min = tonumber(minute),
        sec = tonumber(second),
    })
    if not timestamp then
        return nil
    end
    if zone == "" then
        return timestamp
    end

    local offset = 0
    if zone ~= "Z" and zone ~= "z" then
        local sign, zoneHour, zoneMinute = zone:match("^([%+%-])(%d%d):?(%d%d)$")
        if not sign then
            return nil
        end
        offset = (tonumber(zoneHour) * 60 + tonumber(zoneMinute)) * 60
        if sign == "-" then
            offset = -offset
        end
    end
    return timestamp + utcOffset(timestamp) - offset
end

local function validPercent(value)
    return type(value) == "number"
        and value == value
        and value ~= math.huge
        and value ~= -math.huge
end

local function recordLines(record)
    if type(record) ~= "table" then
        return nil
    end

    local limits = record.limits
    if limits ~= nil and type(limits) ~= "table" then
        return nil
    end

    local lines = {}
    for _, limit in ipairs(limits or {}) do
        if type(limit) ~= "table"
            or type(limit.label) ~= "string"
            or limit.label == ""
            or not validPercent(limit.percent) then
            return nil
        end

        local line = string.format("%s: %.0f%% used", limit.label, limit.percent * 100)
        local reset = limit.resetsAt
        if reset ~= nil and reset ~= "" then
            local timestamp = parseTimestamp(reset)
            if not timestamp then
                return nil
            end
            line = line .. " · resets " .. noctalia.formatTime("%a %d %b, %H:%M %Z", timestamp)
        end
        table.insert(lines, line)
    end

    local status = record.usageStatusText
    if status ~= nil and type(status) ~= "string" then
        return nil
    end
    if status and status ~= "" then
        table.insert(lines, status)
    end
    return #lines > 0 and lines or unavailable()
end

local function render()
    if not open then
        return
    end

    local sections = {}
    for index, agent in ipairs(agents) do
        if index > 1 then
            table.insert(sections, ui.separator({ spacing = 4 }))
        end

        local rows = {
            ui.label({ text = agent.name, fontWeight = "bold", color = "primary" }),
        }
        local lines = agent.loading and { "Loading usage limits…" } or agent.lines
        for _, line in ipairs(lines or unavailable()) do
            table.insert(rows, ui.label({ text = line, color = "on_surface_variant" }))
        end
        table.insert(sections, ui.column({ gap = 5 }, rows))
    end

    panel.render(ui.column({ flexGrow = 1, gap = 12 }, {
        ui.label({ text = "Agent usage limits", fontSize = 16, fontWeight = "bold", color = "on_surface" }),
        ui.scroll({ flexGrow = 1, gap = 12, align = "stretch" }, sections),
    }))
end

local function fetch(agent, currentGeneration)
    local launched = noctalia.runAsync({
        noctalia.expandPath("~/.dotfiles/bin/agent-usage-" .. agent.id),
        "--limits-only",
    }, function(result)
        if not open or generation ~= currentGeneration then
            return
        end

        local lines
        if result.exitCode ~= 0 or result.timedOut or result.stdoutTruncated then
            lines = unavailable()
        else
            local decoded, record = pcall(noctalia.json.decode, result.stdout)
            lines = decoded and recordLines(record) or unavailable()
        end
        agent.loading = false
        agent.lines = lines or unavailable()
        render()
    end)

    if not launched and open and generation == currentGeneration then
        agent.loading = false
        agent.lines = unavailable()
        render()
    end
end

function onOpen(_context)
    generation = generation + 1
    local currentGeneration = generation
    open = true
    for _, agent in ipairs(agents) do
        agent.loading = true
        agent.lines = nil
    end
    render()
    for _, agent in ipairs(agents) do
        fetch(agent, currentGeneration)
    end
end

function onClose()
    open = false
    generation = generation + 1
end
