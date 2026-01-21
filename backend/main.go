package main

import (
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

var(
	activeCon *websocket.Conn
	conMutex sync.Mutex
)

//Upgrade http request to websocket
var wsupgrader=websocket.Upgrader{
	ReadBufferSize: 1024,
	WriteBufferSize: 1024,
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
		c.Writer.Header().Set("Access-Control-Allow-Methods","POST,GET,OPTIONS,PUT,DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers","Content-Type,Authorization")

		if c.Request.Method=="OPTIONS"{
			c.AbortWithStatus(204)
			return
		}
	})

	r.GET("/ws",wsHandler)

	r.POST("/download",startDownloadHandler)

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

var lastBroadcast int=-1

//Bridge function
func SendProgress(fileName string,percent float64){
	conMutex.Lock()
	defer conMutex.Unlock()

	if activeCon==nil{
		return
	}

	//Optimization
	currInt:=int(percent)
	if currInt==lastBroadcast && percent<100.00{
		return
	}

	lastBroadcast=currInt

	msg:=gin.H{
		"event":"progress",
		"fileName":fileName,
		"percent":percent,
	}

	err:=activeCon.WriteJSON(msg)

	if err!=nil{
		log.Println("Error sending in progress: ",err)
		activeCon.Close()
		activeCon=nil
	}
}

//Handler method: Starts when frontend sends the request
func startDownloadHandler(c *gin.Context){
	var req DownloadRequest

	//Read JSON sent by frontend
	if err:=c.ShouldBindJSON(&req);err!=nil{
		c.JSON(http.StatusBadRequest,gin.H{"error":err.Error()})
		return
	}

	if strings.Contains(req.Url,"youtube") || strings.Contains(req.Url,"youtu.be"){
		go downloadYoutube(req.Url,"video.mp4")
	}else{
		go processDownload(req.Url)
	}

	//Response
	c.JSON(http.StatusCreated,gin.H{
		"message":"Download started",
		"url":req.Url,
	})

}