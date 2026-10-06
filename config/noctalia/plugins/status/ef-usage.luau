require("./status-widget.luau")("ef-usage", "robot")

function onRightClick()
    noctalia.togglePanel("dln/status:agent-usage")
end
