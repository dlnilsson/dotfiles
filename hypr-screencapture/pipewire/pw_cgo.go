package pipewire

/*
#cgo pkg-config: libpipewire-0.3
#include "pw_bridge.h"
*/
import "C"

import (
	"log/slog"
	"strings"
	"unsafe"

	"github.com/dlnilsson/dotfiles/hypr-screencapture/messages"
)

var msgChannel chan<- messages.Msg

func SetChannel(ch chan<- messages.Msg) {
	msgChannel = ch
}

func StartPipeWireLoop() int {
	slog.Info("Starting PipeWire loop")
	return int(C.pw_run_loop())
}

func StopPipeWireLoop() {
	slog.Info("Stopping PipeWire loop")
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

/* Called by pw_cgo.go (cgo callbacks) */
func onNodeAdd(id uint32, mediaClass, mediaRole, mediaType, mediaCategory, name, desc, app, portalApp string) {
	var (
		dsLower = strings.ToLower(desc)
		apLower = strings.ToLower(app)
	)
	// slog.Debug("NODE ADD",
	// 	"id", id,
	// 	"mediaclass", mediaClass,
	// 	"mediaRole", mediaRole,
	// 	"mediaType", mediaType,
	// 	"mediaCategory", mediaCategory,
	// 	"name", name,
	// 	"desc", desc,
	// 	"app", app,
	// 	"portalApp", portalApp)

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

	if IsScreenshareNode(mediaClass, name, dsLower, apLower) {
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
			slog.Info("START",
				"id", id,
				"media.class", mediaClass,
				"name", name,
				"app", app,
				"portal", portalApp)
		}
	} else if IsCameraNode(mediaClass, name, dsLower) {
		// slog.Info("CAMERA NODE DETECTED", "id", id, "name", name, "media.class", mediaClass, "desc", desc)
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
		slog.Debug("Total known cameras", "count", len(knownCameras))
		checkCameraUsage()
	} else if mediaClass == "Video/Source" || mediaClass == "Stream/Output/Video" || mediaClass == "Stream/Input/Video" {
		// slog.Debug("VIDEO NODE (not camera)", "id", id, "name", name, "media.class", mediaClass, "desc", desc, "app", app)
		if app != "" {
			slog.Debug("VIDEO NODE created by app - checking if this might link to camera", "app", app)
		}
	}
}

