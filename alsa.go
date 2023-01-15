package midi

// #cgo LDFLAGS: -lasound
// #include <alsa/asoundlib.h>
import "C"

import (
	"fmt"
	"unsafe"
)

const ALSA_TRACE = false

// An ALSA RawMidi device, mostly for connections to outboard gear? TODO: Rename this.
type alsaDevice struct {
	inputDevice, outputDevice *C.snd_rawmidi_t
	deviceName string // ALSA device name, like hw:1:0:0
	name string // friendly name, like "UM-1"
	isInput, isOutput bool
	inputOpen, outputOpen bool
}

func alsaTrace(format string, args ...interface{}) {
	if ALSA_TRACE {
		fmt.Printf(format, args)
	}
}

func (alsa *alsaDevice) Name() string {
	return fmt.Sprintf("%v (%v)", alsa.name, alsa.deviceName)
}

func (alsa *alsaDevice) IsInput() bool {
	return alsa.isInput
}

func (alsa *alsaDevice) IsOutput() bool {
	return alsa.isOutput
}

func (alsa *alsaDevice) OpenInput() error {
    if !alsa.IsInput() {
		return fmt.Errorf("ALSA device %v is not an input device and cannot be opened for input.", alsa.Name())
	}
	name := C.CString(alsa.deviceName)
	defer C.free(unsafe.Pointer(name))
	var result C.int
	result = C.snd_rawmidi_open(&alsa.inputDevice, nil, name, C.SND_RAWMIDI_NONBLOCK)
	if result != 0 {
		return fmt.Errorf("Error opening ALSA MIDI device %v for input.", alsa.Name())
	}
	alsa.inputOpen = true
	return nil
}

func (alsa *alsaDevice) OpenOutput() error {
	if !alsa.IsOutput() {
		return fmt.Errorf("ALSA device %v is not an output device and cannot be opened for output.", alsa.Name())
	}
	name := C.CString(alsa.deviceName)
	defer C.free(unsafe.Pointer(name))
	var result C.int
	result = C.snd_rawmidi_open(nil, &alsa.outputDevice, name, C.SND_RAWMIDI_NONBLOCK)
	if result != 0 {
		return fmt.Errorf("Error opening ALSA MIDI device %v for output.", alsa.Name())
	}
	alsa.outputOpen = true
	return nil
}

func (alsa *alsaDevice) Close() error {
	if alsa.inputOpen {
		result := C.snd_rawmidi_close(alsa.inputDevice)
		if result != 0 {
			return fmt.Errorf("Error closing ALSA MIDI input device %v.", alsa.Name()) // TODO: Error code?
		}
		alsa.inputDevice = nil
		alsa.inputOpen = false
	}

	if alsa.outputOpen {
		result := C.snd_rawmidi_close(alsa.outputDevice)
		if result != 0 {
			return fmt.Errorf("Error closing ALSA MIDI output device %v.", alsa.Name()) // TODO: Error code?
		}
		alsa.outputDevice = nil
		alsa.outputOpen = false
	}
	return nil
}

// ported from amidi.c, list_device()
func listDevice(ctl *C.snd_ctl_t, card, device C.int) ([]Device, error) {

	devices := make([]Device, 0)
	var info *C.snd_rawmidi_info_t
	var subsIn, subsOut C.uint
	var err C.int

	C.snd_rawmidi_info_malloc(&info)
	C.snd_rawmidi_info_set_device(info, C.uint(device))

	C.snd_rawmidi_info_set_stream(info, C.SND_RAWMIDI_STREAM_INPUT)
	err = C.snd_ctl_rawmidi_info(ctl, info)
	if err >= 0 {
		subsIn = C.snd_rawmidi_info_get_subdevices_count(info)
	} else {
		subsIn = 0
	}

	C.snd_rawmidi_info_set_stream(info, C.SND_RAWMIDI_STREAM_OUTPUT)
	err = C.snd_ctl_rawmidi_info(ctl, info)
	if err >= 0 {
		subsOut = C.snd_rawmidi_info_get_subdevices_count(info)
	} else {
		subsOut = 0
	}

	// here's where it starts getting confusing...
	var subs C.uint
	if subsIn > subsOut {
		subs = subsIn
	} else {
		subs = subsOut
    }

	alsaTrace("Found %v subdevices to enumerate.", subs)

	for sub := C.uint(0); sub < subs; sub++ {

		alsaTrace("Found card %v, device %v, subdevice %v.\n", card, device, sub)

		// do not understand. wouldn't this code fail if you had,
		// say, one input-only port and one output-only port? you'd only
		// enumerate over one port, and miss the other one.
		if sub < subsIn {
			C.snd_rawmidi_info_set_stream(info, C.SND_RAWMIDI_STREAM_INPUT)
		} else {
			C.snd_rawmidi_info_set_stream(info, C.SND_RAWMIDI_STREAM_OUTPUT)
		}

		C.snd_rawmidi_info_set_subdevice(info, sub)
		err = C.snd_ctl_rawmidi_info(ctl, info)
		if err < 0 {
			return nil, fmt.Errorf("cannot get rawmidi information %v:%v:%v: %v\n",
				card, device, sub, C.snd_strerror(err))
		}

		name := C.GoString(C.snd_rawmidi_info_get_name(info))
		subName := C.GoString(C.snd_rawmidi_info_get_subdevice_name(info))
		if sub == 0 && len(subName) == 0 {
			// No subdevice name.
			devices = append(devices, &alsaDevice{deviceName: fmt.Sprintf("hw:%v,%v", card, device),
				name: name, isInput: sub < subsIn, isOutput: sub < subsOut})
			break
		} else {
			devices = append(devices, &alsaDevice{deviceName: fmt.Sprintf("hw:%v,%v,%v", card, device, sub),
				name: subName, isInput: sub < subsIn, isOutput: sub < subsOut})
		}
	}

	return devices, nil
}

