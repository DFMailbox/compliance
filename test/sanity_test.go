package main

import (
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/testcontainers/testcontainers-go/log"
)

func TestStack(t *testing.T) {
	t.Parallel()
	env := ReadEnv()
	stack, natPort, err := SetupDefault(env.composePath)
	port := natPort.Port()
	if err != nil {
		t.Error(err)
		return
	}
	log.Printf("Address: localhost:%s", port)
	res, err := http.Get(fmt.Sprintf("http://localhost:%s", port))
	if err != err {
		t.Errorf("Error sending req %v", err)
	}
	body, err := io.ReadAll(res.Body)
	defer res.Body.Close()
	if string(body) != "dfmailbox" {
		t.Errorf("Body is %v not 'dfmailbox'", body)
	}

	defer Teardown(stack)
}
