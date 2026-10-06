require("./status-widget.luau")("vpn")

function onRightClick()
    noctalia.togglePanel("dln/status:tailscale-services")
end
