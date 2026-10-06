local output = barWidget.outputName()

local function switchWorkspace(workspace)
    noctalia.runAsync({
        "hyprctl", "dispatch",
        'hl.dsp.focus({ workspace = "' .. tostring(workspace) .. '" })',
    }, function(result)
        if result.exitCode ~= 0 then
            noctalia.log("Workspace switch failed: " .. result.stdout .. result.stderr)
        end
    end)
end

local function render(state)
    local active = state and state.active[output]
    local occupied = {}
    for _, id in ipairs(state and state.occupied or {}) do
        occupied[id] = true
    end

    local children = {}
    for id = 1, 9 do
        local selected = id == active
        table.insert(children, ui.column({
            key = tostring(id),
            width = 26,
            height = 23,
            align = "center",
            justify = "space_between",
            tooltip = "Switch to workspace " .. tostring(id),
            onClick = function()
                switchWorkspace(id)
            end,
        }, {
            ui.box({ width = 26, height = 3, fill = "#00000000" }),
            ui.label({
                text = tostring(id),
                color = (selected or occupied[id]) and "#d8dee9" or "#4c566a",
            }),
            ui.box({ width = 26, height = 3, fill = selected and "#d08770" or "#00000000" }),
        }))
    end
    barWidget.render(ui.row({ gap = 0, align = "center" }, children))
end

noctalia.state.watch("workspaces", render)
render(noctalia.state.get("workspaces"))

function onScroll(axis, steps)
    if axis == "vertical" and steps ~= 0 then
        switchWorkspace(steps > 0 and "e+1" or "e-1")
    end
end
