package rtmidi

/*
#cgo CXXFLAGS: -g -std=c++11
#cgo LDFLAGS: -g

#cgo freebsd CXXFLAGS: -D__UNIX_JACK__
#cgo freebsd LDFLAGS: -pthread
#cgo linux CXXFLAGS: -D__LINUX_ALSA__
#cgo linux LDFLAGS: -lasound -pthread
#cgo windows CXXFLAGS: -D__WINDOWS_MM__
#cgo windows LDFLAGS: -luuid -lksuser -lwinmm -lole32
#cgo darwin CXXFLAGS: -D__MACOSX_CORE__
#cgo darwin LDFLAGS: -framework CoreServices -framework CoreAudio -framework CoreMIDI -framework CoreFoundation

#include "lib.h"
*/
import "C"
import (
	"errors"
	"sync"
	"unsafe"
)

const (
	// APIUnspecified searches for a working compiled API.
	APIUnspecified API = C.RTMIDI_API_UNSPECIFIED
	// APIMacOSXCore uses Macintosh OS-X CoreMIDI API.
	APIMacOSXCore API = C.RTMIDI_API_MACOSX_CORE
	// APILinuxALSA uses the Advanced Linux Sound Architecture API.
	APILinuxALSA API = C.RTMIDI_API_LINUX_ALSA
	// APIUnixJack uses the JACK Low-Latency MIDI Server API.
	APIUnixJack API = C.RTMIDI_API_UNIX_JACK
	// APIWindowsMM uses the Microsoft Multimedia MIDI API.
	APIWindowsMM API = C.RTMIDI_API_WINDOWS_MM
	// APIDummy is a compilable but non-functional API.
	APIDummy API = C.RTMIDI_API_RTMIDI_DUMMY
)

var (
	ErrClosed = errors.New("MIDI port is closed")
)

// API is an enumeration of possible MIDI API specifiers.
type API C.enum_RtMidiApi

// MIDI interface provides a common, platform-independent API for realtime MIDI
// device enumeration and handling MIDI ports.
type MIDI interface {
	OpenPort(port int, name string) error
	OpenVirtualPort(name string) error
	Close() error
	PortCount() (int, error)
	PortName(port int) (string, error)
}

// MIDIIn interface provides a common, platform-independent API for realtime
// MIDI input. It allows access to a single MIDI input port. Incoming MIDI
// messages are either saved to a queue for retrieval using the Message()
// method or immediately passed to a user-specified callback function. Create
// multiple instances of this class to connect to more than one MIDI device at
// the same time.
type MIDIIn interface {
	MIDI
	API() (API, error)
	IgnoreTypes(midiSysex bool, midiTime bool, midiSense bool) error
	SetCallback(func(MIDIIn, []byte, float64)) error
	CancelCallback() error
	Message() ([]byte, float64, error)
	Destroy()
}

// MIDIOut interface provides a common, platform-independent API for MIDI
// output. It allows one to probe available MIDI output ports, to connect to
// one such port, and to send MIDI bytes immediately over the connection.
// Create multiple instances of this class to connect to more than one MIDI
// device at the same time.
type MIDIOut interface {
	MIDI
	API() (API, error)
	SendMessage([]byte) error
	Destroy()
}

// Private types
type midi struct {
	midi C.RtMidiPtr

	sync.Mutex
	done     chan struct{}
	closeErr error
}

type midiIn struct {
	midi
	in C.RtMidiInPtr
}

type midiOut struct {
	midi
	out C.RtMidiOutPtr
}

// Suppress printing error messages to cerr
func quiet(ptr C.RtMidiPtr) midi {
	C.rtmidi_set_error_quiet(ptr)
	return midi{midi: ptr, done: make(chan struct{})}
}

// CompiledAPI determines the available compiled MIDI APIs.
func CompiledAPI() (apis []API) {
	n := C.rtmidi_get_compiled_api(nil, 0)
	capis := make([]C.enum_RtMidiApi, n, n)
	C.rtmidi_get_compiled_api(&capis[0], C.uint(n))
	for _, capi := range capis {
		apis = append(apis, API(capi))
	}
	return apis
}

// Format an API as a string
func (a API) String() string {
	return a.Name()
}

// Lower case identifier for the API
func (a API) Name() string {
	return C.GoString(C.rtmidi_api_name(C.enum_RtMidiApi(a)))
}

// Display name for the API
func (a API) DisplayName() string {
	return C.GoString(C.rtmidi_api_display_name(C.enum_RtMidiApi(a)))
}

