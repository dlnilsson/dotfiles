#pragma once
#include <pipewire/pipewire.h>
#include <spa/utils/dict.h>
#include <stdint.h>

int  pw_run_loop(void);
void pw_quit_loop(void);

// Go-exported callbacks (note: cgo expects char*, not const char*)
void goOnNodeAdd(uint32_t id, char* media_class, char* media_role, char* media_type, char* media_category, char* name, char* desc, char* app, char* portal_app);
void goOnNodeRemove(uint32_t id);
void goOnLinkAdd(uint32_t id, uint32_t output_node_id, uint32_t input_node_id, int state);
void goOnLinkRemove(uint32_t id);
void goOnLinkStateChange(uint32_t id, int state);
