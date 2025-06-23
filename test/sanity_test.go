package tests

import (
	"fmt"
	"io"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go/log"
)

var _ = Describe("Running test stack", func() {
	It("returns the service name", func() {
		env := ReadEnv()
		stack, natPort, err := SetupDefault(env.composePath)
		Expect(err).Should(BeNil())
		port := natPort.Port()
		log.Printf("Address: localhost:%s", port)
		res, err := http.Get(fmt.Sprintf("http://localhost:%s", port))
		Expect(err).Should(BeNil())
		body, err := io.ReadAll(res.Body)
		Expect(err).Should(BeNil())
		defer res.Body.Close()
		Expect(string(body)).Should(Equal("dfmailbox"))
		defer Teardown(stack)
	})
})
