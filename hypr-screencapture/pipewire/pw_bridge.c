#include "pw_bridge.h"
#include <string.h>
#include <stdlib.h>

static struct pw_main_loop *g_loop = NULL;
static struct pw_context   *g_ctx  = NULL;
static struct pw_core      *g_core = NULL;
static struct pw_registry  *g_reg  = NULL;
static struct spa_hook      g_reg_listener;

static inline const char* dict_get(const struct spa_dict *props, const char *key) {
  if (!props) return NULL;
  return spa_dict_lookup(props, key);
}

static void on_global(void *data,
                      uint32_t id,
                      uint32_t permissions,
                      const char *type,
                      uint32_t version,
                      const struct spa_dict *props)
{
  (void)data; (void)permissions; (void)version;
  if (!type) return;
  if (strcmp(type, PW_TYPE_INTERFACE_Node) != 0) return;

  const char *mc     = dict_get(props, "media.class");
  const char *name   = dict_get(props, "node.name");
  const char *desc   = dict_get(props, "node.description");
  const char *app    = dict_get(props, "application.name");
  if (!app) app      = dict_get(props, "app.name");
  const char *portal = dict_get(props, "access.portal.app.id");

  goOnNodeAdd(id,
    (char*)(mc     ? mc     : ""),
    (char*)(name   ? name   : ""),
    (char*)(desc   ? desc   : ""),
    (char*)((app && app[0]) ? app : ""),
    (char*)(portal ? portal : ""));
}

static void on_global_remove(void *data, uint32_t id)
{
  (void)data;
  goOnNodeRemove(id);
}

static const struct pw_registry_events reg_events = {
  PW_VERSION_REGISTRY_EVENTS,
  .global = on_global,
  .global_remove = on_global_remove,
};

int pw_run_loop(void) {
  pw_init(NULL, NULL);

  g_loop = pw_main_loop_new(NULL);
  if (!g_loop) return -1;

  g_ctx = pw_context_new(pw_main_loop_get_loop(g_loop), NULL, 0);
  if (!g_ctx) return -2;

  g_core = pw_context_connect(g_ctx, NULL, 0);
  if (!g_core) return -3;

  g_reg = pw_core_get_registry(g_core, PW_VERSION_REGISTRY, 0);
  if (!g_reg) return -4;

  pw_registry_add_listener(g_reg, &g_reg_listener, &reg_events, NULL);

  // Roundtrip so we also receive existing objects
  pw_core_sync(g_core, PW_ID_CORE, 0);

  pw_main_loop_run(g_loop);

  if (g_reg)  spa_hook_remove(&g_reg_listener);
  if (g_core) pw_core_disconnect(g_core);
  if (g_ctx)  pw_context_destroy(g_ctx);
  if (g_loop) pw_main_loop_destroy(g_loop);

  pw_deinit();
  return 0;
}

void pw_quit_loop(void) {
  if (g_loop) pw_main_loop_quit(g_loop);
}
