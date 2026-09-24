-- ultradwindle: Lua port of Hyprland's built-in dwindle layout
-- (src/layout/algorithm/tiled/dwindle/DwindleAlgorithm.cpp).
--
-- Usage in hyprland.lua:
--   require("ultradwindle")                -- this file somewhere on package.path
--   hl.config({ general = { layout = "lua:ultradwindle" } })
--
-- Reads the regular dwindle:* config options, so existing dwindle settings apply.
--
-- layoutmsg commands (same as dwindle, plus movewindow):
--   togglesplit, swapsplit, rotatesplit [angle], movetoroot [window] [unstable],
--   preselect <u|d|l|r>, splitratio <delta> [exact], movewindow <u|d|l|r>,
--   togglecenter (lone window: centered SINGLE_WINDOW_ASPECT box <-> full area)

-- Per-workspace state. The provider table is shared by every workspace using
-- the layout, while C++ dwindle gets one algorithm instance per workspace.
local spaces = {}

-- A lone tiled window gets a centered box of this aspect ratio (width / height)
-- instead of the whole area. Toggle per workspace with the "togglecenter" layoutmsg. Only shrinks width, so non-ultrawide monitors are unaffected.
local SINGLE_WINDOW_ASPECT = 16 / 9

local function clamp(x, lo, hi)
    return math.max(lo, math.min(hi, x))
end

local function make_box(x, y, w, h)
    return { x = x, y = y, w = math.max(0, w), h = math.max(0, h) }
end

local function copy_box(b)
    return { x = b.x, y = b.y, w = b.w, h = b.h }
end

-- Config -------------------------------------------------------------------

local function cfg(key, default)
    local ok, v = pcall(hl.get_config, "dwindle:" .. key)
    if not ok or v == nil then
        return default
    end
    return v
end

local function cfg_bool(key, default)
    local v = cfg(key, default)
    return v == true or (type(v) == "number" and v ~= 0)
end

local function read_opts()
    return {
        preserve_split = cfg_bool("preserve_split", false),
        smart_split = cfg_bool("smart_split", false),
        precise_mouse_move = cfg_bool("precise_mouse_move", false),
        permanent_direction_override = cfg_bool("permanent_direction_override", false),
        use_active_for_splits = cfg_bool("use_active_for_splits", true),
        split_width_multiplier = tonumber(cfg("split_width_multiplier", 1.0)) or 1.0,
        default_split_ratio = tonumber(cfg("default_split_ratio", 1.0)) or 1.0,
        force_split = tonumber(cfg("force_split", 0)) or 0,
        split_bias = tonumber(cfg("split_bias", 0)) or 0,
    }
end

-- Helpers --------------------------------------------------------------------

local function target_id(target)
    local window = target.window
    return window and tostring(window.stable_id) or ("index:" .. target.index)
end

local function is_fullscreen(window)
    return window ~= nil and (window.fullscreen or 0) ~= 0
end

local function cursor_pos()
    local ok, pos = pcall(hl.get_cursor_pos)
    if ok and pos then
        return pos
    end
    return { x = 0, y = 0 }
end

local function space_for(ctx)
    local key = "default"
    for _, target in ipairs(ctx.targets) do
        local window = target.window
        local ws = window and window.workspace
        if ws and ws.id then
            key = tostring(ws.id)
            break
        end
    end

    local sp = spaces[key]
    if not sp then
        -- leaves: id -> leaf node, order: ids in ctx.targets order (for swap detection)
        sp = { root = nil, leaves = {}, order = {}, last_active = nil, override_dir = nil, center_single = true }
        spaces[key] = sp
    end
    return sp
end

-- squared distance from point to rectangle, like vecToRectDistanceSquared
local function dist_to_box_sq(p, b)
    local dx = math.max(b.x - p.x, 0, p.x - (b.x + b.w))
    local dy = math.max(b.y - p.y, 0, p.y - (b.y + b.h))
    return dx * dx + dy * dy
end

local function box_contains(b, p)
    return p.x >= b.x and p.x < b.x + b.w and p.y >= b.y and p.y < b.y + b.h
end

local function closest_leaf(sp, point)
    local best, best_dist
    for _, id in ipairs(sp.order) do
        local leaf = sp.leaves[id]
        if leaf and leaf.box then
            local d = dist_to_box_sq(point, leaf.box)
            if not best or d < best_dist then
                best, best_dist = leaf, d
            end
        end
    end
    return best
end

local function first_leaf(sp)
    for _, id in ipairs(sp.order) do
        if sp.leaves[id] then
            return sp.leaves[id]
        end
    end
end

local function replace_child(parent, old, new)
    if parent.children[1] == old then
        parent.children[1] = new
    else
        parent.children[2] = new
    end
