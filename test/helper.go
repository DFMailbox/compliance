package tests

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"sync/atomic"

	openapi "github.com/DFMailbox/go-client"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go/modules/compose"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Pretty please don't mutate this
var keys []string = []string{
	"TESTING0KEYTESTING0KEYTESTING0KEYTESTING000=",
	"TESTING0KEYTESTING0KEYTESTING0KEYTESTING001=",
	"TESTING0KEYTESTING0KEYTESTING0KEYTESTING002=",
	"TESTING0KEYTESTING0KEYTESTING0KEYTESTING003=",
	"TESTING0KEYTESTING0KEYTESTING0KEYTESTING004=",
	"TESTING0KEYTESTING0KEYTESTING0KEYTESTING005=",
	"TESTING0KEYTESTING0KEYTESTING0KEYTESTING006=",
}

var extKeys []ed25519.PrivateKey
var uuidRegex = regexp.MustCompile(`^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[1-5][a-fA-F0-9]{3}-[89abAB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$`)

func init() {
	sKeys := []string{
		"TESTING0KEYTESTINGju9ahiajieng6ohRiu1Ie1zae=",
		"TESTING0KEYTESTINGeR4haaxohghai2Eeg5een1AhV=",
		"TESTING0KEYTESTINGeu6ti3Ohpei7Eej7feid3te7q=",
		"TESTING0KEYTESTINGShaeni7aeLee3ahjeeghaidon=",
		"TESTING0KEYTESTINGeiz5iu8bangeXahcie8Idoobe=",
		"TESTING0KEYTESTINGJlei9ohshoJ5Ey9Busiu4OR6a=",
		"TESTING0KEYTESTINGzohran2aevohyaime5toh6AhT=",
	}
	keyList := make([]ed25519.PrivateKey, len(sKeys))

	for i, str := range sKeys {
		key, err := base64.StdEncoding.DecodeString(str)
		if err != nil {
			panic("key is invalid base64")
		}
		if len(key) != ed25519.SeedSize {
			panic(fmt.Sprintf("key is invalid length %d", len(key)))
		}
		keyList[i] = ed25519.NewKeyFromSeed(key)
	}
	extKeys = keyList
}

func ReadEnv() Environment {
	path, set := os.LookupEnv("DFMC_COMPOSE_FILE")
	if set == false {
		path = "../../compliance-docker-compose.yml"
	}
	return Environment{
		HostGateway: os.Getenv("DFMC_HOST_GATEWAY"),
		ComposePath: path,
	}
}

type Environment struct {
	ComposePath string
	HostGateway string
}

func SetupDefault() (*compose.DockerCompose, *nat.Port, error) {
	env := ReadEnv()
	return Setup(env.ComposePath, map[string]string{
		"DFMC_ADDRESS":      "dfm.example.com",
		"DFMC_PRIVATE_KEY":  keys[0],
		"DFMC_HOST_GATEWAY": env.HostGateway,
	})
}

func Setup(file_path string, env map[string]string) (*compose.DockerCompose, *nat.Port, error) {
	ctx := context.Background()
	stack, err := compose.NewDockerComposeWith(
		// compose.StackIdentifier("dfm_compliance"),
		compose.WithStackFiles(file_path),
	)
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("Failed to create stack %v", err))
	}
	err = stack.WithEnv(env).
		WaitForService("dfmailbox", wait.ForExposedPort()).
		Up(ctx, compose.Wait(true))
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("Failed to start compose stack %v", err))
	}
	container, err := stack.ServiceContainer(ctx, "dfmailbox")
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("Failed to find container %v", err))
	}
	port, err := container.MappedPort(ctx, "8080/tcp")
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("Failed to find container endpoint %v", err))
	}
	return stack, &port, nil
}

func Teardown(stack *compose.DockerCompose) {
	err := stack.Down(
		context.Background(),
		compose.RemoveOrphans(true),
		compose.RemoveImagesLocal,
	)
	if err != nil {
		log.Printf("Failed to start stack: %v", err)

	}
}

func AddPlotAuth(ctx context.Context, name string, plotId int32) context.Context {
	return context.WithValue(ctx, openapi.ContextAPIKeys, map[string]openapi.APIKey{
		"Plot": {Key: fmt.Sprintf("Hypercube/7.2 (%d, %s)", plotId, name)},
	})
}

func StrAsRef(s string) *string { return &s }

func SetupContex(port *nat.Port) context.Context {
	ctx := context.WithValue(context.Background(), openapi.ContextServerVariables, map[string]string{
		"port": port.Port(),
	})
	return context.WithValue(ctx, openapi.ContextServerIndex, 0)
}

func SetupMockServer(key ed25519.PrivateKey) (string, string, *atomic.Int32, *httptest.Server) {
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
