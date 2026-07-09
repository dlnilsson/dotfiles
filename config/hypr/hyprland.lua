-- Hyprland Lua Configuration
-- Converted from hyprland.conf
-- https://wiki.hypr.land/Configuring/Start/

hl.monitor({ output = "desc:Dell Inc. DELL P2717H 4P9HC84RB85S", mode = "1920x1080@60.0", position = "0x0", scale = 1.0 })
hl.monitor({ output = "desc:LG Electronics LG ULTRAGEAR 011NTLE3V064", mode = "3440x1440@99.99", position = "0x0", scale = 1.0 })
hl.monitor({ output = "desc:Lenovo Group Limited 0x403D", mode = "highres", position = "auto", scale = 1.0 })
hl.monitor({ output = "desc:AOC CU34E4CV ZO0RBHA000849", mode = "3440x1440@100.00", position = "0x0", scale = 1.0 })


local terminal = "kitty"
local menu = [[rofi -show drun -display-drun  -run-shell-command '{terminal} -e \" {cmd}; read -n 1 -s\"']]


hl.on("hyprland.start", function()
    hl.exec_cmd("QT_QPA_PLATFORM=xcb copyq --start-server")
    hl.exec_cmd("blueman-applet")
    hl.exec_cmd("flameshot")
    hl.exec_cmd("hyprpm reload --notify")
    hl.exec_cmd("iwgtk --indicators")
    hl.exec_cmd("systemctl --user enable --now hyprpaper.service")
    hl.exec_cmd("caffeine start")
    hl.exec_cmd("dbus-update-activation-environment --systemd WAYLAND_DISPLAY XDG_CURRENT_DESKTOP HYPRLAND_INSTANCE_SIGNATURE DISPLAY")
    hl.exec_cmd([[gsettings set org.gnome.desktop.interface gtk-theme "Nordic"]])
    hl.exec_cmd([[gsettings set org.gnome.desktop.interface color-scheme "prefer-dark"]])
end)

-- Runs on every reload
hl.exec_cmd("pkill waybar; waybar &")


hl.env("XCURSOR_SIZE", "24")
hl.env("HYPRCURSOR_SIZE", "24")
hl.env("GDK_BACKEND", "wayland,x11,*")
hl.env("QT_QPA_PLATFORM", "wayland;xcb")
hl.env("QT_QPA_PLATFORMTHEME", "qt6ct")
hl.env("CLUTTER_BACKEND", "wayland")
hl.env("GTK_THEME", "Nord")
hl.env("QT_STYLE_OVERRIDE", "kvantum")
hl.env("SDL_VIDEODRIVER", "wayland")
hl.env("MOZ_ENABLE_WAYLAND", "1")
hl.env("ELECTRON_OZONE_PLATFORM_HINT", "wayland")
hl.env("OZONE_PLATFORM", "wayland")
hl.env("XDG_SESSION_TYPE", "wayland")
hl.env("XDG_CURRENT_DESKTOP", "Hyprland")
hl.env("XDG_SESSION_DESKTOP", "Hyprland")


hl.config({
    ecosystem = {
        no_donation_nag = true,
        enforce_permissions = 1,
    },
})

