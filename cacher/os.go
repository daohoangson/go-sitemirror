package cacher

import "os"

type realFs struct{}

// NewFs returns a new fs instance that is basically a wrapper for os package functions
func NewFs() Fs {
	return &realFs{}
}

func (fs *realFs) Getwd() (string, error) {
	return os.Getwd()
}

func (fs *realFs) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (fs *realFs) OpenFile(name string, flag int, perm os.FileMode) (File, error) {
	return os.OpenFile(name, flag, perm)
}

func (fs *realFs) RemoveAll(path string) error {
	return os.RemoveAll(path)
}
