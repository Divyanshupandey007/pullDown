package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Config struct {
	Port          string
	MaxConcurrent int
	PartsPerFile  int
	DownloadDir   string
}

// Map JSON data from frontend
type DownloadRequest struct {
	Url string `json:"url"`
}

// To identify which download to pause/resume
type ActionRequest struct {
	Url string `json:"url"`
}

type DownloadManager struct {
	Tasks     []Task
	dataMutex sync.Mutex

	downloadManager map[string]context.CancelFunc
	managerMutex    sync.Mutex

	semaphore chan struct{}

	config   Config
	settings Settings

	limiter *BandwidthMonitor
}

// Global Manager
var (
	clients    = make(map[*websocket.Conn]bool)
	clientsMux sync.Mutex
	manager    *DownloadManager
)

// Upgrade http request to websocket
var wsupgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func main() {

	//Router setup
	r := gin.Default()

	//Enable CORS
	r.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		allowedOrigins := []string{"http://localhost:4200", "http://localhost:8080"}
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				break
			}
		}
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST,GET,DELETE,OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
	})

	manager = NewDownloadManager()

	r.GET("/ws", manager.wsHandler)
	r.POST("/download", manager.StartDownloadHandler)
	r.POST("/pause", manager.PauseDownloadHandler)
	r.POST("/resume", manager.ResumeDownloadHandler)
	r.GET("/settings", manager.GetSettingsHandler)
	r.POST("/settings", manager.UpdateSettingsHandler)
	r.DELETE("/delete", manager.DeleteDownloadHandler)
	r.POST("/mode", manager.SetModeHandler)

	manager.LoadTasks()
	manager.LoadSettings()

	manager.limiter.Start()

	srv := &http.Server{
		Addr:    manager.config.Port,
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down... saving state")

	manager.managerMutex.Lock()
	for url, cancel := range manager.downloadManager {
		cancel()
		delete(manager.downloadManager, url)
	}
	manager.managerMutex.Unlock()

	manager.dataMutex.Lock()
	for i := range manager.Tasks {
		if manager.Tasks[i].Status == "Downloading" {
			manager.Tasks[i].Status = "Paused"
		}
	}

	manager.dataMutex.Unlock()
	manager.SaveTasks()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	manager.limiter.Stop()
	srv.Shutdown(ctx)

	log.Println("Server stopped completely")
}

func NewDownloadManager() *DownloadManager {
	cfg := Config{
		Port:          ":8080",
		MaxConcurrent: 4,
		PartsPerFile:  4,
		DownloadDir:   ".",
	}
	dm := &DownloadManager{
		Tasks:           make([]Task, 0),
		downloadManager: make(map[string]context.CancelFunc),
		semaphore:       make(chan struct{}, cfg.MaxConcurrent),
		config:          cfg,
		settings: Settings{
			DownloadPath:    "C:\\Downloads",
			MaxDownloads:    4,
			MaxConnections:  16,
			ConnTimeout:     30,
			AutoStart:       true,
			CompletionAlert: true,
			ForceHttps:      true,
			AutoRetry:       true,
			NotifComplete:   true,
			NotifError:      true,
		},
		limiter: NewBandwidthMonitor(),
	}
	dm.limiter.SetParts(cfg.PartsPerFile)
	return dm
}

func (dm *DownloadManager) wsHandler(c *gin.Context) {

	//Websocket connection
	con, err := wsupgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Error in connection: ", err)
		return
	}

	//Save the connection
	clientsMux.Lock()
	clients[con] = true
	clientsMux.Unlock()

	//Send tasks immediately upon connection
	dm.dataMutex.Lock()
	initialMsg := gin.H{
		"event": "initial_state",
		"tasks": dm.Tasks,
	}
	dm.dataMutex.Unlock()
	con.WriteJSON(initialMsg)

	log.Println("Client connected via WebSocket!")

	con.SetReadDeadline(time.Now().Add(60 * time.Second))
	con.SetPongHandler(func(string) error {
		con.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	defer func() {
		clientsMux.Lock()
		//Clear the active connection
		delete(clients, con)
		clientsMux.Unlock()
		con.Close()
		log.Println("Client disconnected")
	}()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			if err := con.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}()

	//For keeping connection alive
	for {
		_, _, err := con.ReadMessage()
		if err != nil {
			break
		}
	}

}

