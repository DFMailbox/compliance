package tests

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http/httptest"
	"strings"
	"sync/atomic"

	openapi "github.com/DFMailbox/go-client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go/modules/compose"
)

// TODO: /v0/problems/federation/non-compliance, /v0/problems/instance-introduction/mismatched-address, /v0/problems/instance-introduction/mismatched-public-key
var _ = Describe("Identify and test category: instance", Ordered, Label("federation"), func() {
	var client *openapi.APIClient
	var ctx context.Context
	var stack *compose.DockerCompose
	BeforeAll(func() {
		// setup
		s, port, err := SetupDefault()
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
	When("The instance is compliant", func() {
		var mockAddr string
		var server *httptest.Server
		var pubkey string
		var hits *atomic.Int32
		var bPubKey string
		var bMockAddr string
		var bHits *atomic.Int32
		var bServer *httptest.Server
		BeforeAll(func() {
			pubkey, mockAddr, hits, server = SetupMockServer(extKeys[1])
			bPubKey, bMockAddr, bHits, bServer = SetupMockServer(extKeys[1])
		})
		AfterAll(func() {
			server.Close()
			bServer.Close()
		})
		It("should identify instance", func() {
			resp, err := client.InstanceAPI.IntroduceInstance(ctx).IntroduceInstanceRequest(
				*openapi.NewIntroduceInstanceRequest(pubkey, mockAddr),
			).Execute()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(resp.StatusCode).Should(Equal(200))
			Expect(hits.Load()).Should(Equal(int32(1)))
		})
		It("should reject second instance registration", func() {
			resp, err := client.InstanceAPI.IntroduceInstance(ctx).IntroduceInstanceRequest(
				*openapi.NewIntroduceInstanceRequest(pubkey, mockAddr),
			).Execute()
			Expect(err).Should(HaveOccurred())
			Expect(resp.StatusCode).Should(Equal(409))
			Expect(hits.Load()).Should(Equal(int32(2)))
			var data map[string]any
			json.Unmarshal(err.(*openapi.GenericOpenAPIError).Body(), &data)
			Expect(data).Should(HaveKeyWithValue("type", "/v0/problems/already-exists"))
			Expect(data).Should(HaveKeyWithValue("title", "The resource being created already exists"))
			Expect(data).Should(HaveKeyWithValue("status", 409.0))
		})
		It("should respond with instance", func() {
			oai, resp, err := client.InstanceAPI.LookupInstanceAddress(ctx).
				PublicKey(base64.RawURLEncoding.EncodeToString(extKeys[1].Public().(ed25519.PublicKey))).
				Execute()
			Expect(resp.Header.Get("content-type")).Should(Equal("application/json; charset=utf-8"))
			Expect(err).ShouldNot(HaveOccurred())
			Expect(resp.StatusCode).Should(Equal(200))
			Expect(*oai.LookupInstanceAddress200ResponseOneOf.Instance.Address.Get()).Should(Equal(mockAddr))
		})
		It("should update instance", func() {
			yes := true
			req := client.InstanceAPI.IntroduceInstance(ctx).IntroduceInstanceRequest(
				*&openapi.IntroduceInstanceRequest{
					PublicKey: bPubKey,
					Address:   bMockAddr,
					Update:    &yes,
				},
			)
			resp, err := req.Execute()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(resp.StatusCode).Should(Equal(200))
			Expect(bHits.Load()).Should(Equal(int32(1)))
		})
	})
	When("Other instance isn't compliant", func() {
		It("should fail on a nonexistent instance", func() {
			oai, resp, err := client.InstanceAPI.LookupInstanceAddress(ctx).PublicKey(
				base64.RawURLEncoding.EncodeToString(extKeys[3].Public().(ed25519.PublicKey)),
			).Execute()
			Expect(oai).Should(BeNil())
			Expect(resp.StatusCode).Should(Equal(404))
			Expect(resp.Header.Get("content-type")).Should(Equal("application/problem+json; charset=utf-8"))
			Expect(err).Should(HaveOccurred())
			var data map[string]any
			json.Unmarshal(err.(*openapi.GenericOpenAPIError).Body(), &data)
			Expect(data).Should(HaveKeyWithValue("type", "/v0/problems/unknown-instance"))
			Expect(data).Should(HaveKeyWithValue("public_key", "30TaVy9w1g8W5-JTDJYneuNeYVLRI_NaJgoXwFq_mTI="))
			Expect(data).Should(HaveKeyWithValue("status", 404.0))
			Expect(data).Should(HaveKeyWithValue("title", "Specified instance has not been identified"))
		})
		It("should reject mismatch", func() {
			pubkey, mockAddr, hits, server := SetupMockServer(extKeys[2])
			defer server.Close()
			altAddr := "alt-" + mockAddr
			resp, err := client.InstanceAPI.IntroduceInstance(ctx).IntroduceInstanceRequest(
				*openapi.NewIntroduceInstanceRequest(pubkey, altAddr),
			).Execute()
			Expect(err).Should(HaveOccurred())
			Expect(resp.StatusCode).Should(Equal(400))
			Expect(resp.Header.Get("content-type")).Should(Equal("application/problem+json; charset=utf-8"))
			Expect(hits.Load()).Should(Equal(int32(1)))
			var data map[string]any
			json.Unmarshal(err.(*openapi.GenericOpenAPIError).Body(), &data)
			Expect(data).Should(HaveKeyWithValue("type", "/v0/problems/challenge-failed"))
			Expect(data).Should(HaveKeyWithValue("status", 400.0))
			Expect(data).Should(HaveKeyWithValue("title", "Invalid challenge signature"))
			prefix := base64.RawStdEncoding.EncodeToString([]byte(altAddr))
			bytes := data["challenge_bytes"].(string)
			Expect(strings.HasPrefix(bytes, prefix)).Should(BeTrueBecause("%s should start with %s", bytes, prefix))
		})
		It("should update instance", func() {
			pubkey, mockAddr, hits, server := SetupMockServer(extKeys[3])
			defer server.Close()
			yes := true
			req := client.InstanceAPI.IntroduceInstance(ctx).IntroduceInstanceRequest(
				*&openapi.IntroduceInstanceRequest{
					PublicKey: pubkey,
					Address:   mockAddr,
					Update:    &yes,
				},
			)
			resp, err := req.Execute()
			Expect(err).Should(HaveOccurred())
			Expect(resp.StatusCode).Should(Equal(409))
			Expect(hits.Load()).Should(Equal(int32(1)))
			var data map[string]any
			json.Unmarshal(err.(*openapi.GenericOpenAPIError).Body(), &data)
			Expect(data).Should(HaveKeyWithValue("type", "/v0/problems/no-effect-update"))
			Expect(data).Should(HaveKeyWithValue("title", "The update had no effect"))
			Expect(data).Should(HaveKeyWithValue("status", 409.0))
		})
		It("should reject unreachable instance", func() {
			pub := base64.RawURLEncoding.EncodeToString(extKeys[3].Public().(ed25519.PublicKey))
			resp, err := client.InstanceAPI.IntroduceInstance(ctx).IntroduceInstanceRequest(
				// if 4242 it responds, it means the instance is non compliant lol
				*openapi.NewIntroduceInstanceRequest(pub, "localhost:4242"),
			).Execute()
			Expect(resp.StatusCode).Should(Equal(400))
			var data map[string]any
			json.Unmarshal(err.(*openapi.GenericOpenAPIError).Body(), &data)
			Expect(data).Should(HaveKeyWithValue("type", "/v0/problems/federation/instance-unreachable"))
			Expect(data).Should(HaveKeyWithValue("title", "Cannot reach the introduced instance"))
			Expect(data).Should(HaveKeyWithValue("status", 400.0))
			Expect(data).Should(HaveKeyWithValue("address", "localhost:4242"))
		})
	})
	When("Instance is compliance and using alternate key", func() {
		var mockAddr string
		var server *httptest.Server
		var pubkey string
		var hits *atomic.Int32
		BeforeAll(func() {
			pubkey, mockAddr, hits, server = SetupMockServer(extKeys[2])
		})
		AfterAll(func() {
			server.Close()
		})
		It("should identify instance with key 2", func() {
			resp, err := client.InstanceAPI.IntroduceInstance(ctx).IntroduceInstanceRequest(
				*openapi.NewIntroduceInstanceRequest(pubkey, mockAddr),
			).Execute()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(resp.StatusCode).Should(Equal(200))
			Expect(hits.Load()).Should(Equal(int32(1)))
		})
		It("should respond with instance", func() {
			oai, _, err := client.InstanceAPI.LookupInstanceAddress(ctx).Execute()
			Expect(err).ShouldNot(HaveOccurred())
			oai, resp, err := client.InstanceAPI.LookupInstanceAddress(ctx).
				PublicKey(base64.RawURLEncoding.EncodeToString(extKeys[2].Public().(ed25519.PublicKey))).
				Execute()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(resp.Header.Get("content-type")).Should(Equal("application/json; charset=utf-8"))
			Expect(resp.StatusCode).Should(Equal(200))
			Expect(*oai.LookupInstanceAddress200ResponseOneOf.Instance.Address.Get()).Should(Equal(mockAddr))
		})
	})
})
