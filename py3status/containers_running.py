# -*- coding: utf-8 -*-
"""
Show all containers on current DOCKER_HOST
https://py3status.readthedocs.io/en/latest/writing_modules.html
https://i3wm.org/docs/user-contributed/py3status.html
"""

import docker

class Py3status:
    def __init__(self):
        self.full_text = ''
        self.client = docker.from_env()

    def containers_running(self):
        txt = '🐋 '
        stauses = {
            'created': 0,
            'restarting': 0,
            'running': 0,
            'removing': 0,
            'paused': 0,
            'exited': 0,
            'dead': 0,
        }
        for container in self.client.containers.list(all=True):
            stauses[container.status] += 1
        stauses = {k: v for k, v in stauses.items() if v > 0}

        for key, val in stauses.items():
            txt += " {}: {}".format(key, val)

        self.full_text = txt

        return {
            'full_text': self.full_text,
            'cached_until': self.py3.time_in(30)
        }

if __name__ == "__main__":
    """
    Run module in test mode.
    """
    from py3status.module_test import module_test

    module_test(Py3status)