hl.permission("/usr/(bin|local/bin)/grim", "screencopy", "allow")
hl.permission("/usr/(lib|libexec|lib64)/xdg-desktop-portal-hyprland", "screencopy", "ask")
hl.permission("/usr/(bin|local/bin)/hyprpm", "plugin", "allow")
hl.permission("/usr/(bin|local/bin)/hyprlock", "screencopy", "allow")
hl.permission("/usr/bin/swaylock", "screencopy", "allow")
hl.permission("/usr/bin/hyprlock", "screencopy", "allow")
hl.permission("/usr/bin/hyprpicker", "screencopy", "allow")
hl.permission("/usr/bin/hyprland-preview-share-picker", "screencopy", "allow")
hl.permission("^omkbd-ergodash-rev1\\.2$", "keyboard", "allow")
hl.permission("^yubico-yubikey-otp\\+fido\\+ccid$", "keyboard", "allow")
hl.permission("hl-virtual-keyboard-wtype", "keyboard", "allow")
hl.permission("^omkbd-ergodash-rev1\\.2-(consumer-control|system-control)$", "keyboard", "allow")
hl.permission("^dygma-defy-keyboard-1$", "keyboard", "allow")
hl.permission("^dygma-defy-keyboard$", "keyboard", "allow")
hl.permission("^dygma-defy-mouse$", "keyboard", "allow")
hl.permission("^at-translated-set-2-keyboard$", "keyboard", "allow")
hl.permission("^thinkpad-extra-buttons$", "keyboard", "allow")
hl.permission("^logitech-usb-receiver(-consumer-control|-system-control)?$", "keyboard", "allow")
hl.permission("^keyd-virtual-keyboard$", "keyboard", "allow")
hl.permission("^logiops-virtual-input$", "keyboard", "allow")
hl.permission("^(video-bus|power-button|sleep-button|dp-1|dp-2)$", "keyboard", "allow")
-- Anything else (not currently plugged in) will prompt
hl.permission(".*", "keyboard", "ask")


hl.config({
    cursor = {
        hide_on_key_press = true,
    },

    general = {
        gaps_in = 5,
        gaps_out = 10,

        border_size = 2,

        col = {
            active_border          = { colors = { "rgba(88c0d0ee)", "rgba(8fbcbbee)" }, angle = 30 },
            inactive_border        = "rgba(595959aa)",
            nogroup_border         = "rgba(595959aa)",
            nogroup_border_active  = { colors = { "rgba(88c0d0ee)", "rgba(8fbcbbee)" }, angle = 30 },
        },

        resize_on_border = true,
        allow_tearing = false,
        layout = "dwindle",
    },

    group = {
        col = {
            border_active   = "rgba(33ccffee)",
            border_inactive = "rgba(222222aa)",
        },
        groupbar = {
            enabled                  = true,
            render_titles            = true,
            height                   = 22,
            text_color               = "rgba(ffffffee)",
            font_size                = 12,
            font_family              = "monospace",
            font_weight_active       = "ultraheavy",
            font_weight_inactive     = "normal",
            indicator_height         = 0,
            indicator_gap            = 5,
            gaps_in                  = 5,
            gaps_out                 = 0,
            gradients                = true,
            gradient_rounding        = 0,
            gradient_round_only_edges = false,
        },
    },

    decoration = {
        rounding       = 10,
        rounding_power = 2,

        active_opacity   = 1.0,
        inactive_opacity = 1.0,
        dim_inactive     = true,
        dim_strength     = 0.05,

        shadow = {
            enabled      = true,
            range        = 4,
            render_power = 3,
            color        = "rgba(1a1a1aee)",
        },

        blur = {
            enabled            = true,
            size               = 1,
            noise              = 0.0,
            vibrancy           = 0.1696,
            vibrancy_darkness  = 0.3,
            passes             = 2,
            new_optimizations  = true,
            ignore_opacity     = true,
        },
    },
})


hl.config({
    animations = {
        enabled = true,
    },
})

hl.curve("easeOutQuint",              { type = "bezier", points = { { 0.23, 1 },    { 0.32, 1 }    } })
hl.curve("easeInOutCubic",            { type = "bezier", points = { { 0.65, 0.05 }, { 0.36, 1 }    } })
hl.curve("linear",                    { type = "bezier", points = { { 0, 0 },       { 1, 1 }       } })
hl.curve("almostLinear",              { type = "bezier", points = { { 0.5, 0.5 },   { 0.75, 1.0 }  } })
hl.curve("quick",                     { type = "bezier", points = { { 0.15, 0 },    { 0.1, 1 }     } })
hl.curve("md3_accel",                 { type = "bezier", points = { { 0.05, 0.7 },  { 0.1, 1 }     } })
hl.curve("easeInOutElasticApprox",    { type = "bezier", points = { { 0.68, -0.55 },{ 0.27, 1.55 } } })

