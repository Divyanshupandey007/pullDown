package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"math"
	"mime"
	"net"
	"net/http"
	neturl "net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kkdai/youtube/v2"
)

// Part struct for handling chunks
type Part struct {
	Index int
	Start int64
	End   int64
}

func (dm *DownloadManager) processDownload(ctx context.Context, taskId string, downloadUrl string, customName ...string) {
	dm.semaphore <- struct{}{}
	defer func() {
		<-dm.semaphore
	}()
	log.Println("Starting/Resuming download for: ", downloadUrl)

	res, err := http.Head(downloadUrl)
	if err != nil {
		log.Println("Error fetching HEAD: ", err)
		dm.setTaskError(taskId, "Connection failed")
		SendError(taskId, "Connection failed")
		return
	}
	defer res.Body.Close()

	supportsRange := strings.EqualFold(res.Header.Get("Accept-Ranges"), "bytes")
	numParts := dm.config.PartsPerFile
	if !supportsRange || res.ContentLength <= 0 {
		numParts = 1
		log.Println("Server does not support range requests, using single stream")
	}

	dm.dataMutex.Lock()
	for i := range dm.Tasks {
		if dm.Tasks[i].ID == taskId {
			dm.Tasks[i].TotalSize = res.ContentLength
			break
		}
	}
	dm.dataMutex.Unlock()
	dm.SaveTasks()

	var fileName string
	if len(customName) > 0 {
		fileName = customName[0]
	} else {
		if cd := res.Header.Get("Content-Disposition"); cd != "" {
			_, params, err := mime.ParseMediaType(cd)
			if err == nil {
				fileName = params["filename"]
			}
		}

		if fileName == "" {
			fileName = path.Base(downloadUrl)
		}

		if fileName == "" || !strings.Contains(fileName, ".") {
			fileName = "download_file.bin"
		}
	}

	parts := calculateParts(res.ContentLength, numParts)

	var initialDownloaded int64 = 0

	for i := range parts {
		tmpFileName := fmt.Sprintf("%s_part_%d.tmp", taskHash(taskId), i)
		if stat, err := os.Stat(tmpFileName); err == nil {
			initialDownloaded += stat.Size()
		}
	}

	//Create shared counter
	var downloadBytes = initialDownloaded

	if res.ContentLength > 0 {
		percent := float64(initialDownloaded) / float64(res.ContentLength) * 100
		SendProgress(taskId, fileName, percent, res.ContentLength, 0, 0)
	}

	//For implementing concurrency
	var wg sync.WaitGroup

	errChan := make(chan error, len(parts))

	for i := range parts {
		//Increment the goroutine counter
		wg.Add(1)

		go func(p Part) {
			defer wg.Done()
			maxRetries := 5
			if !dm.settings.AutoRetry {
				maxRetries = 1
			}
			for attempt := 0; attempt < maxRetries; attempt++ {
				if ctx.Err() != nil {
					return
				}
				err := downloadPart(ctx, taskId, downloadUrl, fileName, p, &downloadBytes, res.ContentLength, dm.limiter, dm.settings.ProxyHost, dm.settings.ProxyPort, dm.settings.ConnTimeout, dm.settings.EnableProxy)
				if err == nil {
					return
				}

				if ctx.Err() != nil {
					return
				}
				log.Printf("Part %d failed (Attempt %d %d): %v. Retrying in 2 sec...", p.Index, attempt+1, maxRetries, err)
				time.Sleep(2 * time.Second)
			}
			errChan <- fmt.Errorf("Part %d failed after %d attempts", p.Index, maxRetries)
		}(parts[i])
	}
	wg.Wait()
	close(errChan)

	if len(errChan) > 0 {
		log.Println("Download failed due to network error")
		dm.setTaskError(taskId, "Download failed after retries")
		SendError(taskId, "Download failed after retries")
		return
	}

	if ctx.Err() == nil {
		mergeParts(fileName, len(parts), taskId, dm.config.DownloadDir)
		SendProgress(taskId, fileName, 100.0, res.ContentLength, 0, 0)

		dm.dataMutex.Lock()
		for i := range dm.Tasks {
			if dm.Tasks[i].ID == taskId {
				dm.Tasks[i].Status = "Completed"
				dm.Tasks[i].FileName = fileName
				dm.Tasks[i].Downloaded = res.ContentLength
				dm.Tasks[i].TotalSize = res.ContentLength
				break
			}
		}
		dm.dataMutex.Unlock()
		dm.SaveTasks()
		log.Println("Download Complete")
	} else {
		currBytes := atomic.LoadInt64(&downloadBytes)
		dm.dataMutex.Lock()
		for i := range dm.Tasks {
			if dm.Tasks[i].ID == taskId {
				dm.Tasks[i].Status = "Paused"
				dm.Tasks[i].Downloaded = currBytes
				break
			}
		}
		dm.dataMutex.Unlock()
		dm.SaveTasks()
		log.Println("Download Paused")
	}
}

