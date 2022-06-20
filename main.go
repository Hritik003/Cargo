package main

import (
	_ "encoding/json"
	"flag"
	"fmt"
	agent "goavega-software/cargo/cargo/agent"
	detection "goavega-software/cargo/cargo/detection"
	client "goavega-software/cargo/cargo/gateway"
	"log"
)

func main() {
	installAgent, runAgent, runGateway, localMode, containerize, detectStackver := parseArgs()
	if !installAgent && !runAgent && !runGateway && !containerize && !detectStackver {
		runGateway = true
	}
	if runGateway {
		log.Println("running gateway")
		gateway := client.Gateway{Port: 8080, Debug: localMode}
		gateway.Init()
	}
	if runAgent {
		log.Println("running agent")
		cli := agent.Cli{}
		//TODO: get port from elsewhere
		cli.Port = 8050
		cli.Init()
	}
	if installAgent {
		log.Println("installing agent")
		cli := agent.Cli{}
		cli.Setup()
	}
	if containerize {
		log.Println("containerizing application")
		agent.CreateContainer()
	}

	if detectStackver {

		log.Println("detecting stack")
		stack := detection.Stack{}
		detect, err := stack.Detect("./")
		if err != nil {
			fmt.Println("Error in detecting stack")
			return
		}
		stack.Name = detect.Name

		if stack.Name == "php" {
			stack.Version = detection.Parse_php("./" + "composer.json")
		}
		if detect.Name == "nodejs" {
			stack.Version = detection.Parsenodejs("./" + "package.json")
		}
		if detect.Name == "dotnet" {
			stack.Version = detection.Parse_dotnet("./" + "web.config")
		}

		log.Println(stack.Version)
	}

}

func parseArgs() (bool, bool, bool, bool, bool, bool) {
	installAgent := flag.Bool("ia", false, "install agent on target machine")
	runAgent := flag.Bool("ra", false, "run agent on target machine")
	runGateway := flag.Bool("rg", false, "run gateway")
	localMode := flag.Bool("D", false, "run locally")
	containerize := flag.Bool("c", false, "containerize")
	detectStack := flag.Bool("detect", false, "detect stack")
	flag.Parse()
	return *installAgent, *runAgent, *runGateway, *localMode, *containerize, *detectStack
}
