package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/kkdai/youtube/v2"
)

//Calculate progress across restarts
var globalTotalSize int64=0

// Part struct for handling chunks
type Part struct {
	Index int
	Start int64
	End   int64
}

func processDownload(ctx context.Context,taskId string,downloadUrl string,customName ...string) {
	fmt.Println("Starting/Resuming download for: ", downloadUrl)

	res, err := http.Head(downloadUrl)
	if err != nil {
		log.Println("Error fetching HEAD: ", err)
		return
	}

	globalTotalSize=res.ContentLength

	var fileName string
	if len(customName)>0{
		fileName=customName[0]
	}else{
		fileName = path.Base(downloadUrl)

		if len(fileName)>50 || strings.Contains(fileName,"."){
			fileName="download_file.mp4"
		}
	}

	parts := calculateParts(res.ContentLength, 4)

	var initialDownloaded int64=0

	for i:=range parts{
		tmpFileName:=fmt.Sprintf("part_%d.tmp",i)
		if stat,err:=os.Stat(tmpFileName);err==nil{
			initialDownloaded+=stat.Size()
		}
	}

	//Create shared counter
	var downloadBytes=initialDownloaded

	percent:=float64(initialDownloaded)/float64(res.ContentLength)*100
	SendProgress(taskId,fileName,percent)

	//For implementing concurrency
	var wg sync.WaitGroup
	for i := range parts {
		//Increment the goroutine counter
		wg.Add(1)
		//&wg is reference for the pointer
		go downloadPart(ctx,taskId,downloadUrl, fileName, parts[i], &wg, &downloadBytes, res.ContentLength)
	}

	//It will wait for all parts to download
	wg.Wait()

	if ctx.Err()==nil{
		mergeParts(fileName, len(parts))
		SendProgress(taskId,fileName,100.0)
		fmt.Println("Download Complete")
	}else{
		fmt.Println("Download Paused")
	}
}

func downloadYoutube(ctx context.Context,originalUrl string){
	fmt.Println("Analyzing youtube video:",originalUrl)

	client:=youtube.Client{}
	video,err:=client.GetVideo(originalUrl)
	if err!=nil{
		fmt.Println("Error getting video info:",err)
		return
	}
	formats:=video.Formats.WithAudioChannels()
	if len(formats)==0{
		fmt.Println("Error: No video with audio found")
		return
	}
	format:=&formats[0]
	fmt.Printf("Found format: %s, Quality: %s\n",format.MimeType,format.Quality)

	//Get direct URL
	streamURL,err:=client.GetStreamURL(video,format)
	if err!=nil{
		fmt.Println("Error getting stream URL:",err)
		return
	}

	fmt.Println("Got direct stream URL....")
	safeTitle:=sanitizeFileName(video.Title)+".mp4"
	processDownload(ctx,originalUrl,streamURL,safeTitle)

}

func sanitizeFileName(name string) string{
	invalidChars:=[]string{"/","\\",":","*","?","\"","<",">","|"}
	for _,chars:=range invalidChars{
		name=strings.ReplaceAll(name,chars,"_")
	}
	return name
}

// Logic for calculating size of each part
func calculateParts(totalSize int64, numParts int) []Part {
	var parts []Part
	chunkSize := totalSize / int64(numParts)

	for i := range numParts {
		start := int64(i) * chunkSize
		end := start + chunkSize - 1

		if i == numParts-1 {
			end = totalSize - 1
		}

		parts = append(parts, Part{Index: i, Start: start, End: end})
	}
	return parts
}

func downloadPart(ctx context.Context,taskId string,url string, fileName string, part Part, wg *sync.WaitGroup, progress *int64, totalSize int64) {
	defer wg.Done()

	tmpFileName:=fmt.Sprintf("part_%d.tmp",part.Index)
	var currStart=part.Start

	file,err:=os.OpenFile(tmpFileName,os.O_APPEND|os.O_CREATE|os.O_WRONLY,0644)
	if err!=nil{
		fmt.Println("Error opening temp file:",err)
		return
	}
	defer file.Close()

	stat,err:=file.Stat()
	if err!=nil{
		fileSize:=stat.Size()
		currStart+=fileSize
	}

	if currStart>part.End{
		fmt.Printf("Part %d already finished. Skipping...\n",part.Index)
		return
	}

	req, err := http.NewRequestWithContext(ctx,"GET", url, nil)
	if err != nil {
		log.Println("Error creating request: ", err)
		return
	}

	//Used Sprintf to return formatted string
	rangeHeader := fmt.Sprintf("bytes=%d-%d", currStart, part.End)
	req.Header.Set("Range", rangeHeader)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		if ctx.Err()!=nil{
			return
		}
		log.Println("Error downloading part: ", err)
		return
	}

	//Used defer so that the res object is closed after its functioning
	defer res.Body.Close()

	//Create buffer
	buf := make([]byte, 32*1024)

	for {
		select{
		case<-ctx.Done():
			return
		default:

		}
		//Read data
		n, err := res.Body.Read(buf)

		if err != nil {
			if err == io.EOF {
				break
			}
			if ctx.Err()!=nil{
				return
			}
		}

		//Write data to file upto 'n' bytes
		if n > 0 {
			file.Write(buf[:n])

			//Safely add 'n' bytes to shared counter
			current := atomic.AddInt64(progress, int64(n))
			percent := float64(current) / float64(totalSize) * 100
			if int(current)%10==0{
				SendProgress(taskId,fileName, math.Min(percent,100.0))
			}
		}
	}
}

func mergeParts(fileName string, numParts int) {
	//Create the final file
	outFile, err := os.Create(fileName)
	if err != nil {
		log.Println("Error creating final file: ", err)
		return
	}
	defer outFile.Close()

	for i := 0; i < numParts; i++ {
		partFileName := fmt.Sprintf("part_%d.tmp", i)

		//Reading the partial files
		partFile, err := os.Open(partFileName)

		if err != nil {
			log.Println("Error opening part: ", err)
			return
		}

		io.Copy(outFile, partFile)
		partFile.Close()
		//Remove the partial files
		os.Remove(partFileName)
	}
	fmt.Println("Files merged into:", fileName)
}