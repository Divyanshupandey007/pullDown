# Pulldown

A concurrent file download manager built in Go. It splits files into chunks and downloads them in parallel to maximize speed.

## Features
- 🚀 Concurrent downloading (4 connections)
- 🧩 Automatic file chunking and merging
- 🧹 Auto-cleanup of temporary parts
- 📂 Dynamic filename extraction

## Usage

1. **Build the project:**
   ```bash
   go build -o pulldown