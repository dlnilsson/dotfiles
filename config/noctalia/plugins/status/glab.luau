require("./status-widget.luau")("glab", nil, function()
    noctalia.runAsync({ "xdg-open", "https://gitlab.com/dashboard/merge_requests" }, function() end)
end)
