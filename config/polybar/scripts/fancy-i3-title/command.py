#! /usr/bin/python3

import sys
import i3ipc

id = int(sys.argv[1])

i3 = i3ipc.Connection()
focused = i3.get_tree().find_focused()
if id == focused:
    selected_app = focused
    assert(selected_app)
    selected_app.command('focus')
# selected_app = next(app for app in i3.get_tree().leaves() if app.id == id)


