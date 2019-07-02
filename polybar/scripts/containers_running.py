#!/bin/python
# -*- coding: utf-8 -*-
"""
Show all containers on current DOCKER_HOST
"""

import docker
import argparse


class ListContainers:
    def __init__(self):
        self.client = docker.from_env()

    def containers_running(self, prefix_font):
        prefix = prefix_font or ''
        txt = ''
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

        if len(stauses) > 0:
            txt = f'{prefix} '

        for key, val in stauses.items():
            txt += f' {key}: {val}'

        return txt


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument(
        '--prefix',
        type=str,
        metavar='',
        dest='prefix_font'
    )
    args = parser.parse_args()
    prefix_font = args.prefix_font
    list = ListContainers()
    print(list.containers_running(prefix_font))
