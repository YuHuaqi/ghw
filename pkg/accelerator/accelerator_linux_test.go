//
// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.

package accelerator_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/YuHuaqi/ghw/pkg/accelerator"
	"github.com/YuHuaqi/ghw/pkg/option"
	"github.com/YuHuaqi/ghw/pkg/snapshot"

	"github.com/YuHuaqi/ghw/testdata"
)

func testScenario(t *testing.T, filename string, expectedDevs int) {
	testdataPath, err := testdata.SnapshotsDirectory()
	if err != nil {
		t.Fatalf("Expected nil err, but got %v", err)
	}

	t.Setenv("PCIDB_PATH", testdata.PCIDBChroot())

	workstationSnapshot := filepath.Join(testdataPath, filename)

	tmpRoot, err := os.MkdirTemp("", "ghw-accelerator-testing-*")
	if err != nil {
		t.Fatalf("Unable to create temporary directory: %v", err)
	}

	_, err = snapshot.UnpackInto(workstationSnapshot, tmpRoot, 0)
	if err != nil {
		t.Fatalf("Unable to unpack %q into %q: %v", workstationSnapshot, tmpRoot, err)
	}

	defer func() {
		_ = snapshot.Cleanup(tmpRoot)
	}()

	info, err := accelerator.New(option.WithChroot(tmpRoot))
	if err != nil {
		t.Fatalf("Expected nil err, but got %v", err)
	}
	if info == nil {
		t.Fatalf("Expected non-nil AcceleratorInfo, but got nil")
	}
	if len(info.Devices) != expectedDevs {
		t.Fatalf("Expected %d processing accelerator devices, but found %d.", expectedDevs, len(info.Devices))
	}
}

func TestAcceleratorDefault(t *testing.T) {
	if _, ok := os.LookupEnv("GHW_TESTING_SKIP_ACCELERATOR"); ok {
		t.Skip("Skipping PCI tests.")
	}

	// In this scenario we have 1 processing accelerator device
	testScenario(t, "linux-amd64-accel.tar.gz", 1)

}

func TestAcceleratorNvidia(t *testing.T) {
	if _, ok := os.LookupEnv("GHW_TESTING_SKIP_ACCELERATOR"); ok {
		t.Skip("Skipping PCI tests.")
	}

	// In this scenario we have 1 Nvidia 3D controller device
	testScenario(t, "linux-amd64-accel-nvidia.tar.gz", 1)
}
