package ui

/*
#cgo pkg-config: gtk+-3.0
#include <stdlib.h>
#include <gtk/gtk.h>

extern int openGUI(guint flags, int mode, guint32 *s, int size, char *mtext, char *cfgtext);
*/
import "C"
import (
	"encoding/json"
	"github.com/BambooEngine/ibus-bamboo/config"
	"log"
	"os"
	"unsafe"
)

var engineName string

//export saveFlags
func saveFlags(flags C.guint) {
	var (
		cfg = config.LoadConfig(engineName)
	)
	config.SaveConfig(cfg, engineName)
	cfg.IBflags = uint(flags)
	config.SaveConfig(cfg, engineName)
}

//export saveConfigText
func saveConfigText(text *C.char) {
	var (
		cfgText = C.GoString(text)
		cfgFn   = config.GetConfigPath(engineName)
	)
	err := os.WriteFile(cfgFn, []byte(cfgText), 0644)
	if err != nil {
		log.Printf("cannot save config %s: %v", cfgFn, err)
	}
}

//export saveMacroText
func saveMacroText(text *C.char) {
	var (
		macroText = C.GoString(text)
		macroFP   = config.GetMacroPath(engineName)
	)
	err := os.WriteFile(macroFP, []byte(macroText), 0644)
	if err != nil {
		log.Printf("cannot save macro file %s: %v", macroFP, err)
	}
}

//export saveInputMode
func saveInputMode(mode int) {
	var (
		cfg = config.LoadConfig(engineName)
	)
	cfg.DefaultInputMode = mode
	config.SaveConfig(cfg, engineName)
}

//export saveShortcuts
func saveShortcuts(ptr *C.guint32, length int) {
	var (
		cfg = config.LoadConfig(engineName)
	)
	codes := makeSliceFromPtr(ptr, length)
	cfg.Shortcuts = codes
	config.SaveConfig(cfg, engineName)
}

func makeSliceFromPtr(ptr *C.guint32, size int) [10]uint32 {
	var out [10]uint32
	slice := (*[1 << 28]C.guint32)(unsafe.Pointer(ptr))[:size:size]
	for i, elem := range slice[:size] {
		out[i] = uint32(elem)
	}
	return out
}

func OpenGUI(engName string) {
	engineName = engName
	var (
		cfg           = config.LoadConfig(engineName)
		shortcuts     = cfg.Shortcuts[:]
		s             = (*C.guint32)(&shortcuts[0])
		size          = len(shortcuts)
		macroFilePath = config.GetMacroPath(engineName)
	)
	mText, err := os.ReadFile(macroFilePath)
	if err != nil {
		log.Printf("cannot read macro file %s: %v", macroFilePath, err)
		return
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		log.Printf("cannot encode config: %v", err)
		return
	}
	os.Setenv("GTK_IM_MODULE", "gtk-im-context-simple")
	mTextC := C.CString(string(mText))
	defer C.free(unsafe.Pointer(mTextC))
	dataC := C.CString(string(data))
	defer C.free(unsafe.Pointer(dataC))
	C.openGUI(C.guint(cfg.IBflags), C.int(cfg.DefaultInputMode), s, C.int(size), mTextC, dataC)
}
