package main

import (
	"bufio"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

//go:embed ffmpeg pull.cdnsi.com.crt
var embeddedFiles embed.FS

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("==========================================")
	fmt.Println("       678 RECORDER - ALL-IN-ONE V11      ")
	fmt.Println("==========================================")

	// --- LOGIKA EKSTRAKSI CACHE PERMANEN (SUPER AMAN) ---
	// Kita simpan di folder Home pengguna secara permanen agar tidak hilang oleh sistem
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = os.Getenv("HOME")
	}
	cacheDir := filepath.Join(homeDir, ".678_cache")
	os.MkdirAll(cacheDir, 0755) // Buat folder jika belum ada

	ffmpegPath := filepath.Join(cacheDir, "ffmpeg")
	certPath := filepath.Join(cacheDir, "pull.cdnsi.com.crt")

	// Ekstrak ffmpeg HANYA jika belum ada atau file rusak (kurang dari 1MB)
	ffStat, errFF := os.Stat(ffmpegPath)
	if os.IsNotExist(errFF) || ffStat.Size() < 1000000 {
		ffData, errRead := embeddedFiles.ReadFile("ffmpeg")
		if errRead != nil {
			fmt.Println("[!] FATAL ERROR: Gagal membaca file internal 'ffmpeg'.")
			return
		}
		os.WriteFile(ffmpegPath, ffData, 0755)
	}

	// Ekstrak sertifikat SSL
	certStat, errCert := os.Stat(certPath)
	if os.IsNotExist(errCert) || certStat.Size() == 0 {
		certData, errRead := embeddedFiles.ReadFile("pull.cdnsi.com.crt")
		if errRead != nil {
			fmt.Println("[!] FATAL ERROR: Gagal membaca sertifikat 'pull.cdnsi.com.crt'.")
			return
		}
		os.WriteFile(certPath, certData, 0644)
	}

	// KODE "defer os.Remove" SUDAH KITA HAPUS TOTAL DI SINI AGAR TIDAK HILANG SAAT REMUX

	fmt.Print("\n[+] Tempel link 678 terbaru: ")
	streamURL, _ := reader.ReadString('\n')
	streamURL = strings.TrimSpace(streamURL)
	if streamURL == "" {
		return
	}

	timestamp := time.Now().Format("02Jan_15-04-05")
	tempPath := fmt.Sprintf("/sdcard/Download/678_Live_%s.ts", timestamp)
	finalPath := fmt.Sprintf("/sdcard/Download/678_Live_%s.mp4", timestamp)

	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	referer := "https://678.live/"

	cmd := exec.Command(ffmpegPath,
		"-hide_banner",
		"-loglevel", "quiet",
		"-progress", "pipe:1",
		"-user_agent", ua,
		"-headers", fmt.Sprintf("Referer: %s\r\n", referer),
		"-cafile", certPath,
		"-thread_queue_size", "8192",
		"-i", streamURL,
		"-c", "copy",
		"-reconnect", "1",
		"-reconnect_at_eof", "1",
		"-reconnect_streamed", "1",
		"-reconnect_delay_max", "30",
		"-rw_timeout", "85000000",
		"-y", tempPath)

	stdout, _ := cmd.StdoutPipe()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	if err := cmd.Start(); err != nil {
		fmt.Printf("[!] Error saat menjalankan FFmpeg: %v\n", err)
		return
	}

	fmt.Printf("\n[✓] Merekam ke: %s\n", tempPath)
	fmt.Println("[!] Tekan Ctrl+C untuk save ke MP4")

	done := make(chan error, 1)

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "total_size=") {
				size := strings.Split(line, "=")[1]
				fmt.Printf("\033[2K\r[+] Recording... %s byte", size)
			}
		}
		done <- cmd.Wait()
	}()

	select {
	case <-sigChan:
		fmt.Println("\n\n[!] Menutup rekaman...")
		cmd.Process.Signal(os.Interrupt)
		<-done
	case <-done:
	}

	// --- VALIDASI DAN LOGIKA REMUXING ---

	tsInfo, errStat := os.Stat(tempPath)
	if errStat != nil || tsInfo.Size() == 0 {
		fmt.Println("\n[!] Gagal: File rekaman (.ts) kosong atau tidak ditemukan.")
		os.Remove(tempPath)
		return
	}

	fmt.Println("\n[+] Remuxing ke MP4 (File besar butuh waktu, mohon tunggu)...")
	
	// Remuxing menggunakan ffmpegPath yang sekarang ada di cache permanen
	remuxCmd := exec.Command(ffmpegPath, "-i", tempPath, "-c", "copy", "-movflags", "+faststart", "-y", finalPath)
	out, errRemux := remuxCmd.CombinedOutput()
	
	if errRemux != nil {
		fmt.Printf("[!] Error saat remuxing: %v\n", errRemux)
		fmt.Printf("Detail:\n%s\n", string(out))
		return
	}

	mp4Info, errMp4 := os.Stat(finalPath)
	if errMp4 == nil && mp4Info.Size() > 0 {
		fmt.Printf("[✓] SELESAI DIBUAT: %s\n", finalPath)

		// Media Scanner (Aman dari SIGSYS)
		exec.Command("/system/bin/am", "broadcast", "-a", "android.intent.action.MEDIA_SCANNER_SCAN_FILE", "-d", "file://"+finalPath).Run()

		// Hapus sampah .ts murni via Go
		if err := os.Remove(tempPath); err != nil {
			fmt.Printf("[!] Peringatan: Gagal menghapus file sampah .ts otomatis.\n")
		}
	} else {
		fmt.Println("\n[!] Remux selesai, tapi file MP4 kosong.")
	}
}
