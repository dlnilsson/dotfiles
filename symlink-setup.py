#!/usr/bin/python
    
import os
import sys
import json
import pprint
sys.path.insert(0, os.getcwd() + '/src')
import functions


# Base path to symlink from
home =  os.environ['HOME'] + '/'
current = os.getcwd() + '/'

with open('links.json') as data_file:    
    jsonObject = json.load(data_file)

# All paths are based from $HOME

for item in jsonObject['symlinks']:
    FROM = current + item['from']
    DESTINATION = home + item['to']
    files = functions.build_recursive_dir_tree(FROM)
    pp = pprint.PrettyPrinter(indent=4)
    pp.pprint(files)

    print "Create symlink"
    print "FROM \t\t\t\t\t TO"
    for file in files:
        new_path = DESTINATION + functions.path_leaf(file)
        functions.force_symlink(file, new_path)




