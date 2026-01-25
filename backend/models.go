package main

type Task struct{
	ID string `json:"id"`
	FileName string `json:"fileName"`
	Url string `json:"url"`
	Status string `json:"status"`
	TotalSize int64 `json:"totalSize"`
	Downloaded int64 `json:"downloaded"`
}

