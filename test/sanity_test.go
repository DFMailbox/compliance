package main

import (
	"io"
	"net/http"
	"testing"

	"github.com/testcontainers/testcontainers-go/log"
)

func TestStack(t *testing.T) {
	t.Parallel()
	env := ReadEnv()
	stack, address, err := SetupDefault(env.composePath)
	if err != nil {
		t.Error(err)
		return
	}
	log.Printf("Address: %v", address)
	res, err := http.Get(address)
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
