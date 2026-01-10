package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Frame struct {
	Data []byte
	Size uint64
}

const (
	bifHeaderSize = 64
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		serveStatic()
		return
	}

	input := "/home/aman/Downloads/Dude (2025) (Hindi DD5.1-224Kbps + Tamil) Dual Audio UnCut South Movie HD 1080p ESub.mkv"
	interval := 10
	output := "dude.bif"
	start := time.Now()
	if err := generateBIF(input, output, interval); err != nil {
		fmt.Printf("failed to generate bif : %v", err)
	}
	fmt.Printf("Prossed in %f sec\n", time.Since(start).Seconds())
}

func serveStatic() {
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	port := "8080"
	log.Printf("Serving static files at http://localhost:%s\n", port)
	log.Printf("Open http://localhost:%s/index.html to view the BIF preview\n", port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

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

func generateBIF(video string, output string, interval int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Minute)
	defer cancel()

	const (
		width      = 320
		height     = 180
		headerSize = 64
	)
	intervalMS := interval * 1000 // 10 seconds

	// -------------------------------------------------
	// PASS 1: Extract JPEG frames + sizes
	// -------------------------------------------------
	stdout, cmd, err := startFFmpeg(ctx, video, interval)
	if err != nil {
		return err
	}

	var frames [][]byte
	var sizes []uint64

	err = readMJPEGFrames(stdout, func(jpeg []byte) error {
		frames = append(frames, jpeg)
		sizes = append(sizes, uint64(len(jpeg)))
		return nil
	})
	if err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		return err
	}

	frameCount := len(frames)
	if frameCount == 0 {
		return fmt.Errorf("no frames extracted")
	}

	// -------------------------------------------------
	// Compute OFFSETS (absolute file offsets)
	// -------------------------------------------------
	indexSize := uint64((frameCount + 1) * 8)
	imageStart := uint64(headerSize) + indexSize

	offsets := make([]uint64, 0, frameCount+1)
	cur := imageStart

	for _, sz := range sizes {
		offsets = append(offsets, cur)
		cur += sz
	}
	offsets = append(offsets, cur) // EOF offset

	// -------------------------------------------------
	// PASS 2: Write BIF file
	// -------------------------------------------------
	f, err := os.Create(output)
	if err != nil {
		return err
	}
	defer f.Close()

	// Header
	if err := writeBIFHeader(
		f,
		uint32(frameCount),
		uint32(intervalMS),
		uint32(width),
		uint32(height),
	); err != nil {
		return err
	}

	// Index table
	for _, off := range offsets {
		if err := writeLE(f, off); err != nil {
			return err
		}
	}

	// Image data
	for _, jpeg := range frames {
		if _, err := f.Write(jpeg); err != nil {
			return err
		}
	}

	return nil
}

func readMJPEGFrames(r io.Reader, onFrame func([]byte) error) error {
	const (
		jpegStart = 0xFFD8
		jpegEnd   = 0xFFD9
	)

	buf := make([]byte, 32*1024)
	var frame []byte
	inFrame := false
	var prev byte

	for {
		n, err := r.Read(buf)
		if n > 0 {
			for i := range n {
				b := buf[i]

				if !inFrame {
					if prev == 0xFF && b == 0xD8 {
						inFrame = true
						frame = []byte{0xFF, 0xD8}
					}
				} else {
					frame = append(frame, b)
					if prev == 0xFF && b == 0xD9 {
						if err := onFrame(frame); err != nil {
							return err
						}
						inFrame = false
						frame = nil
					}
				}
				prev = b
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func getDuration(videoPath string) (float64, error) {
	cmd := exec.Command(
		"ffprobe",
		"-v", "error",
		"-show_entries",
		"format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		fmt.Sprintf("%s", videoPath),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("failed to get video duration : %v", string(out))
	}
	s := strings.TrimSpace(string(out))
	duration, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration : %v", err.Error())
	}
	return duration, nil
}

func startFFmpeg(ctx context.Context, video string, interval int) (io.ReadCloser, *exec.Cmd, error) {
	cmd := exec.CommandContext(
		ctx,
		"ffmpeg",
		"-i",
		fmt.Sprintf("%s", video),
		"-vf",
		fmt.Sprintf("fps=1/%d,scale=320:180", interval),
		"-q:v", "10",
		"-f", "image2pipe",
		"-vcodec", "mjpeg",
		"pipe:1",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}

	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}

	return stdout, cmd, nil
}

func writeLE(w io.Writer, v any) error {
	return binary.Write(w, binary.LittleEndian, v)
}