end

-- Tree operations --------------------------------------------------------------

-- Box for the root node: the whole area, or a SINGLE_WINDOW_ASPECT box when the
-- root is a single window. The width comes from the full monitor height and the box
-- is centered on the monitor, so it lines up with a centered bar of the same width
-- (e.g. waybar "width": 2560 on 3440x1440) instead of shrinking by the bar's reserved area.
local function root_box(sp, area)
    if sp.root.children or not sp.center_single then
        return copy_box(area)
    end

    local center, height = area.x + area.w / 2, area.h
    local mon = sp.monitor
    if mon then
        local scale = mon.scale or 1
        center = mon.x + mon.width / scale / 2
        height = mon.height / scale
    end

    local w = math.min(area.w, height * SINGLE_WINDOW_ASPECT)
    local x = clamp(center - w / 2, area.x, area.x + area.w - w)
    return make_box(x, area.y, w, area.h)
end

-- SDwindleNodeData::recalcSizePosRecursive, minus the placing (done afterwards)
local function layout_node(node, opts)
    if not node.children then
        return
    end

    if not opts.preserve_split and not opts.smart_split and not opts.precise_mouse_move then
        node.split_top = node.box.h * opts.split_width_multiplier > node.box.w
    end

    local b = node.box
    local a, c = node.children[1], node.children[2]

    if not node.split_top then
        local first = b.w / 2.0 * node.ratio
        a.box = make_box(b.x, b.y, first, b.h)
        c.box = make_box(b.x + first, b.y, b.w - first, b.h)
    else
        local first = b.h / 2.0 * node.ratio
        a.box = make_box(b.x, b.y, b.w, first)
        c.box = make_box(b.x, b.y + first, b.w, b.h - first)
    end

    layout_node(a, opts)
    layout_node(c, opts)
end

-- CDwindleAlgorithm::addTarget. focal is set when moving a window (movewindow).
-- Callers keep sp.order up to date.
local function add_leaf(sp, id, opts, area, focal)
    local node = { id = id, ratio = 1.0, split_top = false }
    local mouse = focal or cursor_pos()

    local opening
    if focal then
        opening = closest_leaf(sp, focal)
    elseif opts.use_active_for_splits then
        opening = (sp.last_active and sp.leaves[sp.last_active]) or closest_leaf(sp, mouse)
    else
        for _, leafid in ipairs(sp.order) do
            local leaf = sp.leaves[leafid]
            if leaf and leaf.box and box_contains(leaf.box, mouse) then
                opening = leaf
                break
            end
        end
    end

    -- fail-safe, like the "avoid duplicate fullscreens" fallback
    opening = opening or first_leaf(sp)

    sp.leaves[id] = node

    -- first window: take the whole area
    if not opening then
        node.box = copy_box(area)
        sp.root = node
        return
    end

    local parent = {
        box = copy_box(opening.box),
        parent = opening.parent,
        ratio = clamp(opts.default_split_ratio, 0.1, 1.9),
    }

    local new_first
    local horizontal_override, vertical_override = false, false
    local dir = sp.override_dir

    -- ultradwindle: the first split on a workspace (splitting a lone window) is
    -- always side by side, unless a direction was preselected
    local first_split = not opening.parent and not dir

    local side_by_side = first_split or parent.box.w > parent.box.h * opts.split_width_multiplier
    parent.split_top = not side_by_side

    if dir then
        if dir == "u" or dir == "d" then
            vertical_override = true
        else
            horizontal_override = true
        end

        new_first = dir == "u" or dir == "l"

        if not opts.permanent_direction_override then
            sp.override_dir = nil
        end
    elseif opts.smart_split then
        -- ultradwindle: tile shape picks the direction (spiral), the cursor only
        -- picks the side. C++ dwindle uses the cursor for both.
        local b = parent.box
        if side_by_side then
            new_first = not (mouse.x - (b.x + b.w / 2) > 0)
        else
            new_first = not (mouse.y - (b.y + b.h / 2) > 0)
        end
    elseif opts.force_split == 0 or focal then
        local b = parent.box
        new_first = (side_by_side and mouse.x < b.x + b.w / 2) or (not side_by_side and mouse.y < b.y + b.h / 2)
    elseif opts.force_split == 1 then
        new_first = true
    else
        new_first = false
    end

    parent.children = new_first and { node, opening } or { opening, node }

    if opts.split_bias ~= 0 and new_first then
        parent.ratio = 2.0 - parent.ratio
    end

    if opening.parent then
        replace_child(opening.parent, opening, parent)
    else
        sp.root = parent
    end

    opening.parent = parent
    node.parent = parent

    if vertical_override then
        parent.split_top = true
    elseif horizontal_override or first_split then
        parent.split_top = false
    end
