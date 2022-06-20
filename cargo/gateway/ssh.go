package gateway

import (
	"encoding/json"
	"fmt"
	"goavega-software/cargo/cargo/common"
	"log"
	"strings"

	"github.com/sendgrid/rest"
	"golang.org/x/crypto/ssh"
)

const port = 8050
const host = "http://localhost"

type Connection struct {
	*ssh.Client
}

type RemoteAgent struct{}

func (ra *RemoteAgent) Connect(addr, user, password string) (*Connection, error) {
	var con *Connection
	if addr == "127.0.0.1" || addr == "localhost" {
		log.Println("Local address, skipping connecting")
		return con, nil
	}
	sshConfig := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	conn, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, err
	}

	return &Connection{conn}, nil

}

func (ra *RemoteAgent) Start(agent Agent, container common.Container) <-chan bool {
	promise := make(chan bool)
	go func() {
		defer close(promise)
		conn, err := ra.Connect(agent.Credentials.Server, agent.Credentials.UserName, agent.Credentials.Password)
		if err != nil {
			promise <- false
		}
		if conn != nil {
			defer conn.Close()
		}
		queryParams := make(map[string]string)
		queryParams["containerImage"] = container.Image
		queryParams["registry"] = container.Registry
		endpoint := "container/start"
		baseURL := fmt.Sprintf("%s:%d/%s", host, port, endpoint)
		Headers := make(map[string]string)
		//TODO: add authorization
		//Headers["Authorization"] = "Bearer " + key
		method := rest.Post
		request := rest.Request{
			Method:      method,
			BaseURL:     baseURL,
			Headers:     Headers,
			Body:        nil,
			QueryParams: queryParams,
		}
		response, err := rest.Send(request)
		if err != nil {
			promise <- false
			fmt.Println(err)
		} else {
			promise <- true
			fmt.Println(response.StatusCode)
			fmt.Println(response.Body)
			fmt.Println(response.Headers)
		}
	}()
	return promise
}

func (ra *RemoteAgent) AddContainer(agent Agent, container common.Container) <-chan bool {

	promise := make(chan bool)
	go func() {
		defer close(promise)
		conn, err := ra.Connect(agent.Credentials.Server, agent.Credentials.UserName, agent.Credentials.Password)
		if err != nil {
			promise <- false
		}
		if conn != nil {
			defer conn.Close()
		}

		endpoint := "container/pull"
		baseURL := fmt.Sprintf("%s:%d/%s", host, port, endpoint)
		Headers := make(map[string]string)
		//TODO: add authorization
		//Headers["Authorization"] = "Bearer " + key

		Body, err := json.Marshal(container)
		method := rest.Post
		request := rest.Request{
			Method:  method,
			BaseURL: baseURL,
			Headers: Headers,
			Body:    Body,
		}
		response, err := rest.Send(request)
		if err != nil {
			promise <- false
			fmt.Println(err)
		} else {
			promise <- true
			fmt.Println(response.StatusCode)
			fmt.Println(response.Body)
			fmt.Println(response.Headers)
		}
	}()
	return promise
}

func (conn *Connection) Close() {
	conn.Conn.Close()
}

func (conn *Connection) SendCommands(cmds ...string) ([]byte, error) {
	session, err := conn.NewSession()
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	err = session.RequestPty("xterm", 80, 40, modes)
	if err != nil {
		return []byte{}, err
	}
	cmd := strings.Join(cmds, "; ")
	output, err := session.Output(cmd)
	if err != nil {
		return output, fmt.Errorf("failed to execute command '%s' on server: %v", cmd, err)
	}

	return output, err
}
