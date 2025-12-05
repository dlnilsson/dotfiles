package pipewire

/*
#cgo pkg-config: libpipewire-0.3
#include "pw_bridge.h"
*/
import "C"

import (
	"log"
	"strings"
	"unsafe"

	"github.com/dlnilsson/dotfiles/hypr-screencapture/messages"
)

var msgChannel chan<- messages.Msg

func SetChannel(ch chan<- messages.Msg) {
	msgChannel = ch
}

func StartPipeWireLoop() int {
	log.Printf("[pw] Starting PipeWire loop...")
	return int(C.pw_run_loop())
}

func StopPipeWireLoop() {
	log.Printf("[pw] Stopping PipeWire loop...")
	C.pw_quit_loop()
}

// ----- Go-exported functions called from C (must be in this file) -----

//export goOnNodeAdd
func goOnNodeAdd(id C.uint32_t, mediaClass, mediaRole, mediaType, mediaCategory, name, desc, app, portalApp *C.char) {
	var (
		mc   = C.GoString(mediaClass)
		mr   = C.GoString(mediaRole)
		mt   = C.GoString(mediaType)
		mcat = C.GoString(mediaCategory)
		nm   = C.GoString(name)
		ds   = C.GoString(desc)
		ap   = C.GoString(app)
		pa   = C.GoString(portalApp)
	)
	onNodeAdd(uint32(id), mc, mr, mt, mcat, nm, ds, ap, pa)
	// Note: memory owned by C; no need to free here.
	_ = unsafe.Pointer(mediaClass) // pacify linters about cgo pointers
}

//export goOnNodeRemove
func goOnNodeRemove(id C.uint32_t) {
	onNodeRemove(uint32(id))
}

// Called from C code in pw_bridge.c (on_link_info callback)
//
//export goOnLinkAdd
func goOnLinkAdd(id, outputNodeID, inputNodeID C.uint32_t, state C.int) {
	onLinkAdd(uint32(id), uint32(outputNodeID), uint32(inputNodeID), int(state))
}

// Called from C code in pw_bridge.c (on_global_remove callback)
//
//export goOnLinkRemove
func goOnLinkRemove(id C.uint32_t) {
	onLinkRemove(uint32(id))
}

// Called from C code in pw_bridge.c (on_link_info callback)
//
//export goOnLinkStateChange
func goOnLinkStateChange(id C.uint32_t, state C.int) {
	onLinkStateChange(uint32(id), int(state))
}

type nodeProps struct {
	MediaClass    string
	MediaRole     string
	MediaType     string
	MediaCategory string
	Name          string
	DescLower     string
	AppLower      string
	PortalApp     string
}

type linkInfo struct {
	OutputNodeID uint32
	InputNodeID  uint32
	State        int
}

var (
	known        = map[uint32]nodeProps{}
	knownCameras = map[uint32]nodeProps{}
	allNodes     = map[uint32]nodeProps{}
	knownLinks   = map[uint32]linkInfo{}
	cameraInUse  = false
)

func isCameraNode(mediaClass, name, descLower string) bool {
	if strings.HasPrefix(name, "v4l2_input.") {
		log.Printf("isCameraNode: v4l2_input prefix: %+v\n", name)
		return true
	}
	if strings.Contains(descLower, "v4l2") || strings.Contains(descLower, "camera") || strings.Contains(descLower, "webcam") {
		log.Printf("isCameraNode: description match: %+v desc=%+v\n", name, descLower)
		return true
	}
	if mediaClass == "Video/Source" {
		if strings.Contains(name, "v4l2") || strings.Contains(name, "camera") || strings.Contains(name, "webcam") {
			log.Printf("isCameraNode: Video/Source with camera name: %+v\n", name)
			return true
		}
		if !strings.Contains(descLower, "screen") && !strings.Contains(descLower, "display") &&
			!strings.Contains(descLower, "monitor") && name != "xdg-desktop-portal-hyprland" &&
			name != "xdg-desktop-portal-wlr" && name != "xdg-desktop-portal-gnome" &&
			name != "xdg-desktop-portal-kde" && descLower != "" {
			log.Printf("isCameraNode: Video/Source (potential camera): %+v desc=%+v\n", name, descLower)
			return true
		}
	}
	return false
}

