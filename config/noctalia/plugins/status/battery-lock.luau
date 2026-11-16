-- Lockscreen widget: shows bin/battery_lock output, only while running on battery.
local REFRESH_SECONDS = 30

-- Prints nothing and exits non-zero when the AC adapter is online.
local COMMAND = '[ "$(cat /sys/class/power_supply/AC/online 2>/dev/null)" = 0 ] && exec "$HOME/.dotfiles/bin/battery_lock"'

local lines = {}
local pending = false
local tick = 0

local function render()
    local children = {}
    for _, line in ipairs(lines) do
        table.insert(children, ui.label({ text = line.text, color = line.color, fontSize = 18, textAlign = "center" }))
    end
    desktopWidget.render(ui.column({ gap = 4, align = "center" }, children))
end

-- battery_lock prints one "<color>\t<text>" line per label.
local function parse(stdout)
    local parsed = {}
    for line in stdout:gmatch("[^\n]+") do
        local color, text = line:match("^([^\t]*)\t(.*)$")
        if text then
            table.insert(parsed, { text = text, color = color })
        end
    end
    return parsed
end

local function refresh()
    if pending then
        return
    end
    pending = true
    local launched = noctalia.runAsync({ "sh", "-c", COMMAND }, function(result)
        pending = false
        lines = (result.exitCode == 0 and not result.timedOut) and parse(result.stdout) or {}
        render()
    end)
    if not launched then
        pending = false
    end
end

function update()
    tick = tick + 1
    if tick >= REFRESH_SECONDS then
        tick = 0
        refresh()
    end
end

desktopWidget.setWantsSecondTicks(true)
render()
refresh()
