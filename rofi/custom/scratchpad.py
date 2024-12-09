#!/usr/bin/python

import i3ipc
from rofi import Rofi

rofi_args = ['-theme', 'themes/custom-i3scratch.rasi', '-show-icons']
i3 = i3ipc.Connection()
menu = Rofi(rofi_args=rofi_args)

def scratchpad():
    for con in i3.get_tree():
        if(con.name == "__i3_scratch"):
            return con

def get_windows():
    wins = []
    names = []
    for con in scratchpad():
        wins += con
    for con in wins:
        names.append(con.name)
    return wins,names

wins, names = get_windows()
index, key = menu.select("Scratchpad", names)

for i in range((index+1)*2-1):
    i3.command("scratchpad show")

