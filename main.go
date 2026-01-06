package main

import (
	"fmt"
	// "io"
	"net/http"
	// "os"
)

const baseUrl="https://www.w3.org/WAI/ER/tests/xhtml/testfiles/resources/pdf/dummy.pdf"

func main(){

	getFileInfo()

	fmt.Println("File info fetched successfully")

}

func getFileInfo(){
	res,err:=http.Head(baseUrl)
	if err!=nil{
		panic(err)
	}
	fmt.Println(res.ContentLength)

}