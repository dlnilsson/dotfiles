local started = false
noctalia.setUpdateInterval(1000)

function update()
    if started then
        return
    end
    started = noctalia.runStream(
        "python3 ~/.dotfiles/config/noctalia/plugins/status/workspaces.py",
        function(line)
            local data = noctalia.json.decode(line)
            if type(data) == "table" and type(data.active) == "table"
                and type(data.occupied) == "table" then
                noctalia.state.set("workspaces", data)
                if type(data.titles) == "table" then
                    noctalia.state.set("window-titles", data.titles)
                end
            end
        end
    )
end
