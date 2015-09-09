package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/jhoonb/archivex"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

//Versions struct
type Versions struct {
	Latest      Latest    `json:"latest"`
	VersionList []Version `json:"versions"`
}

//Latest struct
type Latest struct {
	Snapshot string `json:"snapshot"`
	Release  string `json:"release"`
}

//Version struct
type Version struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Time        time.Time `json:"time"`
	ReleaseTime time.Time `json:"releaseTime"`
}

var updateInterval int
var debug bool
var releaseOnly bool
var serverPath string
var ramAlloc int
var worldDir string
var backupDir string
var downloadURL string

var serverCmd *exec.Cmd

func main() {
	flag.IntVar(&updateInterval, "updateInterval", 4, "Integer value in hours to check for updates (default: 4)")
	flag.BoolVar(&debug, "debug", false, "Show debug logs")
	flag.BoolVar(&releaseOnly, "releaseOnly", false, "Only download releases")
	flag.StringVar(&serverPath, "serverPath", ".", "Default path is .")
	flag.IntVar(&ramAlloc, "ramAlloc", 2048, "Integer value in Mb (default: 2048)")
	flag.StringVar(&worldDir, "worldDir", "./world", "Default path is ./world")
	flag.StringVar(&backupDir, "backupDir", "./backups", "Default path is ./backups")
	flag.StringVar(&downloadURL, "downloadURL", "https://s3.amazonaws.com/Minecraft.Download/versions", "Download server Url")

	flag.Parse()

	var wg sync.WaitGroup

	for {
		versionData := getVersions()
		newestVersion := versionData.Newest()

		if checkUpdate(newestVersion) {
			if debug {
				fmt.Printf("Updating to %s\n", newestVersion.ID)
			}

			wg.Add(1)
			go downloadNewVersion(newestVersion.ID, &wg)

			wg.Add(1)
			go backupFiles(newestVersion.ID, &wg)

			wg.Wait()
			updateServer(currentServerVersion(), newestVersion.ID)
		}
		time.Sleep(time.Duration(updateInterval) * time.Hour)
	}
}

func perror(err error) {
	if err != nil {
		panic(err)
	}
}

func getVersions() Versions {
	versionsURL := fmt.Sprintf("%s/versions.json", downloadURL)
	res, err := http.Get(versionsURL)
	defer res.Body.Close()
	perror(err)
	body, err := ioutil.ReadAll(res.Body)
	perror(err)
	var versionData Versions
	err = json.Unmarshal(body, &versionData)
	perror(err)

	return versionData
}

func checkUpdate(versionData Version) bool {
	shouldUpdate := false

	if versionData.ID != currentServerVersion() {
		shouldUpdate = true
	}

	return shouldUpdate
}

func currentServerVersion() string {
	fileBuffer, err := ioutil.ReadFile("current_version")
	if err != nil {
		fmt.Println("current_version doesn't exist.")
	}
	currentVersionID := string(fileBuffer)

	if debug {
		fmt.Printf("Current Version %s\n", currentVersionID)
	}

	return currentVersionID
}

func writeCurrentVersion(version string) {
	err := ioutil.WriteFile("current_version", []byte(version), 0644)
	perror(err)
}

func makeBackupDir() {
	err := os.Mkdir(backupDir, 0777)
	perror(err)
}

func backupFiles(versionID string, wg *sync.WaitGroup) {
	defer wg.Done()

	if _, err := os.Stat(backupDir); err != nil {
		if os.IsNotExist(err) {
			makeBackupDir()
		}
	}

	if debug {
		fmt.Printf("backupFiles - %s\n", versionID)
	}

	fileName := fmt.Sprintf("%s%s%s_backup", backupDir, string(os.PathSeparator), versionID)
	worldDir := fmt.Sprintf("%s", worldDir)

	if _, err := os.Stat(backupDir); err != nil {
		if os.IsNotExist(err) {
			if debug {
				fmt.Println("Nothing to backup right now.")
			}
		} else {
			zip := new(archivex.ZipFile)
			zip.Create(fileName)
			fmt.Printf("worldDir - %s\n", worldDir)
			fmt.Printf("fileName - %s\n", fileName)
			zip.AddAll(worldDir, true)
			fmt.Printf("after addall\n")
			zip.Close()
		}
	}
}

func updateServer(oldVersion string, newVersion string) {
	writeCurrentVersion(newVersion)
	stopServer(oldVersion)
	startServer(newVersion)
}

func writeServerPid(pid int) {
	pidString := strconv.Itoa(pid)
	err := ioutil.WriteFile(".pid", []byte(pidString), 0644)
	perror(err)
}

func readServerPid() (int, error) {
	fileBuffer, err := ioutil.ReadFile(".pid")
	if err != nil {
		fmt.Println("current_version doesn't exist.")
		return 0, err
	}
	pidString := string(fileBuffer)
	pid, err := strconv.Atoi(strings.Replace(pidString, "\n", "", -1))
	perror(err)

	return pid, nil
}

func startServer(version string) {
	fileName := fmt.Sprintf("minecraft_server.%s.jar", version)
	command := fmt.Sprintf("java -Xmx%dM -Xms%dM -jar %s nogui", ramAlloc, ramAlloc, fileName)

	fmt.Printf("running %s\n", command)
	serverCmd = exec.Command("java", "-Xmx2048M", "-Xms2048M", "-jar", fileName, "nogui")
	serverCmd.SysProcAttr = &syscall.SysProcAttr{}
	serverCmd.SysProcAttr.Setpgid = true
	// writeServerPid(serverCmd.Process.Pid)
	err := serverCmd.Start()
	perror(err)
	fmt.Printf("Server running...\n")
	go watchServer(serverCmd)
}

func watchServer(cmd *exec.Cmd) {
	fmt.Printf("Watching command %d...\n", cmd.Process.Pid)
	err := cmd.Wait()
	fmt.Printf("Command finished with error: %s\n", err)
}

func stopServer(version string) {
	if serverCmd == nil {
		return
	}

	err := serverCmd.Process.Kill()
	perror(err)
}

func downloadNewVersion(versionID string, wg *sync.WaitGroup) {
	defer wg.Done()

	if debug {
		fmt.Printf("downloading - %s\n", versionID)
	}

	fileName := fmt.Sprintf("minecraft_server.%s.jar", versionID)
	url := fmt.Sprintf("%s/%s/minecraft_server.%s.jar", downloadURL, versionID, versionID)

	if _, err := os.Stat(fileName); err == nil {
		fmt.Printf("%s exists; processing...\n", fileName)
		return
	}

	out, err := os.Create(fileName)
	perror(err)
	defer out.Close()

	resp, err := http.Get(url)
	perror(err)
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	perror(err)
}

//Newest returns newest version
func (versions *Versions) Newest() Version {
	var newestVersion Version

	var versionType string
	if releaseOnly == true {
		versionType = "release"
	} else {
		versionType = "snapshot"
	}

	for _, version := range versions.VersionList {
		if version.Type == versionType {
			newestVersion = version
			break
		}
	}

	if debug {
		fmt.Printf("newestVersion - %s\n", newestVersion.ID)
	}

	return newestVersion
}
