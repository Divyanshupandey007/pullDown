package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

func main(){
	res,err:=http.Get("https://www.w3.org/WAI/ER/tests/xhtml/testfiles/resources/pdf/dummy.pdf")
	
	if err!=nil{
		panic(err)
	}

	defer res.Body.Close()

	file,err:=os.Create("dummy.pdf")

	if err!=nil{
		panic(err)
	}

	defer file.Close()

	io.Copy(file,res.Body)

	fmt.Println("Download Finished")

}