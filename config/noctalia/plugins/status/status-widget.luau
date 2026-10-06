local function attach(source, glyph, onClick)
    local colors = {
        tailscale = "#88c0d0",
        netbird = "#a3be8c",
        disconnected = "#4c566a",
        unavailable = "#bf616a",
        pending = "#d08770",
    }
    local function render(data)
        local visible = data ~= nil and data.text ~= ""
        barWidget.setVisible(visible)
        if not visible then
            return
        end

        local attention = data.class == "action-required"
        local color = attention and "#2e3440" or colors[data.class] or "#81a1c1"
        local children = {}
        if glyph then
            table.insert(children, ui.glyph({ name = glyph, color = color }))
        end
        table.insert(children, ui.label({
            text = data.text,
            color = color,
            fontWeight = attention and "bold" or nil,
        }))
        barWidget.render(ui.row({
            gap = 5,
            align = "center",
            paddingH = attention and 8 or 0,
            paddingV = attention and 3 or 0,
            fill = attention and "#ebcb8b" or "#00000000",
            radius = 4,
            onClick = onClick,
        }, children))
        barWidget.setTooltip(data.tooltip)
    end

    noctalia.state.watch(source, render)
    render(noctalia.state.get(source))
end

return attach
