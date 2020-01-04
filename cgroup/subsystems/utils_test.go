package subsystems

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFindCgroupMountpoint(t *testing.T) {
	s:=FindCgroupMountpoint("memory")
	t.Log(s)
	assert.NotEmpty(t,s)
}
