package gdi

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

var strconvNumError = strconv.NumError{
	Func: "Atoi",
	Num:  "INVALID",
	Err:  strconv.ErrSyntax,
}

func TestType(t *testing.T) {
	assert.Equal(t, Type(0), TypeAudio)
	assert.Equal(t, Type(4), TypeData)
}

func TestIsAudioTrack(t *testing.T) {
	file := File{
		Count: 3,
		Tracks: []Track{
			{
				Number:     1,
				Start:      0,
				Type:       TypeData,
				SectorSize: SectorSize,
				Name:       "track01.bin",
				Zero:       0,
			},
			{
				Number:     2,
				Start:      756,
				Type:       TypeAudio,
				SectorSize: SectorSize,
				Name:       "track02.raw",
				Zero:       0,
			},
			{
				Number:     3,
				Start:      TrackThreeStart,
				Type:       TypeData,
				SectorSize: SectorSize,
				Name:       "track03.bin",
				Zero:       0,
			},
		},
	}

	assert.Equal(t, true, file.Tracks[1].IsAudioTrack())
	assert.Equal(t, false, file.Tracks[0].IsAudioTrack())
}

func TestIsDataTrack(t *testing.T) {
	file := File{
		Count: 3,
		Tracks: []Track{
			{
				Number:     1,
				Start:      0,
				Type:       TypeData,
				SectorSize: SectorSize,
				Name:       "track01.bin",
				Zero:       0,
			},
			{
				Number:     2,
				Start:      756,
				Type:       TypeAudio,
				SectorSize: SectorSize,
				Name:       "track02.raw",
				Zero:       0,
			},
			{
				Number:     3,
				Start:      TrackThreeStart,
				Type:       TypeData,
				SectorSize: SectorSize,
				Name:       "track03.bin",
				Zero:       0,
			},
		},
	}

	assert.Equal(t, true, file.Tracks[0].IsDataTrack())
	assert.Equal(t, false, file.Tracks[1].IsDataTrack())
}

func TestIsValid(t *testing.T) {
	tables := []struct {
		got  File
		want bool
	}{
		{
			File{
				Count: 3,
				Tracks: []Track{
					{
						Number:     1,
						Start:      0,
						Type:       TypeData,
						SectorSize: SectorSize,
						Name:       "track01.bin",
						Zero:       0,
					},
					{
						Number:     2,
						Start:      756,
						Type:       TypeAudio,
						SectorSize: SectorSize,
						Name:       "track02.raw",
						Zero:       0,
					},
					{
						Number:     3,
						Start:      TrackThreeStart,
						Type:       TypeData,
						SectorSize: SectorSize,
						Name:       "track03.bin",
						Zero:       0,
					},
				},
			},
			true,
		},
		{
			File{
				Count: 1,
				Tracks: []Track{
					{
						Number:     1,
						Start:      0,
						Type:       TypeData,
						SectorSize: SectorSize,
						Name:       "track01.bin",
						Zero:       0,
					},
				},
			},
			false,
		},
	}

	for _, table := range tables {
		assert.Equal(t, table.want, table.got.IsValid())
	}
}

func TestUnmarshalText(t *testing.T) {
	tables := []struct {
		got  string
		want *File
		err  error
	}{
		{
			`3
1 0 4 2352 track01.bin 0
2 756 0 2352 "track02.raw" 0
3 45000 4 2352 track03.bin 0
`,
			&File{
				Count: 3,
				Tracks: []Track{
					{
						Number:     1,
						Start:      0,
						Type:       TypeData,
						SectorSize: SectorSize,
						Name:       "track01.bin",
						Zero:       0,
					},
					{
						Number:     2,
						Start:      756,
						Type:       TypeAudio,
						SectorSize: SectorSize,
						Name:       "track02.raw",
						Zero:       0,
					},
					{
						Number:     3,
						Start:      TrackThreeStart,
						Type:       TypeData,
						SectorSize: SectorSize,
						Name:       "track03.bin",
						Zero:       0,
					},
				},
			},
			nil,
		},
		// Unbalanced quotes
		{
			`1
1 0 4 2352 "track01.bin 0
`,
			nil,
			errInvalidTrack,
		},
		// Invalid numeric fields
		{
			`INVALID
`,
			nil,
			&strconvNumError,
		},
		{
			`1
INVALID 0 4 2352 track01.bin 0
`,
			nil,
			&strconvNumError,
		},
		{
			`1
1 INVALID 4 2352 track01.bin 0
`,
			nil,
			&strconvNumError,
		},
		{
			`1
1 0 INVALID 2352 track01.bin 0
`,
			nil,
			&strconvNumError,
		},
		{
			`1
1 0 4 INVALID track01.bin 0
`,
			nil,
			&strconvNumError,
		},
		{
			`1
1 0 4 2352 track01.bin INVALID
`,
			nil,
			&strconvNumError,
		},
		// Invalid track counts
		{
			`1
`,
			nil,
			errNotEnoughTracks,
		},
		{
			`100
`,
			nil,
			errTooManyTracks,
		},
		// Mismatched track count and number of tracks
		{
			`3
1 0 4 2352 track01.bin 0
`,
			nil,
			errInconsistentTracks,
		},
		// Wrong start for track 3
		{
			`3
1 0 4 2352 track01.bin 0
2 756 0 2352 "track02.raw" 0
3 45001 4 2352 track03.bin 0
`,
			nil,
			errInvalidStart,
		},
		// Wrong type for track 1
		{
			`3
1 0 0 2352 track01.bin 0
2 756 0 2352 "track02.raw" 0
3 45000 4 2352 track03.bin 0
`,
			nil,
			errInvalidType,
		},
		// Wrong type for track 2
		{
			`3
1 0 4 2352 track01.bin 0
2 756 4 2352 "track02.raw" 0
3 45000 4 2352 track03.bin 0
`,
			nil,
			errInvalidType,
		},
		// Track starts go backwards
		{
			`3
1 756 4 2352 track01.bin 0
2 0 0 2352 "track02.raw" 0
3 45000 4 2352 track03.bin 0
`,
			nil,
			errOverlappingTracks,
		},
		// Jump in track number
		{
			`3
1 0 4 2352 track01.bin 0
2 756 0 2352 "track02.raw" 0
4 45000 4 2352 track03.bin 0
`,
			nil,
			errNonContinuousTracks,
		},
		// Invalid sector size
		{
			`3
1 0 4 2048 track01.bin 0
2 756 0 2352 "track02.raw" 0
3 45000 4 2352 track03.bin 0
`,
			nil,
			errInvalidSectorSize,
		},
		// Last field not zero
		{
			`3
1 0 4 2352 track01.bin 1
2 756 0 2352 "track02.raw" 0
3 45000 4 2352 track03.bin 0
`,
			nil,
			errFieldNotZero,
		},
	}

	for _, table := range tables {
		f := new(File)
		err := f.UnmarshalText([]byte(table.got))
		assert.Equal(t, table.err, err)
		if err == nil {
			assert.Equal(t, table.want, f)
		}
	}
}

