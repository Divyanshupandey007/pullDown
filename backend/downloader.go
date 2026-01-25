package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"math"
	"mime"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

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


	dataMutex.Lock()
	for i:=range Tasks{
		if Tasks[i].ID==taskId{
			Tasks[i].TotalSize=res.ContentLength
			break
		}
	}
	dataMutex.Unlock()
	saveTasks()

	var fileName string
	if len(customName)>0{
		fileName=customName[0]
	}else{
		if cd:=res.Header.Get("Content-Disposition");cd!=""{
			_,params,err:=mime.ParseMediaType(cd)
			if err==nil{
				fileName=params["filename"]
			}
		}

		if fileName==""{
			fileName=path.Base(downloadUrl)
		}

		if fileName=="" || !strings.Contains(fileName,"."){
			fileName="download_file.bin"
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

	if res.ContentLength>0{
		percent:=float64(initialDownloaded)/float64(res.ContentLength)*100
		SendProgress(taskId,fileName,percent)
	}

	//For implementing concurrency
	var wg sync.WaitGroup

	errChan:=make(chan error,len(parts))

	for i := range parts {
		//Increment the goroutine counter
		wg.Add(1)

		go func(p Part){
			defer wg.Done()
			maxRetries:=5
			for attempt:=0;attempt<maxRetries;attempt++{
				if ctx.Err()!=nil{
					return
				}
				err:=downloadPart(ctx,taskId,downloadUrl,fileName,p,&downloadBytes,res.ContentLength)
				if err==nil{
					return
				}

				if ctx.Err()!=nil{
					return
				}
				fmt.Printf("Part %d failed (Attempt %d %d): %v. Retrying in 2 sec...",p.Index,attempt+1,maxRetries,err)
				time.Sleep(2*time.Second)
			}
			errChan<-fmt.Errorf("Part %d failed after %d attempts",p.Index,maxRetries)
			}(parts[i])
		}
		wg.Wait()
		close(errChan)

		if len(errChan)>0{
			fmt.Println("Download failed due to network error")
			return
		}

	if ctx.Err()==nil{
		mergeParts(fileName, len(parts))
		SendProgress(taskId,fileName,100.0)

		dataMutex.Lock()
		for i:=range Tasks{
			if Tasks[i].ID==taskId{
				Tasks[i].Status="Completed"
				Tasks[i].FileName=fileName
				Tasks[i].Downloaded=res.ContentLength
				Tasks[i].TotalSize=res.ContentLength
				break
			}
		}
		dataMutex.Unlock()
		saveTasks()
		fmt.Println("Download Complete")
	}else{
		currBytes:=atomic.LoadInt64(&downloadBytes)
		dataMutex.Lock()
		for i:=range Tasks{
			if Tasks[i].ID==taskId{
				Tasks[i].Status="Paused"
				Tasks[i].Downloaded=currBytes
				break
			}
		}
		dataMutex.Unlock()
		saveTasks()
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

	dataMutex.Lock()
	for i:=range Tasks{
		if Tasks[i].ID==originalUrl{
			Tasks[i].FileName=safeTitle
			break
		}
	}
	dataMutex.Unlock()
	saveTasks()

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

func downloadPart(ctx context.Context,taskId string,url string, fileName string, part Part, progress *int64, totalSize int64) error {
	// defer wg.Done()

	tmpFileName:=fmt.Sprintf("part_%d.tmp",part.Index)
	var currStart=part.Start

	file,err:=os.OpenFile(tmpFileName,os.O_APPEND|os.O_CREATE|os.O_WRONLY,0644)
	if err!=nil{
		return fmt.Errorf("File open error: %v",err)
	}
	defer file.Close()

	stat,err:=file.Stat()
	if err==nil{
		currStart+=stat.Size()
	}

	expectedBytesRemaining:=(part.End-currStart)+1

	if expectedBytesRemaining<=0{
		return nil
	}

	req, err := http.NewRequestWithContext(ctx,"GET", url, nil)
	if err != nil {
		return err
	}

	//Used Sprintf to return formatted string
	rangeHeader := fmt.Sprintf("bytes=%d-%d", currStart, part.End)
	req.Header.Set("Range", rangeHeader)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return err
	}

	//Used defer so that the res object is closed after its functioning
	defer res.Body.Close()

	if res.StatusCode!=200 && res.StatusCode!=206{
		return fmt.Errorf("Bad status code: %d",res.StatusCode)
	}

	//Create buffer
	buf := make([]byte, 32*1024)
	var bytesDownloadedThisSession int64=0

	for {
		//Read data
		if ctx.Err()!=nil{
			return nil
		}
		n, err := res.Body.Read(buf)

		//Write data to file upto 'n' bytes
		if n > 0 {
			file.Write(buf[:n])

			bytesDownloadedThisSession+=int64(n)
			//Safely add 'n' bytes to shared counter
			current := atomic.AddInt64(progress, int64(n))
			percent := float64(current) / float64(totalSize) * 100
			if int(current)%10==0{
				SendProgress(taskId,fileName, math.Min(percent,100.0))
			}
		}
		if err!=nil{
			if err==io.EOF{
				if bytesDownloadedThisSession<expectedBytesRemaining{
					return fmt.Errorf("Server hung up early. Got %d bytes, expected %d",bytesDownloadedThisSession,expectedBytesRemaining)
				}
				break
			}
			return err
		}
	}
	return nil
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

func updateTaskStatus(taskId string,status string,fileName string){
	dataMutex.Lock()
	defer dataMutex.Unlock()

	for i:=range Tasks{
		if Tasks[i].ID==taskId{
			Tasks[i].Status=status
			if fileName!=""{
				Tasks[i].FileName=fileName
			}
			break
		}
	}
}