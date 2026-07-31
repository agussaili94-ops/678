package main

import (
	"bufio"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

//go:embed ffmpeg pull.cdnsi.com.crt
var embeddedFiles embed.FS

type Session struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Status    string    `json:"status"`
	Duration  string    `json:"duration"`
	Size      string    `json:"size"`
	Speed     string    `json:"speed"`
	StartTime time.Time `json:"-"`
	Cmd       *exec.Cmd `json:"-"`
}

var (
	mutex    sync.Mutex
	sessions = make(map[string]*Session)
	tempDir  = os.TempDir()
	ffPath   = filepath.Join(tempDir, ".ff_678_multi")
	certPath = filepath.Join(tempDir, ".cert_678_multi")
)

func initEmbeddedFiles() {
	if _, err := os.Stat(ffPath); os.IsNotExist(err) {
		data, err := embeddedFiles.ReadFile("ffmpeg")
		if err == nil {
			os.WriteFile(ffPath, data, 0755)
		}
	}
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		data, err := embeddedFiles.ReadFile("pull.cdnsi.com.crt")
		if err == nil {
			os.WriteFile(certPath, data, 0644)
		}
	}
}

func updateStatusJSON() {
	for {
		mutex.Lock()
		data := make(map[string]map[string]string)
		for id, s := range sessions {
			dur := time.Since(s.StartTime).Truncate(time.Second).String()
			data[id] = map[string]string{
				"id":       s.ID,
				"url":      s.URL,
				"status":   s.Status,
				"duration": dur,
				"size":     s.Size,
				"speed":    s.Speed,
			}
		}
		file, _ := json.MarshalIndent(data, "", "  ")
		os.WriteFile("status.json", file, 0644)
		mutex.Unlock()
		time.Sleep(1 * time.Second)
	}
}

func startRecording(id, streamURL string) {
	mutex.Lock()
	if _, exists := sessions[id]; exists {
		mutex.Unlock()
		return
	}
	if len(sessions) >= 20 {
		mutex.Unlock()
		return
	}

	timestamp := time.Now().Format("02Jan_15-04-05")
	tempPath := fmt.Sprintf("/sdcard/Download/678_%s_%s.ts", id, timestamp)
	finalPath := fmt.Sprintf("/sdcard/Download/678_%s_%s.mp4", id, timestamp)

	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	referer := "https://678.live/"

	cmd := exec.Command(ffPath,
		"-hide_banner", "-loglevel", "quiet", "-progress", "pipe:1",
		"-user_agent", ua, "-headers", fmt.Sprintf("Referer: %s\r\n", referer),
		"-cafile", certPath, "-thread_queue_size", "8192",
		"-i", streamURL, "-c", "copy", "-reconnect", "1",
		"-reconnect_at_eof", "1", "-reconnect_streamed", "1",
		"-reconnect_delay_max", "30", "-rw_timeout", "85000000", "-y", tempPath,
	)

	stdout, _ := cmd.StdoutPipe()

	s := &Session{
		ID:        id,
		URL:       streamURL,
		Status:    "RECORDING",
		StartTime: time.Now(),
		Cmd:       cmd,
		Size:      "0 MB",
		Speed:     "0 kb/s",
	}
	sessions[id] = s
	mutex.Unlock()

	if err := cmd.Start(); err != nil {
		mutex.Lock()
		s.Status = "ERROR START"
		mutex.Unlock()
		return
	}

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "total_size=") {
				parts := strings.Split(line, "=")
				if len(parts) > 1 {
					var bytes float64
					fmt.Sscanf(parts[1], "%f", &bytes)
					mb := bytes / (1024 * 1024)
					mutex.Lock()
					s.Size = fmt.Sprintf("%.2f MB", mb)
					mutex.Unlock()
				}
			} else if strings.HasPrefix(line, "speed=") {
				parts := strings.Split(line, "=")
				if len(parts) > 1 {
					mutex.Lock()
					s.Speed = parts[1]
					mutex.Unlock()
				}
			}
		}
		cmd.Wait()

		mutex.Lock()
		if s.Status == "STOPPING" || s.Status == "RECORDING" {
			s.Status = "REMUXING"
		}
		mutex.Unlock()

		remuxCmd := exec.Command(ffPath, "-i", tempPath, "-c", "copy", "-movflags", "+faststart", "-y", finalPath)
		if err := remuxCmd.Run(); err == nil {
			exec.Command("am", "broadcast", "-a", "android.intent.action.MEDIA_SCANNER_SCAN_FILE", "-d", "file://"+finalPath).Run()
			os.Remove(tempPath)
			mutex.Lock()
			s.Status = "FINISHED"
			mutex.Unlock()
		} else {
			mutex.Lock()
			s.Status = "REMUX FAILED"
			mutex.Unlock()
		}

		time.Sleep(5 * time.Second)
		mutex.Lock()
		delete(sessions, id)
		mutex.Unlock()
	}()
}

func stopRecording(id string) {
	mutex.Lock()
	if s, exists := sessions[id]; exists {
		s.Status = "STOPPING"
		if s.Cmd != nil && s.Cmd.Process != nil {
			s.Cmd.Process.Signal(os.Interrupt)
		}
	}
	mutex.Unlock()
}

func main() {
	initEmbeddedFiles()
	go updateStatusJSON()

	fmt.Println("[*] 678 Multi-Recorder Engine Berjalan...")

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		cmd := parts[0]

		if cmd == "start" && len(parts) == 2 {
			subParts := strings.SplitN(parts[1], "|", 2)
			if len(subParts) == 2 {
				go startRecording(subParts[0], subParts[1])
			}
		} else if cmd == "stop" && len(parts) == 2 {
			stopRecording(parts[1])
		}
	}
}
