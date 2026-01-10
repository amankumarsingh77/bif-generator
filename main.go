package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func main() {
	// Parse flags
	serveCmd := flag.Bool("serve", false, "Start static file server")
	input := flag.String("input", "", "Path to video file")
	output := flag.String("output", "output.bif", "Path to output BIF file")
	interval := flag.Int("interval", 10, "Frame interval in seconds")
	parallel := flag.Bool("parallel", false, "Use parallel frame extraction")
	workers := flag.Int("workers", 8, "Number of parallel workers")
	flag.Parse()

	if *serveCmd {
		serveStatic()
		return
	}

	if *input == "" {
		fmt.Println("Usage: bif-generator -input <video.mp4> [-output output.bif] [-interval 10] [--parallel] [-workers 8]")
		fmt.Println("       bif-generator -serve")
		os.Exit(1)
	}

	start := time.Now()
	numWorkers := 1
	if *parallel {
		numWorkers = *workers
	}
	// Create progress bar for CLI mode
	progressBar := newProgressBar(0) // total will be set by generateBIF's internal logic
	progressCallback := func(current, total int) {
		if progressBar.total == 0 && total > 0 {
			progressBar.total = total
		}
		progressBar.add()
	}

	if err := generateBIF(*input, *output, *interval, numWorkers, progressCallback); err != nil {
		log.Fatalf("Failed to generate BIF: %v", err)
	}
	log.Printf("Done! Processed in %.1f seconds. Output: %s", time.Since(start).Seconds(), *output)
}

func serveStatic() {
	os.MkdirAll("./uploads", 0755)
	os.MkdirAll("./outputs", 0755)

	http.HandleFunc("/api/generate", handleGenerateAPI)

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	http.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("./uploads"))))
	http.Handle("/outputs/", http.StripPrefix("/outputs/", http.FileServer(http.Dir("./outputs"))))

	port := "8080"
	log.Printf("Serving at http://localhost:%s\n", port)
	log.Printf("Open http://localhost:%s/index.html to use the BIF generator\n", port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func handleGenerateAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	err := r.ParseMultipartForm(500 * 1024 * 1024)
	if err != nil {
		sendSSEError(w, flusher, "Failed to parse form: "+err.Error())
		return
	}

	file, header, err := r.FormFile("video")
	if err != nil {
		sendSSEError(w, flusher, "Failed to get video file: "+err.Error())
		return
	}
	defer file.Close()

	intervalStr := r.FormValue("interval")
	interval, _ := strconv.Atoi(intervalStr)
	if interval == 0 {
		interval = 10
	}

	parallel := r.FormValue("parallel") == "true"
	workersStr := r.FormValue("workers")
	workers, _ := strconv.Atoi(workersStr)
	if workers == 0 {
		workers = 8
	}

	timestamp := time.Now().UnixNano()
	videoBaseName := strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))
	outputDir := filepath.Join("./outputs", fmt.Sprintf("%d", timestamp))
	os.MkdirAll(outputDir, 0755)

	videoPath := filepath.Join(outputDir, header.Filename)
	bifPath := filepath.Join(outputDir, videoBaseName+".bif")

	sendSSEProgress(w, flusher, 0, "Saving video file...")
	dst, err := os.Create(videoPath)
	if err != nil {
		sendSSEError(w, flusher, "Failed to save video: "+err.Error())
		return
	}
	defer dst.Close()

	written, err := io.Copy(dst, file)
	if err != nil {
		sendSSEError(w, flusher, "Failed to save video: "+err.Error())
		return
	}
	log.Printf("Saved uploaded video: %s (%d bytes)", videoPath, written)

	sendSSEProgress(w, flusher, 5, "Analyzing video...")

	numFrames := 0
	progressCallback := func(current, total int) {
		// Use float to avoid integer division rounding issues
		progress := 5 + int(float64(current)*90.0/float64(total))
		sendSSEProgress(w, flusher, progress, fmt.Sprintf("Extracting frame %d/%d", current, total))
		numFrames = total
	}

	numWorkers := 1
	if parallel {
		numWorkers = workers
	}
	genErr := generateBIF(videoPath, bifPath, interval, numWorkers, progressCallback)

	if genErr != nil {
		sendSSEError(w, flusher, "Generation failed: "+genErr.Error())
		return
	}

	videoURL := fmt.Sprintf("/uploads/%d/%s", timestamp, header.Filename)
	bifURL := fmt.Sprintf("/outputs/%d/%s.bif", timestamp, videoBaseName)

	uploadDir := filepath.Join("./uploads", fmt.Sprintf("%d", timestamp))
	os.MkdirAll(uploadDir, 0755)
	finalVideoPath := filepath.Join(uploadDir, header.Filename)
	os.Rename(videoPath, finalVideoPath)
	videoPath = finalVideoPath

	// Send final progress update at 100%
	sendSSEProgress(w, flusher, 100, fmt.Sprintf("Generated %d frames", numFrames))

	result := map[string]any{
		"done":     true,
		"bifUrl":   bifURL,
		"videoUrl": videoURL,
		"frames":   numFrames,
	}
	sendSSEData(w, flusher, result)

	log.Printf("Generated BIF: %s (%d frames)", bifPath, numFrames)
}

func sendSSEProgress(w http.ResponseWriter, flusher http.Flusher, progress int, status string) {
	data := map[string]any{
		"progress": progress,
		"status":   status,
	}
	sendSSEData(w, flusher, data)
}

func sendSSEError(w http.ResponseWriter, flusher http.Flusher, message string) {
	data := map[string]any{
		"error": message,
	}
	sendSSEData(w, flusher, data)
}

func sendSSEData(w http.ResponseWriter, flusher http.Flusher, data map[string]any) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "data: %s\n\n", jsonData)
	flusher.Flush()
	return nil
}
