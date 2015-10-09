package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// Server gives access to server related commands
type Server struct {
	Version string
}

// NewServer returns a new server object.
func NewServer() *Server {
	server := &Server{
		Version: "",
	}

	return server
}

func (s Server) currentVersion() (string, error) {
	fileBuffer, err := ioutil.ReadFile("current_version")
	if err != nil {
		fmt.Println("current_version doesn't exist.")
		return "", err
	}
	log.Println("current_version doesn't exist.")

	currentVersionID := string(fileBuffer)

	if debug {
		fmt.Printf("Current Version %s\n", currentVersionID)
	}
	log.Printf("Current Version %s\n", currentVersionID)

	return currentVersionID, nil
}

func (s Server) writeCurrentVersion(version string) {
	err := ioutil.WriteFile("current_version", []byte(version), 0644)
	perror(err)
}

func (s Server) writeServerPid(pid int) {
	pidString := strconv.Itoa(pid)
	err := ioutil.WriteFile(".pid", []byte(pidString), 0644)
	perror(err)
}

func (s Server) deleteServerPidFile() {
	err := os.Remove(".pid")
	perror(err)
}

func (s Server) readServerPid() (int, error) {
	fileBuffer, err := ioutil.ReadFile(".pid")
	if err != nil {
		fmt.Println("current_version doesn't exist.")
		log.Println("current_version doesn't exist.")
		return 0, err
	}
	pidString := string(fileBuffer)
	pid, err := strconv.Atoi(strings.Replace(pidString, "\n", "", -1))
	perror(err)

	return pid, nil
}

func (s Server) makeServerRunning() {
	if _, err := os.Stat(".pid"); err == nil {
		fmt.Println("Server already running")
		return
	}

	version, err := s.currentVersion()
	if err != nil {
		fmt.Println("No current version found. Can't start server.")
		log.Println("No current version found. Can't start server.")
		return
	}

	s.startServer(version)
}

func (s Server) startServer(version string) {
	fileName := fmt.Sprintf("minecraft_server.%s.jar", version)
	command := fmt.Sprintf("java -Xmx%dM -Xms%dM -jar %s nogui", ramAlloc, ramAlloc, fileName)

	fmt.Printf("running %s\n", command)
	log.Printf("running %s\n", command)
	serverCmd := exec.Command("java", "-Xmx2048M", "-Xms2048M", "-jar", fileName, "nogui")
	serverCmd.SysProcAttr = &syscall.SysProcAttr{}
	serverCmd.SysProcAttr.Setpgid = true

	err := serverCmd.Start()
	perror(err)

	s.writeServerPid(serverCmd.Process.Pid)

	fmt.Printf("Server running...\n")
	log.Printf("Server running...\n")
	go s.watchServer(serverCmd)
}

func (s Server) watchServer(cmd *exec.Cmd) {
	fmt.Printf("Watching command %d...\n", cmd.Process.Pid)
	log.Printf("Watching command %d...\n", cmd.Process.Pid)
	err := cmd.Wait()
	fmt.Printf("Command finished with error: %s\n", err)
	log.Printf("Command finished with error: %s\n", err)
}

func (s Server) stopServer() {
	pid, err := s.readServerPid()

	if err != nil {
		return
	}

	p, err := os.FindProcess(pid)
	perror(err)

	err = p.Kill()
	perror(err)

	s.deleteServerPidFile()
}
