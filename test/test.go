package test

import (
	"path/filepath"
	"runtime"

	"github.com/stretchr/testify/suite"
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
