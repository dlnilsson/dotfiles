local urls = {}
local message = "Loading…"
local status = nil
local statusMessage = "Loading…"
local country, city, dns = nil, nil, nil
local applying = false
local actionMessage = nil
local opened = false
local generation = 0
local statusRequest = 0
local render, fetchStatus

local statusCommand = [[tailscale status --json | jq -ce '{
    online: .Self.Online,
    exitNode: ([.Peer[]? | select(.ExitNode) | (.DNSName // .HostName) | rtrimstr(".")][0]),
    countries: ([.Peer[]?
        | select((.Tags // []) | index("tag:mullvad-exit-node"))
        | select(.DNSName != null and .DNSName != "")
        | {country: (.Location.Country // "Unknown country"),
           city: (.Location.City // "Unknown city"), dns: (.DNSName | rtrimstr("."))}]
        | group_by(.country)
        | map({name: .[0].country, cities: (group_by(.city)
            | map({name: .[0].city, nodes: (map(.dns) | sort)}))}))
}']]

local function decode(result)
    if result.exitCode ~= 0 or result.timedOut or result.stdoutTruncated then
        return nil
    end
    local ok, data = pcall(noctalia.json.decode, result.stdout)
    return ok and type(data) == "table" and data or nil
end

local function failure(prefix, result)
    local detail = result.stderr and result.stderr:match("^%s*(.-)%s*$")
    return detail and detail ~= "" and (prefix .. ": " .. detail) or prefix
end

local function setExitNode(name)
    if applying then
        return
    end
    applying = true
    actionMessage = "Applying exit node…"
    render()
    local function complete(result)
        applying = false
        if result.exitCode ~= 0 or result.timedOut then
            dns = nil
            actionMessage = failure("Could not set exit node", result)
            render()
            return
        end
        actionMessage = nil
        if opened then
            fetchStatus()
        end
    end
    if not noctalia.runAsync({ "tailscale", "set", "--exit-node=" .. name }, complete) then
        complete({ exitCode = -1, stderr = "Could not start tailscale" })
    end
end

local function selector(key, placeholder, choices, selected, onChange)
    local options = { placeholder }
    local selectedIndex = 0
    for index, choice in ipairs(choices) do
        table.insert(options, type(choice) == "table" and choice.name or choice)
        if choice == selected then
            selectedIndex = index
        end
    end
    return ui.select({
        key = key,
        options = options,
        selectedIndex = selectedIndex,
        enabled = not applying and #choices > 0,
        onChange = function(value)
            if not applying then
                onChange(choices[tonumber(value)])
            end
        end,
    })
end

local function open(url)
    noctalia.runAsync({ "xdg-open", url })
    panel.close()
end

render = function()
    if not opened then
        return
    end
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

    local exitRows = {
        ui.label({ text = "Mullvad exit node", fontWeight = "medium", color = "on_surface_variant" }),
        selector("country", "Country", status and status.countries or {}, country, function(value)
            country, city, dns = value, nil, nil
            render()
        end),
        selector("city", "City", country and country.cities or {}, city, function(value)
            city, dns = value, nil
            render()
        end),
        selector("dns", "DNS name", city and city.nodes or {}, dns, function(value)
            dns = value
            if dns then
                setExitNode(dns)
            else
                render()
            end
        end),
    }
    if status and status.exitNode and not applying then
        table.insert(exitRows, ui.button({
            key = "disable-exit-node",
            text = "Disable exit node",
            variant = "ghost",
            contentAlign = "start",
            onClick = function()
                setExitNode("")
            end,
        }))
    end
    if status and #status.countries == 0 then
        table.insert(exitRows, ui.label({ text = "No Mullvad exit nodes available", color = "on_surface_variant" }))
    end
    if actionMessage then
        table.insert(exitRows, ui.label({ text = actionMessage, color = applying and "on_surface_variant" or "#bf616a" }))
    end

    panel.render(ui.column({ flexGrow = 1, gap = 12 }, {
        ui.label({ text = "Tailscale", fontSize = 16, fontWeight = "bold", color = "on_surface" }),
        ui.column({ gap = 4 }, statusRows),
        ui.scroll({ flexGrow = 1, gap = 12, align = "stretch" }, {
            ui.column({ gap = 8 }, exitRows),
            ui.label({ text = "Services", fontWeight = "medium", color = "on_surface_variant" }),
            ui.column({ gap = 4 }, rows),
        }),
    }))
end

fetchStatus = function()
    statusRequest = statusRequest + 1
    local request, currentGeneration = statusRequest, generation
    status = nil
    statusMessage = "Loading…"
    country, city, dns = nil, nil, nil
    render()
    local function complete(result)
        if not opened or generation ~= currentGeneration or statusRequest ~= request then
            return
        end
        status = decode(result)
        if not status then
            statusMessage = failure("tailscale status failed", result)
        else
            for _, candidateCountry in ipairs(status.countries) do
                for _, candidateCity in ipairs(candidateCountry.cities) do
                    for _, candidateDns in ipairs(candidateCity.nodes) do
                        if candidateDns == status.exitNode then
                            country, city, dns = candidateCountry, candidateCity, candidateDns
                        end
                    end
                end
            end
        end
        render()
    end
    if not noctalia.runAsync(statusCommand, complete) then
        complete({ exitCode = -1, stderr = "Could not start tailscale status" })
    end
end

function onOpen(_context)
    opened = true
    generation = generation + 1
    local currentGeneration = generation
    urls = {}
    message = "Loading…"
    if not applying then
        actionMessage = nil
    end
    fetchStatus()
    local function complete(result)
        if not opened or generation ~= currentGeneration then
            return
        end
        local services = decode(result)
        if not services then
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
    end
    if not noctalia.runAsync({ "tailscale", "service", "list", "--json" }, complete) then
        complete({ exitCode = -1 })
    end
end

function onClose()
    opened = false
    generation = generation + 1
end
