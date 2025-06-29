package tests

import (
	"context"
	"encoding/json"
	"log"

	// "net/http/httptest"
	// "sync/atomic"

	openapi "github.com/DFMailbox/go-client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go/modules/compose"
)

var _ = Describe("Registering plots", Ordered, func() {
	var ctx context.Context
	var client *openapi.APIClient
	var stack *compose.DockerCompose
	BeforeAll(func() {
		env := ReadEnv()
		s, port, err := SetupDefault(env.composePath)
		stack = s
		Expect(err).ShouldNot(HaveOccurred())

		ctx = SetupContex(port)

		config := openapi.NewConfiguration()
		client = openapi.NewAPIClient(config)
	})
	AfterAll(func() {
		Teardown(stack)
	})

	Describe("Internal plots", Ordered, func() {
		It("will register Notch", func() {
			RegisterCheckPlot(client, ctx, "Notch", "069a79f4-44e9-4726-a5be-fca90e38aaf5", 123)
		})
		It("will register Jeb_", func() {
			RegisterCheckPlot(client, ctx, "jeb_", "853c80ef-3c37-49fd-aa49-938b674adae6", 456)
		})
		It("will register dinnerbone", func() {
			RegisterCheckPlot(client, ctx, "dinnerbone", "61699b2e-d327-4a01-9f1e-0ea8c3f06bc6", 2147483647)
		})
	})
	Describe("External plots", Ordered, func() {
		It("will fail to register unidentfied instance", func() {
			ctx := AddPlotAuth(ctx, "NOTCH", 666)
			pubKey := "UUt_RAzgNOQlxrUsRqpei5HdCXLCTIiyY1FjX5hd2DA="
			resp, err := client.PlotAPI.RegisterPlot(ctx).UpdateInstanceRequest(
				*openapi.NewUpdateInstanceRequest(*openapi.NewNullableString(&pubKey)),
			).Execute()
			Expect(err).Should(HaveOccurred())
			Expect(resp.StatusCode).Should(Equal(409))
			var data map[string]any
			json.Unmarshal(err.(*openapi.GenericOpenAPIError).Body(), &data)
			Expect(data).Should(HaveKeyWithValue("type", "/v0/problems/unknown-instance"))
			Expect(data).Should(HaveKeyWithValue("title", "Specified instance has not been identified"))
			Expect(data).Should(HaveKeyWithValue("status", 409.0))
			Expect(data).Should(HaveKeyWithValue("public_key", pubKey))

		})
		It("will return unregistered error", func() {
			ctx := AddPlotAuth(ctx, "NOTCH", 666)
			plot, resp, err := client.PlotAPI.GetPlotInfo(ctx).Execute()
			Expect(err).Should(HaveOccurred())
			Expect(resp.StatusCode).Should(Equal(403))
			Expect(plot).Should(BeNil())
			var data map[string]any
			json.Unmarshal(err.(*openapi.GenericOpenAPIError).Body(), &data)
			Expect(data).Should(HaveKeyWithValue("type", "/v0/problems/expected-role/any"))
			Expect(data).Should(HaveKeyWithValue("title", "Expected any registration"))
			Expect(data).Should(HaveKeyWithValue("status", 403.0))
			Expect(data).Should(HaveKeyWithValue("expected", ContainElements("host", "registered")))
			Expect(data).Should(HaveKeyWithValue("received", "unregistered"))
		})
	})

	// TODO: /v0/problems/instance-key-compromised (POST, PUT /plot)
	// DELETE /plot
	Describe("Client refuses to use auth", func() {
		It("will not be authorized when getting own plot info", func() {
			plot, resp, err := client.PlotAPI.GetPlotInfo(ctx).Execute()
			Expect(err).Should(HaveOccurred())
			Expect(resp.StatusCode).Should(Equal(401))
			Expect(plot).Should(BeNil())
			var data map[string]any
			json.Unmarshal(err.(*openapi.GenericOpenAPIError).Body(), &data)
			Expect(data).Should(HaveKeyWithValue("type", "https://tools.ietf.org/html/rfc9110#section-15.5.2"))
			Expect(data).Should(HaveKeyWithValue("title", "Unauthorized"))
			Expect(data).Should(HaveKeyWithValue("status", 401.0))
		})
		It("will not be authorized when registering a plot", func() {
			resp, err := client.PlotAPI.RegisterPlot(ctx).UpdateInstanceRequest(
				*openapi.NewUpdateInstanceRequest(*openapi.NewNullableString(nil)),
			).Execute()
			var data map[string]any
			log.Printf("%+v", err)
			json.Unmarshal(err.(*openapi.GenericOpenAPIError).Body(), &data)

			Expect(resp.StatusCode).Should(Equal(401))
			Expect(err).Should(HaveOccurred())
			Expect(data).Should(HaveKeyWithValue("type", "https://tools.ietf.org/html/rfc9110#section-15.5.2"))
			Expect(data).Should(HaveKeyWithValue("title", "Unauthorized"))
			Expect(data).Should(HaveKeyWithValue("status", 401.0))
		})
		It("will not be authorized when updating a plot", func() {
			resp, err := client.PlotAPI.UpdateInstance(ctx).UpdateInstanceRequest(
				*openapi.NewUpdateInstanceRequest(*openapi.NewNullableString(nil)),
			).Execute()
			Expect(err).Should(HaveOccurred())
			Expect(resp.StatusCode).Should(Equal(401))
			var data map[string]any
			json.Unmarshal(err.(*openapi.GenericOpenAPIError).Body(), &data)
			Expect(data).Should(HaveKeyWithValue("type", "https://tools.ietf.org/html/rfc9110#section-15.5.2"))
			Expect(data).Should(HaveKeyWithValue("title", "Unauthorized"))
			Expect(data).Should(HaveKeyWithValue("status", 401.0))
		})
	})
	/*
		Describe("External plots", func() {
			var mockAddr string
			var server *httptest.Server
			var pubkey string
			var hits *atomic.Int32
			BeforeAll(func() {
				pubkey, mockAddr, hits, server = SetupMockServer(extKeys[1])
			})
			AfterAll(func() {
				server.Close()
			})

		})
	*/
})

func RegisterCheckPlot(client *openapi.APIClient, ctx1 context.Context, username string, uuid string, plotId int32) {
	ctx := AddPlotAuth(ctx1, username, plotId)
	log.Printf("Context value: %+v", ctx.Value(openapi.ContextServerVariables))
	resp, err := client.PlotAPI.RegisterPlot(ctx).UpdateInstanceRequest(
		*openapi.NewUpdateInstanceRequest(*openapi.NewNullableString(nil)),
	).Execute()
	Expect(err).ShouldNot(HaveOccurred())
	Expect(resp.StatusCode).Should(Equal(201))

	plot, resp, err := client.PlotAPI.GetPlotInfo(ctx).Execute()
	Expect(err).ShouldNot(HaveOccurred())
	Expect(resp.StatusCode).Should(Equal(200))
	Expect(*plot).Should(Equal(openapi.Plot{
		PlotId:       plotId,
		Owner:        uuid,
		PublicKey:    *openapi.NewNullableString(nil),
		Address:      *openapi.NewNullableString(nil),
		MailboxMsgId: 0,
	}))
}
