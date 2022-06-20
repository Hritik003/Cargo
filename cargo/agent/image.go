package agent

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"goavega-software/cargo/cargo/common"
	"goavega-software/cargo/cargo/gateway"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
)

var dockerPostgres string = `
FROM python:3.8-alpine
RUN mkdir /app
ADD . /app
WORKDIR /app
RUN \
set -ex && \
apk add --no-cache postgresql-libs libffi-dev && \
apk add --no-cache --virtual .build-deps gcc musl-dev postgresql-dev && \
pip install -r requirements.txt && \
apk --purge del .build-deps

EXPOSE 8080
CMD ["python", "start.py"]

`
var pythonMysql string = `
FROM python:3.8-alpine
RUN mkdir /app
ADD . /app
WORKDIR /app
RUN \
set -ex && \
apk add --no-cache mysql-libs libffi-dev && \
apk add --no-cache --virtual .build-deps gcc musl-dev mysql-dev && \
pip install -r requirements.txt && \
apk --purge del .build-deps

EXPOSE 8080
CMD ["python", "start.py"]

`

func getContainerId(image string, cli *client.Client, ctx context.Context) string {
	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}
	for _, container := range containers {
		if container.Image == image {
			return container.ID
		}
	}
	return ""
}

func mapEnvToString(settings []common.Setting) []string {
	envs := make([]string, 0)
	for _, setting := range settings {
		envs = append(envs, setting.Key+"="+setting.Value)
	}
	return envs
}

func PullContainer(image common.Container) {
	agentDb := gateway.AgentDb{}

	hasContainer, err := agentDb.AddContainer(image)
	log.Println(hasContainer)
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	var authStr string
	fmt.Println("username is", image.Username)
	if len(image.Username) != 0 {
		authConfig := types.AuthConfig{
			Username: image.Username,
			Password: image.Password,
		}

		encodedJSON, err := json.Marshal(authConfig)
		if err != nil {
			panic(err)
		}
		authStr = base64.URLEncoding.EncodeToString(encodedJSON)
	}
	imageToPull := fmt.Sprintf("%s/%s", image.Registry, image.Image)
	reader, err := cli.ImagePull(ctx, imageToPull, types.ImagePullOptions{RegistryAuth: authStr})
	if err != nil {
		panic(err)
	}
	io.Copy(os.Stdout, reader)

	if hasContainer {
		// stop the current container
		StopContainer(image.Image, cli, ctx)
	}
	log.Println("Starting container")
	StartContainer(image)
}

func StopContainer(name string, cli *client.Client, ctx context.Context) {
	containerId := getContainerId(name, cli, ctx)
	if containerId == "" {
		log.Println("No running container for ", name)
	}
	t, _ := time.ParseDuration("1000ms")
	err := cli.ContainerStop(ctx, containerId, &t)
	if err != nil {
		log.Println(err)
	}
}

func getPortMapping(ports string) (string, string) {
	portsArray := strings.Split(ports, ":")
	return portsArray[0], portsArray[1]
}

func StartContainer(image common.Container) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	log.Println("starting container", image.Image, image.Port, image.Registry)
	hostPort, containerPort := getPortMapping(image.Port)
	log.Println("Got port mapping")
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: fmt.Sprintf("%s/%s", image.Registry, image.Image),
		Env:   mapEnvToString(image.Env),
		ExposedPorts: nat.PortSet{
			nat.Port(containerPort): {},
		},
		Tty:          true,
		OpenStdin:    true,
		AttachStdout: true,
		AttachStderr: true,
	}, &container.HostConfig{
		PortBindings: nat.PortMap{
			nat.Port(containerPort): []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: hostPort}},
		},
	}, nil, nil, "")
	log.Println("created container")
	if err != nil {
		panic(err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}
	log.Println("started container")
	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			panic(err)
		}
	case <-statusCh:
	}

	out, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		panic(err)
	}

	stdcopy.StdCopy(os.Stdout, os.Stderr, out)

}

func Stop(name string) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	StopContainer(name, cli, ctx)
}

func GetLogs(containerImage string) string {
	buf := new(strings.Builder)
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	containerId := getContainerId(containerImage, cli, ctx)
	if containerId == "" {
		return ""
	}
	options := types.ContainerLogsOptions{ShowStdout: true}
	out, err := cli.ContainerLogs(ctx, containerId, options)
	if err != nil {
		panic(err)
	}

	_, err = io.Copy(buf, out)
	return buf.String()
}

func CreateContainer() {
	content, err := os.ReadFile("./.cargo")
	if err != nil {
		log.Println("Setup file not found, exiting")
		return
	}
	_, err = os.Stat("./Dockerfile")
	var fileContent string = ""
	if err != nil {
		log.Println("dockerfile exists, exiting")
		return
	}
	stringContent := string(content)
	log.Println("Scanning stack....")
	if strings.Contains(stringContent, "python") {
		log.Println("python detected....")
	}
	if strings.Contains(stringContent, "php") {
		log.Println("python detected....")
	}
	if strings.Contains(stringContent, "mysql") {
		fileContent = pythonMysql
	}
	if strings.Contains(stringContent, "postgres") {
		log.Println("postgres detected....")
		fileContent = dockerPostgres
	}
	d1 := []byte(fileContent)
	os.WriteFile("./Dockerfile", d1, 0644)
	return
}