// ported from amidi.c, list_card_devices()
// Return all the devices on the given card.
func listCardDevices(card C.int) ([]Device, error) {

	var ctl *C.snd_ctl_t
	var device C.int
	devices := make([]Device, 0)

	name := C.CString(fmt.Sprintf("hw:%v", card))
	defer C.free(unsafe.Pointer(name))

	// Open this card...
	if err := C.snd_ctl_open(&ctl, name, 0); err < 0 {
		return nil, fmt.Errorf("cannot open control for card %v: %v", card, C.snd_strerror(err))
	}
	defer C.snd_ctl_close(ctl)

	// ...and enumerate all the devices on it.
	device = -1
	for {
		if err := C.snd_ctl_rawmidi_next_device(ctl, &device); err < 0 {
			return nil, fmt.Errorf("cannot determine device number: %v\n", C.snd_strerror(err))
		}
		if device < 0 {
			break
		}

		alsaTrace("Checking card %v, device %v.\n", card, device)

		devs, strerr := listDevice(ctl, card, device)
		if strerr == nil {
			devices = append(devices, devs...)
		} else {
			return nil, strerr
		}
	}

	return devices, nil
}

// ported from amidi.c 
// DO NOT USE amidilist.c as a reference it does not get us the info we need
func ListAllAlsaDevices() ([]Device,error) {

	devices := make([]Device, 0)

	var card C.int
	var err C.int

	card = -1

	// Check if we can get at least one card.
	if err = C.snd_card_next(&card); err < 0 {
		strerror := C.snd_strerror(err)
		return nil, fmt.Errorf("cannot determine card number: %v", strerror)
	}
	if (card < 0) {
		return nil, fmt.Errorf("no sound card found")
	}

	// Enumerate cards.
	for (card >= 0) {

		alsaTrace("Checking card %v.\n", card)

		devs, strerr := listCardDevices(card)
		if strerr != nil {
			return nil, strerr
		}
		devices = append(devices, devs...)

		if err = C.snd_card_next(&card); err < 0 {
			return nil, fmt.Errorf("cannot determine card number: %v", C.snd_strerror(err))
		}

	}

	return devices, nil
}

func (alsa *alsaDevice) Receive(msg *Message) error {

	if !alsa.inputOpen {
		return fmt.Errorf("cannot read from a non-input device")
	}

	var cmdBuf [1]byte

	// TODO: Once we see how this behaves (blocking? nonblocking?) handle the potential EOF.
	_, err := C.snd_rawmidi_read(alsa.inputDevice, unsafe.Pointer(&cmdBuf), 1)
	if err != nil {
		return err
	}

	command := (cmdBuf[0] & 0xF0) >> 4

	switch command {

	case 0x8, 0x9, 0xA, 0xB, 0xE:
		// two-byte commands
		var dataBuf [2]byte
		_, err := C.snd_rawmidi_read(alsa.inputDevice, unsafe.Pointer(&dataBuf), 2)
		if err != nil {
			return err
		}
		msg.Command = command
		msg.Channel = cmdBuf[0] & 0x0F
		msg.Data1 = dataBuf[0]
		msg.Data2 = dataBuf[1]
		return nil

	case 0xC, 0xD:
		// one-byte commands
		var dataBuf [1]byte
		_, err := C.snd_rawmidi_read(alsa.inputDevice, unsafe.Pointer(&dataBuf), 1)
		if err != nil {
			return err
		}
		msg.Command = command
		msg.Channel = cmdBuf[0] & 0x0F
		msg.Data1 = dataBuf[0]
		return nil

	case 0xF:
		// TODO: SysEx commands
		return fmt.Errorf("no SysEx yet sorry")
	}

	return fmt.Errorf("Unknown MIDI command: %v", command)

}

func (alsa *alsaDevice) NoteOn(channel byte, note byte, velocity byte) {
	if alsa.outputOpen {
		var msg [3]byte
		msg[0] = 0x90 | (channel-1)
		msg[1] = note
		msg[2] = velocity
		C.snd_rawmidi_write(alsa.outputDevice, unsafe.Pointer(&msg), 3)
	}
}

func (alsa *alsaDevice) NoteOff(channel byte, note byte, velocity byte) {
	if alsa.outputOpen {
		var msg [3]byte
		msg[0] = 0x80 | (channel-1)
		msg[1] = note
		msg[2] = velocity
		C.snd_rawmidi_write(alsa.outputDevice, unsafe.Pointer(&msg), 3)
	}
}

func (alsa *alsaDevice) KeyAftertouch(channel byte, key byte, touch byte) {
	if alsa.outputOpen {
		var msg [3]C.uchar
		msg[0] = C.uchar(0xA0 | (channel-1))
		msg[1] = C.uchar(key)
		msg[2] = C.uchar(touch)
		C.snd_rawmidi_write(alsa.outputDevice, unsafe.Pointer(&msg), 3)
	}
}

func (alsa *alsaDevice) ControllerChange(channel byte, controller byte, value byte) {
	if alsa.outputOpen {
		var msg [3]C.uchar
		msg[0] = C.uchar(0xB0 | (channel-1))
		msg[1] = C.uchar(controller)
		msg[2] = C.uchar(value)
		C.snd_rawmidi_write(alsa.outputDevice, unsafe.Pointer(&msg), 3)
	}
}

func (device *alsaDevice) ProgramChange(channel byte, program byte) {
}

func (device *alsaDevice) ChannelAftertouch(channel byte, touch byte) {
}

func (device *alsaDevice) PitchBend(channel byte, amount uint16) {
}

func (device *alsaDevice) ChannelMode(channel byte, mode byte) {
}

