/*
Package gdi implements parsing of Sega Dreamcast GDI files. Basic
checks are performed pre-marshalling or post-unmarshalling to ensure it
is valid.
*/
package gdi

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

const (
	// Extension is the conventional file extension used
	Extension = ".gdi"
	// SectorSize is the standard sector size used for tracks
	SectorSize = 2352
	// TrackThreeStart is the starting sector for track three, the
	// beginning of the high density area
	TrackThreeStart = 45000
)

const (
	minTracks = 3
	maxTracks = 99
)

// Type represents the type of track
type Type int

const (
	// TypeAudio is used for audio tracks
	TypeAudio Type = iota
	_
	_
	_
	// TypeData is used for data tracks
	TypeData
)

// Flag represents additional formatting tweaks
type Flag int

const (
	// NoWhitespace disables padding/alignment with additional spaces
	NoWhitespace Flag = 1 << iota
)

var (
	errInvalidTrack        = errors.New("invalid track")
	errNotEnoughTracks     = errors.New("not enough tracks")
	errTooManyTracks       = errors.New("too many tracks")
	errInconsistentTracks  = errors.New("inconsistent tracks")
	errInvalidStart        = errors.New("invalid start")
	errInvalidType         = errors.New("invalid track type")
	errNonContinuousTracks = errors.New("non-continuous tracks")
	errInvalidSectorSize   = errors.New("invalid sector size")
	errFieldNotZero        = errors.New("field not zero")
)

// File represents a GDI file
type File struct {
	// Count is the number of tracks in the GDI file
	Count int
	// Tracks contains each track
	Tracks []Track
	// Flags manages any additional formatting tweaks
	Flags Flag
}

// Track represents a single track within a GDI file
type Track struct {
	// Number is the track number
	Number int
	// Start refers to the first sector of the track
	Start int
	// Type refers to the type of track, audio or data
	Type Type
	// SectorSize is the sector size used by the track
	SectorSize int
	// Name is the filename of the track relative to the GDI file
	Name string
	// Zero is always set to zero
	Zero int
}

const (
	trackNumber = iota
	trackStart
	trackType
	trackSectorSize
	trackName
	trackZero
	trackFields
)

func split(s string) ([]string, error) {
	var withinQuotes = false
	fields := strings.FieldsFunc(s, func(c rune) bool {
		if c == '"' {
			withinQuotes = !withinQuotes
		}
		return unicode.IsSpace(c) && !withinQuotes
	})

	if withinQuotes || len(fields) != trackFields {
		return nil, errInvalidTrack
	}

	return fields, nil
}

func (f *File) validate() error {
	if f.Count < minTracks {
		return errNotEnoughTracks
	}

	if f.Count > maxTracks {
		return errTooManyTracks
	}

	if len(f.Tracks) != f.Count {
		return errInconsistentTracks
	}

	for i, track := range f.Tracks {
		switch i {
		case 2: // 3rd track, always starts at 45000
			if track.Start != TrackThreeStart {
				return errInvalidStart
			}
			fallthrough
		case 0: // 1st (and 3rd) tracks, should be data
			if track.Type != TypeData {
				return errInvalidType
			}
		case 1: // 2nd track, should be audio
			if track.Type != TypeAudio {
				return errInvalidType
			}
		}

		if track.Number != i+1 {
			return errNonContinuousTracks
		}

		if track.SectorSize != SectorSize {
			return errInvalidSectorSize
		}

		if track.Zero != 0 {
			return errFieldNotZero
		}
	}

	return nil
}

// MarshalText encodes the GDI file into textual form
func (f File) MarshalText() ([]byte, error) {
	if err := f.validate(); err != nil {
		return nil, err
	}

	b := new(bytes.Buffer)

	last := f.Tracks[len(f.Tracks)-1]
	numberWidth := len(strconv.FormatUint(uint64(last.Number), 10))
	if f.Flags&NoWhitespace != 0 {
		numberWidth = 1
	}
	startWidth := len(strconv.FormatUint(uint64(last.Start), 10))
	if f.Flags&NoWhitespace != 0 {
		startWidth = 1
	}

	fmt.Fprintf(b, "%d\n", len(f.Tracks))

	for _, track := range f.Tracks {
		name := track.Name
		if strings.ContainsAny(name, " ") {
			name = `"` + name + `"`
		}

		fmt.Fprintf(b, "%*d %*d %d %d %s %d\n", numberWidth, track.Number, startWidth, track.Start, track.Type, track.SectorSize, name, track.Zero)
	}

	return b.Bytes(), nil
}

// UnmarshalText decodes the GDI file from textual form
func (f *File) UnmarshalText(text []byte) error {
	// Clear out any existing state
	f.Count, f.Tracks, f.Flags = 0, []Track{}, 0

	s, i := bufio.NewScanner(bytes.NewReader(text)), 0
	for s.Scan() {
		switch i {
		case 0:
			var err error
			total, err := strconv.Atoi(s.Text())
			if err != nil {
				return err
			}
			f.Count = total
		default:
			fields, err := split(s.Text())
			if err != nil {
				return err
			}

			track := Track{}

			track.Number, err = strconv.Atoi(fields[trackNumber])
			if err != nil {
				return err
			}

			track.Start, err = strconv.Atoi(fields[trackStart])
			if err != nil {
				return err
			}

			t, err := strconv.Atoi(fields[trackType])
			if err != nil {
				return err
			}
			track.Type = Type(t)

			track.SectorSize, err = strconv.Atoi(fields[trackSectorSize])
			if err != nil {
				return err
			}

			track.Name = strings.Trim(fields[trackName], `"`)

			track.Zero, err = strconv.Atoi(fields[trackZero])
			if err != nil {
				return err
			}

			f.Tracks = append(f.Tracks, track)
		}
		i++
	}
	if err := s.Err(); err != nil {
		return err
	}

	return f.validate()
}