func taskHash(taskId string) string {
	h := md5.Sum([]byte(taskId))
	return hex.EncodeToString(h[:])[:8]
}

func (dm *DownloadManager) downloadYoutube(ctx context.Context, originalUrl string) {
	log.Println("Analyzing youtube video:", originalUrl)

	client := youtube.Client{}
	video, err := client.GetVideo(originalUrl)
	if err != nil {
		log.Println("Error getting video info:", err)
		dm.setTaskError(originalUrl, "Youtube: "+err.Error())
		SendError(originalUrl, "Failed to analyze video")
		return
	}
	formats := video.Formats.WithAudioChannels()

	var bestFormat *youtube.Format
	for i := range formats {
		if formats[i].QualityLabel != "" {
			bestFormat = &formats[i]
			break
		}
	}

	if bestFormat == nil {
		if len(formats) == 0 {
			log.Println("Error: No formats found for", originalUrl)
			dm.setTaskError(originalUrl, "No downloadable formats found")
			SendError(originalUrl, "No downloadable formats found")
			return
		}
		bestFormat = &formats[0]
	}
	log.Printf("Found format: %s, Quality: %s\n", bestFormat.MimeType, bestFormat.QualityLabel)
	//Get direct URL
	streamURL, err := client.GetStreamURL(video, bestFormat)
	if err != nil {
		log.Println("Error getting stream URL:", err)
		dm.setTaskError(originalUrl, "Failed to get stream URL")
		SendError(originalUrl, "Failed to get stream URL")
		return
	}

	log.Println("Got direct stream URL....")
	safeTitle := sanitizeFileName(video.Title) + ".mp4"

	dm.dataMutex.Lock()
	for i := range dm.Tasks {
		if dm.Tasks[i].ID == originalUrl {
			dm.Tasks[i].FileName = safeTitle
			break
		}
	}
	dm.dataMutex.Unlock()
	dm.SaveTasks()

	dm.processDownload(ctx, originalUrl, streamURL, safeTitle)

}

func sanitizeFileName(name string) string {
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	for _, chars := range invalidChars {
		name = strings.ReplaceAll(name, chars, "_")
	}
	return name
}

