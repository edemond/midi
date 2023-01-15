package midi

import (
	"fmt"
	"os"
	"strings"
)

type ossDevice struct {
	filename string
	input *os.File
	output *os.File
}

func (oss *ossDevice) Name() string {
	return oss.filename
}

func (oss *ossDevice) IsInput() bool {
	// TODO
	return true
}

func (oss *ossDevice) IsOutput() bool {
	// TODO
	return true
}

func ListAllOssDevices() ([]Device, error) {
	dev, err := os.OpenFile("/dev/snd", os.O_RDONLY|os.O_SYNC, os.ModeDir)
	if err != nil {
		return nil, err
	}
	defer dev.Close()

	filenames, err := dev.Readdirnames(0)
	if err != nil {
		return nil, err
	}

	devices := make([]Device, 0)
	for _,filename := range filenames {
		if strings.Contains(filename, "dmmidi") {
			continue
		}
		if strings.Contains(filename, "midi") {
			devices = append(devices, &ossDevice{filename: "/dev/snd/" + filename})
		}
	}
	return devices, nil
}

func (oss *ossDevice) OpenInput() error {
	device, err := os.OpenFile(oss.filename, os.O_RDONLY|os.O_SYNC, os.ModeDevice)
	if err != nil {
		return fmt.Errorf("Could not open OSS MIDI input '%v': %v", oss.filename, err)
	}
	oss.input = device
	return nil
}

func (oss *ossDevice) OpenOutput() error {
	device, err := os.OpenFile(oss.filename, os.O_WRONLY|os.O_SYNC, os.ModeDevice)
	if err != nil {
		return fmt.Errorf("Could not open OSS MIDI output '%v': %v", oss.filename, err)
	}
	oss.output = device
	return nil
}

func (oss *ossDevice) Receive(msg *Message) error {

	if oss.input == nil {
		return fmt.Errorf("Device not open for input.")
	}

	var cmdBuf [1]byte

	// TODO: Once we see how this behaves (blocking? nonblocking?) handle the potential EOF.
	_, err := oss.input.Read(cmdBuf[:])
	if err != nil {
		return err
	}

	command := (cmdBuf[0] & 0xF0) >> 4

	switch command {

	case 0x8, 0x9, 0xA, 0xB, 0xE:
		// two-byte commands
		var dataBuf [2]byte
		_, err := oss.input.Read(dataBuf[:])
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
		_, err := oss.input.Read(dataBuf[:])
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

func (oss *ossDevice) Close() error {
	// TODO: fix this so it can return both errors
	if oss.input != nil {
		err := oss.input.Close()
		if err != nil {
			return fmt.Errorf("Error closing OSS MIDI device: %v", err)
		}
		oss.input = nil
	}
	if oss.output != nil {
		err := oss.output.Close()
		if err != nil {
			return fmt.Errorf("Error closing OSS MIDI device: %v", err)
		}
		oss.output = nil
	}
	return nil
}

func (oss *ossDevice) NoteOn(channel byte, note byte, velocity byte) {
	if oss.output == nil {
		return
	}
	var msg [3]byte
	msg[0] = 0x90 | (channel - 1)
	msg[1] = note
	msg[2] = velocity
	_, err := oss.output.Write(msg[:])
	if err != nil {
		// TODO: handle somehow
	}
}

func (oss *ossDevice) NoteOff(channel byte, note byte, velocity byte) {
	if oss.output == nil {
		return
	}
	var msg [3]byte
	msg[0] = 0x80 | (channel - 1)
	msg[1] = note
	msg[2] = velocity
	_, err := oss.output.Write(msg[:])
	if err != nil {
		// TODO: handle somehow
	}
}

func (oss *ossDevice) KeyAftertouch(channel byte, key byte, touch byte) {
	if oss.output == nil {
		return
	}
	var msg [3]byte
	msg[0] = 0xA0 | (channel - 1)
	msg[1] = key
	msg[2] = touch
	_, err := oss.output.Write(msg[:])
	if err != nil {
		// TODO: handle somehow
	}
}

func (device *ossDevice) ControllerChange(channel byte, controller byte, value byte) {
}

func (device *ossDevice) ProgramChange(channel byte, program byte) {
}

func (device *ossDevice) ChannelAftertouch(channel byte, touch byte) {
}

func (device *ossDevice) PitchBend(channel byte, amount uint16) {
}

func (device *ossDevice) ChannelMode(channel byte, mode byte) {
}
