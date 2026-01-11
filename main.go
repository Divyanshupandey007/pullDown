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

type Part struct{
	Index int
	Start int64
	End int64
}

func main() {
	if len(os.Args)<2{
		fmt.Println("Please provide a URL")
		os.Exit(1)
	}

	//Take the link from cli
	url:=os.Args[1]

	//Extracts the file name from the URL
	fileName:=path.Base(url)

	res,err:=http.Head(url)
	if err!=nil{
		log.Fatal(err)
	}

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
		log.Fatal(err)
	}

	//Used Sprintf to return formatted string
	rangeHeader:=fmt.Sprintf("bytes=%d-%d",part.Start,part.End)
	req.Header.Set("Range",rangeHeader)

	client:=&http.Client{}
	res,err:=client.Do(req)
	if err!=nil{
		log.Fatal(err)
	}

	//Used defer so that the res object is closed after its functioning
	defer res.Body.Close()

	fileName:=fmt.Sprintf("part_%d.tmp",part.Index)
	file,err:=os.Create(fileName)
	if err!=nil{
		log.Fatal(err)
	}

	defer file.Close()

	io.Copy(file,res.Body)

	fmt.Println("Download finished: ",fileName)
}

func mergeParts(fileName string,numParts int){
	//Create the final file
	outFile,err:=os.Create(fileName)
	if err!=nil{
		log.Fatal(err)
	}
	defer outFile.Close()

	for i:=0;i<numParts;i++{
		partFileName:=fmt.Sprintf("part_%d.tmp",i)

		//Reading the partial files
		partFile,err:=os.Open(partFileName)

		if err!=nil{
			log.Fatal(err)
		}

		io.Copy(outFile,partFile)
		partFile.Close()
		//Remove the partial files
		os.Remove(partFileName)
	}
	fmt.Println("Files merged")
}