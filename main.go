package main

import (
	"io"
	"net/http"
	"os"
)

const baseUrl="https://www.w3.org/WAI/ER/tests/xhtml/testfiles/resources/pdf/dummy.pdf"
func main() {
	req,err:=http.NewRequest("GET",baseUrl,nil)
	if err!=nil{
		panic(err)
	}
	req.Header.Set("Range","bytes=0-1023")

	client:=&http.Client{}
	res,err:=client.Do(req)
	if err!=nil{
		panic(err)
	}
	defer res.Body.Close()

	file,err:=os.Create("part1.pdf")
	if err!=nil{
		panic(err)
	}
	io.Copy(file,res.Body)

}