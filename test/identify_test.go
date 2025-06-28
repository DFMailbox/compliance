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
	// "time"

	openapi "github.com/DFMailbox/go-client"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go/modules/compose"
)

var _ = Describe("Identify and test category: instance", Ordered, Label("federation"), func() {
	var client *openapi.APIClient
	var ctx context.Context
	var stack *compose.DockerCompose
	BeforeAll(func() {
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
	AfterAll(func() {
		Teardown(stack)
	})
	When("The instance is compliant", Ordered, func() {
		var mockAddr string
		It("should identify instance", func() {
			pubkey, lMockAddr, hits, testServer := setupMockServer(extKeys[1])
			mockAddr = lMockAddr
			defer testServer.Close()
			resp, err := client.InstanceAPI.IntroduceInstance(ctx).IntroduceInstanceRequest(
				*openapi.NewIntroduceInstanceRequest(pubkey, lMockAddr),
			).Execute()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(resp.StatusCode).Should(Equal(200))
			Expect(resp.StatusCode, 200)
			Expect(hits.Load()).Should(Equal(int32(1)))
		})
		It("should respond with instance", func() {
			oai, _, err := client.InstanceAPI.LookupInstanceAddress(ctx).
				Execute()
			Expect(err).ShouldNot(HaveOccurred())
			log.Printf("More: %+v", oai.LookupInstanceAddress200ResponseOneOf1.Instances)
			oai, resp, err := client.InstanceAPI.LookupInstanceAddress(ctx).
				PublicKey(base64.RawURLEncoding.EncodeToString(extKeys[1].Public().(ed25519.PublicKey))).
				Execute()
			log.Printf("%+v", oai)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(resp.StatusCode).Should(Equal(200))
			Expect(*oai.LookupInstanceAddress200ResponseOneOf.Instance.Address.Get()).Should(Equal(mockAddr))
		})
		It("should fail on a nonexistent instance", func() {
			oai, resp, err := client.InstanceAPI.LookupInstanceAddress(ctx).PublicKey(
				base64.RawURLEncoding.EncodeToString(extKeys[2].Public().(ed25519.PublicKey)),
			).Execute()
			Expect(oai).Should(BeNil())
			Expect(resp.StatusCode).Should(Equal(404))
			Expect(err).Should(HaveOccurred())
			var data map[string]any
			json.Unmarshal(err.(*openapi.GenericOpenAPIError).Body(), &data)
			log.Printf("%+v", data)
			Expect(data).To(HaveKeyWithValue("type", "/v0/problems/unknown-instance"))
			Expect(data).To(HaveKeyWithValue("public_key", "Gp6a-nGu8TCRsMWSRuIzlt-_KYOJsBJgaQ2DRIIkvF4="))
			Expect(data).To(HaveKeyWithValue("status", 404.0))
			Expect(data).To(HaveKeyWithValue("title", "Specified instance has not been identified"))
		})
	})
	It("should identify instance with key 2", func() {
		pubkey, mockAddr, hits, testServer := setupMockServer(extKeys[2])
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

func setupMockServer(key ed25519.PrivateKey) (string, string, *atomic.Int32, *httptest.Server) {
	pubkey := key.Public().(ed25519.PublicKey)
	encodedPubkey := base64.RawURLEncoding.EncodeToString(pubkey)
	var hits atomic.Int32
	addrChan := make(chan string, 1)
	listener, err := net.Listen("tcp", "0.0.0.0:0")
	Expect(err).ShouldNot(HaveOccurred())
	ts := &httptest.Server{
		Listener: listener,
		Config:   &http.Server{Handler: http.HandlerFunc(compliantHandleIdentifyInstanceOwnership(key, addrChan, encodedPubkey, &hits))},
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

func compliantHandleIdentifyInstanceOwnership(key ed25519.PrivateKey, addrChan chan string, pubkey string, hits *atomic.Int32) func(w http.ResponseWriter, r *http.Request) {
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
		sig := ed25519.Sign(key, challenge)
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
