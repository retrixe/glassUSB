package main

import (
	"os"
	"testing"

	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/partition/gpt"
)

func TestWipeStaleGPTMetadata(t *testing.T) {
	img := t.TempDir() + "/test.img"
	f, err := os.Create(img)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(64 * 1024 * 1024); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	disk, err := diskfs.Open(img, diskfs.WithOpenMode(diskfs.ReadWrite))
	if err != nil {
		t.Fatal(err)
	}
	table := &gpt.Table{
		ProtectiveMBR: true,
		Partitions: []*gpt.Partition{
			{Start: 2048, End: 2048 + 1024 - 1, Type: gpt.EFISystemPartition, Name: "EFI"},
		},
	}
	if err := disk.Partition(table); err != nil {
		t.Fatal(err)
	}
	disk.Close()

	if err := FormatDiskForUEFINTFS(img, false); err != nil {
		t.Fatalf("FormatDiskForUEFINTFS: %v", err)
	}

	const sectorSize = 512
	checkZero := func(offset int64, label string) {
		t.Helper()
		buf := make([]byte, sectorSize)
		f, err := os.Open(img)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if _, err := f.ReadAt(buf, offset); err != nil {
			t.Fatalf("read %s: %v", label, err)
		}
		for i, b := range buf {
			if b != 0 {
				t.Fatalf("%s byte %d is 0x%02x, want 0", label, i, b)
			}
		}
	}

	checkZero(sectorSize, "primary GPT header (LBA 1)")
	checkZero(sectorSize*2, "primary GPT partition entry (LBA 2)")
	stat, err := os.Stat(img)
	if err != nil {
		t.Fatal(err)
	}
	checkZero(stat.Size()-sectorSize, "backup GPT header (last LBA)")
}

func TestFormatDiskForUEFINTFSWithValidStaleGPT(t *testing.T) {
	img := t.TempDir() + "/test.img"
	f, err := os.Create(img)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(64 * 1024 * 1024); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	// Create a valid GPT with one partition, then reformat as MBR without wiping GPT.
	disk, err := diskfs.Open(img, diskfs.WithOpenMode(diskfs.ReadWrite))
	if err != nil {
		t.Fatal(err)
	}
	table := &gpt.Table{
		ProtectiveMBR: true,
		Partitions: []*gpt.Partition{
			{Start: 2048, End: 2048 + 1024 - 1, Type: gpt.EFISystemPartition, Name: "EFI"},
		},
	}
	if err := disk.Partition(table); err != nil {
		t.Fatal(err)
	}
	disk.Close()

	if err := FormatDiskForUEFINTFS(img, false); err != nil {
		t.Fatalf("FormatDiskForUEFINTFS: %v", err)
	}
	if err := WriteUEFINTFSToPartition(img, 2); err != nil {
		t.Fatalf("WriteUEFINTFSToPartition: %v", err)
	}
}