hl.animation({ leaf = "global",        enabled = true,  speed = 10,   bezier = "default" })
hl.animation({ leaf = "border",        enabled = true,  speed = 5.39, bezier = "easeOutQuint" })
hl.animation({ leaf = "windows",       enabled = true,  speed = 1.79, bezier = "easeOutQuint" })
hl.animation({ leaf = "windowsIn",     enabled = true,  speed = 1.5,  bezier = "easeOutQuint",    style = "gnomed" })
hl.animation({ leaf = "windowsOut",    enabled = true,  speed = 1.5,  bezier = "easeInOutCubic",  style = "gnomed" })
hl.animation({ leaf = "fadeIn",        enabled = true,  speed = 0.73, bezier = "almostLinear" })
hl.animation({ leaf = "fadeOut",       enabled = true,  speed = 0.46, bezier = "almostLinear" })
hl.animation({ leaf = "fade",          enabled = true,  speed = 0.03, bezier = "quick" })
hl.animation({ leaf = "layers",        enabled = true,  speed = 3.81, bezier = "easeOutQuint" })
hl.animation({ leaf = "layersIn",      enabled = true,  speed = 4,    bezier = "easeOutQuint",    style = "fade" })
hl.animation({ leaf = "layersOut",     enabled = true,  speed = 1.5,  bezier = "linear",          style = "fade" })
hl.animation({ leaf = "fadeLayersIn",  enabled = true,  speed = 1.79, bezier = "almostLinear" })
hl.animation({ leaf = "fadeLayersOut", enabled = true,  speed = 1.39, bezier = "almostLinear" })
hl.animation({ leaf = "workspaces",    enabled = true,  speed = 1.94, bezier = "almostLinear",    style = "fade" })
hl.animation({ leaf = "workspacesIn",  enabled = true,  speed = 1.21, bezier = "almostLinear",    style = "fade" })
hl.animation({ leaf = "workspacesOut", enabled = true,  speed = 1.94, bezier = "almostLinear",    style = "fade" })
hl.animation({ leaf = "zoomFactor",    enabled = true,  speed = 7,    bezier = "quick" })
-- hl.animation({ leaf = "hyprfocusIn",  enabled = true, speed = 1.7, bezier = "md3_accel" })
-- hl.animation({ leaf = "hyprfocusOut", enabled = true, speed = 1.7, bezier = "md3_accel" })


-- "Smart gaps" / "No gaps when only"
hl.workspace_rule({ workspace = "w[tv1]", gaps_out = 0, gaps_in = 0 })
hl.workspace_rule({ workspace = "f[1]",   gaps_out = 0, gaps_in = 0 })

hl.window_rule({
    name  = "no-gaps-wtv1-border",
    match = { float = false, workspace = "w[tv1]s[false]" },
    border_size = 0,
})
hl.window_rule({
    name  = "no-gaps-wtv1-rounding",
    match = { float = false, workspace = "w[tv1]s[false]" },
    rounding = 0,
})
hl.window_rule({
    name  = "no-gaps-f1-border",
    match = { float = false, workspace = "f[1]s[false]" },
    border_size = 0,
})
hl.window_rule({
    name  = "no-gaps-f1-rounding",
    match = { float = false, workspace = "f[1]s[false]" },
    rounding = 0,
})

-- Gromit-mpx annotation special workspace
hl.workspace_rule({ workspace = "special:gromit", gaps_in = 0, gaps_out = 0, on_created_empty = "gromit-mpx -a" })


hl.window_rule({
    name  = "pip",
    match = { title = "^(Picture in Picture|Picture-in-Picture)$" },
    float       = true,
    opacity     = "1.0 1.0",
    no_blur     = true,
    border_size = 0,
    no_dim      = true,
    no_anim     = true,
    no_shadow   = true,
    rounding    = 0,
    pin         = true,
})

hl.window_rule({
    name  = "system-utils-float",
    match = { class = "^(org.twosheds.iwgtk|blueman-manager|com.saivert.pwvucontrol|org.pulseaudio.pavucontrol|iwgtk)$" },
    float = true,
})

hl.window_rule({
    name  = "tag-browser",
    match = { class = "^(zen|chromium|Chromium|firefox|Opera|vivaldi-stable)$" },
    tag   = "browser",
})

