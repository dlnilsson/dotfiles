local output = barWidget.outputName()

local function render(titles)
    local title = titles and titles[output]
    local text = title and title.text or ""
    barWidget.setVisible(text ~= "")
    barWidget.setTooltip(title and title.tooltip or "")
    barWidget.render(ui.label({ text = text, maxWidth = 400, maxLines = 1 }))
end

noctalia.state.watch("window-titles", render)
render(noctalia.state.get("window-titles"))
