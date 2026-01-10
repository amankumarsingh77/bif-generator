package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

type Frame struct {
	Idx  int
	Data []byte
}

const (
	bifHeaderSize = 64
)

func writeBIFHeader(
	w io.Writer,
	frameCount uint32,
	intervalMS uint32,
	width uint32,
	height uint32,
) error {

	w.Write([]byte{'B', 'I', 'F', 0, 0, 0, 0, 0})
	writeLE(w, uint32(1)) // version
	writeLE(w, frameCount)
	writeLE(w, intervalMS)
	writeLE(w, width)
	writeLE(w, height)
	w.Write(make([]byte, 36)) // reserved
	return nil
}

func writeBIF(frames [][]byte, output string, interval int) error {
	const (
		width      = 320
		height     = 180
		headerSize = 64
	)
	intervalMS := interval * 1000

	f, err := os.Create(output)
	if err != nil {
		return err
	}
	defer f.Close()

	frameCount := len(frames)
	if frameCount == 0 {
		return fmt.Errorf("no frames to write")
	}

	if err := writeBIFHeader(
		f,
		uint32(frameCount),
		uint32(intervalMS),
		uint32(width),
		uint32(height),
	); err != nil {
		return err
	}

	sizes := make([]uint64, len(frames))
	for i, frame := range frames {
		sizes[i] = uint64(len(frame))
	}

	indexSize := uint64((frameCount + 1) * 8)
	imageStart := uint64(headerSize) + indexSize

	offsets := make([]uint64, 0, frameCount+1)
	cur := imageStart

	for _, sz := range sizes {
		offsets = append(offsets, cur)
		cur += sz
	}
	offsets = append(offsets, cur)

	for _, off := range offsets {
		if err := writeLE(f, off); err != nil {
			return err
		}
	}

	for _, jpeg := range frames {
		if _, err := f.Write(jpeg); err != nil {
			return err
		}
	}

	return nil
}

func writeLE(w io.Writer, v any) error {
	return binary.Write(w, binary.LittleEndian, v)
}
