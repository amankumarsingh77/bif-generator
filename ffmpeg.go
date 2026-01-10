package main

import (
	"fmt"
	"io"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

type progressBar struct {
	total   int
	current int
	mu      sync.Mutex
}

func newProgressBar(total int) *progressBar {
	return &progressBar{total: total}
}

func (p *progressBar) add() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.current++
	width := 40
	percent := float64(p.current) / float64(p.total)
	filled := int(percent * float64(width))

	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	fmt.Printf("\r[%s] %d/%d (%.1f%%)", bar, p.current, p.total, percent*100)

	if p.current == p.total {
		fmt.Println()
	}
}

func getDuration(videoPath string) (float64, error) {
	cmd := exec.Command(
		"ffprobe",
		"-v", "error",
		"-show_entries",
		"format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		videoPath,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("failed to get video duration: %v", string(out))
	}
	s := strings.TrimSpace(string(out))
	duration, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration: %v", err)
	}
	return duration, nil
}

func extractFrame(video string, ts float64) ([]byte, error) {
	cmd := exec.Command(
		"ffmpeg",
		"-ss", fmt.Sprintf("%.3f", ts),
		"-i", video,
		"-frames:v", "1",
		"-vf", "scale=240:160:force_original_aspect_ratio=decrease:flags=bicubic:sws_dither=none",
		"-qscale:v", "4",
		"-f", "image2pipe",
		"-vcodec", "mjpeg",
		"pipe:1",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start ffmpeg: %w", err)
	}

	data, err := io.ReadAll(stdout)
	if err != nil {
		cmd.Process.Kill()
		cmd.Wait()
		return nil, fmt.Errorf("read frame: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("ffmpeg wait: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("empty frame data")
	}

	return data, nil
}

// generateBIF generates a BIF file with optional parallel processing and progress callback
// progressFn can be nil for no progress reporting (CLI uses progressBar, API uses SSE)
func generateBIF(video string, output string, interval int, numWorkers int, progressFn func(current, total int)) error {
	duration, err := getDuration(video)
	if err != nil {
		return err
	}

	numFrames := int(duration/float64(interval)) + 1
	if numFrames == 0 {
		return fmt.Errorf("video too short for interval %d seconds", interval)
	}

	log.Printf("Video duration: %.0fs, extracting %d frames", duration, numFrames)

	// Sequential if workers=1, otherwise parallel
	if numWorkers <= 1 {
		return generateBIFSequential(video, output, interval, numFrames, progressFn)
	}
	return generateBIFParallel(video, output, interval, numWorkers, numFrames, progressFn)
}

func generateBIFSequential(video string, output string, interval int, numFrames int, progressFn func(current, total int)) error {
	frames := make([][]byte, numFrames)
	successCount := 0

	for i := 0; i < numFrames; i++ {
		timestamp := float64(i * interval)
		data, err := extractFrame(video, timestamp)
		if err != nil {
			log.Printf("Frame %d (t=%.1fs) failed: %v", i, timestamp, err)
		} else {
			frames[i] = data
			successCount++
		}
		if progressFn != nil {
			progressFn(i+1, numFrames)
		}
	}

	if successCount == 0 {
		return fmt.Errorf("no frames were successfully extracted")
	}

	log.Printf("Extracted %d/%d frames", successCount, numFrames)
	return writeBIF(frames, output, interval)
}

func generateBIFParallel(video string, output string, interval int, numWorkers int, numFrames int, progressFn func(current, total int)) error {
	jobs := make(chan int, numFrames)
	results := make(chan Frame, numFrames)

	var wg sync.WaitGroup
	for w := range numWorkers {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for frameIdx := range jobs {
				timestamp := float64(frameIdx * interval)
				data, err := extractFrame(video, timestamp)
				if err != nil {
					log.Printf("[Worker %d] Frame %d (t=%.1fs) failed: %v", workerID, frameIdx, timestamp, err)
				}
				results <- Frame{Idx: frameIdx, Data: data}
			}
		}(w)
	}

	go func() {
		for i := range numFrames {
			jobs <- i
		}
		close(jobs)
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	frames := make([][]byte, numFrames)
	successCount := 0
	completedCount := 0
	for frame := range results {
		if frame.Data != nil {
			frames[frame.Idx] = frame.Data
			successCount++
		}
		completedCount++
		if progressFn != nil {
			progressFn(completedCount, numFrames)
		}
	}

	if successCount == 0 {
		return fmt.Errorf("no frames were successfully extracted")
	}

	log.Printf("Extracted %d/%d frames", successCount, numFrames)
	return writeBIF(frames, output, interval)
}
