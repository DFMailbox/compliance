package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	openapi "github.com/DFMailbox/go-client"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestBasicIdentify(t *testing.T) {
	// setup
	t.Parallel()
	env := ReadEnv()
	stack, port, err := SetupDefault(env.composePath)
	assert.NoError(t, err)
	defer Teardown(stack)

	ctx := SetupContex(port)
	log.Printf("Container address: http://localhost:%d/", port.Int())

	config := openapi.NewConfiguration()
	client := openapi.NewAPIClient(config)

	// build listener
	pubkey := extKeys[0].Public().(ed25519.PublicKey)
	encodedPubkey := base64.RawURLEncoding.EncodeToString(pubkey)
	var hits atomic.Int32
	addrChan := make(chan string, 1)
	listener, err := net.Listen("tcp", "0.0.0.0:36479")
	assert.NoError(t, err)
	ts := &httptest.Server{
		Listener: listener,
		Config:   &http.Server{Handler: http.HandlerFunc(handleIdentifyInstanceOwnership(t, addrChan, encodedPubkey, &hits))},
	}
	ts.Start()
	unprocessedAddr := listener.Addr()
	tcpAddr, ok := unprocessedAddr.(*net.TCPAddr)
	assert.True(t, ok)
	defer ts.Close()
	mockAddr := fmt.Sprintf("host.docker.internal:%d", tcpAddr.Port)
	addrChan <- mockAddr

	log.Printf("Mock address: http://%s", mockAddr)

	// actual test
	resp, err := client.InstanceAPI.IntroduceInstance(ctx).IntroduceInstanceRequest(
		*openapi.NewIntroduceInstanceRequest(encodedPubkey, mockAddr),
	).Execute()
	assert.NoError(t, err)

	assert.Equal(t, 200, resp.StatusCode)
	log.Printf("%s %+v", resp.Body, mockAddr)
	assert.Equal(t, int32(1), hits.Load())
}

func handleIdentifyInstanceOwnership(t *testing.T, addrChan chan string, pubkey string, hits *atomic.Int32) func(w http.ResponseWriter, r *http.Request) {
	// Yes, this is just a dfmailbox complianct /v0/federation/instance
	return func(w http.ResponseWriter, r *http.Request) {
		// This is probably not how you are supposed to do this
		// But I shoulnd't dwell on this too long and I should come back when I am better at go
		addr := <-addrChan
		addrChan <- addr

		assert.Equal(t, r.URL.Path, "/v0/federation/instance")
		assert.Equal(t, r.Method, "GET")
		query := r.URL.Query()
		challengeStrUuid := query.Get("challenge")
		challengeUuid, err := uuid.Parse(challengeStrUuid)
		assert.True(t, uuidRegex.Match([]byte(challengeStrUuid)))
		assert.NoError(t, err)
		challengeBytes := challengeUuid[:]
		challenge := append([]byte(addr), challengeBytes...)
		sig := ed25519.Sign(extKeys[0], challenge)
		encoded, err := json.Marshal(
			openapi.VerifyIdentity200Response{
				PublicKey: pubkey,
				Signature: base64.RawStdEncoding.EncodeToString(sig),
				Address:   addr,
			},
		)
		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(encoded)
		assert.NoError(t, err)
		hits.Add(1)
	}
}
