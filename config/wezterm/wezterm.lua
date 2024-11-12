-- Pull in the wezterm API
local wezterm = require 'wezterm'



-- This table will hold the configuration.
local config = {}

-- In newer versions of wezterm, use the config_builder which will
-- help provide clearer error messages
if wezterm.config_builder then
  config = wezterm.config_builder()
end

-- This is where you actually apply your config choices

-- For example, changing the color scheme:
config.color_scheme = 'nord'
-- and finally, return the configuration to wezterm


config.font = wezterm.font 'JetBrains Mono'

config.hide_tab_bar_if_only_one_tab = true
config.tab_bar_at_bottom = true
config.window_background_opacity = 0.98

config.webgpu_force_fallback_adapter = true

-- window:spawn_tab { cwd = '/tmp' }

-- config.default_cwd = '/home/dln'
config.keys = {
  {
    key = 'y',
    mods = 'CMD',
    action = wezterm.action.SpawnCommandInNewTab {
      args = { 'htop' },
    },
  },
  {
    key = 'T',
    mods = 'SHIFT|CTRL',
    action = wezterm.action.SpawnCommandInNewTab {
      domain = 'CurrentPaneDomain',
      cwd = wezterm.home_dir,

    },
  },
  {
    key = 'Q',
    mods = 'SHIFT|CTRL',
    action = wezterm.action.CloseCurrentTab { confirm = true },
  },
}
-- config.keys = {
--   -- Turn off the default CMD-m Hide action, allowing CMD-m to
--   -- be potentially recognized and handled by the tab
--   {
--     key = 'LeftArrow',
--     mods = 'CMD',
--     action = wezterm.action.ActivateTabRelative(1),
--   },
-- }
-- { key = 'PageDown', mods = 'CTRL', action = act.ActivateTabRelative(1) },


return config