end

-- CDwindleAlgorithm::removeTarget
local function remove_leaf(sp, id)
    local node = sp.leaves[id]
    if not node then
        return
    end
    sp.leaves[id] = nil

    local parent = node.parent
    if not parent then
        sp.root = nil
        return
    end

    local sibling = parent.children[1] == node and parent.children[2] or parent.children[1]
    sibling.parent = parent.parent

    if parent.parent then
        replace_child(parent.parent, parent, sibling)
    else
        sp.root = sibling
    end
end

-- Bring the tree in line with ctx.targets. The Lua layout API only exposes the
-- current target list, so additions, removals and swaps are diffed here.
local function sync(sp, ctx, opts)
    local present, targets = {}, {}
    local active_id
    sp.monitor = nil -- for root_box

    for _, target in ipairs(ctx.targets) do
        local id = target_id(target)
        present[id] = true
        targets[id] = target
        if target.window and target.window.active then
            active_id = id
        end
        sp.monitor = sp.monitor or (target.window and target.window.monitor)
    end

    -- removals
    for id in pairs(sp.leaves) do
        if not present[id] then
            remove_leaf(sp, id)
        end
    end

    -- swaps: swapwindow reorders ctx.targets. Map the new relative order of the
    -- known ids onto the leaves they held before.
    local prev = {}
    for _, id in ipairs(sp.order) do
        if sp.leaves[id] then
            table.insert(prev, id)
        end
    end

    local cur = {}
    for _, target in ipairs(ctx.targets) do
        local id = target_id(target)
        if sp.leaves[id] then
            table.insert(cur, id)
        end
    end

    local nodes = {}
    for k, id in ipairs(prev) do
        nodes[k] = sp.leaves[id]
    end
    for k, id in ipairs(cur) do
        nodes[k].id = id
        sp.leaves[id] = nodes[k]
    end
    sp.order = cur

    -- the focused window, if already tiled, is the split target for new windows
    if active_id and sp.leaves[active_id] then
        sp.last_active = active_id
    end

    -- additions
    for _, target in ipairs(ctx.targets) do
        local id = target_id(target)
        if not sp.leaves[id] then
            add_leaf(sp, id, opts, ctx.area)
            table.insert(sp.order, id)
            -- lay out right away so the next insertion sees real boxes
            sp.root.box = root_box(sp, ctx.area)
            layout_node(sp.root, opts)
        end
    end

    return targets
end

local function active_leaf(sp, ctx)
    for _, target in ipairs(ctx.targets) do
        local window = target.window
        if window and window.active then
            return sp.leaves[target_id(target)], window
        end
    end
end

-- Layout messages ----------------------------------------------------------------

local function toggle_split(node, window)
    if not node or not node.parent or is_fullscreen(window) then
        return false
    end
    node.parent.split_top = not node.parent.split_top
    return true
end

local function swap_split(node, window)
    if not node or not node.parent or is_fullscreen(window) then
        return false
    end
    local c = node.parent.children
    c[1], c[2] = c[2], c[1]
    return true
end

local function rotate_split(node, window, angle)
    if not node or not node.parent or is_fullscreen(window) then
        return
    end

    local quarter = angle / 90
    quarter = quarter >= 0 and math.floor(quarter) or math.ceil(quarter) -- C++ int division
    local normalized = quarter % 4

    local parent = node.parent
    local should_swap = false

    if normalized == 1 then
        should_swap = parent.split_top
        parent.split_top = not parent.split_top
    elseif normalized == 2 then
        should_swap = true
    elseif normalized == 3 then
        should_swap = not parent.split_top
        parent.split_top = not parent.split_top
    end

    if should_swap then
        parent.children[1], parent.children[2] = parent.children[2], parent.children[1]
    end
end

local function move_to_root(node, window, stable)
    if not node or not node.parent or is_fullscreen(window) then
        return false
    end

    -- already at root
    if not node.parent.parent then
        return false
    end

    local parent = node.parent
    local ancestor, root = node, parent
    while root.parent do
        ancestor = root
        root = root.parent
    end

    local node_slot = parent.children[1] == node and 1 or 2
    local swap_slot = root.children[1] == ancestor and 2 or 1
    local swap = root.children[swap_slot]

    parent.children[node_slot] = swap
    root.children[swap_slot] = node
    swap.parent = parent
    node.parent = root

    -- stable: the focused window keeps its side of the screen
    if stable then
        root.children[1], root.children[2] = root.children[2], root.children[1]
    end

    return true
end

