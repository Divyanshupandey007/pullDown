package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

//Map JSON data from frontend
type DownloadRequest struct{
	Url string `json:"url"`
}

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

	r.GET("/ping",func(c *gin.Context){
		c.JSON(200,gin.H{
			"message":"pong",
		})
	})

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
	defer con.Close()

	log.Println("Client connected via WebSocket!")

	//For keeping connection alive
	for{
		_,_,err:=con.ReadMessage()
		if err!=nil{
			log.Println("Client disconnected")
			break
		}
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

	//Function runs in background to avoid starvation 
	go processDownload(req.Url)

	//Response
	c.JSON(http.StatusCreated,gin.H{
		"message":"Download started",
		"url":req.Url,
	})

}