func onNodeRemove(id uint32) {
	delete(allNodes, id)

	if props, ok := known[id]; ok {
		delete(known, id)

		if msgChannel != nil {
			select {
			case msgChannel <- messages.Msg{Kind: messages.ScOff, Source: messages.SourcePipeWire}:
			default:
			}
		} else {
			slog.Info("STOP", "id", id, "name", props.Name, "media.class", props.MediaClass)
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
			slog.Info("CAMERA STOP", "id", id, "name", props.Name, "media.class", props.MediaClass)
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

	// cameraNote := ""
	// if outputIsCamera || inputIsCamera {
	// 	cameraNote = " [CAMERA LINK]"
	// }

	var outputMediaClass, inputMediaClass string
	if props, ok := allNodes[outputNodeID]; ok {
		outputMediaClass = props.MediaClass
	}
	if props, ok := allNodes[inputNodeID]; ok {
		inputMediaClass = props.MediaClass
	}

	// slog.Debug("LINK ADD",
	// 	"id", id,
	// 	"output_node", outputNodeID,
	// 	"output_name", outputName,
	// 	"output_mediaclass", outputMediaClass,
	// 	"input_node", inputNodeID,
	// 	"input_name", inputName,
	// 	"input_mediaclass", inputMediaClass,
	// 	"state", state,
	// 	"camera_note", cameraNote)

	if outputIsCamera || inputIsCamera {
		cameraNodeID := func() uint32 {
			if outputIsCamera {
				return outputNodeID
			}
			return inputNodeID
		}()
		slog.Info("CAMERA LINK DETECTED",
			"link", id,
			"camera_node", cameraNodeID,
			"output", outputNodeID,
			"input", inputNodeID,
			"state", state)
		slog.Debug("CAMERA LINK INFO",
			"camera_node", cameraNodeID,
			"in_knownCameras", func() bool {
				_, ok := knownCameras[cameraNodeID]
				return ok
			}())
	} else if outputMediaClass == "Video/Source" || inputMediaClass == "Video/Source" ||
		outputMediaClass == "Stream/Input/Video" || inputMediaClass == "Stream/Input/Video" ||
		outputMediaClass == "Stream/Output/Video" || inputMediaClass == "Stream/Output/Video" {
		slog.Debug("VIDEO LINK (not camera)",
			"link", id,
			"output", outputNodeID,
			"output_name", outputName,
			"input", inputNodeID,
			"input_name", inputName,
			"state", state)
		slog.Debug("VIDEO LINK CHECK",
			"output_node", outputNodeID,
			"output_in_knownCameras", func() bool { _, ok := knownCameras[outputNodeID]; return ok }(),
			"input_node", inputNodeID,
			"input_in_knownCameras", func() bool { _, ok := knownCameras[inputNodeID]; return ok }())
	}
	knownLinks[id] = linkInfo{
		OutputNodeID: outputNodeID,
		InputNodeID:  inputNodeID,
		State:        state,
	}
	// slog.Debug("Total known links", "count", len(knownLinks))
	checkCameraUsage()
}

func onLinkRemove(id uint32) {
	// slog.Debug("LINK REMOVE", "id", id)
	delete(knownLinks, id)
	// slog.Debug("Total known links", "count", len(knownLinks))
	checkCameraUsage()
}

func onLinkStateChange(id uint32, state int) {
	if link, exists := knownLinks[id]; exists {
		// var (
		// 	outputName string
		// 	inputName  string
		// )
		// if props, ok := knownCameras[link.OutputNodeID]; ok {
		// 	outputName = props.Name
		// } else if props, ok := allNodes[link.OutputNodeID]; ok {
		// 	outputName = props.Name
		// }
		// if props, ok := knownCameras[link.InputNodeID]; ok {
		// 	inputName = props.Name
		// } else if props, ok := allNodes[link.InputNodeID]; ok {
		// 	inputName = props.Name
		// }

		// slog.Debug("LINK STATE CHANGE",
		// 	"id", id,
		// 	"output_node", link.OutputNodeID,
		// 	"output_name", outputName,
		// 	"input_node", link.InputNodeID,
		// 	"input_name", inputName,
		// 	"old_state", link.State,
		// 	"new_state", state)

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
			slog.Debug("CHECK: link camera node state",
				"link", linkID,
				"camera_node", cameraNodeID,
				"camera", cameraName,
				"state", link.State)
			if link.State == 4 {
				slog.Debug("CHECK: Found active camera link", "link", linkID, "camera_node", cameraNodeID)
				newCameraInUse = true
				break
			}
		}
	}
	// if len(knownCameras) > 0 {
	// 	slog.Debug("CHECK: known cameras", "cameras", func() []uint32 {
	// 		var ids []uint32
	// 		for id := range knownCameras {
	// 			ids = append(ids, id)
	// 		}
	// 		return ids
	// 	}())
	// }

	if newCameraInUse != cameraInUse {
		cameraInUse = newCameraInUse
		if msgChannel != nil {
			var msg messages.Msg
			if cameraInUse {
				msg = messages.Msg{Kind: messages.CameraOn, Source: messages.SourcePipeWire}
				slog.Debug("SENDING CameraOn message to channel")
			} else {
				msg = messages.Msg{Kind: messages.CameraOff, Source: messages.SourcePipeWire}
				slog.Debug("SENDING CameraOff message to channel")
			}
			select {
			case msgChannel <- msg:
				slog.Debug("Successfully sent camera message", "message", msg)
			default:
				slog.Error("Failed to send camera message (channel full)", "message", msg)
			}
		} else {
			slog.Warn("msgChannel is nil, cannot send camera message")
			if cameraInUse {
				slog.Info("CAMERA IN USE (active link detected)")
			} else {
				slog.Info("CAMERA NOT IN USE (no active links)")
			}
		}
	}
}
