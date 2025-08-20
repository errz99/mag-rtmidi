package rtmidi

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"
)

func ExampleCompiledAPI() {
	for _, api := range CompiledAPI() {
		log.Println("Compiled API: ", api)
	}
}

func ExampleMIDIIn_Message() {
	in, err := NewMIDIInDefault()
	if err != nil {
		log.Fatal(err)
	}
	defer in.Destroy()
	if err := in.OpenPort(0, "RtMidi"); err != nil {
		log.Fatal(err)
	}
	defer in.Close()

	for {
		m, t, err := in.Message()
		if len(m) > 0 {
			log.Println(m, t, err)
		}
	}
}

func ExampleMIDIIn_SetCallback() {
	in, err := NewMIDIInDefault()
	if err != nil {
		log.Fatal(err)
	}
	defer in.Destroy()
	if err := in.OpenPort(0, "RtMidi"); err != nil {
		log.Fatal(err)
	}
	defer in.Close()
	in.SetCallback(func(m MIDIIn, msg []byte, t float64) {
		log.Println(msg, t)
	})
	<-make(chan struct{})
}

//
// Tests
//

var IsDummy bool

// Ensure there is at least one API available
func TestCompiledAPI(t *testing.T) {
	apis := CompiledAPI()
	if len(apis) < 1 {
		t.Errorf("Compiled API list is empty")
	}
}

func TestAPIString(t *testing.T) {
	for a := APIUnspecified; a <= APIDummy; a++ {
		if a.String() == "" {
			t.Errorf("Missing string for API value %d", int(a))
		}
		if a == APIUnspecified {
			continue
		}
		if a.DisplayName() == "" || a.DisplayName() == "Unknown" {
			t.Errorf("Missing display name for API value %d", int(a))
		}
	}
	if (APIDummy + API(1)).String() != "" {
		t.Error("More valid API strings than expected")
	}
	if (APIDummy + API(1)).DisplayName() != "Unknown" {
		t.Error("More valid API display names than expected")
	}
}

// Helper to close a port when the test is complete
func closeAfter(t *testing.T, m MIDI) {
	t.Cleanup(func() {
		t.Run("close", func(t *testing.T) {
			err := m.Close()
			if err != nil {
				t.Error(err)
			}
		})
	})
}

