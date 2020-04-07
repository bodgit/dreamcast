package dreamcast

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/bodgit/dreamcast/gdi"
)

const (
	cueExtension = ".cue"
)

// Reader is the interface implemented by an object that can be used as a
// source for reading a Dreamcast game image from disk
type Reader interface {
	// Close closes the source
	Close() error
	// FindGDIFile returns an io.ReadCloser opened on, and the filename
	// of, the first GDI file found
	FindGDIFile() (io.ReadCloser, string, error)
	// FindCueFile returns an io.ReadCloser opened on, and the filename
	// of, the first cue sheet found
	FindCueFile() (io.ReadCloser, string, error)
	// OpenFile returns an io.ReadCloser opened on the named file
	OpenFile(string) (io.ReadCloser, error)
	// FileSize returns the size of the named file
	FileSize(string) (uint64, error)
}

// DirectoryReader reads a Dreamcast game from a directory
type DirectoryReader struct {
	directory *os.File
}

// NewDirectoryReader returns a DirectoryReader using the passed directory path
func NewDirectoryReader(directory string) (*DirectoryReader, error) {
	file, err := os.Open(directory)
	if err != nil {
		return nil, err
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}

	if !info.IsDir() {
		file.Close()
		return nil, &os.PathError{"open", directory, syscall.ENOTDIR}
	}

	r := &DirectoryReader{
		directory: file,
	}

	return r, nil
}

// Close closes the directory
func (r DirectoryReader) Close() error {
	return r.directory.Close()
}

func (r DirectoryReader) findFileByExtension(extension string) (io.ReadCloser, string, error) {
	// Rewind to the beginning of the directory again
	if _, err := r.directory.Seek(0, os.SEEK_SET); err != nil {
		return nil, "", err
	}

	names, err := r.directory.Readdirnames(0)
	if err != nil {
		return nil, "", err
	}

	for _, name := range names {
		if strings.HasSuffix(name, extension) {
			file, err := os.Open(filepath.Join(r.directory.Name(), name))
			if err != nil {
				return nil, "", err
			}
			return file, name, nil
		}
	}

	return nil, "", &os.PathError{"open", r.directory.Name(), syscall.ENOENT}
}

// FindCueFile reads the directory and returns an io.ReadCloser for, and the
// filename of, the first cue file found
func (r DirectoryReader) FindCueFile() (io.ReadCloser, string, error) {
	return r.findFileByExtension(cueExtension)
}

// FindGDIFile reads the directory and returns an io.ReadCloser for, and the
// filename of, the first GDI file found
func (r DirectoryReader) FindGDIFile() (io.ReadCloser, string, error) {
	return r.findFileByExtension(gdi.Extension)
}

// OpenFile returns an io.ReadCloser for the named file
func (r DirectoryReader) OpenFile(filename string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(r.directory.Name(), filename))
}

// FileSize returns the size of the named file
func (r DirectoryReader) FileSize(filename string) (uint64, error) {
	info, err := os.Stat(filepath.Join(r.directory.Name(), filename))
	if err != nil {
		return 0, err
	}

	return uint64(info.Size()), nil
}

// ZipFileReader reads a Dreamcast game from a zip archive
type ZipFileReader struct {
	filename string
	reader   *zip.ReadCloser
}

// NewZipFileReader returns a ZipFileReader using the passed zip file path
func NewZipFileReader(zipFile string) (*ZipFileReader, error) {
	reader, err := zip.OpenReader(zipFile)
	if err != nil {
		return nil, err
	}

	r := &ZipFileReader{
		filename: zipFile,
		reader:   reader,
	}

	return r, nil
}

// Close closes the zip file
func (r ZipFileReader) Close() error {
	return r.reader.Close()
}

func (r ZipFileReader) findFileByExtension(extension string) (io.ReadCloser, string, error) {
	for _, file := range r.reader.File {
		if strings.HasSuffix(file.Name, extension) {
			f, err := file.Open()
			if err != nil {
				return nil, "", err
			}
			return f, file.Name, nil
		}
	}
	return nil, "", &os.PathError{"open", r.filename, syscall.ENOENT}
}

// FindCueFile reads the zip file and returns an io.ReadCloser for, and the
// filename of, the first cue file found
func (r ZipFileReader) FindCueFile() (io.ReadCloser, string, error) {
	return r.findFileByExtension(cueExtension)
}

// FindGDIFile reads the zip file and returns an io.ReadCloser for, and the
// filename of, the first GDI file found
func (r ZipFileReader) FindGDIFile() (io.ReadCloser, string, error) {
	return r.findFileByExtension(gdi.Extension)
}

// OpenFile returns an io.ReadCloser for the named file
func (r ZipFileReader) OpenFile(filename string) (io.ReadCloser, error) {
	for _, file := range r.reader.File {
		if file.Name == filename {
			return file.Open()
		}
	}
	return nil, &os.PathError{"open", r.filename, syscall.ENOENT}
}

// FileSize returns the size of the named file
func (r ZipFileReader) FileSize(filename string) (uint64, error) {
	for _, file := range r.reader.File {
		if file.Name == filename {
			return file.UncompressedSize64, nil
		}
	}
	return 0, &os.PathError{"stat", r.filename, syscall.ENOENT}
}
