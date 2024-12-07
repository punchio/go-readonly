package pkg_check

import (
	"testing"
)

func TestPackage(t *testing.T) {
	Check("../parser/testdata")
}