// Bridge function
func SendProgress(taskId string, fileName string, percent float64, totalSize int64, speed float64, eta float64) {
	clientsMux.Lock()
	defer clientsMux.Unlock()

	msg := gin.H{
		"event":     "progress",
		"id":        taskId,
		"fileName":  fileName,
		"percent":   percent,
		"totalSize": totalSize,
		"speed":     speed,
		"eta":       eta,
	}

	for con := range clients {
		if err := con.WriteJSON(msg); err != nil {
			con.Close()
			delete(clients, con)
		}
	}
}

func SendError(taskId string, message string) {
	clientsMux.Lock()
	defer clientsMux.Unlock()

	msg := gin.H{
		"event":   "error",
		"id":      taskId,
		"message": message,
	}

	for con := range clients {
		if err := con.WriteJSON(msg); err != nil {
			con.Close()
			delete(clients, con)
		}
	}
}

// Handler method: Starts when frontend sends the request
func (dm *DownloadManager) StartDownloadHandler(c *gin.Context) {
	var req DownloadRequest

	//Read JSON sent by frontend
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	parsedUrl, urlErr := url.ParseRequestURI(req.Url)
	if urlErr != nil || (parsedUrl.Scheme != "http" && parsedUrl.Scheme != "https") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid URL. Only http:// and https:// are supported."})
		return
	}

	if dm.settings.ForceHttps && parsedUrl.Scheme == "http" {
		parsedUrl.Scheme = "https"
		req.Url = parsedUrl.String()
	}

	dm.dataMutex.Lock()
	var taskFound bool = false

	for i := range dm.Tasks {
		if dm.Tasks[i].ID == req.Url {
			dm.Tasks[i].Status = "Downloading"
			taskFound = true
			break
		}
	}

	if !taskFound {
		newTask := Task{
			ID:         req.Url,
			Url:        req.Url,
			FileName:   "Pending...",
			Status:     "Downloading",
			TotalSize:  0,
			Downloaded: 0,
		}
		dm.Tasks = append(dm.Tasks, newTask)
	}

	dm.dataMutex.Unlock()
	dm.SaveTasks()

	dm.managerMutex.Lock()
	if _, alreadyRunning := dm.downloadManager[req.Url]; alreadyRunning {
		dm.managerMutex.Unlock()
		c.JSON(http.StatusOK, gin.H{
			"message": "Download Already Running",
		})
		return
	}
	dm.managerMutex.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	dm.managerMutex.Lock()
	dm.downloadManager[req.Url] = cancel
	dm.managerMutex.Unlock()

	if strings.Contains(req.Url, "youtube") || strings.Contains(req.Url, "youtu.be") {
		go dm.downloadYoutube(ctx, req.Url)
	} else {
		go dm.processDownload(ctx, req.Url, req.Url)
	}

	//Response
	c.JSON(http.StatusOK, gin.H{
		"message": "Download started",
	})

}

func (dm *DownloadManager) setTaskError(taskId string, errMsg string) {
	dm.managerMutex.Lock()
	if cancel, exists := dm.downloadManager[taskId]; exists {
		cancel()
		delete(dm.downloadManager, taskId)
	}
	dm.managerMutex.Unlock()

	dm.dataMutex.Lock()
	for i := range dm.Tasks {
		if dm.Tasks[i].ID == taskId {
			dm.Tasks[i].Status = "Error"
			break
		}
	}
	dm.dataMutex.Unlock()
	dm.SaveTasks()
	log.Println("Task error:", taskId, "-", errMsg)
}

func (dm *DownloadManager) PauseDownloadHandler(c *gin.Context) {
	var req ActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dm.managerMutex.Lock()
	cancel, exists := dm.downloadManager[req.Url]
	if exists {
		cancel()
		delete(dm.downloadManager, req.Url)
	}
	dm.managerMutex.Unlock()

	dm.dataMutex.Lock()
	for i := range dm.Tasks {
		if dm.Tasks[i].ID == req.Url {
			dm.Tasks[i].Status = "Paused"
			break
		}
	}
	dm.dataMutex.Unlock()
	dm.SaveTasks()
	if exists {
		c.JSON(http.StatusOK, gin.H{"message": "Download Paused"})
	} else {
		c.JSON(http.StatusNotFound, gin.H{"message": "Download not running"})
	}
}

func (dm *DownloadManager) ResumeDownloadHandler(c *gin.Context) {
	dm.StartDownloadHandler(c)
}