hl.window_rule({
    name  = "terminal",
    match = { class = "(Alacritty|kitty|com.mitchellh.ghostty)" },
    tag   = "terminal",
})

hl.window_rule({
    name  = "starship",
    match = { tag = "^(starship)$" },
    float           = true,
    center          = true,
    no_screen_share = true,
    size            = "1024 768",
})

hl.window_rule({
    name  = "copyq",
    match = { class = "^(copyq)$" },
    float          = true,
    center         = true,
    no_blur        = true,
    no_dim         = true,
    no_screen_share = true,
    dim_around     = true,
    suppress_event = "fullscreen",
    opacity        = "0.95 0.95",
    animation      = "popin 85%",
    rounding       = 10,
    border_size    = 0,
    stay_focused   = true,
})

hl.window_rule({
    name    = "zed",
    match   = { class = "^(dev\\.zed\\.Zed)$" },
    opacity = "0.98 0.98",
})

hl.window_rule({
    name  = "sensitive-apps-noscreenshare",
    match = { class = "^(discord|Proton Mail|Bitwarden|1Password|org.gnome.seahorse.Application)$" },
    no_screen_share = true,
})

hl.window_rule({
    name  = "vlc-float",
    match = { class = "^(vlc)$" },
    float = true,
})

hl.window_rule({
    name           = "suppress-event-maximize",
    match          = { class = ".*" },
    suppress_event = "maximize",
})

hl.window_rule({
    name    = "firefox-no-blur",
    match   = { class = "^(firefox)$" },
    no_blur = true,
})

hl.window_rule({
    name  = "nofocus-empty-xwayland",
    match = {
        class      = "^$",
        title      = "^$",
        xwayland   = true,
        fullscreen = false,
    },
    no_focus = true,
})

hl.window_rule({
    name  = "slack-pin",
    match = { class = "^(Slack)$", title = "^(Slack)$" },
    pin   = true,
})

hl.window_rule({
    name  = "flameshot",
    match = { class = "flameshot" },
    float       = true,
    monitor     = 0,
    move        = "0 0",
    no_anim     = true,
    border_size = 0,
    rounding    = 0,
})

hl.window_rule({
    name  = "meeting",
    match = { tag = "^(meeting)$" },
    float          = true,
    no_blur        = true,
    opacity        = "1.0 1.0",
    border_size    = 0,
    no_dim         = true,
    no_shadow      = true,
    rounding       = 0,
    suppress_event = "fullscreen",
    animation      = "popin",
})

hl.window_rule({
    name  = "xwayland-video-bridge-fixes",
    match = { class = "xwaylandvideobridge" },
    no_initial_focus = true,
    no_focus         = true,
    no_anim          = true,
    no_blur          = true,
    max_size         = "1 1",
    opacity          = "0.0",
})

-- JetBrains IDE fixes
-- https://github.com/hyprwm/Hyprland/issues/1947#issuecomment-2690914693
hl.window_rule({
    name  = "jetbrains-float-focus",
    match = { class = "^(jetbrains-.*)$", float = true },
    focus_on_activate = true,
})

hl.window_rule({
    name  = "jetbrains-fix-splashscreen-focus-takeovers",
    match = { class = "^(jetbrains-.*)$", title = "^(splash)$", float = true },
    float       = true,
    no_focus    = true,
    border_size = 0,
})

hl.window_rule({
    name  = "jetbrains-fix-popups",
    match = { class = "^(jetbrains-.*)$", title = "^( )$", float = true },
    center       = true,
    stay_focused = true,
    border_size  = 0,
})

hl.window_rule({
    name  = "jetbrains-fix-autocomplete-flicker",
    match = { class = "^(jetbrains-.*)$", title = "^(win.*)$", float = true },
    no_initial_focus = true,
})

hl.window_rule({
    name  = "xdg-desktop-portal-gtk",
    match = { class = "^([Xx]dg-desktop-portal-gtk)$" },
    float       = true,
    center      = true,
    no_blur     = true,
    rounding    = 0,
    border_size = 0,
    no_shadow   = true,
    size        = "900 600",
})

