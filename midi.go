package midi

import(
	"fmt"
)

type Message struct {
	Command byte
	Channel byte
	Data1 byte
	Data2 byte
	SysEx []byte
}

func (m *Message) String() string {
	return fmt.Sprintf("{0x%X ch:%v %v %v}", m.Command, m.Channel, m.Data1, m.Data2)
}

type Device interface {
	Name() string
	IsInput() bool
	IsOutput() bool
	OpenInput() error
	OpenOutput() error
	Close() error

	// Channel Voice messages
	NoteOn(channel byte, note byte, velocity byte)
	NoteOff(channel byte, note byte, velocity byte)
	KeyAftertouch(channel byte, key byte, touch byte)
	ControllerChange(channel byte, controller byte, value byte)
	ProgramChange(channel byte, program byte)
	ChannelAftertouch(channel byte, touch byte)
	PitchBend(channel byte, amount uint16)

	// Channel Mode messages
	ChannelMode(channel byte, mode byte)

	// TODO: System Exclusive (SysEx) messages

	// TODO: System Common messages

	// TODO: System Realtime messages

	// Synchronously receive a single MIDI message into the given struct.
	Receive(*Message) error
	//Send(*Message) error
}

func GetDevices(driver string) ([]Device, error) {
	switch driver {
	case "oss":
		devices, err := ListAllOssDevices()
		return devices, err
	case "alsa":
		devices, err := ListAllAlsaDevices()
		return devices, err
	default:
		return nil, fmt.Errorf("Unrecognized MIDI driver '%v'.", driver)
	}
}
