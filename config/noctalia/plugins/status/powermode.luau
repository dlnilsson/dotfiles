require("./status-widget.luau")("powermode", nil, function()
    local request = noctalia.state.get("powermode-toggle") or 0
    noctalia.state.set("powermode-toggle", request + 1)
end)
