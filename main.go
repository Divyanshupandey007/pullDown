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
	p1:=Part{Index: 0,Start: 0,End: 500}
	downloadPart(baseUrl,p1)

	p2:=Part{Index: 1,Start: 501,End: 1024}
	downloadPart(baseUrl,p2)
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
