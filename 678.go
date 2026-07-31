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
	fmt.Println("       678 RECORDER - ALL-IN-ONE V10      ")
	fmt.Println("==========================================")

	// --- LOGIKA EKSTRAKSI TERSEMBUNYI DENGAN VALIDASI ---
	tempDir := os.TempDir()
	ffmpegPath := filepath.Join(tempDir, ".ff_678")
	certPath := filepath.Join(tempDir, ".cert_678")

	// Ekstrak ffmpeg hanya jika belum ada di temp
	if _, err := os.Stat(ffmpegPath); os.IsNotExist(err) {
		ffData, errRead := embeddedFiles.ReadFile("ffmpeg")
		if errRead != nil {
			fmt.Println("[!] FATAL ERROR: Gagal membaca file internal 'ffmpeg'.")
			fmt.Println("[!] Pastikan file 'ffmpeg' ada di folder saat proses 'go build'.")
			return
		}
		os.WriteFile(ffmpegPath, ffData, 0755)
	}

	// Ekstrak sertifikat dengan validasi
	certData, errCert := embeddedFiles.ReadFile("pull.cdnsi.com.crt")
	if errCert != nil {
		fmt.Println("[!] FATAL ERROR: Gagal membaca sertifikat 'pull.cdnsi.com.crt'.")
		fmt.Println("[!] Proses dihentikan karena koneksi HTTPS (SSL) pasti akan ditolak oleh server.")
		return
	}
	os.WriteFile(certPath, certData, 0644)

	// Hapus file temp saat program benar-benar berhenti
	defer os.Remove(ffmpegPath)
	defer os.Remove(certPath)

	fmt.Print("\n[+] Tempel link 678 terbaru: ")
	streamURL, _ := reader.ReadString('\n')
	streamURL = strings.TrimSpace(streamURL)
	if streamURL == "" {
		return
	}

	timestamp := time.Now().Format("02Jan_15-04-05")
	tempPath := fmt.Sprintf("/sdcard/Download/678_Live_%s.ts", timestamp)
	finalPath := fmt.Sprintf("/sdcard/Download/678_Live_%s.mp4", timestamp)

	// Menggunakan User-Agent standar agar tidak mudah diblokir server
	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	referer := "https://678.live/"

	cmd := exec.Command(ffmpegPath,
		"-hide_banner",
		"-loglevel", "quiet",
		"-progress", "pipe:1",
		"-user_agent", ua,
		"-headers", fmt.Sprintf("Referer: %s\r\n", referer),
		"-cafile", certPath, // Menggunakan sertifikat yang diekstrak tadi
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
	
	// 1. Cek apakah file TS ada dan berisi data
	tsInfo, errStat := os.Stat(tempPath)
	if errStat != nil || tsInfo.Size() == 0 {
		fmt.Println("\n[!] Gagal: File rekaman (.ts) kosong atau tidak ditemukan.")
		fmt.Println("[!] Pastikan link stream valid dan jaringan lancar.")
		os.Remove(tempPath) // Bersihkan jika ada file 0 byte
		return
	}

	fmt.Println("\n[+] Remuxing ke MP4 (File besar butuh waktu, mohon tunggu)...")
	remuxCmd := exec.Command(ffmpegPath, "-i", tempPath, "-c", "copy", "-movflags", "+faststart", "-y", finalPath)
	
	// Tangkap output jika terjadi error saat remuxing
	out, errRemux := remuxCmd.CombinedOutput()
	if errRemux != nil {
		fmt.Printf("[!] Error saat remuxing: %v\n", errRemux)
		fmt.Printf("Detail:\n%s\n", string(out))
		return
	}

	// 2. Cek apakah file MP4 sukses terbuat
	mp4Info, errMp4 := os.Stat(finalPath)
	if errMp4 == nil && mp4Info.Size() > 0 {
		fmt.Printf("[✓] SELESAI DIBUAT: %s\n", finalPath)
		
		// 3. Pemicu Media Scanner agar file MP4 langsung terbaca di File Manager HP
		exec.Command("am", "broadcast", "-a", "android.intent.action.MEDIA_SCANNER_SCAN_FILE", "-d", "file://"+finalPath).Run()
		
		// 4. Hapus file sampah .ts (dengan fallback perintah 'rm' jika diblokir sistem Android)
		if err := os.Remove(tempPath); err != nil {
			errRm := exec.Command("rm", "-f", tempPath).Run()
			if errRm != nil {
				fmt.Printf("[!] File MP4 sukses, tapi sistem memblokir penghapusan otomatis file .ts.\n")
				fmt.Printf("[!] Silakan hapus manual: %s\n", tempPath)
			}
		}
	} else {
		fmt.Println("\n[!] Remux selesai, tapi file MP4 kosong atau tidak ditemukan.")
		fmt.Println("[!] Cek sisa memori internal HP Anda. Pastikan ruang kosong mencukupi.")
	}
}
