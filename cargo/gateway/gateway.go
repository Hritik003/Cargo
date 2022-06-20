package gateway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"goavega-software/cargo/cargo/common"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

var debug = false

type Gateway struct {
	Port  int
	Debug bool
}

func (g Gateway) Init() {
	debug = g.Debug
	http.HandleFunc("/api/agent/container/manage", manageContainerHandler)
	http.HandleFunc("/api/agent/container", addContainerHandler)
	http.HandleFunc("/api/agent", agentHandler)
	http.Handle("/public/", http.StripPrefix("/public/", http.FileServer(http.Dir("./resources/public/"))))
	http.HandleFunc("/", viewHandler)

	http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", g.Port), nil)
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Serving URL", r.URL.Path)
	if r.URL.Path == "/login" {
		http.ServeFile(w, r, "./resources/gateway/views/login.html")
		return
	}
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	files := []string{
		"./resources/gateway/views/list.html",
		"./resources/gateway/views/index.html",
	}
	ts, err := template.ParseFiles(files...)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", 500)
		return
	}
	err = ts.Execute(w, nil)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", 500)
	}
}

func manageContainerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not Alowed", http.StatusMethodNotAllowed)
		return
	}
	agentId := r.URL.Query().Get("agentId")

	id, err := strconv.Atoi(agentId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	agent, err := GetAgent(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var result map[string]string
	response := false
	// Unmarshal or Decode the JSON to the interface.
	json.NewDecoder(r.Body).Decode(&result)
	action := result["action"]
	if action == "start" {
		ra := RemoteAgent{}
		container := common.Container{Image: result["image"], Registry: result["registry"]}
		response = <-ra.Start(agent, container)
	}
	// Print the data type of result variable
	fmt.Println(result["image"], id)
	w.Header().Add("Content-Type", "application/json")
	fmt.Fprint(w, fmt.Sprintf("{\"success\": %t}", response))
}

func addContainerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		agentId := r.URL.Query().Get("agentId")
		containers := make([]common.Container, 0)
		err := json.NewDecoder(r.Body).Decode(&containers)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		id, err := strconv.Atoi(agentId)
		err = AddContainer(id, containers)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		agent, err := GetAgent(id)
		ra := RemoteAgent{}
		response := <-ra.AddContainer(agent, containers[len(containers)-1])
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, fmt.Sprintf("{\"success\": %t}", response))
	}
}

func agentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {

		agent := Agent{}
		// Try to decode the request body into the struct. If there is an error,
		// respond to the client with the error message and a 400 status code.
		err := json.NewDecoder(r.Body).Decode(&agent)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_, e := AddAgent(agent)
		if e != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, "{\"success\": true}")
		go func() {
			if agent.Csp == "azure" {
				d1 := []byte(common.AzureDeploy)
				os.WriteFile("./cargo.deploy.ps1", d1, 0644)
				container := agent.Containers[0]
				envMap := make(map[string]string)
				for _, e := range container.Env {
					envMap[e.Key] = e.Value
					if strings.Contains(e.Key, "DATABASE") {
						dbString := strings.Replace(strings.Replace(strings.Replace(strings.Replace(e.Value, "%username%", agent.Db.UserName, -1), "%password%", agent.Db.Password, -1), "%host%", agent.Name+".postgres.database.azure.com", -1), "%db%", agent.Db.Name, -1)
						envMap[e.Key] = dbString
					}
				}
				envString, _ := json.Marshal(envMap)
				log.Println(envString)
				os.WriteFile("./.env.properties", []byte(envString), 0644)
				args := []string{"./cargo.deploy.ps1", "-NoProfile", "-NonInteractive",
					"-resource", agent.Name, "-registry", container.Registry, "-tag", container.Image, "-rUser", container.Username, "-rPassword", container.Password,
					"-dbName", agent.Db.Name, "-dbUser", agent.Db.UserName, "-dbPassword", agent.Db.Password}
				cmd := exec.Command("powershell.exe", args...)

				var stdout bytes.Buffer
				var stderr bytes.Buffer
				cmd.Stdout = &stdout
				cmd.Stderr = &stderr
				log.Println("deploying application")
				err = cmd.Run()
				log.Println(stdout.String())
				log.Println(stderr.String())
				log.Println("application deployed")
				os.Remove("./.env.properties")
				os.Remove("./cargo.deploy.ps1")
			}
		}()
		// skip on local
		if debug {
			log.Println("Running in debug mode, skipping setup")
			return
		}
		// go func() {
		// 	connection, err := Connect(agent.Credentials.Server, agent.Credentials.UserName, agent.Credentials.Password)
		// 	if err != nil {
		// 		// we should set the status of this agent as error and log the error somewhere
		// 		log.Print(err)
		// 		return
		// 	}
		// 	response, err := connection.SendCommands("whoami")
		// 	if err != nil {
		// 		log.Print(err)
		// 		return
		// 	}
		// 	log.Print(string(response))
		// }()
		return
	}
	if r.Method == "GET" {
		agents, _ := GetAgents()
		agentsStr, _ := json.Marshal(agents)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, string(agentsStr))
		return
	}
	if r.Method == "PUT" {
		// let's just update it - ideally we would have liked id in URL but that's ok
	}
	http.Error(w, "Method not Alowed", http.StatusMethodNotAllowed)
}
