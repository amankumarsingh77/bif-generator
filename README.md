<div align="center">

# ✨ BIF Generator

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)](LICENSE)
[![FFmpeg](https://img.shields.io/badge/requires-FFmpeg-green.svg?style=flat-square)](https://ffmpeg.org/)

**Generate professional video seek thumbnails in seconds**

</div>

---

## About

BIF Generator is a high-performance command-line tool and web service that generates BIF (Binary Image Format) files — the same thumbnail preview format used by major streaming platforms like Netflix, Hulu, and YouTube. When you hover over a video progress bar and see those slick thumbnail previews, you're looking at a BIF file in action.

This tool extracts frames from your videos at regular intervals, compresses them as JPEG images, and bundles everything into a single optimized `.bif` file with an embedded index table for instant random access. The result is a professional-grade seek thumbnail experience that works anywhere.

Perfect for video platforms, media libraries, streaming services, or anyone who wants to add polish to their video player experience.

## Features

- **Fast processing** — A 10-minute video takes about ~15 seconds
- **Parallel extraction** — Uses multiple CPU cores to speed up frame extraction
- **Lightweight** — Only requires FFmpeg and FFprobe
- **Web interface** — Upload and generate BIF files through a browser UI
- **CLI interface** — Scriptable for automated workflows

---

## Quick Start

### Prerequisites

You need [FFmpeg](https://ffmpeg.org/download.html) installed:

```bash
# macOS
brew install ffmpeg

# Ubuntu/Debian
sudo apt install ffmpeg

# Windows
# Download from https://ffmpeg.org/download.html
```

### Installation

```bash
git clone https://github.com/yourusername/bif-generator-tool.git
cd bif-generator-tool
go build -o bif-generator
```

### Usage

**Web UI** (recommended for first-time users)

```bash
./bif-generator -serve
# Open http://localhost:8080
```

**CLI** (for automation & scripts)

```bash
# Basic: 10-second interval (default)
./bif-generator -input video.mp4 -output video.bif

# Custom interval (every 5 seconds)
./bif-generator -input video.mp4 -interval 5

# Parallel mode — warp speed
./bif-generator -input video.mp4 --parallel --workers 8

# Pipe directly to stdout
./bif-generator -input video.mp4 > video.bif
```

---

## Demos

### CLI Generation

<p align="center">
  <video src="static/cli.mp4" width="600" controls></video>
</p>

### Web UI Generation

<p align="center">
  <video src="static/ui.mp4" width="600" controls></video>
</p>

---

## How It Works

1. **Extract frames** — FFmpeg pulls JPEG frames from your video at regular intervals
2. **Build index** — Create a lookup table for instant random access
3. **Bundle it up** — Pack everything into a single `.bif` file
4. **Done** — Drop it into any BIF-compatible video player

---

## Options

```
-input       string    Input video file path
-output      string    Output BIF file path (default: input.bif)
-interval    int       Thumbnail interval in seconds (default: 10)
-parallel              Enable parallel processing
-workers     int       Number of parallel workers (default: CPU cores)
-serve                 Start web server mode
```

---

## Why BIF?

BIF isn't the only thumbnail format, but it's one of the best:

- **Compact** — Efficient JPEG compression
- **Fast** — Built-in index table means zero seeking
- **Compatible** — Used by major streaming platforms
- **Simple** — Just a binary format anyone can parse

---

## Contributing

Found a bug? Have a feature idea? Pull requests are welcome.

---

## License

MIT — feel free to use this in your projects.

---

<div align="center">

Made with ❤️ by [@amankumarsingh77](https://github.com/amankumarsingh77)

</div>
