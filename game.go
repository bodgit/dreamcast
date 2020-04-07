package dreamcast

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"os"

	"github.com/bodgit/dreamcast/gdi"
	"github.com/vchimishuk/chub/cue"
)

const (
	pauseData = 150
	preGap    = 75
)

var (
	errInvalidType             = errors.New("invalid track type")
	errInvalidSize             = errors.New("invalid track size")
	errInvalidCueFile          = errors.New("invalid cue file")
	errInvalidGame             = errors.New("invalid game")
	errInconsistentAudioTracks = errors.New("inconsistent audio tracks")
)

// Game represents a Sega Dreamcast game image
type Game struct {
	// GDIFile is the name of the GDI file that was read
	GDIFile string
	// CueFile is the name of the cue file that was read
	CueFile string
	// IPBin represents the IP.BIN initial program found in the third track
	IPBin *IPBin

	reader  Reader
	gdiFile *gdi.File
}

var cueTrackTypeToGDIType = map[cue.TrackDataType]gdi.Type{
	cue.DataTypeAudio:      gdi.TypeAudio,
	cue.DataTypeMode1_2352: gdi.TypeData,
}

func (g *Game) newFromCueFile() error {
	r, filename, err := g.reader.FindCueFile()
	if err != nil {
		return err
	}
	g.CueFile = filename

	sheet, err := cue.Parse(r)
	if err != nil {
		return err
	}

	start := 0
	for _, file := range sheet.Files {
		for _, t := range file.Tracks {
			trackType, ok := cueTrackTypeToGDIType[t.DataType]
			if !ok {
				return errInvalidType
			}

			track := gdi.Track{
				Number:     t.Number,
				Start:      start,
				Type:       trackType,
				SectorSize: gdi.SectorSize,
				Name:       file.Name,
				Zero:       0,
			}

			switch t.Number {
			case 2:
				start = gdi.TrackThreeStart
			default:
				size, err := g.reader.FileSize(file.Name)
				if err != nil {
					return err
				}

				if size%gdi.SectorSize != 0 {
					return errInvalidSize
				}

				start += int(size / uint64(gdi.SectorSize))
			}

			g.gdiFile.Tracks = append(g.gdiFile.Tracks, track)
		}
	}
	g.gdiFile.Count = len(g.gdiFile.Tracks)

	// This checks the tracks are all of the correct type
	if !g.gdiFile.IsValid() {
		return errInvalidCueFile
	}

	return nil
}

// NewGame returns a Game object read using the passed Reader. A GDI file is
// searched for first, followed by a cue sheet.
func NewGame(reader Reader) (*Game, error) {
	game := &Game{
		reader:  reader,
		gdiFile: new(gdi.File),
	}

	r, filename, err := game.reader.FindGDIFile()
	if err != nil {
		if e, ok := err.(*os.PathError); !ok || !os.IsNotExist(e) {
			return nil, err
		}

		if err := game.newFromCueFile(); err != nil {
			return nil, err
		}
	} else {
		game.GDIFile = filename

		b, err := ioutil.ReadAll(r)
		if err != nil {
			return nil, err
		}

		if err := game.gdiFile.UnmarshalText(b); err != nil {
			return nil, err
		}
	}

	if err := game.readIPBin(); err != nil {
		return nil, err
	}

	return game, nil
}

func (g *Game) readIPBin() error {
	file, err := g.reader.OpenFile(g.gdiFile.Tracks[2].Name)
	if err != nil {
		return err
	}
	defer file.Close()

	buf := new(bytes.Buffer)
	buf.Grow(ipBinLength) // Size the buffer to 32 KiB

	// Loop over the first 16 sectors
	for i := 0; i < 16; i++ {
		// Skip over the sync data
		if _, err := io.CopyN(ioutil.Discard, file, 16); err != nil {
			return err
		}

		// Read 2048 bytes
		if _, err := io.CopyN(buf, file, 2048); err != nil {
			return err
		}

		// Skip the rest of the sector
		if _, err := io.CopyN(ioutil.Discard, file, gdi.SectorSize-2064); err != nil {
			return err
		}
	}

	g.IPBin = new(IPBin)
	if err := g.IPBin.UnmarshalBinary(buf.Bytes()); err != nil {
		return err
	}

	return nil
}

func (g Game) isValid() error {
	if !g.gdiFile.IsValid() {
		return errInvalidGame
	}

	for _, track := range g.gdiFile.Tracks {
		size, err := g.reader.FileSize(track.Name)
		if err != nil {
			return err
		}

		if size%gdi.SectorSize != 0 {
			return errInvalidSize
		}
	}

	return nil
}

func (g Game) isRedump() (bool, error) {
	if err := g.isValid(); err != nil {
		return false, err
	}

	audioTracks, redumpTracks := 0, 0
	for _, track := range g.gdiFile.Tracks {
		if !track.IsAudioTrack() {
			continue
		}

		audioTracks++

		file, err := g.reader.OpenFile(track.Name)
		if err != nil {
			return false, err
		}
		defer file.Close()

		buf := new(bytes.Buffer)
		if _, err := io.CopyN(buf, file, 16); err != nil {
			return false, err
		}

		if bytes.Compare(buf.Bytes(), bytes.Repeat([]byte{0}, 16)) == 0 {
			redumpTracks++
		}

		file.Close()
	}

	if redumpTracks > 0 && redumpTracks < audioTracks {
		return false, errInconsistentAudioTracks
	}

	return redumpTracks == audioTracks, nil
}

func writeGDIFile(writer Writer, gdiFile *gdi.File) error {
	if writer.Config().TrimWhitespace {
		gdiFile.Flags = gdi.TrimWhitespace
	}

	b, err := gdiFile.MarshalText()
	if err != nil {
		return err
	}

	file, err := writer.CreateFile(writer.Config().GDIFile)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.Write(b); err != nil {
		return err
	}

	return nil
}

func writeCueFile(writer Writer, gdiFile *gdi.File) error {
	// TODO
	return nil
}

func (g Game) Write(writer Writer) error {
	isRedump, err := g.isRedump()
	if err != nil {
		return err
	}

	gdiFile := g.gdiFile.Copy()

	var dst io.WriteCloser
	for i, track := range g.gdiFile.Tracks {
		src, err := g.reader.OpenFile(track.Name)
		if err != nil {
			return err
		}
		defer src.Close()

		if isRedump {
			switch {
			case track.IsDataTrack() && track.Number == g.gdiFile.Count && track.Number > 3:
				if _, err := io.CopyN(dst, src, preGap*gdi.SectorSize); err != nil {
					return err
				}
				gdiFile.Tracks[i].Start += preGap
				fallthrough
			case track.IsAudioTrack():
				if _, err := io.CopyN(ioutil.Discard, src, pauseData*gdi.SectorSize); err != nil {
					return err
				}
				gdiFile.Tracks[i].Start += pauseData
			}
		}

		if writer.Config().TrackRename != nil {
			gdiFile.Tracks[i].Name = writer.Config().TrackRename(track)
		}

		if i > 0 {
			dst.Close()
		}

		dst, err = writer.CreateFile(gdiFile.Tracks[i].Name)
		if err != nil {
			return err
		}
		defer dst.Close()

		if _, err := io.Copy(dst, src); err != nil {
			return err
		}

		src.Close()
	}

	dst.Close()

	if writer.Config().GDIFile != "" {
		if err := writeGDIFile(writer, gdiFile); err != nil {
			return err
		}
	}

	if writer.Config().CueFile != "" {
		if err := writeCueFile(writer, gdiFile); err != nil {
			return err
		}
	}

	return nil
}