func (dm *DownloadManager) DeleteDownloadHandler(c *gin.Context) {
	var req ActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dm.managerMutex.Lock()
	if cancel, exists := dm.downloadManager[req.Url]; exists {
		cancel()
		delete(dm.downloadManager, req.Url)
	}
	dm.managerMutex.Unlock()

	time.Sleep(500 * time.Millisecond)

	dm.dataMutex.Lock()
	for i := range dm.Tasks {
		if dm.Tasks[i].ID == req.Url {
			dm.Tasks = append(dm.Tasks[:i], dm.Tasks[i+1:]...)
			break
		}
	}
	dm.dataMutex.Unlock()

	hash := taskHash(req.Url)
	cleanupPattern := hash + "_part_*.tmp"
	matches, _ := filepath.Glob(cleanupPattern)
	for _, f := range matches {
		os.Remove(f)
	}

	dm.SaveTasks()
	c.JSON(http.StatusOK, gin.H{"message": "Download deleted"})
}

func (dm *DownloadManager) GetSettingsHandler(c *gin.Context) {
	dm.dataMutex.Lock()
	defer dm.dataMutex.Unlock()
	c.JSON(http.StatusOK, dm.settings)
}

func (dm *DownloadManager) UpdateSettingsHandler(c *gin.Context) {
	var newSettings Settings
	if err := c.ShouldBindJSON(&newSettings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dm.dataMutex.Lock()
	dm.settings = newSettings
	dm.dataMutex.Unlock()

	dm.SaveSettings()

	dm.config.DownloadDir = newSettings.DownloadPath
	dm.config.MaxConcurrent = newSettings.MaxDownloads
	dm.config.PartsPerFile = newSettings.MaxConnections
	dm.limiter.SetParts(newSettings.MaxConnections)

	dm.managerMutex.Lock()
	activeCount := len(dm.downloadManager)
	dm.managerMutex.Unlock()

	if activeCount == 0 {
		dm.semaphore = make(chan struct{}, newSettings.MaxDownloads)
	} else {
		log.Println("Warning: Cannot update max concurrent downloads while downloads are active. Will take effect after current downloads finish")
	}

	c.JSON(http.StatusOK, gin.H{"message": "Settings saved"})
}

// Saves memory list to a file
func (dm *DownloadManager) SaveTasks() {
	dm.dataMutex.Lock()
	defer dm.dataMutex.Unlock()

	bytes, err := json.MarshalIndent(dm.Tasks, "", " ")
	if err != nil {
		log.Println("Error marshalling:", err)
		return
	}

	tmpFile := "tasks.json.tmp"
	if writeErr := os.WriteFile(tmpFile, bytes, 0644); writeErr != nil {
		log.Println("Error writing tasks temp file:", writeErr)
		return
	}

	if renameErr := os.Rename(tmpFile, "tasks.json"); renameErr != nil {
		log.Println("Error renaming tasks file:", renameErr)
	}
}

func (dm *DownloadManager) LoadTasks() {
	dm.dataMutex.Lock()
	defer dm.dataMutex.Unlock()

	bytes, err := os.ReadFile("tasks.json")
	if err != nil {
		log.Println("Error reading json file:", err)
		return
	}

	if err := json.Unmarshal(bytes, &dm.Tasks); err != nil {
		log.Println("Error parsing tasks.json", err)
		return
	}

	for i := range dm.Tasks {
		if dm.Tasks[i].Status == "Downloading" {
			dm.Tasks[i].Status = "Paused"
		}
	}
}

func (dm *DownloadManager) SaveSettings() {
	dm.dataMutex.Lock()
	defer dm.dataMutex.Unlock()

	bytes, err := json.MarshalIndent(dm.settings, "", " ")
	if err != nil {
		log.Println("Error marshalling settings:", err)
		return
	}

	tmpFile := "settings.json.tmp"
	if err := os.WriteFile(tmpFile, bytes, 0644); err != nil {
		log.Println("Error writing settings temp file:", err)
		return
	}

	if err := os.Rename(tmpFile, "settings.json"); err != nil {
		log.Println("Error renaming settings file:", err)
	}
}

func (dm *DownloadManager) LoadSettings() {
	dm.dataMutex.Lock()
	defer dm.dataMutex.Unlock()

	bytes, err := os.ReadFile("settings.json")
	if err != nil {
		log.Println("No settings file found, using defaults")
		return
	}

	if err := json.Unmarshal(bytes, &dm.settings); err != nil {
		log.Println("Error parsing settings.json:", err)
	}
}

func (dm *DownloadManager) SetModeHandler(c *gin.Context) {
	var req struct {
		Mode string `json:"mode"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	dm.limiter.SetMode(req.Mode)
	log.Println("Speed mode set to:", req.Mode)
	c.JSON(http.StatusOK, gin.H{"message": "Mode set to " + req.Mode})
}