// Logic for calculating size of each part
func calculateParts(totalSize int64, numParts int) []Part {
	if totalSize <= 0 || totalSize < int64(numParts) {
		return []Part{{
			Index: 0,
			Start: 0,
			End:   -1,
		},
		}
	}

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

func downloadPart(ctx context.Context, taskId string, downloadUrl string, fileName string, part Part, progress *int64, totalSize int64, limiter *BandwidthMonitor, proxyHost string, proxyPort int, connTimeout int, enableProxy bool) error {

	tmpFileName := fmt.Sprintf("%s_part_%d.tmp", taskHash(taskId), part.Index)
	var currStart = part.Start

	file, err := os.OpenFile(tmpFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("File open error: %v", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err == nil {
		currStart += stat.Size()
	}

	expectedBytesRemaining := (part.End - currStart) + 1

	if expectedBytesRemaining <= 0 {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET", downloadUrl, nil)
	if err != nil {
		return err
	}

	//Used Sprintf to return formatted string
	if part.End >= 0 {
		rangeHeader := fmt.Sprintf("bytes=%d-%d", currStart, part.End)
		req.Header.Set("Range", rangeHeader)
	}

	dialer := &net.Dialer{
		Timeout: time.Duration(connTimeout) * time.Second,
	}
	transport := &http.Transport{
		DialContext: dialer.DialContext,
	}
	if enableProxy && proxyHost != "" {
		proxyAddr, _ := neturl.Parse(fmt.Sprintf("http://%s:%d", proxyHost, proxyPort))
		transport.Proxy = http.ProxyURL(proxyAddr)
	}

	client := &http.Client{
		Transport: transport,
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}

	//Used defer so that the res object is closed after its functioning
	defer res.Body.Close()

	if res.StatusCode != 200 && res.StatusCode != 206 {
		return fmt.Errorf("Bad status code: %d", res.StatusCode)
	}

	//Create buffer
	buf := make([]byte, 32*1024)
	var bytesDownloadedThisSession int64 = 0
	lastSent := time.Now()
	lastBytes := atomic.LoadInt64(progress)

	for {
		//Read data
		if ctx.Err() != nil {
			return nil
		}
		n, err := res.Body.Read(buf)

		//Write data to file upto 'n' bytes
		if n > 0 {
			if _, writeErr := file.Write(buf[:n]); writeErr != nil {
				return fmt.Errorf("disk write error: %w", writeErr)
			}

			limiter.AddBytes(n)

			bytesDownloadedThisSession += int64(n)
			//Safely add 'n' bytes to shared counter
			current := atomic.AddInt64(progress, int64(n))
			percent := float64(current) / float64(totalSize) * 100
			if time.Since(lastSent) > 500*time.Millisecond {
				elapsed := time.Since(lastSent).Seconds()
				currBytes := atomic.LoadInt64(progress)
				speed := float64(currBytes-lastBytes) / elapsed
				var eta float64
				if speed > 0 {
					eta = float64(totalSize-currBytes) / speed
				}
				SendProgress(taskId, fileName, math.Min(percent, 100.0), totalSize, speed, eta)
				lastSent = time.Now()
				lastBytes = currBytes
			}
			limiter.Wait(n)
		}
		if err != nil {
			if err == io.EOF {
				if bytesDownloadedThisSession < expectedBytesRemaining {
					return fmt.Errorf("Server hung up early. Got %d bytes, expected %d", bytesDownloadedThisSession, expectedBytesRemaining)
				}
				break
			}
			return err
		}
	}
	return nil
}

func mergeParts(fileName string, numParts int, taskId string, downloadDir string) {

	outputPath := filepath.Join(downloadDir, fileName)
	os.MkdirAll(downloadDir, 0755)

	if _, err := os.Stat(outputPath); err == nil {
		ext := filepath.Ext(fileName)
		base := strings.TrimSuffix(fileName, ext)
		for i := 1; ; i++ {
			candidate := filepath.Join(downloadDir, fmt.Sprintf("%s(%d)%s", base, i, ext))
			if _, err := os.Stat(candidate); os.IsNotExist(err) {
				outputPath = candidate
				break
			}
		}
	}
	//Create the final file
	outFile, err := os.Create(outputPath)
	if err != nil {
		log.Println("Error creating final file: ", err)
		return
	}
	defer outFile.Close()

	for i := 0; i < numParts; i++ {
		partFileName := fmt.Sprintf("%s_part_%d.tmp", taskHash(taskId), i)

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
	log.Println("Files merged into:", fileName)
}