// Tests specific to a MIDIIn port
func testInputPort(t *testing.T, m MIDIIn) {
	_, err := m.API()
	if err != nil {
		t.Error(err)
	}

	t.Run("ignore", func(t *testing.T) {
		for i := 0; i < 8; i++ {
			sysex := (i & 1)
			sense := ((i >> 1) & 1)
			timing := ((i >> 2) & 1)

			k := fmt.Sprintf("%d%d%d", sysex, timing, sense)
			t.Run(k, func(t *testing.T) {
				err := m.IgnoreTypes(sysex == 1, timing == 1, sense == 1)
				if err != nil {
					t.Error(err)
				}
			})
		}
	})

	t.Run("callback", func(t *testing.T) {
		callback := func(MIDIIn, []byte, float64) {
			// do nothing
		}
		err := m.SetCallback(callback)
		if err != nil {
			t.Fatal(err)
		}
		m.(*midiIn).testCallback()
		err = m.CancelCallback()
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("message", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		go func() {
			_, _, err := m.Message()
			if err != nil {
				t.Error(err)
			}
		}()

		<-ctx.Done()
	})
}

// Tests specific to a MIDIOut port
func testOutputPort(t *testing.T, m MIDIOut) {
	_, err := m.API()
	if err != nil {
		t.Error(err)
	}

	messages := []struct {
		name  string
		bytes []byte
	}{
		{"note-on", []byte{0x90, 0x30, 0x60}},
		{"note-off", []byte{0x80, 0x30, 0x00}},
	}

	t.Run("send", func(t *testing.T) {
		for _, msg := range messages {
			t.Run(msg.name, func(t *testing.T) {
				err := m.SendMessage(msg.bytes)
				if err != nil {
					t.Error(err)
				}
			})
		}
	})
}

func testVirtualPort(m MIDI, err error) func(t *testing.T) {
	return func(t *testing.T) {
		if err != nil {
			t.Fatal(err)
		}
		closeAfter(t, m)

		err = m.OpenVirtualPort("RtMidiVirtual")
		if err != nil {
			t.Error(err)
		}

		switch mm := m.(type) {
		case MIDIIn:
			testInputPort(t, mm)
		case MIDIOut:
			testOutputPort(t, mm)
		default:
			t.Fatalf("Unexpected port type %T", mm)
		}
	}
}

func testExistingPort(m MIDI, err error) func(t *testing.T) {
	return func(t *testing.T) {
		if err != nil {
			t.Fatal(err)
		}
		closeAfter(t, m)

		var n int
		var name string

		t.Run("count", func(t *testing.T) {
			n, err = m.PortCount()
			if err != nil {
				t.Error(err)
			}
		})

		if IsDummy || testing.Short() {
			return
		}

		if n < 1 {
			t.Fatal("There were zero available ports")
		}

		t.Run("name", func(t *testing.T) {
			name, err = m.PortName(0)
			if err != nil {
				t.Error(err)
			}
		})

		if name == "" {
			t.Fatal("Port name is an empty string")
		}

		t.Run("open", func(t *testing.T) {
			err = m.OpenPort(0, name)
			if err != nil {
				t.Error(err)
			}
		})

		switch mm := m.(type) {
		case MIDIIn:
			testInputPort(t, mm)
		case MIDIOut:
			testOutputPort(t, mm)
		default:
			t.Fatalf("Unexpected port type %T", mm)
		}
	}
}

func testErrs(m MIDI, err error) func(t *testing.T) {
	return func(t *testing.T) {
		if err != nil {
			t.Fatal(err)
		}

		t.Run("open", func(t *testing.T) {
			err := m.OpenPort(123, "unnamed")
			if !IsDummy && err == nil {
				t.Error("No error opening an invalid port")
			}
		})

		t.Run("name", func(t *testing.T) {
			n, err := m.PortName(123)
			if !IsDummy && (err == nil || n != "") {
				t.Error("No error getting port name for an invalid port")
			}
		})
	}
}

// Run tests for each API discovered
func TestAPIs(t *testing.T) {
	for _, api := range CompiledAPI() {
		name := api.String()
		if name == "" {
			name = fmt.Sprintf("RtMidiApi(%d)", int(api))
			t.Errorf("API %s is unnamed", name)
		}

		t.Run(name, func(t *testing.T) {
			t.Run("output", testExistingPort(NewMIDIOut(api, "RtMidi")))
			t.Run("input", testExistingPort(NewMIDIIn(api, "RtMidi", 1024)))
		})
	}
}

func TestDefaults(t *testing.T) {
	t.Run("virtual", func(t *testing.T) {
		t.Run("output", testVirtualPort(NewMIDIOutDefault()))
		t.Run("input", testVirtualPort(NewMIDIInDefault()))
	})

	t.Run("default", func(t *testing.T) {
		t.Run("output", testExistingPort(NewMIDIOutDefault()))
		t.Run("input", testExistingPort(NewMIDIInDefault()))
	})

	t.Run("errors", func(t *testing.T) {
		t.Run("output", testErrs(NewMIDIOutDefault()))
		t.Run("input", testErrs(NewMIDIInDefault()))
	})
}

func TestDestroy(t *testing.T) {
	out, err := NewMIDIOutDefault()
	if err != nil {
		t.Fatal(err)
	}

	in, err := NewMIDIInDefault()
	if err != nil {
		t.Fatal(err)
	}

	out.Destroy()
	in.Destroy()
}

func TestMain(m *testing.M) {
	apis := CompiledAPI()
	if len(apis) == 1 && apis[0] == APIDummy {
		IsDummy = true
	}
	os.Exit(m.Run())
}

//
// Benchmarks
//
func benchmarkSend(b *testing.B, msg []byte) {
	b.SetBytes(int64(len(msg)))

	out, err := NewMIDIOutDefault()
	if err != nil {
		b.Fatal(err)
	}
	defer out.Close()

	err = out.OpenVirtualPort("RtMidiVirtual")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = out.SendMessage(msg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func benchmarkSysEx(b *testing.B, size int) {
	msg := make([]byte, size)
	msg[0] = 0xf0
	msg[len(msg)-1] = 0xf7
	benchmarkSend(b, msg)
}

func benchmarkNotes(b *testing.B, size int) {
	msg := make([]byte, 0, 6*size)
	for i := 0; i < size; i++ {
		msg = append(msg, []byte{0x90, 0x32, 0x90, 0x80, 0x32, 0x00}...)
	}
	benchmarkSend(b, msg)
}

func BenchmarkNoteOn(b *testing.B) {
	benchmarkSend(b, []byte{0x90, 0x32, 0x90})
}

func BenchmarkNotes24(b *testing.B) {
	benchmarkNotes(b, 24)
}

func BenchmarkNotes96(b *testing.B) {
	benchmarkNotes(b, 96)
}

func BenchmarkNotes256(b *testing.B) {
	benchmarkNotes(b, 256)
}

func BenchmarkNotes1024(b *testing.B) {
	benchmarkNotes(b, 1024)
}

func BenchmarkSysEx7(b *testing.B) {
	benchmarkSysEx(b, 7)
}

func BenchmarkSysEx60(b *testing.B) {
	benchmarkSysEx(b, 60)
}

func BenchmarkSysEx512(b *testing.B) {
	benchmarkSysEx(b, 512)
}

func BenchmarkSysEx1024(b *testing.B) {
	benchmarkSysEx(b, 1024)
}

func BenchmarkSysEx2048(b *testing.B) {
	benchmarkSysEx(b, 2048)
}

func BenchmarkSysEx4096(b *testing.B) {
	benchmarkSysEx(b, 4096)
}

func BenchmarkSysEx65535(b *testing.B) {
	benchmarkSysEx(b, 65535)
}
