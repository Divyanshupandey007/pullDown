package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

const baseUrl="https://www.w3.org/WAI/ER/tests/xhtml/testfiles/resources/pdf/dummy.pdf"

type Part struct{
	Index int
	Start int64
	End int64
}

func main() {
	res,err:=http.Head(baseUrl)
	if err!=nil{
		log.Fatal(err)
	}

	parts:=calculateParts(res.ContentLength,4)

	fmt.Println(parts)
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

func downloadPart(url string,part Part){

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

	fileName:=fmt.Sprintf("part_%d.pdf",part.Index)
	file,err:=os.Create(fileName)
	if err!=nil{
		log.Fatal(err)
	}

	defer file.Close()

	io.Copy(file,res.Body)

	fmt.Println("Download finished: ",fileName)
}
