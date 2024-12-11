package readonly

import (
	"testing"
)

func TestFromTestdata(t *testing.T) {
	output := Run("./testdata")
	for _, s := range output {
		t.Log(s)
	}
}
