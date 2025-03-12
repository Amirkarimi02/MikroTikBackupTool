
# 📦 MikroTik Backup Tool

A powerful **Go-based** automation tool to backup and export MikroTik router configurations via **SSH** and **SFTP**. This tool connects to multiple MikroTik devices, performs backups, exports configuration files, and securely downloads them to a local directory.

## 🔍 Features

- **Automated Backup & Export**:
   - Creates both `.backup` and `.rsc` configuration files.
- **Multi-Device Support**:
   - Works with multiple MikroTik routers concurrently.
- **Retry Mechanism**:
   - Automatically retries failed attempts with customizable retry counts.
- **Secure File Transfer**:
   - Uses **SSH** for command execution and **SFTP** for file transfer.
- **File Management**:
   - Downloads backups locally and deletes them from the router after a successful transfer.
- **Detailed Logging**:
   - Displays clear logs for successful and failed operations.

## 📊 How It Works

1. Connects to MikroTik devices via **SSH** (port `2006` by default).
2. Executes:
   - `/system backup save` to create a binary backup (`.backup`).
   - `/export` to generate a readable script (`.rsc`).
3. Downloads the files via **SFTP** to the local directory.
4. Deletes the backups from the MikroTik device once successfully downloaded.

## ⚙️ Prerequisites

- **MikroTik Routers** with SSH (Port `22`) enabled.
- Go **v1.21+** installed on your machine.

## 🚀 Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/Amirkarimi02/MikroTikBackupTool.git
   cd MikroTikBackupTool
   ```
2. Build the tool:
   ```bash
   go build -o mikrotik-backup
   ```

## 🛠️ Usage

1. Update the following constants in `main.go`:

   - `username` – MikroTik SSH username
   - `password` – MikroTik SSH password
   - `backupDir` – Local path to store backups
   - `mikrotikDevices` – List of MikroTik IP addresses

2. Run the tool:
   ```bash
   ./mikrotik-backup
   ```

## 📂 Backup Output Structure

Backups are saved in the following format:

```
Backup_<IP>_<SystemIdentity>_<Timestamp>.backup
Export_<IP>_<SystemIdentity>_<Timestamp>.rsc
```

Example:
```
Backup_192.168.1.1_RouterOffice_20240312_120045.backup
Export_192.168.1.1_RouterOffice_20240312_120045.rsc
```

## 📝 Custom Configuration

- **Max Retries**: Adjust `maxRetries` for the retry count.
- **Retry Delay**: Modify `retryDelay` to change the wait time between retries.

## 📌 To-Do

- Add multi-threaded processing for faster backups.
- Implement email notifications for failures.
- Enhance error handling and logging.

---

⭐ **Contributions are welcome! Feel free to fork and submit pull requests.**
