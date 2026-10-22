local output = barWidget.outputName()
-- Ultrawide outputs (3440px or wider) get a longer title.
local maxWidth = 400

local function render(titles)
    local title = titles and titles[output]
    local text = title and title.text or ""
    barWidget.setVisible(text ~= "")
    barWidget.setTooltip(title and title.tooltip or "")
    barWidget.render(ui.label({ text = text, maxWidth = maxWidth, maxLines = 1 }))
end

-- The plugin API does not expose output size, so ask Hyprland.
noctalia.runAsync({ "hyprctl", "monitors", "-j" }, function(result)
    if result.exitCode ~= 0 then
        return
    end
    local monitors = noctalia.json.decode(result.stdout)
    if type(monitors) ~= "table" then
        return
    end
    for _, monitor in ipairs(monitors) do
        if monitor.name == output and type(monitor.width) == "number" and monitor.width >= 3440 then
            maxWidth = 950
            render(noctalia.state.get("window-titles"))
        end
    end
end)

noctalia.state.watch("window-titles", render)
render(noctalia.state.get("window-titles"))
