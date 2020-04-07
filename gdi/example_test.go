package gdi

import (
	"fmt"
)

func ExampleFile_IsValid() {
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
		Flags: 0,
	}

	fmt.Println(file.IsValid())
	// Output: true
}

func ExampleTrack_IsAudioTrack() {
	track := Track{
		Number:     2,
		Start:      756,
		Type:       TypeAudio,
		SectorSize: SectorSize,
		Name:       "track02.raw",
		Zero:       0,
	}

	fmt.Println(track.IsAudioTrack())
	// Output: true
}

func ExampleTrack_IsDataTrack() {
	track := Track{
		Number:     1,
		Start:      0,
		Type:       TypeData,
		SectorSize: SectorSize,
		Name:       "track01.bin",
		Zero:       0,
	}

	fmt.Println(track.IsDataTrack())
	// Output: true
}

func ExampleFile_MarshalText() {
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
		Flags: 0,
	}

	gdi, err := file.MarshalText()
	if err != nil {
		panic(err)
	}

	fmt.Println(string(gdi))
	// Output: 3
	// 1     0 4 2352 track01.bin 0
	// 2   756 0 2352 track02.raw 0
	// 3 45000 4 2352 track03.bin 0
}

func ExampleFile_UnmarshalText() {
	gdi := `3
1     0 4 2352 track01.bin 0
2   756 0 2352 track02.raw 0
3 45000 4 2352 track03.bin 0
`

	file := new(File)
	if err := file.UnmarshalText([]byte(gdi)); err != nil {
		panic(err)
	}

	fmt.Println(file)
	// Output: &{3 [{1 0 4 2352 track01.bin 0} {2 756 0 2352 track02.raw 0} {3 45000 4 2352 track03.bin 0}] 0}
}
