package main

type Task struct {
	ID         string `json:"id"`
	FileName   string `json:"fileName"`
	Url        string `json:"url"`
	Status     string `json:"status"`
	TotalSize  int64  `json:"totalSize"`
	Downloaded int64  `json:"downloaded"`
}

type Settings struct {
	DownloadPath    string `json:"downloadPath"`
	MaxDownloads    int    `json:"maxDownloads"`
	MaxConnections  int    `json:"maxConnections"`
	ConnTimeout     int    `json:"connTimeout"`
	ProxyHost       string `json:"proxyHost"`
	ProxyPort       int    `json:"proxyPort"`
	AutoStart       bool   `json:"autoStart"`
	CompletionAlert bool   `json:"completionAlert"`
	EnableProxy     bool   `json:"enableProxy"`
	ForceHttps      bool   `json:"forceHttps"`
	AutoExtract     bool   `json:"autoExtract"`
	AutoRetry       bool   `json:"autoRetry"`
	Scheduler       bool   `json:"scheduler"`
	NotifComplete   bool   `json:"notifComplete"`
	NotifError      bool   `json:"notifError"`
	SoundEffects    bool   `json:"soundEffects"`
}
