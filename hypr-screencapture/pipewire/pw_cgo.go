package pipewire

/*
#cgo pkg-config: libpipewire-0.3
#include "pw_bridge.h"
*/
import "C"

import (
	"fmt"
	"strings"
	"time"
	"unsafe"

	"github.com/dlnilsson/dotfiles/hypr-screencapture/messages"
)

var msgChannel chan<- messages.Msg

func SetChannel(ch chan<- messages.Msg) {
	msgChannel = ch
}

func StartPipeWireLoop() int { return int(C.pw_run_loop()) }

func StopPipeWireLoop() { C.pw_quit_loop() }

// ----- Go-exported functions called from C (must be in this file) -----

//export goOnNodeAdd
func goOnNodeAdd(id C.uint32_t, mediaClass, name, desc, app, portalApp *C.char) {
	mc := C.GoString(mediaClass)
	nm := C.GoString(name)
	ds := C.GoString(desc)
	ap := C.GoString(app)
	pa := C.GoString(portalApp)

	onNodeAdd(uint32(id), mc, nm, ds, ap, pa)
	// Note: memory owned by C; no need to free here.
	_ = unsafe.Pointer(mediaClass) // pacify linters about cgo pointers
}

//export goOnNodeRemove
func goOnNodeRemove(id C.uint32_t) {
	onNodeRemove(uint32(id))
}

type nodeProps struct {
	MediaClass string
	Name       string
	DescLower  string
	AppLower   string
	PortalApp  string
}

var known = map[uint32]nodeProps{}

func ts() string { return time.Now().Format("2006-01-02 15:04:05") }

func isScreenshareNode(mc, name, descLower, appLower string) bool {
	// fmt.Printf("mc %+v name %+v descLower %+v appLower %+v\n", mc, name, descLower, appLower)
	// Ignore obvious physical cameras (v4l2)
	if strings.HasPrefix(name, "v4l2_input.") || strings.Contains(descLower, "v4l2") || strings.Contains(descLower, "camera") {
		return false
	}
	// Portal-produced virtual screen source (common names)
	if mc == "Video/Source" && (name == "xdg-desktop-portal-hyprland" ||
		name == "xdg-desktop-portal-wlr" ||
		name == "xdg-desktop-portal-gnome" ||
		name == "xdg-desktop-portal-kde") {
		return true
	}
	// App’s capture sink (browser/Zoom receives)
	if mc == "Stream/Input/Video" && appLower != "" && appLower != "pipewire" {
		return true
	}
	// Fallback: generic non-camera virtual source
	if mc == "Video/Source" &&
		!strings.Contains(descLower, "camera") &&
		!strings.HasPrefix(name, "v4l2_input.") {
		return true
	}
	return false
}

/* Called by pw_cgo.go (cgo callbacks) */
func onNodeAdd(id uint32, mediaClass, name, desc, app, portalApp string) {
	dsLower := strings.ToLower(desc)
	apLower := strings.ToLower(app)

	if isScreenshareNode(mediaClass, name, dsLower, apLower) {
		known[id] = nodeProps{
			MediaClass: mediaClass,
			Name:       name,
			DescLower:  dsLower,
			AppLower:   apLower,
			PortalApp:  portalApp,
		}

		if msgChannel != nil {
			select {
			case msgChannel <- messages.Msg{Kind: messages.ScOn}:
			default:
			}
		} else {
			fmt.Printf("%s [pw] START id=%d media.class=%q name=%q app=%q portal=%q\n",
				ts(), id, mediaClass, name, app, portalApp)
		}
	}
}

func onNodeRemove(id uint32) {
	if props, ok := known[id]; ok {
		delete(known, id)

		if msgChannel != nil {
			select {
			case msgChannel <- messages.Msg{Kind: messages.ScOff}:
			default:
			}
		} else {
			fmt.Printf("%s [pw] STOP  id=%d name=%q media.class=%q\n",
				ts(), id, props.Name, props.MediaClass)
		}
	}
}
