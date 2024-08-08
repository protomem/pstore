package blobstore

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
)

const (
	_filePermMode   = 0o755
	_folderPremMode = 0o755
)

var _ Storage = (*FS)(nil)

type FSOptions struct {
	Path string
}

type FS struct {
	Options  FSOptions
	basePath string
}

func NewFS(opts FSOptions) (*FS, error) {
	basePath := filepath.Clean(opts.Path)
	fs := &FS{
		Options:  opts,
		basePath: basePath,
	}
	return fs, fs.initFolder("")
}

func (fs *FS) Get(_ context.Context, foldername, filename string) (Object, error) {
	file, err := os.OpenFile(fs.fmtFilename(foldername, filename), os.O_RDONLY, _filePermMode)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return emptyObject(), ErrObjectNotFound
		}

		return emptyObject(), nil
	}

	return NewObject(file), nil
}

func (fs *FS) Set(_ context.Context, foldername, filename string, obj Object) error {
	if err := fs.initFolder(foldername); err != nil {
		return err
	}

	filename = fs.fmtFilename(foldername, filename)
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, _filePermMode)
	if err != nil {
		return err
	}

	if _, err := io.CopyN(file, obj.Body, 1024); err != nil {
		return err
	}

	return nil
}

func (fs *FS) Put(_ context.Context, foldername, filename string, obj Object) error {
	if err := fs.initFolder(foldername); err != nil {
		return err
	}

	filename = fs.fmtFilename(foldername, filename)
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, _filePermMode)
	if err != nil {
		return err
	}

	if _, err := io.CopyN(file, obj.Body, 1024); err != nil {
		return err
	}

	return nil
}

func (fs *FS) Delete(_ context.Context, foldername, filename string) error {
	return os.Remove(fs.fmtFilename(foldername, filename))
}

func (fs *FS) Close(_ context.Context) error {
	return nil
}

func (fs *FS) initFolder(parent string) error {
	err := os.Mkdir(fs.fmtFoldername(parent), _folderPremMode)
	if err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	return nil
}

func (fs *FS) fmtFoldername(parent string) string {
	return filepath.Join(fs.basePath, parent)
}

func (fs *FS) fmtFilename(parent, name string) string {
	return filepath.Join(fs.fmtFoldername(parent), name)
}
