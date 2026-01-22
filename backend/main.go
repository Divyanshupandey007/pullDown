package main

import (
	"context"
	"log"
	"net/http"
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
		managerMutex.Unlock()
		c.JSON(http.StatusOK,gin.H{"message":"Download Paused"})
	}else{
		managerMutex.Unlock()
		c.JSON(http.StatusNotFound,gin.H{"message":"Downlod not found or already stopped"})
	}
}

func resumeDownloadHandler(c *gin.Context){
	startDownloadHandler(c)
}