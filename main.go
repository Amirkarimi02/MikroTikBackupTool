package main

import (
	"fmt"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MikroTik device list
var mikrotikDevices = []string{
	"192.168.5.47",
	"192.168.0.1",
	"192.168.1.112",
	"192.168.2.204",
	"192.168.7.200",
	"192.168.2.202",
	"192.168.1.251",
	"192.168.3.188",
	"192.168.1.131",
	"192.168.2.203",
	"192.168.2.15",
	"192.168.1.106",
	"192.168.2.197",
	"192.168.2.233",
	"192.168.1.164",
	"192.168.2.166",
	"192.168.2.137",
	"192.168.3.39",
	"192.168.1.154",
	"192.168.2.91",
	"192.168.1.153",
	"192.168.1.146",
	"192.168.2.9",
	"192.168.1.139",
	"192.168.1.144",
	"192.168.3.160",
	"192.168.3.174",
	"192.168.1.49",
	"192.168.1.245",
	"192.168.1.160",
	"192.168.1.148",
	"192.168.2.220",
	"192.168.1.143",
	"192.168.110.1",
	"192.168.100.1",
}

const (
	username   = "admin"                                        // MikroTik username
	password   = "ParlarNetwork@1402"                           // MikroTik password
	backupDir  = "H:\\Karimi\\My Mikrotik's\\Backups\\Automate" // Directory to save backups
	maxRetries = 5                                              // Maximum retry count
	retryDelay = 5 * time.Second                                // Delay between retries
)

func main() {
	// Ensure backup directory exists
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		os.Mkdir(backupDir, os.ModePerm)
	}

	// Process each MikroTik device
	for _, ip := range mikrotikDevices {
		if err := processMikroTik(ip); err != nil {
			fmt.Printf("❌ Error processing %s: %v\n", ip, err)
			fmt.Println("==========================================================")
		} else {
			fmt.Printf("✅ Backup completed successfully for %s\n", ip)
			fmt.Println("==========================================================")
		}
	}
}

// Retry function to attempt an action multiple times
func retry(action func() error) error {
	var err error
	for attempts := 1; attempts <= maxRetries; attempts++ {
		err = action()
		if err == nil {
			return nil
		}
		fmt.Printf("🔄 Attempt %d/%d failed: %v. Retrying...\n", attempts, maxRetries, err)
		time.Sleep(retryDelay)
	}
	return fmt.Errorf("after %d attempts, last error: %v", maxRetries, err)
}

// Process a MikroTik device (backup and export)
func processMikroTik(ip string) error {
	fmt.Println("🔹 Connecting to MikroTik:", ip)

	// Establish SSH connection
	client, err := connectSSH(ip)
	if err != nil {
		return fmt.Errorf("SSH connection failed: %v", err)
	}
	defer client.Close()

	fmt.Printf("✅ Connection established: %s - ", ip)

	// Get the system identity (device name)
	systemIdentity, err := getSystemIdentity(client)
	if err != nil {
		return fmt.Errorf("failed to fetch system identity: %v", err)
	}

	fmt.Printf("%s\n", systemIdentity)

	// Create unique filenames based on system identity, IP, and timestamp
	timestamp := time.Now().Format("20060102_150405")
	backupFile := fmt.Sprintf("Backup_%s_%s_%s.backup", ip, systemIdentity, timestamp)
	exportFile := fmt.Sprintf("Export_%s_%s_%s.rsc", ip, systemIdentity, timestamp)

	// Attempt to create the backup (.backup)
	var backupCreated bool
	if err := retry(func() error {
		return executeCommand(client, fmt.Sprintf("/system backup save name=%s", backupFile))
	}); err != nil {
		fmt.Printf("❌ Failed to create .backup for %s: %v\n", ip, err)
	} else {
		fmt.Printf("📄 Backup created: %s\n", backupFile)
		backupCreated = true
	}

	// Attempt to create the export (.rsc)
	var exportCreated bool
	if err := retry(func() error {
		return executeCommand(client, fmt.Sprintf("/export file=%s", exportFile))
	}); err != nil {
		fmt.Printf("❌ Failed to create .rsc export for %s: %v\n", ip, err)
	} else {
		fmt.Printf("📄 Export created: %s\n", exportFile)
		exportCreated = true
	}

	// Allow MikroTik to register the files
	time.Sleep(2 * time.Second)

	// Download available files
	if backupCreated {
		if err := handleFile(client, ip, backupFile); err != nil {
			fmt.Printf("❌ Error handling .backup for %s: %v\n", ip, err)
		}
	}
	if exportCreated {
		if err := handleFile(client, ip, exportFile); err != nil {
			fmt.Printf("❌ Error handling .rsc for %s: %v\n", ip, err)
		}
	}

	// If nothing was created, return an error
	if !backupCreated && !exportCreated {
		return fmt.Errorf("both .backup and .rsc export failed")
	}

	return nil
}

