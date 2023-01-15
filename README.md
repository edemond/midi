# MIDI

A little Go MIDI library with support for ALSA and OSS on Linux. Don't use; not really complete. Mainly for development on [https://github.com/edemond/abstract](https://github.com/edemond/abstract).

It's able to do a little device enumeration in ALSA, which was hard to figure out from the ALSA documentation and not sure if I'm doing it in the best way.

## Prerequisites

Requires ALSA development libraries, as it binds to those with cgo.
