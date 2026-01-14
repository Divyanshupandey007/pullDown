package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

//Map JSON data from frontend
type DownloadRequest struct{
	Url string `json:"url"`
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

	r.GET("/ping",func(c *gin.Context){
		c.JSON(200,gin.H{
			"message":"pong",
		})
	})

	r.POST("/download",startDownloadHandler)

	//Run Server
	r.Run()
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