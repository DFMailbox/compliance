package tests

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"

	openapi "github.com/DFMailbox/go-client"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go/modules/compose"
)

var _ = Describe("Identify", Label("federation"), func() {
	var client *openapi.APIClient
	var ctx context.Context
	var stack *compose.DockerCompose
	BeforeEach(func() {
		// setup
		env := ReadEnv()
		s, port, err := SetupDefault(env.composePath)
		stack = s
		Expect(err).ShouldNot(HaveOccurred())

		ctx = SetupContex(port)
		log.Printf("Container address: http://localhost:%d/", port.Int())

		config := openapi.NewConfiguration()
		client = openapi.NewAPIClient(config)
	})
	AfterEach(func() {
		Teardown(stack)
	})
	When("The host instance is compliant", Ordered, func() {
		It("should identify instance with key 0", func() {
			pubkey, mockAddr, hits, testServer := setupMockServer(extKeys[0])
			defer testServer.Close()
			resp, err := client.InstanceAPI.IntroduceInstance(ctx).IntroduceInstanceRequest(
				*openapi.NewIntroduceInstanceRequest(pubkey, mockAddr),
			).Execute()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(resp.StatusCode).Should(Equal(200))
			log.Printf("%s %+v", resp.Body, mockAddr)
			Expect(resp.StatusCode, 200)
			Expect(hits.Load()).Should(Equal(int32(1)))
		})
		It("should identify instance with key 1", func() {
			pubkey, mockAddr, hits, testServer := setupMockServer(extKeys[1])
			defer testServer.Close()
			resp, err := client.InstanceAPI.IntroduceInstance(ctx).IntroduceInstanceRequest(
				*openapi.NewIntroduceInstanceRequest(pubkey, mockAddr),
			).Execute()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(resp.StatusCode).Should(Equal(200))
			log.Printf("%s %+v", resp.Body, mockAddr)
			Expect(resp.StatusCode, 200)
			Expect(hits.Load()).Should(Equal(int32(1)))
		})
	})
})

func setupMockServer(key ed25519.PrivateKey) (string, string, *atomic.Int32, *httptest.Server) {
	pubkey := key.Public().(ed25519.PublicKey)
	encodedPubkey := base64.RawURLEncoding.EncodeToString(pubkey)
	var hits atomic.Int32
	addrChan := make(chan string, 1)
	listener, err := net.Listen("tcp", "0.0.0.0:0")
	Expect(err).ShouldNot(HaveOccurred())
	ts := &httptest.Server{
		Listener: listener,
		Config:   &http.Server{Handler: http.HandlerFunc(compliantHandleIdentifyInstanceOwnership(addrChan, encodedPubkey, &hits))},
	}
	ts.Start()
	unprocessedAddr := listener.Addr()
	tcpAddr, ok := unprocessedAddr.(*net.TCPAddr)
	Expect(ok).Should(BeTrue())
	mockAddr := fmt.Sprintf("host.docker.internal:%d", tcpAddr.Port)
	addrChan <- mockAddr

	log.Printf("Mock address: http://%s", mockAddr)
	return encodedPubkey, mockAddr, &hits, ts
}

func compliantHandleIdentifyInstanceOwnership(addrChan chan string, pubkey string, hits *atomic.Int32) func(w http.ResponseWriter, r *http.Request) {
	// Yes, this is just a dfmailbox complianct /v0/federation/instance
	return func(w http.ResponseWriter, r *http.Request) {
		// This is probably not how you are supposed to do this
		// But I shoulnd't dwell on this too long and I should come back when I am better at go
		addr := <-addrChan
		addrChan <- addr

		Expect(r.URL.Path).Should(Equal("/v0/federation/instance"))
		Expect(r.Method, "GET")
		query := r.URL.Query()
		challengeStrUuid := query.Get("challenge")
		challengeUuid, err := uuid.Parse(challengeStrUuid)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(uuidRegex.Match([]byte(challengeStrUuid))).Should(BeTrue())
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
		Expect(err).ShouldNot(HaveOccurred())
		hits.Add(1)
	}
}
