#!/usr/bin/python

import shutil
import errno
import ntpath
import os
import sys
import json
import pprint

# Base path to symlink from
# All paths are based from $HOME
home =  os.environ['HOME'] + '/'
current = os.getcwd() + '/'

with open('links.json') as data_file:
    jsonObject = json.load(data_file)


def path_leaf(path):
    head, tail = ntpath.split(path)
    return tail or ntpath.basename(head)

def select_files(root, files):
    """
    select all files in root directory
    root  - root directory
    files - files
    """
    selected_files = []

    for file in files:
        #do concatenation here to get full path
        full_path = os.path.join(root, file)
        ext = os.path.splitext(file)[1]
        selected_files.append(full_path)

    return selected_files

def build_recursive_dir_tree(path):
    """
    path    -    where to begin folder scan
    """
    selected_files = []

    for root, dirs, files in os.walk(path):
        selected_files += select_files(root, files)
    return selected_files

def force_symlink(file1, file2):
    """
    Creates a symlink, remove existing file if it exits
    file1    -    file to link
    file2    -    destination
    """
    print file1 + u'\t\u2192\t' + file2
    try:
        os.symlink(file1, file2)
    except OSError, e:
        if e.errno == errno.EEXIST:
            if os.path.isdir(file2):
                shutil.rmtree(file2)
            else:
                os.remove(file2)
            os.symlink(file1, file2)


for item in jsonObject['symlinks']:
    FROM = current + item['from']
    DESTINATION = home + item['to']
    files = build_recursive_dir_tree(FROM)
    pp = pprint.PrettyPrinter(indent=4)
    pp.pprint(files)

    print "Create symlink"
    print "FROM \t\t\t\t\t TO"
    for file in files:
        new_path = DESTINATION + path_leaf(file)
        force_symlink(file, new_path)



