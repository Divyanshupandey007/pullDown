package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

//Map JSON data from frontend
type DownloadRequest struct{
	Url string `json:"url"`
}

//To identify which download to pause/resume
type ActionRequest struct{
	Url string `json:"url"`
}

//Global Manager
var(
	activeCon *websocket.Conn
	conMutex sync.Mutex

	//Store running downloads
	downloadManager=make(map[string]context.CancelFunc)
	managerMutex sync.Mutex

	Tasks []Task
	dataMutex sync.Mutex
)

//Upgrade http request to websocket
var wsupgrader=websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool{
		return true
	},
}


func main() {

	//Router setup
	r:=gin.Default()

	//Enable CORS
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin","*")
		c.Writer.Header().Set("Access-Control-Allow-Methods","POST,GET,OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers","Content-Type")

		if c.Request.Method=="OPTIONS"{
			c.AbortWithStatus(204)
			return
		}
	})

	r.GET("/ws",wsHandler)
	r.POST("/download",startDownloadHandler)
	r.POST("/pause",pauseDownloadHandler)
	r.POST("/resume",resumeDownloadHandler)

	loadTasks()
	
	//Run Server
	r.Run()
}

func wsHandler(c *gin.Context){

	//Websocket connection
	con,err:=wsupgrader.Upgrade(c.Writer,c.Request,nil)
	if err!=nil{
		log.Println("Error in connection: ",err)
		return
	}

	//Save the connection
	conMutex.Lock()
	activeCon=con
	conMutex.Unlock()

	//Send tasks immediately upon connection
	dataMutex.Lock()
	initialMsg:=gin.H{
		"event":"initial_state",
		"tasks":Tasks,
	}
	dataMutex.Unlock()
	activeCon.WriteJSON(initialMsg)

	log.Println("Client connected via WebSocket!")

	defer func(){
		conMutex.Lock()
		//Clear the active connection
		activeCon=nil
		conMutex.Unlock()
		con.Close()
	}()

	//For keeping connection alive
	for{
		_,_,err:=con.ReadMessage()
		if err!=nil{
			break
		}
	}

}

//Bridge function
func SendProgress(taskId string,fileName string,percent float64){
	conMutex.Lock()
	defer conMutex.Unlock()

	if activeCon==nil{
		return
	}

	msg:=gin.H{
		"event":"progress",
		"id":taskId,
		"fileName":fileName,
		"percent":percent,
	}

	activeCon.WriteJSON(msg)
}

//Handler method: Starts when frontend sends the request
func startDownloadHandler(c *gin.Context){
	var req DownloadRequest

	//Read JSON sent by frontend
	if err:=c.ShouldBindJSON(&req);err!=nil{
		c.JSON(http.StatusBadRequest,gin.H{"error":err.Error()})
		return
	}

	dataMutex.Lock()
	var taskFound bool=false

	for i:=range Tasks{
		if Tasks[i].ID==req.Url{
			Tasks[i].Status="Downloading"
			taskFound=true
			break
		}
	}

	if !taskFound{
		newTask:=Task{
			ID: req.Url,
			Url: req.Url,
			FileName: "Pending...",
			Status: "Downloading",
			TotalSize: 0,
			Downloaded: 0,
		}
		Tasks=append(Tasks, newTask)
	}

	dataMutex.Unlock()
	saveTasks()

	ctx,cancel:=context.WithCancel(context.Background())
	managerMutex.Lock()
	downloadManager[req.Url]=cancel
	managerMutex.Unlock()

	

	if strings.Contains(req.Url,"youtube") || strings.Contains(req.Url,"youtu.be"){
		go downloadYoutube(ctx,req.Url)
	}else{
		go processDownload(ctx,req.Url,req.Url)
	}

	//Response
	c.JSON(http.StatusOK,gin.H{
		"message":"Download started",
	})

}

func pauseDownloadHandler(c *gin.Context){
	var req ActionRequest
	if err:=c.ShouldBindJSON(&req);err!=nil{
		c.JSON(http.StatusBadRequest,gin.H{"error":err.Error()})
		return
	}

	managerMutex.Lock()
	cancel,exists:=downloadManager[req.Url]
	if exists{
		cancel()
		delete(downloadManager,req.Url)
	}
	managerMutex.Unlock()

	dataMutex.Lock()
	for i:=range Tasks{
		if Tasks[i].ID==req.Url{
			Tasks[i].Status="Paused"
			break
		}
	}
	dataMutex.Unlock()
	if exists{
		c.JSON(http.StatusOK,gin.H{"message":"Download Paused"})
	}else{
		c.JSON(http.StatusNotFound,gin.H{"message":"Downlod not running"})
	}
}

func resumeDownloadHandler(c *gin.Context){
	startDownloadHandler(c)
}

//Saves memory list to a file
func saveTasks(){
	dataMutex.Lock()
	defer dataMutex.Unlock()

	bytes,err:=json.MarshalIndent(Tasks,""," ")
	if err!=nil{
		fmt.Println("Error marshalling:",err)
		return
	}

	writeErr:=os.WriteFile("tasks.json",bytes,0644)
	if writeErr!=nil{
		fmt.Println(writeErr)
		return
	}
}

func loadTasks(){
	dataMutex.Lock()
	defer dataMutex.Unlock()

	bytes,err:=os.ReadFile("tasks.json")
	if err!=nil{
		fmt.Println("Error reading json file:",err)
		return
	}

	json.Unmarshal(bytes,&Tasks)

	for i:=range Tasks{
		if Tasks[i].Status=="Downloading"{
			Tasks[i].Status="Paused"
		}
	}
}