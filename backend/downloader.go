package main

import (
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path"
	"sync"
	"sync/atomic"

	"github.com/kkdai/youtube/v2"
)

// Part struct for handling chunks
type Part struct {
	Index int
	Start int64
	End   int64
}

func processDownload(url string) {
	fmt.Println("Starting download for: ", url)

	res, err := http.Head(url)
	if err != nil {
		log.Println("Error fetching HEAD: ", err)
		return
	}

	fileName := path.Base(url)

	parts := calculateParts(res.ContentLength, 4)

	//Create shared counter
	var downloadBytes int64 = 0

	//For implementing concurrency
	var wg sync.WaitGroup
	for i := range parts {
		//Increment the goroutine counter
		wg.Add(1)
		//&wg is reference for the pointer
		go downloadPart(url, fileName, parts[i], &wg, &downloadBytes, res.ContentLength)
	}

	//It will wait for all parts to download
	wg.Wait()

	mergeParts(fileName, len(parts))
	fmt.Println("Download Complete")
}

func downloadYoutube(url string,fileName string){
	fmt.Println("Starting download for:",url)

	client:=youtube.Client{}
	video,err:=client.GetVideo(url)
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

	stream,_,err:=client.GetStream(video,format)
	if err!=nil{
		fmt.Println("Error getting stream:",err)
		return
	}
	defer stream.Close()

	file, err := os.Create(fileName)
	if err != nil {
		log.Println("Error creating file: ", err)
		return
	}

	defer file.Close()

	//Create buffer
	buf := make([]byte, 32*1024)
	var totalBytes int64=0
	totalSize:=format.ContentLength
	for {
		//Read data
		n, err := stream.Read(buf)

		if err != nil {
			if err == io.EOF {
				break
			}
			log.Println("Error reading stream:", err)
			return
		}

		//Write data to file upto 'n' bytes
		if n > 0 {
			file.Write(buf[:n])
			totalBytes+=int64(n)
			var percent float64
			if totalSize>0{
				percent = float64(totalBytes) / float64(totalSize) * 100
			}else{
				percent = float64((totalBytes%(10*1024*1024))) / (10*1024*1024)*100
			}
			SendProgress(fileName, percent)
			if totalBytes%(1024*1024)==0{
				fmt.Printf("Downloaded: %dMB\n",totalBytes/1024/1024)
			}
		}
	}
	SendProgress(fileName,100.0)
	fmt.Println("Download complete!")
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

func downloadPart(url string, fileName string, part Part, wg *sync.WaitGroup, progress *int64, totalSize int64) {
	defer wg.Done()

	//Used NewRequest instead of Get() to add custom headers
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println("Error creating request: ", err)
		return
	}

	//Used Sprintf to return formatted string
	rangeHeader := fmt.Sprintf("bytes=%d-%d", part.Start, part.End)
	req.Header.Set("Range", rangeHeader)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Println("Error downloading part: ", err)
		return
	}

	//Used defer so that the res object is closed after its functioning
	defer res.Body.Close()

	//Check if server supports range requests
	if res.StatusCode == 200 {
		log.Printf("WARNING: Server returned 200 OK instead of 206 Partial Content. Range headers not supported for part %d", part.Index)
	} else if res.StatusCode != 206 {
		log.Printf("ERROR: Unexpected status code %d for part %d", res.StatusCode, part.Index)
		return
	}

	tmpFile := fmt.Sprintf("part_%d.tmp", part.Index)
	file, err := os.Create(tmpFile)
	if err != nil {
		log.Println("Error creating temp file: ", err)
		return
	}

	defer file.Close()

	//Create buffer
	buf := make([]byte, 32*1024)

	for {
		//Read data
		n, err := res.Body.Read(buf)

		if err != nil {
			if err == io.EOF {
				break
			}
			log.Println("Error reading: ", err)
			return
		}

		//Write data to file upto 'n' bytes
		if n > 0 {
			file.Write(buf[:n])

			//Safely add 'n' bytes to shared counter
			current := atomic.AddInt64(progress, int64(n))
			percent := float64(current) / float64(totalSize) * 100

			//Cap progress at 100% to handle servers that ignore Range headers
			percent = math.Min(percent, 100.0)

			SendProgress(fileName, percent)
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