package tests

import (
	"context"
	"log"

	openapi "github.com/DFMailbox/go-client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go/modules/compose"
)

var _ = Describe("Registering plots", func() {
	Describe("Internal plots", Ordered, func() {
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
