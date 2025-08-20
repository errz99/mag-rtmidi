#include <stdlib.h>
#include <stdint.h>
#include "rtmidi/rtmidi_c.h"


void rtmidi_set_error_quiet (RtMidiPtr device);

extern void goMIDIInCallback(double ts, unsigned char *msg, size_t msgsz, void *arg);

static inline void midiInCallback(double ts, const unsigned char *msg, size_t msgsz, void *arg) {
    goMIDIInCallback(ts, (unsigned char*) msg, msgsz, arg);
}

static inline void cgoSetCallback(RtMidiPtr in, void *arg) {
    rtmidi_in_set_callback(in, midiInCallback, arg);
}

static void testCallback(RtMidiPtr in) {
    double ts = 3.14159;
    unsigned char buf[] = {0x90, 0x30, 0x00};

    goMIDIInCallback(ts, buf, sizeof(buf), (void *) in);
}
