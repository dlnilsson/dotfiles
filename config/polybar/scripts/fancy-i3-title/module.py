#! /usr/bin/python3

import os
import asyncio
import getpass
import sys
import i3ipc
import platform
from time import sleep
import argparse
import subprocess

from icon_resolver import IconResolver

#: Max length of single window title
MAX_LENGTH = 85
#: Base 1 index of the font that should be used for icons
ICON_FONT = 3
OUTPUT = 'DP-1'

def screens():
    output = [l for l in subprocess.check_output(
        ["xrandr"]).decode("utf-8").splitlines()]
    return [l.split()[0] for l in output if " connected " in l]

SCREENS = screens()
HOSTNAME = platform.node()
USER = getpass.getuser()

ICONS = [
    ('name=mutt', '\uf199'),
    ('name=thunderbird', '\uf199'),
    ('class=thunderbird', '\uf199'),
    ('name=youtube', '\uf167'),
    ('title=lazylocker', '\uf308'),
    ('title=ranger', '\uf413'),
    ('title=steam', '\uf9d2'),
    ('class=atom', '\ue764'),
    ('class=chrome', '\uf268'),
    ('class=code-oss', '\ue70c'),
    ('class=discord', '\ufb6e'),
    ('class=firefox', '\uf269'),
    ('class=Google-chrome', '\uf268'),
    ('class=kitty', '\uf120'),
    ('class=pcmanfm', '\uf07b'),
    ('class=pdf', '\uf725'),
    ('class=Rocket.Chat', '\uf135'),
    ('class=Slack', '\uf198'),
    ('class=Microsoft Teams - Preview', '\uf871'),
    ('class=spotify', '\uf1bc'),
    ('class=steam', '\uf9d2'),
    ('class=URxvt', '\ue795'),
    ('class=vlc', '\ufa7b'),

    ('*', ''),
    # ('*', '\ufaae'),
]

FORMATERS = {
    'chromium': lambda title: title.replace(' - Chromium', ''),
    'Google-chrome': lambda title: title.replace(' - Google Chrome', ''),
    'firefox': lambda title: title.replace(' - Mozilla Firefox', ''),
    'urxvt': lambda title: title.replace('%s@%s: ' % (USER, HOSTNAME), ''),
    'code-oss': lambda title: title.replace(' - Code - OSS', ''),
    'jetbrains-idea-ce': lambda title: title.replace(' - IntelliJ IDEA', ''),
    'jetbrains-pycharm-ce': lambda title: title.replace(' - PyCharm', ''),
    'kitty': lambda title: title.replace('%s@%s: ' % (USER, HOSTNAME), ''),
    'Microsoft Teams - Preview': lambda title: title.replace(' | Microsoft Teams', ''),
    'Slack': lambda title: title.replace('Slack | ', ''),

}

SCRIPT_DIR = os.path.dirname(os.path.realpath(__file__))
COMMAND_PATH = os.path.join(SCRIPT_DIR, 'command.py')

icon_resolver = IconResolver(ICONS)


def on_change(i3, e):
    render_apps(i3)


def render_apps(i3):
    global SCREENS
    workspaces = i3.get_workspaces()
    for w in workspaces:
        if w.focused and w.visible:
            tree = i3.get_tree()
            focused = tree.find_focused()
            if focused is None:
                return

            if w.output == OUTPUT:
                # [for screen in SCREENS if screen == OUTPUT]
                for screen in SCREENS:
                    if screen == OUTPUT:
                        print(format_entry(focused), flush=True)
                        return


def format_entry(app):
    title = make_title(app)

    u_color = '#b4619a' if app.focused else\
        '#e84f4f' if app.urgent else\
        '#404040'

    return '%%{u%s} %s %%{u-}' % (u_color, title)


def make_title(app):
    out = '{0}  {1}'.format(get_prefix(app), format_title(app))

    if app.focused:
        out = '%{F#fff}' + out + '%{F-}'

    return '%%{A1:%s %s:}%s%%{A-}' % (COMMAND_PATH, app.id, out)

def get_prefix(app):
    try:
        icon = icon_resolver.resolve({
            'class': app.window_class,
            'name': app.name,
            'title': app.window_title,
        })

        return '{:13}'.format(('%%{T%s}%s%%{T-}' % (ICON_FONT, icon)))
    except:
        return '\ufaaf'


def format_title(app):
    try:
        class_name = app.window_class
        name = app.name

        title = FORMATERS[class_name](
            name) if class_name in FORMATERS else name

        if len(title) > MAX_LENGTH:
            title = title[:MAX_LENGTH - 3] + '...'

        return title
    except:
        return ''


def main():
    i3 = i3ipc.Connection()

    i3.on('workspace::focus', on_change)
    i3.on('window::focus', on_change)
    i3.on('window', on_change)

    loop = asyncio.get_event_loop()

    loop.run_in_executor(None, i3.main)

    render_apps(i3)

    loop.run_forever()


if __name__ == '__main__':
    try:
        parser = argparse.ArgumentParser()
        parser.add_argument(
            '-t',
            '--trunclen',
            type=int,
            metavar='trunclen'
        )
        parser.add_argument(
            '-o',
            '--output',
            metavar='output'
        )
        parser.add_argument(
            '-f',
            '--font',
            type=int,
            metavar='font'
        )
        args = parser.parse_args()

        if args.trunclen is not None:
            MAX_LENGTH = args.trunclen

        if args.font is not None:
            ICON_FONT = args.font

        if args.output is not None:
            OUTPUT = args.output

        main()
    except KeyboardInterrupt:
        print('Interrupted')
        try:
            sys.exit(0)
        except SystemExit:
            os._exit(0)
