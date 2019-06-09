#!/usr/bin/python
"""Setup dotfiles symlinks.

Mirror directory tree of the git repositroy based on
links_{UNAME}.json
all links are relative from $HOME.

"""
import shutil
import os
import json
import platform

def copytree(src, dst, symlinks=False):
    names = os.listdir(src)
    os.makedirs(dst, exist_ok=True)
    errors = []
    for name in names:
        srcname = os.path.join(src, name)
        dstname = os.path.join(dst, name)
        try:
            if symlinks and os.path.islink(srcname):
                linkto = os.readlink(srcname)
                os.symlink(linkto, dstname)
            elif os.path.isdir(srcname):
                copytree(srcname, dstname, symlinks)
            else:
                os.symlink(srcname, dstname)
        except FileExistsError:
            pass
        except OSError as why:
            errors.append((srcname, dstname, str(why)))
        except Error as err:
            errors.extend(err.args[0])
    try:
        shutil.copystat(src, dst)
    except OSError as why:
        errors.extend((src, dst, str(why)))
    if errors:
        raise Exception(errors)

def unlink_all(directory):
    for root, _, files in os.walk(directory):
        for f in files:
            os.unlink(os.path.join(os.path.abspath(root), f))

def main(cfg, home, current):
    for item in cfg['symlinks']:
        src_dir = current + item['from']
        dest_dir = home + item['to']

        try:
            if src_dir == "root":
                unlink_all(dest_dir)
            print('{:45} \u2192 {}'.format(src_dir, dest_dir))
            copytree(src_dir, dest_dir, symlinks=True)
        except Exception as error:
            print('Failed to setup symlink: {}'.format(error))


if __name__ == '__main__':
    # Base path to symlink from
    # All paths are based from $HOME
    HOME = os.environ['HOME'] + '/'
    CURRENT = os.getcwd() + '/'
    CFG = 'links_%s.json' % platform.uname().system

    with open(CFG) as data_file:
        JSON_CFG = json.load(data_file)

    main(JSON_CFG, HOME, CURRENT)