-- CDwindleAlgorithm::moveTargetInDirection, within the workspace only
local function move_window(sp, node, dir, opts, area)
    if not node then
        return
    end

    local b = node.box
    local focal
    if dir == "u" then
        focal = { x = b.x + b.w / 2, y = b.y - 1 }
    elseif dir == "d" then
        focal = { x = b.x + b.w / 2, y = b.y + b.h + 1 }
    elseif dir == "l" then
        focal = { x = b.x - 1, y = b.y + b.h / 2 }
    else
        focal = { x = b.x + b.w + 1, y = b.y + b.h / 2 }
    end

    -- C++ moves to another monitor here; not possible from the Lua API
    if not box_contains(area, focal) then
        return
    end

    -- moving toward the nearest divider with a single window partner:
    -- force the window onto the far side of that partner
    local p = node.parent
    if p then
        local c = p.children
        if (dir == "u" and p.split_top and c[2] == node and not c[1].children)
            or (dir == "d" and p.split_top and c[1] == node and not c[2].children)
            or (dir == "l" and not p.split_top and c[2] == node and not c[1].children)
            or (dir == "r" and not p.split_top and c[1] == node and not c[2].children)
        then
            sp.override_dir = dir
        end
    end

    -- sp.order stays as is: ctx.targets order does not change on a move
    local id = node.id
    remove_leaf(sp, id)
    if sp.root then
        sp.root.box = root_box(sp, area)
        layout_node(sp.root, opts)
    end
    add_leaf(sp, id, opts, area, focal)
end

local function parse_dir(s)
    local c = s and s:sub(1, 1) or ""
    if c == "u" or c == "t" then
        return "u"
    elseif c == "d" or c == "b" then
        return "d"
    elseif c == "l" then
        return "l"
    elseif c == "r" then
        return "r"
    end
end

-- Registration -------------------------------------------------------------------

hl.layout.register("ultradwindle", {
    recalculate = function(ctx)
        if #ctx.targets == 0 then
            return
        end

        local opts = read_opts()
        local sp = space_for(ctx)
        local targets = sync(sp, ctx, opts)

        -- dwindle leaves positioning to the fullscreen handler while a window is fullscreen
        for _, target in pairs(targets) do
            if is_fullscreen(target.window) then
                return
            end
        end

        if not sp.root then
            return
        end

        sp.root.box = root_box(sp, ctx.area)
        layout_node(sp.root, opts)

        for id, leaf in pairs(sp.leaves) do
            local target = targets[id]
            if target then
                target:place(leaf.box)
            end
        end
    end,

    layout_msg = function(ctx, msg)
        local opts = read_opts()
        local sp = space_for(ctx)
        sync(sp, ctx, opts)

        local args = {}
        for word in msg:gmatch("%S+") do
            table.insert(args, word)
        end

        local command = args[1]
        local node, window = active_leaf(sp, ctx)

        if command == "togglesplit" then
            if node and not toggle_split(node, window) then
                return "can't togglesplit in the current workspace"
            end
        elseif command == "swapsplit" then
            if node and not swap_split(node, window) then
                return "can't swapsplit in the current workspace"
            end
        elseif command == "rotatesplit" then
            local angle = 90
            if args[2] then
                angle = tonumber(args[2])
                if not angle then
                    return "Invalid angle argument"
                end
            end
            rotate_split(node, window, angle)
        elseif command == "movetoroot" then
            local stable = args[3] ~= "unstable"
            if args[2] then
                local ok, w = pcall(hl.get_window, args[2])
                if ok and w then
                    node, window = sp.leaves[tostring(w.stable_id)], w
                end
            end
            if not move_to_root(node, window, stable) then
                return "can't movetoroot in the current workspace"
            end
        elseif command == "preselect" then
            if not args[2] then
                return "No direction for preselect"
            end
            -- any other character resets the direction (for permanent_direction_override)
            sp.override_dir = parse_dir(args[2])
        elseif command == "splitratio" then
            if not args[2] then
                return "splitratio requires an arg"
            end
            local delta = tonumber(args[2])
            if not node or not node.parent then
                return "cannot alter split ratio on no / single node"
            end
            if not delta then
                return string.format('failed to parse "%s" as a delta', args[2])
            end
            local exact = args[3] ~= nil and args[3]:sub(1, 5) == "exact"
            local ratio = exact and delta or (node.parent.ratio + delta)
            node.parent.ratio = clamp(ratio, 0.1, 1.9)
        elseif command == "togglecenter" then
            sp.center_single = not sp.center_single
        elseif command == "movewindow" then
            local dir = parse_dir(args[2])
            if not dir then
                return "movewindow expects a direction: u, d, l or r"
            end
            move_window(sp, node, dir, opts, ctx.area)
        else
            return "Unknown ultradwindle layoutmsg: " .. msg
        end

        -- Hyprland calls recalculate after layout_msg
        return true
    end,
})
