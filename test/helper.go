package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	openapi "github.com/DFMailbox/go-client"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go/modules/compose"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Pretty please don't mutate this
var Keys = []string{
	"TESTING0KEYTESTING0KEYTESTING0KEYTESTING000=",
	"TESTING0KEYTESTING0KEYTESTING0KEYTESTING001=",
	"TESTING0KEYTESTING0KEYTESTING0KEYTESTING002=",
	"TESTING0KEYTESTING0KEYTESTING0KEYTESTING003=",
	"TESTING0KEYTESTING0KEYTESTING0KEYTESTING004=",
}

func ReadEnv() Environment {
	path, set := os.LookupEnv("DFMC_COMPOSE_FILE")
	if set == false {
		path = "../../compliance-docker-compose.yml"
	}
	return Environment{
		composePath: path,
	}
}

type Environment struct {
	composePath string
}

func SetupDefault(file_path string) (*compose.DockerCompose, *nat.Port, error) {
	return Setup(file_path, map[string]string{
		"DFMC_ADDRESS":     "dfm.example.com",
		"DFMC_PRIVATE_KEY": Keys[0],
	})
}

func Setup(file_path string, env map[string]string) (*compose.DockerCompose, *nat.Port, error) {
	ctx := context.Background()
	stack, err := compose.NewDockerComposeWith(
		// compose.StackIdentifier("dfm_compliance"),
		compose.WithStackFiles(file_path),
	)
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("Failed to create stack %v", err))
	}
	err = stack.WithEnv(env).
		WaitForService("dfmailbox", wait.ForExposedPort()).
		Up(ctx, compose.Wait(true))
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("Failed to start compose stack %v", err))
	}
	container, err := stack.ServiceContainer(ctx, "dfmailbox")
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("Failed to find container %v", err))
	}
	port, err := container.MappedPort(ctx, "8080/tcp")
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("Failed to find container endpoint %v", err))
	}
	return stack, &port, nil
}

func Teardown(stack *compose.DockerCompose) {
	err := stack.Down(
		context.Background(),
		compose.RemoveOrphans(true),
		compose.RemoveImagesLocal,
	)
	if err != nil {
		log.Printf("Failed to start stack: %v", err)

	}
}

func AddPlotAuth(ctx context.Context, name string, plotId int32) context.Context {
	return context.WithValue(ctx, openapi.ContextAPIKeys, map[string]openapi.APIKey{
		"Plot": {Key: fmt.Sprintf("Hypercube/7.2 (%d, %s)", plotId, name)},
	})
}

func StrAsRef(s string) *string { return &s }

func SetupContex(port *nat.Port) context.Context {
	ctx := context.WithValue(context.Background(), openapi.ContextServerVariables, map[string]string{
		"port": port.Port(),
	})
	return context.WithValue(ctx, openapi.ContextServerIndex, 0)
}
