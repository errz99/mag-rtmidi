#if defined(GO_COVERAGE_TEST)
#undef __MACOSX_CORE__
#undef __LINUX_ALSA__
#undef __UNIX_JACK__
#undef __WINDOWS_MM__
#undef __WEB_MIDI_API__
#define __RTMIDI_DUMMY__
#endif

#include "rtmidi/RtMidi.h"

#define RTMIDI_SOURCE_INCLUDED
#include "rtmidi/RtMidi.cpp"
#include "rtmidi/rtmidi_c.cpp"


static void quietErr(RtMidiError::Type type, const std::string &errorText, void *userData) {
    if ( type != RtMidiError::WARNING && type != RtMidiError::DEBUG_WARNING ) {
        throw RtMidiError( errorText, type );
    }
}

extern "C" void rtmidi_set_error_quiet (RtMidiPtr device) {
    try {
        ((RtMidiIn*) device->ptr)->setErrorCallback (quietErr, NULL);
    } catch (const RtMidiError & err) {
        device->ok  = false;
        device->msg = err.what ();
    }
}

