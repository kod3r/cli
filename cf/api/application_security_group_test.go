package api_test

import (
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/cloudfoundry/cli/cf/configuration"
	"github.com/cloudfoundry/cli/cf/net"
	testapi "github.com/cloudfoundry/cli/testhelpers/api"
	testconfig "github.com/cloudfoundry/cli/testhelpers/configuration"
	testnet "github.com/cloudfoundry/cli/testhelpers/net"

	. "github.com/cloudfoundry/cli/cf/api"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("app security group api", func() {
	var (
		testServer  *httptest.Server
		testHandler *testnet.TestHandler
		configRepo  configuration.ReadWriter
		repo        ApplicationSecurityGroupRepo
	)

	BeforeEach(func() {
		configRepo = testconfig.NewRepositoryWithDefaults()
		gateway := net.NewCloudControllerGateway((configRepo), time.Now)
		repo = NewApplicationSecurityGroupRepo(configRepo, gateway)
	})

	AfterEach(func() {
		testServer.Close()
	})

	setupTestServer := func(reqs ...testnet.TestRequest) {
		testServer, testHandler = testnet.NewServer(reqs)
		configRepo.SetApiEndpoint(testServer.URL)
	}

	It("can create an app security group, given some attributes", func() {
		req := testapi.NewCloudControllerTestRequest(testnet.TestRequest{
			Method: "POST",
			Path:   "/v2/app_security_groups",
			// FIXME: this matcher depend on the order of the key/value pairs in the map
			Matcher: testnet.RequestBodyMatcher(`{
				"name": "mygroup",
				"rules": [{"my-house": "my-rules"}],
				"space_guids": ["myspace"]
			}`),
			Response: testnet.TestResponse{Status: http.StatusCreated},
		})
		setupTestServer(req)

		err := repo.Create(ApplicationSecurityGroupFields{
			Name:       "mygroup",
			Rules:      []map[string]string{{"my-house": "my-rules"}},
			SpaceGuids: []string{"myspace"},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(testHandler).To(testnet.HaveAllRequestsCalled())
	})
})