hl.window_rule({
    name  = "electron-file-dialogs",
    match = { class = "^(electron|chromium|slack)$", title = "^(Save (As|File)|Open Files?)$" },
    float       = true,
    center      = true,
    no_blur     = true,
    border_size = 0,
    size        = "900 600",
})

hl.window_rule({
    name  = "localsend-dialogs",
    match = { class = "^(localsend)$", title = "^(Open File|Choose Directory)$" },
    float  = true,
    center = true,
})

hl.window_rule({
    name  = "firefox-empty-title",
    match = { class = "^(zen|firefox)$", title = "^$" },
    float  = true,
    center = true,
    size   = "900 600",
})

hl.window_rule({
    name  = "firefox-about",
    match = { class = "^(zen|firefox)$", title = "^About Mozilla Firefox\\b" },
    float  = true,
    center = true,
    size   = "900 600",
})

hl.window_rule({
    name  = "slack-images",
    match = { class = "^(Slack|slack)$", title = "\\.(png|PNG|jpg|JPG|jpeg|JPEG)$" },
    float  = true,
    center = true,
})


hl.layer_rule({
    name  = "swaync-control-center",
    match = { namespace = "swaync-control-center" },
    blur       = true,
    dim_around = true,
})

hl.layer_rule({
    name  = "swaync-notification-window",
    match = { namespace = "swaync-notification-window" },
    no_screen_share = true,
})

hl.layer_rule({
    name  = "rofi",
    match = { namespace = "rofi" },
    dim_around = true,
})


-- Plugin config (uncomment when plugins are loaded)
-- hl.config({
--     plugin = {
--         hyprwinwrap = {
--             class = "kitty-bg",
--             title = "kitty-bg",
--         },
--         hyprexpo = {
--             columns          = 3,
--             gap_size         = 5,
--             bg_col           = "rgba(3b42522a)",
--             workspace_method = "first 1",
--             skip_empty       = true,
--         },
--     },
-- })


hl.config({
    dwindle = {
        preserve_split              = true,
        smart_split                 = true,
        permanent_direction_override = true,
    },

    master = {
        new_status = "master",
    },

    scrolling = {
        fullscreen_on_one_column = true,
        column_width             = 0.9,
        direction                = "right",
    },
})


hl.config({
    misc = {
        focus_on_activate      = true,
        disable_splash_rendering = true,
        force_default_wallpaper  = 0,
        disable_hyprland_logo    = false,
    },

    binds = {
        scroll_event_delay = 50,
    },
})


hl.config({
    input = {
        kb_layout  = "se",
        kb_variant = "",
        kb_model   = "",
        kb_options = "",
        kb_rules   = "",

        follow_mouse = 1,
        sensitivity  = 0,

        touchpad = {
            natural_scroll = false,
        },
    },
})

hl.device({ name = "omkbd-ergodash-rev1.2", kb_layout = "eu" })
hl.device({ name = "dygma-defy-keyboard",   kb_layout = "eu" })
hl.device({ name = "dygma-defy-keyboard-1", kb_layout = "eu" })


local mainMod = "SUPER"

-- Scrolling layout binds
hl.bind(mainMod .. " + period",         hl.dsp.layout("move +col"))
hl.bind(mainMod .. " + comma",          hl.dsp.layout("move -col"))
hl.bind(mainMod .. " + SHIFT + period", hl.dsp.layout("swapcol r"))
hl.bind(mainMod .. " + SHIFT + comma",  hl.dsp.layout("swapcol l"))

-- Core binds
hl.bind(mainMod .. " + Return",       hl.dsp.exec_cmd(terminal))
hl.bind("SUPER + SHIFT + Q",          hl.dsp.window.close())

-- Kitty quick-access terminal
hl.bind(mainMod .. " + TAB",          hl.dsp.exec_cmd("kitten quick-access-terminal"))

-- CopyQ: pass keys when CopyQ is focused
hl.bind(mainMod .. " + V",            hl.dsp.pass({ window = "class:^(copyq)$" }))
hl.bind(mainMod .. " + F",            hl.dsp.pass({ window = "class:^(copyq)$" }))

