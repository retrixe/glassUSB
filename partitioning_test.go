package main

import (
	"os"
	"testing"

	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/partition/gpt"
)

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
