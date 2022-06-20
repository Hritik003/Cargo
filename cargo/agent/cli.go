package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"goavega-software/cargo/cargo/common"
	"goavega-software/cargo/cargo/gateway"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"runtime"
	"syscall"
	"time"
)

var done chan bool

type Cli struct {
	Port int
}

func canInstall() bool {
	if runtime.GOOS == "windows" {
		log.Fatal("agent can only be installed on linux")
		return false
	}
	cmd := exec.Command("cat", "/etc/os-release")
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal("could not get OS info", err)
		return false
	}
	r := regexp.MustCompile(`NAME=(?P<NAME>.+)`)
	matches := r.FindStringSubmatch(out.String())
	if len(matches) != 2 {
		log.Fatal("could not get OS info", err)
		return false
	}
	osName := matches[1]
	if osName != "ubuntu" {
		log.Fatal("setup is only supported on Ubuntu, running on", osName)
		return false
	}

	return true
}

func watchDog() {
	fmt.Println("getting containers")
	db := gateway.AgentDb{}
	_ = db.GetContainers()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			fmt.Println("Done!")
			return
		case t := <-ticker.C:

			fmt.Println("Current time: ", t)
		}
	}
}

func (cli *Cli) Init() {
	http.HandleFunc("/", cliHandler)
	go func() {
		http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", cli.Port), nil)
	}()
	done = make(chan bool)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		done <- true
	}()
	go watchDog()
	<-done
}

func (cli *Cli) Setup() {
	log.Println("Attempting installation on ", runtime.GOOS)

	if !canInstall() {
		return
	}
	serviceTemplate := GetServiceInstallationTemplate()
	log.Println(serviceTemplate)
	//TODO: install the service template
}

func cliHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Seek and you shall find.", http.StatusMethodNotAllowed)
	}
	log.Print(r.URL.Path)
	containerImage := r.URL.Query().Get("containerImage")
	registry := r.URL.Query().Get("registry")
	if r.URL.Path == "/container/pull" {
		container := common.Container{}
		err := json.NewDecoder(r.Body).Decode(&container)
		if err != nil {
			http.Error(w, "Ship ahoy.", http.StatusBadRequest)
			return
		}
		w.Header().Add("Content-Type", "application/json")
		pullContainer(container)
		w.Write([]byte("{\"success\": true}"))
		return
	}
	if r.URL.Path == "/container/logs" {
		if containerImage == "" {
			http.Error(w, "Ship ahoy.", http.StatusBadRequest)
			return
		}
		w.Header().Add("Content-Type", "application/json")

		fmt.Fprintf(w, "{\"success\": true, \"data\": \"%s\"}", getContainerLogs(containerImage))
		return
	}
	if r.URL.Path == "/container/stop" {
		if containerImage == "" {
			http.Error(w, "Ship ahoy.", http.StatusBadRequest)
			return
		}
		stopContainer(containerImage)
		w.Header().Add("Content-Type", "application/json")
		w.Write([]byte("{\"success\": true}"))
		return
	}
	if r.URL.Path == "/container/start" {
		if containerImage == "" {
			http.Error(w, "Ship ahoy.", http.StatusBadRequest)
			return
		}
		container := common.Container{Image: containerImage, Registry: registry}
		startContainer(container)
	}
	http.Error(w, "You have sunk the cargo.", http.StatusNotFound)
}

func pullContainer(container common.Container) {
	go func() {
		PullContainer(container)
	}()
}

func startContainer(request common.Container) {
	go func() {
		db := gateway.AgentDb{}
		container := db.GetContainer(request.Image, request.Registry)
		if container.PullPolicy == "always" {
			PullContainer(container)
			return
		}
		StartContainer(container)
	}()
}

func stopContainer(containerImage string) {
	go func() {
		Stop(containerImage)
	}()
}

func getContainerLogs(containerImage string) string {
	return GetLogs(containerImage)
}
