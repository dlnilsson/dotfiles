-- Red dot while a screencast is active. It blinks with a 1-second period,
-- 0.5 seconds on and 0.5 seconds off.
local recording = false
local lit = true

local function render()
    barWidget.setVisible(recording)
    if not recording then
        return
    end
    barWidget.render(ui.row({
        align = "center",
        onClick = function()
            noctalia.runAsync({ noctalia.expandPath("~/.dotfiles/bin/screen-recorder") })
        end,
    }, {
        ui.label({ text = "●", color = lit and "#ff0000" or "#00000000" }),
    }))
end

noctalia.setUpdateInterval(500)

function update()
    if recording then
        lit = not lit
        render()
    end
end

noctalia.state.watch("recording", function(value)
    recording = value == true
    lit = true
    render()
end)
recording = noctalia.state.get("recording") == true
render()
