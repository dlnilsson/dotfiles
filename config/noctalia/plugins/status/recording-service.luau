local started = false
noctalia.setUpdateInterval(1000)

function update()
    if started then
        return
    end
    -- runStream has no exit callback, so the loop restarts the subscription
    -- when the daemon is not running yet or restarts.
    started = noctalia.runStream(
        "while :; do ~/go/bin/hypr-screencapture status -subscribe; echo; sleep 2; done",
        function(line)
            noctalia.state.set("recording", line:match("%S") ~= nil)
        end
    )
end
