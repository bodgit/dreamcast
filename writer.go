package dreamcast

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/bodgit/dreamcast/gdi"
	"github.com/bodgit/plumbing"
)

// Writer is the interface implemented by an object that can be used as a
// destination for writing a Dreamcast game image to disk
type Writer interface {
	// Close closes the destination
	Close() error
	// CreateFile returns an io.WriteCloser opened on the named file
	CreateFile(string) (io.WriteCloser, error)
	// Config returns the WriterConfig associated with this writer
	Config() WriterConfig
	// Tx returns the number of bytes written
	Tx() uint64
}

// WriterConfig contains the configuration of the Writer
type WriterConfig struct {
	// CueFile is the target filename for a cue file
	CueFile string
	// GDIFile is the target filename for a GDI file
	GDIFile string
	// TrackRename is a function to rename tracks. The function is passed
	// a gdi.Track object and returns a string representing the desired
	// filename
	TrackRename func(gdi.Track) string
	// TrimWhitespace controls whether extra passing whitespace is removed
	// from either the GDI or cue file where applicable
	TrimWhitespace bool
}

// GDemuTrackName is a track renaming function that names each track how a
// GDemu device expects them
func GDemuTrackName(track gdi.Track) string {
	switch {
	case track.IsAudioTrack():
		return fmt.Sprintf("track%02d.raw", track.Number)
	case track.IsDataTrack():
		return fmt.Sprintf("track%02d.bin", track.Number)
	default:
		return track.Name
	}
}

// DirectoryWriter writes a Dreamcast game to a directory
type DirectoryWriter struct {
	directory string
	config    WriterConfig
	tx        plumbing.WriteCounter
}

// NewDirectoryWriter returns a DirectoryWriter using the passed directory
// path and config
func NewDirectoryWriter(directory string, config WriterConfig) (*DirectoryWriter, error) {
	if err := os.MkdirAll(directory, os.ModePerm); err != nil {
		return nil, err
	}

	w := &DirectoryWriter{
		directory: directory,
		config:    config,
	}

	return w, nil
}

// Close closes the directory
func (w DirectoryWriter) Close() error {
	return nil
}

// CreateFile creates the named file in the directory and returns an
// io.WriteCloser for it
func (w *DirectoryWriter) CreateFile(filename string) (io.WriteCloser, error) {
	file, err := os.Create(filepath.Join(w.directory, filename))
	if err != nil {
		return nil, err
	}
	return plumbing.MultiWriteCloser(file, plumbing.NopWriteCloser(&w.tx)), nil
}

// Config returns the WriterConfig associated with this writer
func (w DirectoryWriter) Config() WriterConfig {
	return w.config
}

// Tx returns the number of bytes written
func (w DirectoryWriter) Tx() uint64 {
	return w.tx.Count()
}

// ZipFileWriter writes a Dreamcast game to a zip archive
type ZipFileWriter struct {
	file   *os.File
	writer *zip.Writer
	config WriterConfig
	tx     plumbing.WriteCounter
}

// NewZipFileWriter returns a ZipFileWriter using the passed zip file path
// and config
func NewZipFileWriter(filename string, config WriterConfig) (*ZipFileWriter, error) {
	file, err := os.Create(filename)
	if err != nil {
		return nil, err
	}

	w := &ZipFileWriter{
		file:   file,
		config: config,
	}
	w.writer = zip.NewWriter(io.MultiWriter(file, &w.tx))

	return w, nil
}

// Close closes the zip file
func (w ZipFileWriter) Close() error {
	if err := w.writer.Close(); err != nil {
		return err
	}

	return w.file.Close()
}

// CreateFile create the named file in the zip file and returns an
// io.WriteCloser for it
func (w ZipFileWriter) CreateFile(filename string) (io.WriteCloser, error) {
	writer, err := w.writer.Create(filename)
	if err != nil {
		return nil, err
	}
	return plumbing.NopWriteCloser(writer), nil
}

// Config returns the WriterConfig associated with this writer
func (w ZipFileWriter) Config() WriterConfig {
	return w.config
}

// Tx returns the number of bytes written
func (w ZipFileWriter) Tx() uint64 {
	return w.tx.Count()
}