-- Normal behavior for non-CopyQ windows
hl.bind(mainMod .. " + B",            hl.dsp.window.float({ action = "toggle" }))
hl.bind(mainMod .. " + F",            hl.dsp.window.fullscreen())

hl.bind("ALT + TAB",                  hl.dsp.focus({ urgent_or_last = true }))

hl.bind(mainMod .. " + K",            hl.dsp.exec_cmd([[swayosd-client --custom-message="$(date '+%H:%M %A')"]]))
hl.bind(mainMod .. " + T",            hl.dsp.exec_cmd("sleep 0.1 && swaync-client -t -sw"))

hl.bind(mainMod .. " + D",            hl.dsp.exec_cmd(menu))
hl.bind(mainMod .. " + P",            hl.dsp.window.pseudo())
hl.bind(mainMod .. " + E",            hl.dsp.layout("togglesplit"))

hl.bind(mainMod .. " + R",            hl.dsp.exec_cmd("/home/dln/go/bin/hyprtabs"))

hl.bind(mainMod .. " + L",            hl.dsp.exec_cmd("/home/dln/.dotfiles/bin/lock"))
hl.bind(mainMod .. " + O",            hl.dsp.exec_cmd("/home/dln/.dotfiles/bin/wofi-ykman"))
hl.bind("SUPER + SHIFT + V",          hl.dsp.exec_cmd("QT_QPA_PLATFORM=xcb copyq show"))
hl.bind("Print",                       hl.dsp.exec_cmd("sh -c 'flameshot gui --clipboard'"))
hl.bind(mainMod .. " + Y",            hl.dsp.window.pin())

-- Clamshell mode
hl.bind(mainMod .. " + code:108",     hl.dsp.exec_cmd("/home/dln/.dotfiles/bin/clamshell"))
hl.bind(mainMod .. " + code:34",      hl.dsp.exec_cmd("/home/dln/.dotfiles/bin/clamshell"))

-- Lid switch
hl.bind("switch:off:Lid Switch", hl.dsp.exec_cmd("/home/dln/.dotfiles/bin/clamshell open"),  { locked = true })
hl.bind("switch:on:Lid Switch",  hl.dsp.exec_cmd("/home/dln/.dotfiles/bin/clamshell close"), { locked = true })

-- Move focus with arrow keys
hl.bind(mainMod .. " + left",  hl.dsp.focus({ direction = "left" }))
hl.bind(mainMod .. " + right", hl.dsp.focus({ direction = "right" }))
hl.bind(mainMod .. " + up",    hl.dsp.focus({ direction = "up" }))
hl.bind(mainMod .. " + down",  hl.dsp.focus({ direction = "down" }))

-- Switch workspaces with mainMod + [0-9]
-- Move active window to a workspace with mainMod + SHIFT + [0-9]
for i = 1, 10 do
    local key = i % 10
    hl.bind(mainMod .. " + " .. key,           hl.dsp.focus({ workspace = i }))
    hl.bind(mainMod .. " + SHIFT + " .. key,   hl.dsp.window.move({ workspace = i }))
end

-- Scroll through workspaces
hl.bind(mainMod .. " + mouse_down", hl.dsp.focus({ workspace = "e+1" }))
hl.bind(mainMod .. " + mouse_up",  hl.dsp.focus({ workspace = "e-1" }))

-- Cursor zoom with Ctrl+Super + scroll
-- Lua parser disables `hyprctl keyword`; set the option in-VM via `hyprctl eval` instead.
local zoomIn  = [[hyprctl -q eval 'local z = hl.get_config("cursor:zoom_factor") * 1.1 hl.config({ cursor = { zoom_factor = z } })']]
local zoomOut = [[hyprctl -q eval 'local z = hl.get_config("cursor:zoom_factor") * 0.9 if z < 1 then z = 1 end hl.config({ cursor = { zoom_factor = z } })']]
local zoomReset = [[hyprctl -q eval 'hl.config({ cursor = { zoom_factor = 1 } })']]

