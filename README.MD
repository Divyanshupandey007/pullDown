
# Pulldown 🚀

  

**Pulldown** is a high-performance, concurrent download manager built with **Golang** and **Angular**. It emulates the core logic of tools like IDM (Internet Download Manager) by splitting files into chunks and downloading them in parallel to maximize speed.

  

![Go](https://img.shields.io/badge/Backend-Golang-blue?logo=go)

![Angular](https://img.shields.io/badge/Frontend-Angular-red?logo=angular)

![Gin](https://img.shields.io/badge/Framework-Gin-green)

![License](https://img.shields.io/badge/License-MIT-grey)

  

## 🌟 Key Features

  

*  **⚡ Concurrent Downloading:** Uses Go **Goroutines** to split files into 4 chunks and download them simultaneously using HTTP Range headers.

*  **📡 Real-Time Progress:** Broadcasts atomic progress updates from the backend to the frontend via **WebSockets**.

*  **🧩 Automatic Assembly:** Merges file parts (`.tmp`) into the final file automatically upon completion.

*  **🧠 Thread-Safe Logic:** Uses `sync.Mutex` and `sync/atomic` to safely manage concurrent writes and progress calculations.

*  **🎨 Modern UI:** Angular standalone components with `NgZone` optimization for smooth, high-performance UI updates.

*  **🛠️ Robust Error Handling:** Handles server edge cases (like servers rejecting concurrent connections) gracefully.

  

## 🏗️ Architecture

  

1.  **Frontend (Angular):** Accepts a URL and connects to the backend via HTTP (for commands) and WebSockets (for data).

2.  **Backend (Go/Gin):**

*  **Handler:** Receives the download request.

*  **Calculator:** Fetches file headers (`HEAD`) and calculates chunk ranges.

*  **Hydra Engine:** Spawns 4 concurrent Goroutines. Each routine manages its own memory buffer and writes directly to the disk.

*  **Broadcaster:** Throttles and sends JSON progress updates over the active WebSocket connection.

  

## 📂 Project Structure

  

```text

pulldown/

├── backend/ # Golang API Server

│ ├── main.go # API Routes & WebSocket Handler

│ ├── downloader.go # Core download logic (Chunking & Merging)

│ ├── go.mod # Go dependencies

│ └── (Downloads appear here)

│

└── frontend/ # Angular Client

├── src/

│ ├── app/components # UI Components

│ └── app/services # WebSocket & HTTP Logic

├── angular.json

└── package.json
```
  

## 🚀 Getting Started

  

**Prerequisites**

Go (v1.20+)

Node.js (v18+) & Angular CLI

1. Setup the Backend

Navigate to the backend folder and start the Go server.

```Bash

cd backend

go mod tidy

go run .
```
The server will start on http://localhost:8080.

2. Setup the Frontend

Open a new terminal, navigate to the frontend folder, and start the Angular app.

```Bash

cd frontend

npm install

ng serve -o
```
The application will open at http://localhost:4200.

## 🕹️ Usage

Open the web interface.

Paste a direct download link (e.g., a PDF, Zip, or Image).

Test Link: http://212.183.159.230/5MB.zip

Click Download.

Watch the progress bar update in real-time as the backend fetches and merges the file.

Check the backend/ folder to see your downloaded file.

## 🧠 Technical Highlights (What I Learned)

**Concurrency:** Managing WaitGroups to synchronize multiple download threads.

**Memory Management:** Replacing io.Copy with manual buffer loops to track bytes as they flow.

**Race Conditions:** Using sync/atomic to prevent data corruption when multiple threads update the progress counter simultaneously.

**Zone Pollution:** Debugging Angular's change detection cycle using NgZone.run() to force UI updates from external WebSocket events.


## 📄 License

  

This project is open-source and available under the MIT License.
