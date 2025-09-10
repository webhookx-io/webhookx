package test

import (
	"github.com/stretchr/testify/suite"
	"path/filepath"
	"runtime"
)

var dir string // test dir

func init() {
	_, filename, _, _ := runtime.Caller(0)
	dir = filepath.Dir(filename)
}

func FilePath(filename string) string {
	return filepath.Join(dir, filename)
}

type BasicSuite struct {
	suite.Suite
}

func (s *BasicSuite) SetupSuite() {

}