// Only close once
func (m *midi) close(in C.RtMidiInPtr, out C.RtMidiOutPtr) error {
	m.Lock()
	defer m.Unlock()

	select {
	case <-m.done:
	default:
		C.rtmidi_close_port(C.RtMidiPtr(m.midi))
		if !m.midi.ok {
			m.closeErr = errors.New(C.GoString(m.midi.msg))
		}
		if in != nil {
			C.rtmidi_in_free(in)
		}
		if out != nil {
			C.rtmidi_out_free(out)
		}
		close(m.done)
	}
	return m.closeErr
}

// Open a MIDI input connection given by enumeration number.
func (m *midi) OpenPort(port int, name string) error {
	p := C.CString(name)
	defer C.free(unsafe.Pointer(p))
	C.rtmidi_open_port(m.midi, C.uint(port), p)
	if !m.midi.ok {
		return errors.New(C.GoString(m.midi.msg))
	}
	return nil
}

// Create a virtual input port, with optional name, to allow software connections
// (OS X, JACK and ALSA only).
func (m *midi) OpenVirtualPort(name string) error {
	p := C.CString(name)
	defer C.free(unsafe.Pointer(p))
	C.rtmidi_open_virtual_port(m.midi, p)
	if !m.midi.ok {
		return errors.New(C.GoString(m.midi.msg))
	}
	return nil
}

// Return a string identifier for the specified MIDI input port number.
func (m *midi) PortName(port int) (string, error) {
	bufLen := C.int(0)

	C.rtmidi_get_port_name(m.midi, C.uint(port), nil, &bufLen)
	if !m.midi.ok {
		return "", errors.New(C.GoString(m.midi.msg))
	}

	if bufLen < 1 {
		return "", nil
	}

	bufOut := make([]byte, int(bufLen))
	p := (*C.char)(unsafe.Pointer(&bufOut[0]))

	C.rtmidi_get_port_name(m.midi, C.uint(port), p, &bufLen)
	if !m.midi.ok {
		return "", errors.New(C.GoString(m.midi.msg))
	}

	return string(bufOut[0 : bufLen-1]), nil
}

// Return the number of available MIDI input ports.
func (m *midi) PortCount() (int, error) {
	n := C.rtmidi_get_port_count(m.midi)
	if !m.midi.ok {
		return 0, errors.New(C.GoString(m.midi.msg))
	}
	return int(n), nil
}

// Close an open MIDI connection.
func (m *midi) Close() error {
	return m.close(nil, nil)
}

// Open a default MIDIIn port.
func NewMIDIInDefault() (MIDIIn, error) {
	in := C.rtmidi_in_create_default()
	if !in.ok {
		defer C.rtmidi_in_free(in)
		return nil, errors.New(C.GoString(in.msg))
	}
	return &midiIn{in: in, midi: quiet(C.RtMidiPtr(in))}, nil
}

// Open a single MIDIIn port using the given API. One can provide a
// custom port name and a desired queue size for the incomming MIDI messages.
func NewMIDIIn(api API, name string, queueSize int) (MIDIIn, error) {
	p := C.CString(name)
	defer C.free(unsafe.Pointer(p))
	in := C.rtmidi_in_create(C.enum_RtMidiApi(api), p, C.uint(queueSize))
	if !in.ok {
		defer C.rtmidi_in_free(in)
		return nil, errors.New(C.GoString(in.msg))
	}
	return &midiIn{in: in, midi: quiet(C.RtMidiPtr(in))}, nil
}

// Return the MIDI API specifier for the current instance of RtMidiIn.
func (m *midiIn) API() (API, error) {
	api := C.rtmidi_in_get_current_api(m.in)
	if !m.in.ok {
		return APIUnspecified, errors.New(C.GoString(m.in.msg))
	}
	return API(api), nil
}

// Close an open MIDI connection (if one exists).
func (m *midiIn) Close() error {
	unregisterMIDIIn(m)
	return m.midi.close(m.in, nil)
}

func (m *midiIn) Destroy() {
	m.Close()
}

// Specify whether certain MIDI message types should be queued or ignored during input.
//
// By default, MIDI timing and active sensing messages are ignored
// during message input because of their relative high data rates.
// MIDI sysex messages are ignored by default as well.  Variable
// values of "true" imply that the respective message type will be
// ignored.
func (m *midiIn) IgnoreTypes(midiSysex bool, midiTime bool, midiSense bool) error {
	C.rtmidi_in_ignore_types(m.in, C._Bool(midiSysex), C._Bool(midiTime), C._Bool(midiSense))
	if !m.in.ok {
		return errors.New(C.GoString(m.in.msg))
	}
	return nil
}

