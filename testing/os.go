package testing

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/daohoangson/go-sitemirror/cacher"
)

type fakeFs struct {
	logger *logrus.Logger
	mutex  sync.Mutex

	root *fakeNode
	wd   string
}

type fakeNode struct {
	logger *logrus.Entry
	mutex  sync.Mutex

	path  string
	perm  os.FileMode
	nodes map[string]*fakeNode
	bytes []byte
}

type fakeFile struct {
	fs    *fakeFs
	node  *fakeNode
	bytes []byte
	pos   int64
}

// NewFs returns an in memory file system
func NewFs() cacher.Fs {
	logger := Logger()

	rootNode := &fakeNode{
		logger: logger.WithField("path", "/"),
		path:   "/",
		perm:   os.ModePerm,
		nodes:  make(map[string]*fakeNode),
	}

	return &fakeFs{
		logger: logger,
		root:   rootNode,
		wd:     "/",
	}
}

// FsCreate works similar to os.Create
func FsCreate(fs cacher.Fs, name string) (cacher.File, error) {
	return fs.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
}

// FsReadFile works similar to ioutil.ReadFile
func FsReadFile(fs cacher.Fs, name string) ([]byte, error) {
	f, err := fs.OpenFile(name, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(f)
}

func (fs *fakeFs) Getwd() (string, error) {
	return fs.wd, nil
}

func (fs *fakeFs) MkdirAll(name string, perm os.FileMode) error {
	if !path.IsAbs(name) {
		name = path.Join(fs.wd, name)
	}
	if !strings.HasPrefix(name, "/") {
		panic(fmt.Sprintf("name=%s does not start with /", name))
	}

	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	loggerContext := fs.logger.WithFields(logrus.Fields{
		"name": name,
		"perm": perm,
	})

	parts := strings.Split(name, "/")
	node := fs.root
	for i := 1; i < len(parts); i++ {
		if node.isFile() {
			loggerContext.WithField("node", node).Error("MkdirAll: is file")
			return fmt.Errorf("%s is file", node.path)
		}

		nextNode, ok := node.nodes[parts[i]]
		if ok {
			node = nextNode
			continue
		}

		newNode, err := node.newDir(parts[i], perm)
		if err != nil {
			return err
		}

		node = newNode
		loggerContext.WithField("node", node).Debug("MkdirAll: created")
	}

	loggerContext.WithField("node", node).Debug("MkdirAll: ok")

	return nil
}

func (fs *fakeFs) OpenFile(name string, flag int, perm os.FileMode) (cacher.File, error) {
	if !path.IsAbs(name) {
		name = path.Join(fs.wd, name)
	}
	if !strings.HasPrefix(name, "/") {
		panic(fmt.Sprintf("name=%s does not start with /", name))
	}

	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	loggerContext := fs.logger.WithFields(logrus.Fields{
		"name": name,
		"flag": flag,
		"perm": perm,
	})

	parts := strings.Split(name, "/")
	node := fs.root
	for i := 1; i < len(parts); i++ {
		if node.isFile() {
			loggerContext.WithField("node", node).Error("OpenFile: is file")
			return nil, fmt.Errorf("%s is file", node.path)
		}

		nextNode, ok := node.nodes[parts[i]]
		if !ok {
			notOkLoggerContext := loggerContext.WithFields(logrus.Fields{
				"parent":  node,
				"element": parts[i],
				"i":       i,
				"parts":   len(parts),
			})

			if i == len(parts)-1 {
				// try to create the file
				if flag&os.O_CREATE != 0 {
					newNode, err := node.newFile(parts[i], perm)
					if err != nil {
						return nil, err
					}

					node = newNode
					loggerContext.WithField("node", node).Debug("OpenFile: created")
					continue
				} else {
					notOkLoggerContext.Debug("OpenFile: O_CREATE not set")
				}
			}

			notOkLoggerContext.Error("OpenFile: does not exists")
			return nil, fmt.Errorf("%s/%s does not exists", node.path, parts[i])
		}

		node = nextNode
	}

	if node.isDir() {
		loggerContext.WithField("node", node).Error("OpenFile: is dir")
		return nil, fmt.Errorf("%s is dir", node.path)
	}

	f := &fakeFile{fs: fs, node: node}
	node.mutex.Lock()
	f.bytes = make([]byte, len(node.bytes))
	copy(f.bytes, node.bytes)
	node.mutex.Unlock()

	if flag&os.O_APPEND != 0 {
		f.Seek(0, os.SEEK_END)
	}

	fs.logger.WithFields(logrus.Fields{
		"name": name,
		"flag": flag,
		"perm": perm,
		"node": node,
	}).Debug("OpenFile: ok")

	return f, nil
}

func (fs *fakeFs) RemoveAll(path string) error {
	panic("Not implemented")
}

func (fn *fakeNode) isDir() bool {
	return fn.nodes != nil
}

func (fn *fakeNode) isFile() bool {
	return !fn.isDir()
}

func (fn *fakeNode) newDir(name string, perm os.FileMode) (*fakeNode, error) {
	fn.mutex.Lock()
	defer fn.mutex.Unlock()

	if _, ok := fn.nodes[name]; ok {
		return nil, fmt.Errorf("%s/%s already exists", fn.path, name)
	}

	newNode := newFakeNode(fn, name, perm, true)
	fn.nodes[name] = newNode

	return newNode, nil
}

func (fn *fakeNode) newFile(name string, perm os.FileMode) (*fakeNode, error) {
	fn.mutex.Lock()
	defer fn.mutex.Unlock()

	if _, ok := fn.nodes[name]; ok {
		return nil, fmt.Errorf("%s/%s already exists", fn.path, name)
	}

	newNode := newFakeNode(fn, name, perm, false)
	fn.nodes[name] = newNode

	return newNode, nil
}

func newFakeNode(parent *fakeNode, name string, perm os.FileMode, isDir bool) *fakeNode {
	path := path.Join(parent.path, name)

	fn := &fakeNode{
		logger: parent.logger.WithField("path", path),
		path:   path,
		perm:   perm,
	}

	if isDir {
		fn.nodes = make(map[string]*fakeNode)
	} else {
		fn.bytes = make([]byte, 0)
	}

	return fn
}

func (ff *fakeFile) Read(p []byte) (int, error) {
	ff.node.logger.Debug("File.Read...")
	ff.node.mutex.Lock()
	defer ff.node.mutex.Unlock()

	if ff.pos >= int64(len(ff.bytes)) {
		return 0, io.EOF
	}

	n := copy(p, ff.bytes[ff.pos:])
	ff.pos += int64(n)

	ff.node.logger.WithFields(logrus.Fields{
		"read": n,
		"pos":  ff.pos,
	}).Debug("File.Read: ok")

	return n, nil
}

func (ff *fakeFile) Write(p []byte) (int, error) {
	ff.node.logger.WithField("len", len(p)).Debug("File.Write...")
	ff.node.mutex.Lock()
	defer ff.node.mutex.Unlock()

	before := ff.bytes[:int(ff.pos)]
	var after []byte
	afterOffset := int(ff.pos) + len(p)
	if afterOffset < len(ff.bytes) {
		after = ff.bytes[afterOffset:]
	}

	ff.bytes = make([]byte, len(before)+len(p)+len(after))
	copy(ff.bytes, before)
	copy(ff.bytes[len(before):], p)
	if after != nil {
		copy(ff.bytes[len(before)+len(p):], after)
	}

	written := len(p)
	ff.pos = int64(len(before) + written)

	ff.node.logger.WithFields(logrus.Fields{
		"written": written,
		"pos":     ff.pos,
	}).Debug("File.Write: ok")

	return written, nil
}

func (ff *fakeFile) WriteAt(p []byte, off int64) (int, error) {
	ff.Seek(off, os.SEEK_SET)
	return ff.Write(p)
}

func (ff *fakeFile) Close() error {
	ff.node.logger.Debug("File.Close...")
	ff.node.mutex.Lock()
	defer ff.node.mutex.Unlock()

	ff.node.bytes = make([]byte, len(ff.bytes))
	copy(ff.node.bytes, ff.bytes)

	ff.node.logger.WithField("len", len(ff.bytes)).Debug("File.Close: ok")

	return nil
}

func (ff *fakeFile) Seek(offset int64, whence int) (int64, error) {
	ff.node.logger.WithFields(logrus.Fields{
		"offset": offset,
		"whence": whence,
	}).Debug("File.Seek...")
	ff.node.mutex.Lock()
	defer ff.node.mutex.Unlock()

	switch whence {
	case os.SEEK_SET:
		ff.pos = offset
	case os.SEEK_CUR:
		ff.pos += offset
	case os.SEEK_END:
		ff.pos = int64(len(ff.bytes)-1) - offset
	}

	return ff.pos, nil
}

func (ff *fakeFile) Name() string {
	return path.Base(ff.node.path)
}

func (ff *fakeFile) Truncate(size int64) error {
	ff.node.logger.Debug("File.Truncate...")
	ff.node.mutex.Lock()
	defer ff.node.mutex.Unlock()

	if size != 0 {
		panic("Non zero size is not implemented")
	}

	ff.bytes = make([]byte, 0)
	ff.pos = 0

	return nil
}
