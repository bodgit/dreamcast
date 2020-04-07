package dreamcast

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	ipBinLength = 0x8000
	lastSector  = 0x861b4
	space       = " "
)

const (
	offsetHardwareID        = 0x000
	offsetMakerID           = 0x010
	offsetDeviceInformation = 0x020
	offsetAreaSymbols       = 0x030
	offsetPeripherals       = 0x038
	offsetProductNumber     = 0x040
	offsetProductVersion    = 0x04a
	offsetReleaseDate       = 0x050
	offsetBootFilename      = 0x060
	offsetProducer          = 0x070
	offsetSoftwareName      = 0x080
	offsetTOC               = 0x100
)

// Region represents the permitted regions
type Region [offsetPeripherals - offsetAreaSymbols]byte

// IsRegionJapan returns true if Japan region is permitted
func (r Region) IsRegionJapan() bool {
	return r[0] == 'J'
}

// IsRegionUSA returns true if USA region is permitted
func (r Region) IsRegionUSA() bool {
	return r[1] == 'U'
}

// IsRegionEurope returns true if Europe region is permitted
func (r Region) IsRegionEurope() bool {
	return r[2] == 'E'
}

func (r Region) String() string {
	return string(r[:])
}

// Peripheral maps to each peripheral option
type Peripheral int

// These are the individual peripherals
const (
	PeripheralWindowsCE Peripheral = 1 << iota
	_
	_
	_
	PeripheralVGABox
	_
	_
	_
	PeripheralOtherExpansions
	PeripheralVibrationPack
	PeripheralMicrophone
	PeripheralMemoryCard
	PeripheralStartABDirections
	PeripheralCButton
	PeripheralDButton
	PeripheralXButton
	PeripheralYButton
	PeripheralZButton
	PeripheralExpandedDirections
	PeripheralRTrigger
	PeripheralLTrigger
	PeripheralHorizontal
	PeripheralVertical
	PeripheralExpandedHorizontal
	PeripheralExpandedVertical
	PeripheralGun
	PeripheralKeyboard
	PeripheralMouse
)

// IsSet returns true if the given peripheral is set
func (p Peripheral) IsSet(peripherals uint32) bool {
	return (peripherals & uint32(p)) != 0
}

// IPBin represents the IP.BIN initial program. It implements the
// encoding.BinaryUnmarshaler interface.
type IPBin struct {
	bytes          []byte
	HardwareID     string
	MakerID        string
	CRC            uint16
	Disc           int
	TotalDiscs     int
	Regions        Region
	Peripherals    uint32
	ProductNumber  string
	ProductVersion string
	ReleaseDate    time.Time
	BootFilename   string
	Producer       string
	SoftwareName   string
	TOC            []Track
}

func crc(b []byte) uint16 {
	n := uint16(0xffff)
	for _, x := range b {
		n ^= uint16(x) << 8
		for i := 0; i < 8; i++ {
			if n&0x8000 != 0 {
				n = n<<1 ^ 4129
			} else {
				n = n << 1
			}
		}
	}
	return n & 0xffff
}

// UnmarshalBinary decodes the IP.BIN from binary form
func (ip *IPBin) UnmarshalBinary(b []byte) error {
	if len(b) != ipBinLength {
		return errors.New("incorrect amount of bytes for IP.BIN")
	}

	ip.bytes = b

	// Copy out all of the simple space-padded strings
	ip.HardwareID = strings.TrimRight(string(ip.bytes[offsetHardwareID:offsetMakerID]), space)
	ip.MakerID = strings.TrimRight(string(ip.bytes[offsetMakerID:offsetDeviceInformation]), space)
	ip.ProductNumber = strings.TrimRight(string(ip.bytes[offsetProductNumber:offsetProductVersion]), space)
	ip.ProductVersion = strings.TrimRight(string(ip.bytes[offsetProductVersion:offsetReleaseDate]), space)
	ip.BootFilename = strings.TrimRight(string(ip.bytes[offsetBootFilename:offsetProducer]), space)
	ip.Producer = strings.TrimRight(string(ip.bytes[offsetProducer:offsetSoftwareName]), space)
	ip.SoftwareName = strings.TrimRight(string(ip.bytes[offsetSoftwareName:offsetTOC]), space)

	// Copy out the CRC and decode it back into a number
	crc, err := hex.DecodeString(string(ip.bytes[offsetDeviceInformation : offsetDeviceInformation+4]))
	if err != nil {
		return err
	}
	ip.CRC = binary.BigEndian.Uint16(crc)

	// Extract the current disc and total disc counts
	if _, err = fmt.Sscanf(string(ip.bytes[offsetDeviceInformation+5:offsetPeripherals]), "GD-ROM%d/%d", &ip.Disc, &ip.TotalDiscs); err != nil {
		return err
	}

	// Extract the regions
	copy(ip.Regions[:], ip.bytes[offsetAreaSymbols:offsetPeripherals])

	// Extract the peripheral bitmask. The number is seven digits long and
	// needs padding to an even number of digits in order to decode it
	peripherals, err := hex.DecodeString("0" + string(ip.bytes[offsetPeripherals:offsetProductNumber-1]))
	if err != nil {
		return err
	}
	ip.Peripherals = binary.BigEndian.Uint32(peripherals)

	// Parse the release date into a proper time object
	if ip.ReleaseDate, err = time.Parse("20060102", strings.TrimRight(string(ip.bytes[offsetReleaseDate:offsetBootFilename]), space)); err != nil {
		return err
	}

	// 99 tracks potentially on a GDROM, but the first two are in the low
	// density area so we ignore those
	for i := 0; i < 97; i++ {
		start := ip.bytes[offsetTOC+4+i*4 : offsetTOC+7+i*4]
		track := Track{
			Start: int(start[0]) + int(start[1])<<8 + int(start[2])<<16 - pauseData,
			Type:  int(ip.bytes[offsetTOC+7+i*4]),
		}
		if !track.IsAudioTrack() && !track.IsDataTrack() {
			break
		}
		ip.TOC = append(ip.TOC, track)
	}

	for i := 0; i < len(ip.TOC)-1; i++ {
		ip.TOC[i].Length = ip.TOC[i+1].Start - ip.TOC[i].Start - pauseData
	}
	ip.TOC[len(ip.TOC)-1].Length = lastSector - ip.TOC[len(ip.TOC)-1].Start - pauseData

	return nil
}

func (ip IPBin) String() string {
	return hex.Dump(ip.bytes)
}

// Track represents a TOC entry found in the IP.BIN initial program
type Track struct {
	// Start refers to the first sector of the track
	Start int
	// Length refers to the length of the track in sectors
	Length int
	// Type is the type of track, audio or data
	Type int
}

const (
	typeAudio = 0x01
	typeData  = 0x41
)

// IsAudioTrack returns true if the track is an audio track
func (t Track) IsAudioTrack() bool {
	return t.Type == typeAudio
}

// IsDataTrack returns true if the track is a data track
func (t Track) IsDataTrack() bool {
	return t.Type == typeData
}
