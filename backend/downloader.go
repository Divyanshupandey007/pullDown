package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"sync"
)

//Part struct for handling chunks
type Part struct{
	Index int
	Start int64
	End int64
}

func processDownload(url string) {
	fmt.Println("Staring download for: ",url)

	res,err:=http.Head(url)
	if err!=nil{
		log.Println("Error fetching HEAD: ",err)
		return
	}

	fileName:=path.Base(url)

	parts:=calculateParts(res.ContentLength,4)

	//For implementing concurrency
	var wg sync.WaitGroup
	for i:=range parts{
		//Increment the goroutine counter
		wg.Add(1)
		//&wg is reference for the pointer
		go downloadPart(url,parts[i],&wg)
	}
	wg.Wait() //It will wait for all parts to download

	mergeParts(fileName,len(parts))
	fmt.Println("Download Completed for",fileName)
}

//Logic for calculating size of each part
func calculateParts(totalSize int64,numParts int) []Part{
	var parts []Part
	chunkSize:=totalSize/int64(numParts)
	
	for i:=range numParts{
		start:=int64(i)*chunkSize
		end:=start+chunkSize-1

		if i==numParts-1{
			end=totalSize-1
		}

		parts=append(parts, Part{Index: i,Start: start,End: end})
	}
	return parts
}

func downloadPart(url string,part Part,wg *sync.WaitGroup){
	defer wg.Done()

	//Used NewRequest instead of Get() to add custom headers
	req,err:=http.NewRequest("GET",url,nil)
	if err!=nil{
		log.Println("Error creating request: ",err)
		return
	}

	//Used Sprintf to return formatted string
	rangeHeader:=fmt.Sprintf("bytes=%d-%d",part.Start,part.End)
	req.Header.Set("Range",rangeHeader)

	client:=&http.Client{}
	res,err:=client.Do(req)
	if err!=nil{
		log.Println("Error downloading part: ",err)
		return
	}

	//Used defer so that the res object is closed after its functioning
	defer res.Body.Close()

	fileName:=fmt.Sprintf("part_%d.tmp",part.Index)
	file,err:=os.Create(fileName)
	if err!=nil{
		log.Println("Error creating temp file: ",err)
		return
	}

	defer file.Close()

	io.Copy(file,res.Body)

	fmt.Println("Part finished: ",fileName)
}

func mergeParts(fileName string,numParts int){
	//Create the final file
	outFile,err:=os.Create(fileName)
	if err!=nil{
		log.Println("Error creating final file: ",err)
		return
	}
	defer outFile.Close()

	for i:=0;i<numParts;i++{
		partFileName:=fmt.Sprintf("part_%d.tmp",i)

		//Reading the partial files
		partFile,err:=os.Open(partFileName)

		if err!=nil{
			log.Println("Error opening part: ",err)
			return
		}

		io.Copy(outFile,partFile)
		partFile.Close()
		//Remove the partial files
		os.Remove(partFileName)
	}
	fmt.Println("Files merged into:",fileName)
}