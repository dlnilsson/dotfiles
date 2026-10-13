-- Desktop/lockscreen widget showing the output of bin/days_since.
local REFRESH_SECONDS = 60

local text = ""
local pending = false
local tick = 0

local function render()
    desktopWidget.render(ui.column({ gap = 4, align = "center" }, {
        ui.label({ text = text, fontSize = 18, textAlign = "center", maxLines = 8 }),
    }))
end

local function refresh()
    if pending then
        return
    end
    pending = true
    local launched = noctalia.runAsync({
        "python3",
        noctalia.expandPath("~/.dotfiles/bin/days_since"),
    }, function(result)
        pending = false
        if result.exitCode ~= 0 or result.timedOut then
            text = "days_since failed"
        else
            text = (result.stdout:gsub("%s+$", ""))
        end
        noctalia.log("days-since: exit=" .. tostring(result.exitCode) .. " out=" .. result.stdout .. " err=" .. result.stderr)
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

noctalia.log("days-since: loaded")
desktopWidget.setWantsSecondTicks(true)
render()
refresh()
