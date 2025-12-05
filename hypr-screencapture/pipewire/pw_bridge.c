#include "pw_bridge.h"
#include <pipewire/link.h>
#include <string.h>
#include <stdlib.h>

static struct pw_main_loop *g_loop = NULL;
static struct pw_context   *g_ctx  = NULL;
static struct pw_core      *g_core = NULL;
static struct pw_registry  *g_reg  = NULL;
static struct spa_hook      g_reg_listener;

struct link_data {
  uint32_t id;
  uint32_t output_node_id;
  uint32_t input_node_id;
  struct pw_link *link;
  struct spa_hook listener;
  struct link_data *next;
};

static struct link_data *g_links = NULL;

static inline const char* dict_get(const struct spa_dict *props, const char *key) {
  if (!props) return NULL;
  return spa_dict_lookup(props, key);
}

static void on_link_info(void *data, const struct pw_link_info *info)
{
  if (!data || !info) return;

  struct link_data *ld = (struct link_data *)data;
  if (!ld->link) return;

  int is_new = (ld->output_node_id == 0 && ld->input_node_id == 0);
  uint32_t old_output = ld->output_node_id;
  uint32_t old_input = ld->input_node_id;
  ld->output_node_id = info->output_node_id;
  ld->input_node_id = info->input_node_id;

  if (is_new) {
    goOnLinkAdd(ld->id, info->output_node_id, info->input_node_id, (int)info->state);
  } else if (old_output != 0 && old_input != 0) {
    goOnLinkStateChange(ld->id, (int)info->state);
  }
}

static const struct pw_link_events link_events = {
  PW_VERSION_LINK_EVENTS,
  .info = on_link_info,
};

static void on_global(void *data,
                      uint32_t id,
                      uint32_t permissions,
                      const char *type,
                      uint32_t version,
                      const struct spa_dict *props)
{
  (void)data; (void)permissions; (void)version;
  if (!type) return;

  if (strcmp(type, PW_TYPE_INTERFACE_Node) == 0) {
    const char *mc     = dict_get(props, "media.class");
    const char *mr     = dict_get(props, "media.role");
    const char *mt     = dict_get(props, "media.type");
    const char *mcat   = dict_get(props, "media.category");
    const char *name   = dict_get(props, "node.name");
    const char *desc   = dict_get(props, "node.description");
    const char *app    = dict_get(props, "application.name");
    if (!app) app      = dict_get(props, "app.name");
    const char *portal = dict_get(props, "access.portal.app.id");

    goOnNodeAdd(id,
      (char*)(mc     ? mc     : ""),
      (char*)(mr     ? mr     : ""),
      (char*)(mt     ? mt     : ""),
      (char*)(mcat   ? mcat   : ""),
      (char*)(name   ? name   : ""),
      (char*)(desc   ? desc   : ""),
      (char*)((app && app[0]) ? app : ""),
      (char*)(portal ? portal : ""));
  } else if (strcmp(type, PW_TYPE_INTERFACE_Link) == 0) {
    struct pw_link *link = pw_registry_bind(g_reg, id, type, version, 0);
    if (!link) return;

    struct link_data *ld = calloc(1, sizeof(struct link_data));
    if (!ld) {
      pw_proxy_destroy((struct pw_proxy *)link);
      return;
    }

    ld->id = id;
    ld->link = link;
    ld->output_node_id = 0;
    ld->input_node_id = 0;
    ld->next = g_links;
    g_links = ld;

    pw_link_add_listener(link, &ld->listener, &link_events, ld);
  }
}

static void on_global_remove(void *data, uint32_t id)
{
  (void)data;

  struct link_data **prev = &g_links;
  struct link_data *current = g_links;

  while (current) {
    if (current->id == id) {
      *prev = current->next;
      spa_hook_remove(&current->listener);
      if (current->link) {
        pw_proxy_destroy((struct pw_proxy *)current->link);
        current->link = NULL;
      }
      goOnLinkRemove(id);
      free(current);
      return;
    }
    prev = &current->next;
    current = current->next;
  }

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

  while (g_links) {
    struct link_data *next = g_links->next;
    spa_hook_remove(&g_links->listener);
    if (g_links->link) {
      pw_proxy_destroy((struct pw_proxy *)g_links->link);
    }
    free(g_links);
    g_links = next;
  }

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