func isScreenshareNode(mc, name, descLower, appLower string) bool {
	if isCameraNode(mc, name, descLower) {
		return false
	}
	log.Printf("isScreenshareNode: MC %+v NAME %+v DESCLOWER %+v APPLOWER %+v\n", mc, name, descLower, appLower)
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
func onNodeAdd(id uint32, mediaClass, mediaRole, mediaType, mediaCategory, name, desc, app, portalApp string) {
	var (
		dsLower = strings.ToLower(desc)
		apLower = strings.ToLower(app)
	)
	log.Printf("[pw] NODE ADD id=%d mediaclass=%q mediaRole=%q mediaType=%q mediaCategory=%q name=%q desc=%q app=%q portalApp=%q\n",
		id, mediaClass, mediaRole, mediaType, mediaCategory, name, desc, app, portalApp)

	allNodes[id] = nodeProps{
		MediaClass:    mediaClass,
		MediaRole:     mediaRole,
		MediaType:     mediaType,
		MediaCategory: mediaCategory,
		Name:          name,
		DescLower:     dsLower,
		AppLower:      apLower,
		PortalApp:     portalApp,
	}

	if isScreenshareNode(mediaClass, name, dsLower, apLower) {
		known[id] = nodeProps{
			MediaClass:    mediaClass,
			MediaRole:     mediaRole,
			MediaType:     mediaType,
			MediaCategory: mediaCategory,
			Name:          name,
			DescLower:     dsLower,
			AppLower:      apLower,
			PortalApp:     portalApp,
		}

		if msgChannel != nil {
			select {
			case msgChannel <- messages.Msg{
				Kind:   messages.ScOn,
				Source: messages.SourcePipeWire,
			}:
			default:
			}
		} else {
			log.Printf("[pw] START id=%d media.class=%q name=%q app=%q portal=%q\n",
				id, mediaClass, name, app, portalApp)
		}
	} else if isCameraNode(mediaClass, name, dsLower) {
		log.Printf("[pw] CAMERA NODE DETECTED id=%d name=%q media.class=%q desc=%q\n", id, name, mediaClass, desc)
		knownCameras[id] = nodeProps{
			MediaClass:    mediaClass,
			MediaRole:     mediaRole,
			MediaType:     mediaType,
			MediaCategory: mediaCategory,
			Name:          name,
			DescLower:     dsLower,
			AppLower:      apLower,
			PortalApp:     portalApp,
		}
		log.Printf("[pw] Total known cameras: %d\n", len(knownCameras))
		checkCameraUsage()
	} else if mediaClass == "Video/Source" || mediaClass == "Stream/Output/Video" || mediaClass == "Stream/Input/Video" {
		log.Printf("[pw] VIDEO NODE (not camera) id=%d name=%q media.class=%q desc=%q app=%q\n", id, name, mediaClass, desc, app)
		if app != "" {
			log.Printf("[pw] VIDEO NODE created by app: %q - checking if this might link to camera", app)
		}
	}
}

func onNodeRemove(id uint32) {
	if props, ok := allNodes[id]; ok {
		log.Printf("[pw] NODE REMOVE id=%d name=%q media.class=%q\n", id, props.Name, props.MediaClass)
		delete(allNodes, id)
	}

	if props, ok := known[id]; ok {
		delete(known, id)

		if msgChannel != nil {
			select {
			case msgChannel <- messages.Msg{Kind: messages.ScOff, Source: messages.SourcePipeWire}:
			default:
			}
		} else {
			log.Printf("[pw] STOP  id=%d name=%q media.class=%q\n",
				id, props.Name, props.MediaClass)
		}
	} else if props, ok := knownCameras[id]; ok {
		delete(knownCameras, id)
		checkCameraUsage()

		if msgChannel != nil {
			select {
			case msgChannel <- messages.Msg{Kind: messages.CameraOff, Source: messages.SourcePipeWire}:
			default:
			}
		} else {
			log.Printf("[pw] CAMERA STOP  id=%d name=%q media.class=%q\n",
				id, props.Name, props.MediaClass)
		}
	}
}

func onLinkAdd(id, outputNodeID, inputNodeID uint32, state int) {
	var (
		outputName     string
		inputName      string
		outputIsCamera bool
		inputIsCamera  bool
	)
	if props, ok := knownCameras[outputNodeID]; ok {
		outputName = props.Name
		outputIsCamera = true
	} else if props, ok := allNodes[outputNodeID]; ok {
		outputName = props.Name
	}
	if props, ok := knownCameras[inputNodeID]; ok {
		inputName = props.Name
		inputIsCamera = true
	} else if props, ok := allNodes[inputNodeID]; ok {
		inputName = props.Name
	}

	cameraNote := ""
	if outputIsCamera || inputIsCamera {
		cameraNote = " [CAMERA LINK]"
	}

	var outputMediaClass, inputMediaClass string
	if props, ok := allNodes[outputNodeID]; ok {
		outputMediaClass = props.MediaClass
	}
	if props, ok := allNodes[inputNodeID]; ok {
		inputMediaClass = props.MediaClass
	}

	log.Printf("[pw] LINK ADD id=%d output_node=%d (%q, %q) input_node=%d (%q, %q) state=%d%s\n",
		id, outputNodeID, outputName, outputMediaClass, inputNodeID, inputName, inputMediaClass, state, cameraNote)

	if outputNodeID == 73 || outputNodeID == 75 || inputNodeID == 73 || inputNodeID == 75 {
		log.Printf("[pw] *** LINK INVOLVES CAMERA NODE *** link=%d output=%d input=%d (camera nodes: 73, 75)",
			id, outputNodeID, inputNodeID)
	}

	if outputIsCamera || inputIsCamera {
		cameraNodeID := func() uint32 {
			if outputIsCamera {
				return outputNodeID
			}
			return inputNodeID
		}()
		log.Printf("[pw] *** CAMERA LINK DETECTED *** link=%d camera_node=%d (output=%d, input=%d) state=%d\n",
			id, cameraNodeID, outputNodeID, inputNodeID, state)
		log.Printf("[pw] *** CAMERA LINK INFO *** camera_node=%d is in knownCameras: %v\n",
			cameraNodeID, func() bool {
				_, ok := knownCameras[cameraNodeID]
				return ok
			}())
	} else if outputMediaClass == "Video/Source" || inputMediaClass == "Video/Source" ||
		outputMediaClass == "Stream/Input/Video" || inputMediaClass == "Stream/Input/Video" ||
		outputMediaClass == "Stream/Output/Video" || inputMediaClass == "Stream/Output/Video" {
		log.Printf("[pw] *** VIDEO LINK (not camera) *** link=%d output=%d (%q) input=%d (%q) state=%d\n",
			id, outputNodeID, outputName, inputNodeID, inputName, state)
		log.Printf("[pw] *** VIDEO LINK CHECK *** output_node %d in knownCameras: %v, input_node %d in knownCameras: %v\n",
			outputNodeID, func() bool { _, ok := knownCameras[outputNodeID]; return ok }(),
			inputNodeID, func() bool { _, ok := knownCameras[inputNodeID]; return ok }())
	}
	knownLinks[id] = linkInfo{
		OutputNodeID: outputNodeID,
		InputNodeID:  inputNodeID,
		State:        state,
	}
	log.Printf("[pw] Total known links: %d\n", len(knownLinks))
	checkCameraUsage()
}

func onLinkRemove(id uint32) {
	log.Printf("[pw] LINK REMOVE id=%d\n", id)
	delete(knownLinks, id)
	log.Printf("[pw] Total known links: %d\n", len(knownLinks))
	checkCameraUsage()
}

func onLinkStateChange(id uint32, state int) {
	if link, exists := knownLinks[id]; exists {
		var (
			outputName string
			inputName  string
		)
		if props, ok := knownCameras[link.OutputNodeID]; ok {
			outputName = props.Name
		} else if props, ok := allNodes[link.OutputNodeID]; ok {
			outputName = props.Name
		}
		if props, ok := knownCameras[link.InputNodeID]; ok {
			inputName = props.Name
		} else if props, ok := allNodes[link.InputNodeID]; ok {
			inputName = props.Name
		}

		log.Printf("[pw] LINK STATE CHANGE id=%d output_node=%d (%q) input_node=%d (%q) old_state=%d new_state=%d\n",
			id, link.OutputNodeID, outputName, link.InputNodeID, inputName, link.State, state)

		if link.OutputNodeID == 73 || link.OutputNodeID == 75 || link.InputNodeID == 73 || link.InputNodeID == 75 {
			log.Printf("[pw] *** LINK STATE CHANGE INVOLVES CAMERA NODE *** link=%d output=%d input=%d state=%d",
				id, link.OutputNodeID, link.InputNodeID, state)
		}
		link.State = state
		knownLinks[id] = link
		checkCameraUsage()
	}
}

func checkCameraUsage() {
	newCameraInUse := false
	for linkID, link := range knownLinks {
		var (
			cameraNodeID uint32
			cameraName   string
			isCamera     bool
		)

		if props, ok := knownCameras[link.OutputNodeID]; ok {
			cameraNodeID = link.OutputNodeID
			cameraName = props.Name
			isCamera = true
		} else if props, ok := knownCameras[link.InputNodeID]; ok {
			cameraNodeID = link.InputNodeID
			cameraName = props.Name
			isCamera = true
		}

		if isCamera {
			log.Printf("[pw] CHECK: link=%d camera_node=%d (camera: %q) state=%d (state 4 = active)\n",
				linkID, cameraNodeID, cameraName, link.State)
			if link.State == 4 {
				log.Printf("[pw] CHECK: Found active camera link! link=%d camera_node=%d", linkID, cameraNodeID)
				newCameraInUse = true
				break
			}
		}
	}
	if len(knownCameras) > 0 {
		log.Printf("[pw] CHECK: known cameras: %v\n", func() []uint32 {
			var ids []uint32
			for id := range knownCameras {
				ids = append(ids, id)
			}
			return ids
		}())
	}

	if newCameraInUse != cameraInUse {
		cameraInUse = newCameraInUse
		if msgChannel != nil {
			var msg messages.Msg
			if cameraInUse {
				msg = messages.Msg{Kind: messages.CameraOn, Source: messages.SourcePipeWire}
				log.Printf("[pw] SENDING CameraOn message to channel")
			} else {
				msg = messages.Msg{Kind: messages.CameraOff, Source: messages.SourcePipeWire}
				log.Printf("[pw] SENDING CameraOff message to channel")
			}
			select {
			case msgChannel <- msg:
				log.Printf("[pw] Successfully sent camera message: %+v", msg)
			default:
				log.Printf("[pw] ERROR: Failed to send camera message (channel full): %+v", msg)
			}
		} else {
			log.Printf("[pw] WARNING: msgChannel is nil, cannot send camera message")
			if cameraInUse {
				log.Printf("[pw] CAMERA IN USE (active link detected)")
			} else {
				log.Printf("[pw] CAMERA NOT IN USE (no active links)")
			}
		}
	}
}