hl.bind("SUPER + CTRL + mouse_up",   hl.dsp.exec_cmd(zoomIn))
hl.bind("SUPER + CTRL + mouse_down", hl.dsp.exec_cmd(zoomOut))
hl.bind(mainMod .. " + HOME",      hl.dsp.exec_cmd(zoomReset))
hl.bind(mainMod .. " + minus",     hl.dsp.exec_cmd(zoomOut), { repeating = true })

-- Move/resize windows with mouse
hl.bind(mainMod .. " + mouse:272", hl.dsp.window.drag(),   { mouse = true })
hl.bind(mainMod .. " + mouse:273", hl.dsp.window.resize(), { mouse = true })

-- OSD client helper (focused monitor)
local osdclient = [[swayosd-client --monitor "$(hyprctl monitors -j | jq -r '.[] | select(.focused == true).name')"]]

-- Volume and brightness (locked + repeating + description)
hl.bind("XF86AudioRaiseVolume",  hl.dsp.exec_cmd(osdclient .. " --output-volume raise"),  { locked = true, repeating = true, description = "Volume up" })
hl.bind("XF86AudioLowerVolume",  hl.dsp.exec_cmd(osdclient .. " --output-volume lower"),  { locked = true, repeating = true, description = "Volume down" })
hl.bind("XF86AudioMute",         hl.dsp.exec_cmd(osdclient .. " --output-volume mute-toggle"), { locked = true, repeating = true, description = "Mute" })
hl.bind("XF86AudioMicMute",      hl.dsp.exec_cmd(osdclient .. " --input-volume mute-toggle"),  { locked = true, repeating = true, description = "Mute microphone" })
hl.bind("XF86MonBrightnessUp",   hl.dsp.exec_cmd(osdclient .. " --brightness raise"),     { locked = true, repeating = true, description = "Brightness up" })
hl.bind("XF86MonBrightnessDown", hl.dsp.exec_cmd(osdclient .. " --brightness lower"),     { locked = true, repeating = true, description = "Brightness down" })

-- Precise 1% adjustments with Alt modifier
hl.bind("ALT + XF86AudioRaiseVolume",  hl.dsp.exec_cmd(osdclient .. " --output-volume +1"),  { locked = true, repeating = true, description = "Volume up precise" })
hl.bind("ALT + XF86AudioLowerVolume",  hl.dsp.exec_cmd(osdclient .. " --output-volume -1"),  { locked = true, repeating = true, description = "Volume down precise" })
hl.bind("ALT + XF86MonBrightnessUp",   hl.dsp.exec_cmd(osdclient .. " --brightness +1"),     { locked = true, repeating = true, description = "Brightness up precise" })
hl.bind("ALT + XF86MonBrightnessDown", hl.dsp.exec_cmd(osdclient .. " --brightness -1"),     { locked = true, repeating = true, description = "Brightness down precise" })

-- Media keys (locked + description)
hl.bind("XF86AudioNext",  hl.dsp.exec_cmd(osdclient .. " --playerctl next"),       { locked = true, description = "Next track" })
hl.bind("XF86AudioPause", hl.dsp.exec_cmd(osdclient .. " --playerctl play-pause"), { locked = true, description = "Pause" })
hl.bind("XF86AudioPlay",  hl.dsp.exec_cmd(osdclient .. " --playerctl play-pause"), { locked = true, description = "Play" })
hl.bind("XF86AudioPrev",  hl.dsp.exec_cmd(osdclient .. " --playerctl previous"),   { locked = true, description = "Previous track" })


-- bind = SUPER, g, hyprexpo:expo, toggle
-- bind = $mainMod, code:191, hyprexpo:expo, toggle
-- bind = , code:191, hyprexpo:expo, toggle
-- bindm = ALT, mouse:277, hyprexpo:expo toggle
-- bindel = , code:201, hyprexpo:expo, toggle
-- bindel = SUPER SHIFT, XF86Assistant, hyprexpo:expo, toggle
-- bind = , XF86Assistant, hyprexpo:expo, toggle
-- bindeld = ,XF86Assistant, Foo, hyprexpo:expo, toggle
-- bind = $mainMod, z, easymotion, action:hyprctl dispatch focuswindow address:{}