// Set a callback function to be invoked for incoming MIDI messages.
func (m *midiIn) SetCallback(cb func(MIDIIn, []byte, float64)) error {
	return registerMIDIIn(m, cb)
}

// Cancel use of the current callback function (if one exists).
func (m *midiIn) CancelCallback() error {
	return unregisterMIDIIn(m)
}

// Fill a byte buffer with the next available MIDI message in the input queue
// and return the event delta-time in seconds.
//
// This function returns immediately whether a new message is available or not.
func (m *midiIn) Message() ([]byte, float64, error) {
	msg := make([]C.uchar, 64*1024, 64*1024)
	sz := C.size_t(len(msg))
	r := C.rtmidi_in_get_message(m.in, &msg[0], &sz)
	if !m.in.ok {
		return nil, 0, errors.New(C.GoString(m.in.msg))
	}
	b := make([]byte, int(sz), int(sz))
	for i, c := range msg[:sz] {
		b[i] = byte(c)
	}
	return b, float64(r), nil
}

// Open a default MIDIOut port.
func NewMIDIOutDefault() (MIDIOut, error) {
	out := C.rtmidi_out_create_default()
	if !out.ok {
		defer C.rtmidi_out_free(out)
		return nil, errors.New(C.GoString(out.msg))
	}
	return &midiOut{out: out, midi: quiet(C.RtMidiPtr(out))}, nil
}

// Open a single MIDIIn port using the given API with the given port name.
func NewMIDIOut(api API, name string) (MIDIOut, error) {
	p := C.CString(name)
	defer C.free(unsafe.Pointer(p))
	out := C.rtmidi_out_create(C.enum_RtMidiApi(api), p)
	if !out.ok {
		defer C.rtmidi_out_free(out)
		return nil, errors.New(C.GoString(out.msg))
	}
	return &midiOut{out: out, midi: quiet(C.RtMidiPtr(out))}, nil
}

// Return the MIDI API specifier for the current instance of RtMidiOut.
func (m *midiOut) API() (API, error) {
	api := C.rtmidi_out_get_current_api(m.out)
	if !m.out.ok {
		return APIUnspecified, errors.New(C.GoString(m.out.msg))
	}
	return API(api), nil
}

// Close an open MIDI connection.
func (m *midiOut) Close() error {
	return m.midi.close(nil, m.out)
}

func (m *midiOut) Destroy() {
	m.Close()
}

// Immediately send a single message out an open MIDI output port.
func (m *midiOut) SendMessage(b []byte) error {
	m.midi.Lock()
	defer m.midi.Unlock()

	select {
	case <-m.done:
		return ErrClosed
	default:
	}

	p := C.CBytes(b)
	defer C.free(unsafe.Pointer(p))
	C.rtmidi_out_send_message(m.out, (*C.uchar)(p), C.int(len(b)))
	if !m.out.ok {
		return errors.New(C.GoString(m.out.msg))
	}
	return nil
}

// Callback registry
var (
	regmtx   = sync.RWMutex{}
	registry = map[unsafe.Pointer]func([]byte, float64){}
)

func registerMIDIIn(m *midiIn, cb func(MIDIIn, []byte, float64)) error {
	regmtx.Lock()
	defer regmtx.Unlock()

	p := unsafe.Pointer(m.in)
	if _, exists := registry[p]; exists {
		C.rtmidi_in_cancel_callback(m.in)
		if !m.in.ok {
			return errors.New(C.GoString(m.in.msg))
		}
		delete(registry, p)
	}

	if cb != nil {
		C.cgoSetCallback(m.in, p)
		if !m.in.ok {
			return errors.New(C.GoString(m.in.msg))
		}
		registry[p] = func(data []byte, ts float64) {
			cb(m, data, ts)
		}
	}

	return nil
}

func unregisterMIDIIn(m *midiIn) error {
	return registerMIDIIn(m, nil)
}

func dispatchCallback(p unsafe.Pointer, data []byte, ts float64) {
	regmtx.RLock()
	defer regmtx.RUnlock()

	if cb, ok := registry[p]; ok {
		cb(data, ts)
	}
}

//export goMIDIInCallback
func goMIDIInCallback(ts C.double, msg *C.uchar, msgsz C.size_t, arg unsafe.Pointer) {
	dispatchCallback(arg, C.GoBytes(unsafe.Pointer(msg), C.int(msgsz)), float64(ts))
}

func (m *midiIn) testCallback() {
	C.testCallback(m.in)
}
