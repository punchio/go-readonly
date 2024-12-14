package readonly

import (
	"testing"
)

func TestFromTestdata(t *testing.T) {
	output := Run("./testdata/interface_check")
	for _, s := range output {
		t.Log(s)
	}
}