func TestMarshalText(t *testing.T) {
	tables := []struct {
		got  File
		want string
		err  error
	}{
		{
			File{
				Count: 3,
				Tracks: []Track{
					{
						Number:     1,
						Start:      0,
						Type:       TypeData,
						SectorSize: SectorSize,
						Name:       "track01.bin",
						Zero:       0,
					},
					{
						Number:     2,
						Start:      756,
						Type:       TypeAudio,
						SectorSize: SectorSize,
						Name:       "track02.raw",
						Zero:       0,
					},
					{
						Number:     3,
						Start:      TrackThreeStart,
						Type:       TypeData,
						SectorSize: SectorSize,
						Name:       "track03.bin",
						Zero:       0,
					},
				},
			},
			`3
1     0 4 2352 track01.bin 0
2   756 0 2352 track02.raw 0
3 45000 4 2352 track03.bin 0
`,
			nil,
		},
		{
			File{
				Count: 3,
				Tracks: []Track{
					{
						Number:     1,
						Start:      0,
						Type:       TypeData,
						SectorSize: SectorSize,
						Name:       "track01.bin",
						Zero:       0,
					},
					{
						Number:     2,
						Start:      756,
						Type:       TypeAudio,
						SectorSize: SectorSize,
						Name:       "track02.raw",
						Zero:       0,
					},
					{
						Number:     3,
						Start:      TrackThreeStart,
						Type:       TypeData,
						SectorSize: SectorSize,
						Name:       "track03.bin",
						Zero:       0,
					},
				},
				Flags: TrimWhitespace,
			},
			`3
1 0 4 2352 track01.bin 0
2 756 0 2352 track02.raw 0
3 45000 4 2352 track03.bin 0
`,
			nil,
		},
		{
			File{
				Count: 3,
				Tracks: []Track{
					{
						Number:     1,
						Start:      0,
						Type:       TypeData,
						SectorSize: SectorSize,
						Name:       "track01.bin",
						Zero:       0,
					},
					{
						Number:     2,
						Start:      756,
						Type:       TypeAudio,
						SectorSize: SectorSize,
						Name:       "track 02.raw",
						Zero:       0,
					},
					{
						Number:     3,
						Start:      TrackThreeStart,
						Type:       TypeData,
						SectorSize: SectorSize,
						Name:       "track03.bin",
						Zero:       0,
					},
				},
			},
			`3
1     0 4 2352 track01.bin 0
2   756 0 2352 "track 02.raw" 0
3 45000 4 2352 track03.bin 0
`,
			nil,
		},
		{
			File{
				Count: 1,
				Tracks: []Track{
					{
						Number:     1,
						Start:      0,
						Type:       TypeData,
						SectorSize: SectorSize,
						Name:       "track01.bin",
						Zero:       0,
					},
				},
			},
			"",
			errNotEnoughTracks,
		},
	}

	for _, table := range tables {
		b, err := table.got.MarshalText()
		assert.Equal(t, table.err, err)
		if err == nil {
			assert.Equal(t, table.want, string(b))
		}
	}
}

func TestCopy(t *testing.T) {
	file := &File{
		Count: 1,
		Tracks: []Track{
			{
				Number:     1,
				Start:      0,
				Type:       TypeData,
				SectorSize: SectorSize,
				Name:       "track01.bin",
				Zero:       0,
			},
		},
	}
	clone := file.Copy()

	assert.NotSame(t, file, clone)
	assert.Equal(t, file, clone)
	file.Tracks[0].Type = TypeAudio
	assert.NotEqual(t, file, clone)
}
