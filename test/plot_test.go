package main

import (
	"reflect"
	"testing"

	openapi "github.com/DFMailbox/go-client"
	"github.com/stretchr/testify/assert"
)

func TestNotch(t *testing.T) {
	PlotTest(t, "Notch", "069a79f4-44e9-4726-a5be-fca90e38aaf5", 123)
}

func TestJeb(t *testing.T) {
	PlotTest(t, "jeb_", "853c80ef-3c37-49fd-aa49-938b674adae6", 456)
}
func TestHigh(t *testing.T) {
	PlotTest(t, "dinnerbone", "61699b2e-d327-4a01-9f1e-0ea8c3f06bc6", 2147483647)
}

func PlotTest(t *testing.T, username string, uuid string, plotId int32) {
	t.Parallel()
	env := ReadEnv()
	stack, port, err := SetupDefault(env.composePath)
	assert.NoError(t, err)
	defer Teardown(stack)

	ctx := SetupContex(port)
	ctx = AddPlotAuth(ctx, username, plotId)

	config := openapi.NewConfiguration()
	client := openapi.NewAPIClient(config)

	resp, err := client.PlotAPI.RegisterPlot(ctx).UpdateInstanceRequest(
		*openapi.NewUpdateInstanceRequest(*openapi.NewNullableString(nil)),
	).Execute()
	assert.NoError(t, err)
	assert.Equal(t, resp.StatusCode, 201)

	plot, resp, err := client.PlotAPI.GetPlotInfo(ctx).Execute()
	assert.NoError(t, err)
	cmpPlot := openapi.Plot{
		PlotId:       plotId,
		Owner:        uuid,
		PublicKey:    *openapi.NewNullableString(nil),
		Address:      *openapi.NewNullableString(nil),
		MailboxMsgId: 0,
	}
	assert.Equal(t, resp.StatusCode, 200)
	if !reflect.DeepEqual(*plot, cmpPlot) {
		t.Errorf("Got %+v expected %+v", plot, cmpPlot)
		return
	}
}
