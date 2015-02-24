package docker

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/sgotti/baci/Godeps/_workspace/src/github.com/appc/spec/schema/types"
)

const (
	tstprefix = "baci"
)

func TestGetMountPoints(t *testing.T) {
	dir, err := ioutil.TempDir("", tstprefix)
	if err != nil {
		t.Fatalf("error creating tempdir: %v", err)
	}
	defer os.RemoveAll(dir)

	tests := []struct {
		dockerfile string
		expected   []types.MountPoint
	}{
		{
			`
			VOLUME [ "/mnt/vol00" ]
			`,
			[]types.MountPoint{
				types.MountPoint{
					Name: "volume0",
					Path: "/mnt/vol00",
				},
			},
		},
		{
			`
			VOLUME [ "/mnt/vol00", "/mnt/vol01" ]
			VOLUME /mnt/vol02 /mnt/vol03
			# Duplicate volumes
			VOLUME /mnt/vol00 /mnt/vol03
			`,
			[]types.MountPoint{
				types.MountPoint{
					Name: "volume0",
					Path: "/mnt/vol00",
				},
				types.MountPoint{
					Name: "volume1",
					Path: "/mnt/vol01",
				},
				types.MountPoint{
					Name: "volume2",
					Path: "/mnt/vol02",
				},
				types.MountPoint{
					Name: "volume3",
					Path: "/mnt/vol03",
				},
			},
		},
	}

	for _, tt := range tests {
		ioutil.WriteFile(filepath.Join(dir, "Dockerfile"), []byte(tt.dockerfile), 0644)

		builder, err := NewDockerBuilder("", dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		mountPoints, err := builder.GetMountPoints()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(tt.expected, mountPoints) {
			t.Errorf("wrong mountPoints, want: %v, got: %v", tt.expected, mountPoints)
		}
	}
}
