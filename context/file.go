package context

import (
	"io"
	"mime/multipart"
	"os"
	"path"
)

// A file in the file system.
type File struct {
	FileHeader *multipart.FileHeader
}

// Move Moves the file to a new location.
func (f *File) Move(directory string, name ...string) (bool, error) {
	src, err := f.FileHeader.Open()
	if err != nil {
		return false, err
	}
	defer src.Close()

	fname := f.FileHeader.Filename

	if len(name) > 0 {
		fname = name[0]
	}

	dst := path.Join(directory, fname)

	out, err := os.Create(dst)
	if err != nil {
		return false, err
	}
	defer out.Close()

	io.Copy(out, src)

	return true, nil
}
