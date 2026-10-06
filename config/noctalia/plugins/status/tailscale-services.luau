-- Tailscale connection and exit node status, then one button per service.
-- Clicking a button opens https://<hostname> in the browser.
local urls = {}
local message = "Loading…"
local status = nil
local statusMessage = "Loading…"

local function open(url)
    noctalia.runAsync({ "xdg-open", url })
    panel.close()
end

local function render()
    local rows = {}
    for _, url in ipairs(urls) do
        table.insert(rows, ui.button({
            key = url,
            text = url,
            variant = "ghost",
            contentAlign = "start",
            onClick = function()
                open(url)
            end,
        }))
    end
    if #rows == 0 then
        table.insert(rows, ui.label({ text = message, color = "on_surface_variant" }))
    end

    local statusRows
    if status then
        statusRows = {
            ui.label({
                text = status.online and "Connected" or "Offline",
                color = status.online and "#a3be8c" or "#bf616a",
            }),
            ui.label({ text = "Exit node: " .. (status.exitNode or "none"), color = "on_surface_variant" }),
        }
    else
        statusRows = { ui.label({ text = statusMessage, color = "on_surface_variant" }) }
    end

    panel.render(ui.column({ flexGrow = 1, gap = 12 }, {
        ui.label({ text = "Tailscale", fontSize = 16, fontWeight = "bold", color = "on_surface" }),
        ui.column({ gap = 4 }, statusRows),
        ui.label({ text = "Services", fontWeight = "medium", color = "on_surface_variant" }),
        ui.scroll({ flexGrow = 1, gap = 4, align = "stretch" }, rows),
    }))
end

function onOpen(_context)
    urls = {}
    message = "Loading…"
    status = nil
    statusMessage = "Loading…"
    render()
    noctalia.runAsync(
        "tailscale status --json | jq -c '{online: .Self.Online, exitNode: ([.Peer[]? | select(.ExitNode) | .HostName][0])}'",
        function(result)
            local data = result.exitCode == 0 and noctalia.json.decode(result.stdout)
            if type(data) ~= "table" then
                statusMessage = "tailscale status failed"
            else
                status = {
                    online = data.online == true,
                    exitNode = type(data.exitNode) == "string" and data.exitNode or nil,
                }
            end
            render()
        end
    )
    noctalia.runAsync({ "tailscale", "service", "list", "--json" }, function(result)
        local services = result.exitCode == 0 and noctalia.json.decode(result.stdout)
        if type(services) ~= "table" then
            message = "tailscale service list failed"
            render()
            return
        end
        urls = {}
        for _, service in ipairs(services) do
            if type(service.Hostname) == "string" then
                table.insert(urls, "https://" .. service.Hostname)
            end
        end
        message = "No services"
        render()
    end)
end
