#pragma once
#include <pipewire/pipewire.h>
#include <spa/utils/dict.h>
#include <stdint.h>

int  pw_run_loop(void);
void pw_quit_loop(void);

// Go-exported callbacks (note: cgo expects char*, not const char*)
void goOnNodeAdd(uint32_t id, char* media_class, char* name, char* desc, char* app, char* portal_app);
void goOnNodeRemove(uint32_t id);