// Handle the file: locate, download, and delete
func handleFile(client *ssh.Client, ip, filename string) error {
	remotePath, err := findBackupFile(client, filename)
	if err != nil {
		return fmt.Errorf("failed to locate file: %s, error: %v", filename, err)
	}

	fmt.Printf("📥 Downloading file from MikroTik (%s): %s\n", ip, filename)

	if err := retry(func() error {
		return downloadBackupSFTP(ip, remotePath, filename)
	}); err != nil {
		return fmt.Errorf("failed to download file: %s, error: %v", filename, err)
	}

	fmt.Printf("✅ Successfully downloaded: %s\n", filename)

	if err := retry(func() error {
		return executeCommand(client, fmt.Sprintf("/file remove \"%s\"", filename))
	}); err != nil {
		return fmt.Errorf("failed to delete file: %s, error: %v", filename, err)
	}

	fmt.Printf("🗑️ Deleted file from MikroTik: %s\n", filename)
	return nil
}

// Retrieve MikroTik system identity (device name)
func getSystemIdentity(client *ssh.Client) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput("/system identity print")
	if err != nil {
		return "", fmt.Errorf("failed to get system identity: %v", err)
	}

	for _, line := range strings.Split(string(output), "\n") {
		if strings.Contains(line, "name:") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1]), nil
			}
		}
	}
	return "", fmt.Errorf("system identity not found")
}

// Establish SSH connection
func connectSSH(ip string) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}
	return ssh.Dial("tcp", ip+":2006", config)
}

// Execute MikroTik command
func executeCommand(client *ssh.Client, command string) error {
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()
	return session.Run(command)
}

// Locate backup file on MikroTik
func findBackupFile(client *ssh.Client, backupFile string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput("/file print detail")
	if err != nil {
		return "", fmt.Errorf("failed to execute file print: %v", err)
	}

	for _, line := range strings.Split(string(output), "\n") {
		if strings.Contains(line, backupFile) {
			if strings.Contains(line, "flash/") {
				return "/flash/" + backupFile, nil
			}
			return "/" + backupFile, nil
		}
	}
	return "", fmt.Errorf("file not found: %s", backupFile)
}

// Download the backup using SFTP
func downloadBackupSFTP(ip, remoteFile, localFileName string) error {
	sshClient, err := connectSSH(ip)
	if err != nil {
		return fmt.Errorf("failed to connect via SSH: %v", err)
	}
	defer sshClient.Close()

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		return fmt.Errorf("failed to create SFTP client: %v", err)
	}
	defer sftpClient.Close()

	localPath := filepath.Join(backupDir, localFileName)
	dstFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %v", err)
	}
	defer dstFile.Close()

	srcFile, err := sftpClient.Open(remoteFile)
	if err != nil {
		return fmt.Errorf("failed to open remote file: %v", err)
	}
	defer srcFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}
