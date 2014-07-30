package actors_test

import (
	"github.com/cloudfoundry/cli/cf/actors"
	"github.com/cloudfoundry/cli/cf/api/fakes"
	"github.com/cloudfoundry/cli/cf/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Services", func() {
	var (
		actor                     actors.ServiceActor
		brokerRepo                *fakes.FakeServiceBrokerRepo
		serviceRepo               *fakes.FakeServiceRepo
		servicePlanRepo           *fakes.FakeServicePlanRepo
		servicePlanVisibilityRepo *fakes.FakeServicePlanVisibilityRepository
		orgRepo                   *fakes.FakeOrgRepository
	)

	BeforeEach(func() {
		brokerRepo = &fakes.FakeServiceBrokerRepo{}
		serviceRepo = &fakes.FakeServiceRepo{}
		servicePlanRepo = &fakes.FakeServicePlanRepo{}
		servicePlanVisibilityRepo = &fakes.FakeServicePlanVisibilityRepository{}
		orgRepo = &fakes.FakeOrgRepository{}

		actor = actors.NewServiceHandler(brokerRepo, serviceRepo, servicePlanRepo, servicePlanVisibilityRepo, orgRepo)

		serviceBroker1 := models.ServiceBroker{Guid: "my-service-broker-guid", Name: "my-service-broker"}
		serviceBroker2 := models.ServiceBroker{Guid: "my-service-broker-guid2", Name: "my-service-broker2"}

		brokerRepo.FindByNameServiceBroker = serviceBroker2

		brokerRepo.ServiceBrokers = []models.ServiceBroker{
			serviceBroker1,
			serviceBroker2,
		}

		serviceRepo.ListServicesFromBrokerReturns = map[string][]models.ServiceOffering{
			"my-service-broker-guid": {},
			"my-service-broker-guid2": {
				{ServiceOfferingFields: models.ServiceOfferingFields{Guid: "service-guid", Label: "my-service"}},
				{ServiceOfferingFields: models.ServiceOfferingFields{Guid: "service-guid2", Label: "my-service2"}},
			},
		}

		service2 := models.ServiceOffering{ServiceOfferingFields: models.ServiceOfferingFields{Label: "my-service2", Guid: "service-guid2"}}

		serviceRepo.FindServiceOfferingByLabelServiceOffering = service2

		servicePlanRepo.SearchReturns = map[string][]models.ServicePlanFields{
			"service-guid": {{Name: "service-plan", Guid: "service-plan-guid", ServiceOfferingGuid: "service-guid"},
				{Name: "other-plan", Guid: "other-plan-guid", ServiceOfferingGuid: "service-guid", Public: true}},
			"service-guid2": {{Name: "service-plan2", Guid: "service-plan2-guid", ServiceOfferingGuid: "service-guid2"}},
		}

		servicePlanVisibilityRepo.ListReturns([]models.ServicePlanVisibilityFields{
			{ServicePlanGuid: "service-plan2-guid", OrganizationGuid: "org-guid"},
			{ServicePlanGuid: "service-plan-guid", OrganizationGuid: "org-guid"},
			{ServicePlanGuid: "service-plan2-guid", OrganizationGuid: "org2-guid"},
			{ServicePlanGuid: "other-plan-guid", OrganizationGuid: "org-guid"},
		}, nil)

		org1 := models.Organization{}
		org1.Name = "org1"
		org1.Guid = "org-guid"

		org2 := models.Organization{}
		org2.Name = "org2"
		org2.Guid = "org2-guid"

		orgRepo.Organizations = []models.Organization{
			org1,
			org2,
		}
	})

	Describe("GetBrokersWithVisibilityFromASingleOrg", func() {
		It("Returns a slice of brokers containing Services/Service Plans visible to the org", func() {
			servicePlanRepo.SearchReturns = map[string][]models.ServicePlanFields{
				"service-guid": {{Name: "service-plan", Guid: "service-plan-guid", ServiceOfferingGuid: "service-guid"},
					{Name: "other-plan", Guid: "other-plan-guid", ServiceOfferingGuid: "service-guid", Public: true}},
			}
			serviceRepo.GetServiceOfferingByGuidReturns.ServiceOffering = models.ServiceOffering{
				ServiceOfferingFields: models.ServiceOfferingFields{Guid: "service-guid", Label: "my-service", BrokerGuid: "my-service-broker-guid"},
			}
			serviceRepo.GetServiceOfferingByGuidReturns.Error = nil
			brokerRepo.FindByGuidServiceBroker = models.ServiceBroker{Guid: "my-service-broker-guid", Name: "my-service-broker"}

			brokers, err := actor.GetBrokersWithVisibilityFromASingleOrg("org1")
			Expect(err).NotTo(HaveOccurred())

			Expect(len(brokers)).To(Equal(1))
			Expect(len(brokers[0].Services)).To(Equal(1))
			Expect(len(brokers[0].Services[0].Plans)).To(Equal(2))

			Expect(brokers[0].Services[0].Guid).To(Equal("service-guid"))
			Expect(brokers[0].Services[0].Plans[0].Name).To(Equal("service-plan"))
			Expect(brokers[0].Services[0].Plans[0].OrgNames).To(Equal([]string{"org1"}))
			Expect(brokers[0].Services[0].Plans[1].Name).To(Equal("other-plan"))
		})

		It("ignores any service that does not have a a broker guid attached", func() {
			//The service offering fixtures we use don't have broker guids attached, and thus we ignore them.
			brokers, err := actor.GetBrokersWithVisibilityFromASingleOrg("org1")
			Expect(err).NotTo(HaveOccurred())

			Expect(len(brokers)).To(Equal(0))
		})
	})

	Describe("GetBrokerWithSingleService", func() {
		It("Returns a single broker contained in a slice with all dependencies populated", func() {
			brokers, err := actor.GetBrokerWithSingleService("my-service2")
			Expect(err).NotTo(HaveOccurred())

			Expect(len(brokers)).To(Equal(1))
			Expect(len(brokers[0].Services)).To(Equal(1))

			Expect(brokers[0].Services[0].Guid).To(Equal("service-guid2"))
			Expect(brokers[0].Services[0].Plans[0].Name).To(Equal("service-plan2"))
			Expect(brokers[0].Services[0].Plans[0].OrgNames).To(Equal([]string{"org1", "org2"}))
		})
	})

	Describe("GetBrokerWithDependencies", func() {
		It("Returns a single broker contained in a slice with all dependencies populated", func() {
			brokers, err := actor.GetBrokerWithDependencies("my-service-broker2")
			Expect(err).NotTo(HaveOccurred())

			Expect(len(brokers)).To(Equal(1))
			Expect(len(brokers[0].Services)).To(Equal(2))

			Expect(brokers[0].Services[0].Guid).To(Equal("service-guid"))
			Expect(brokers[0].Services[0].Plans[0].Name).To(Equal("service-plan"))
			Expect(brokers[0].Services[0].Plans[1].Name).To(Equal("other-plan"))
			Expect(brokers[0].Services[1].Guid).To(Equal("service-guid2"))
			Expect(brokers[0].Services[1].Plans[0].Name).To(Equal("service-plan2"))
			Expect(brokers[0].Services[1].Plans[0].OrgNames).To(Equal([]string{"org1", "org2"}))
		})

	})

	Describe("GetAllBrokersWithDependencies", func() {
		It("Returns a slice of all brokers with all their dependencies populated", func() {
			brokers, err := actor.GetAllBrokersWithDependencies()
			Expect(err).NotTo(HaveOccurred())

			Expect(len(brokers)).To(Equal(2))
			Expect(len(brokers[0].Services)).To(Equal(0))
			Expect(len(brokers[1].Services)).To(Equal(2))

			Expect(brokers[1].Services[0].Guid).To(Equal("service-guid"))
			Expect(brokers[1].Services[0].Plans[0].Name).To(Equal("service-plan"))
			Expect(brokers[1].Services[0].Plans[1].Name).To(Equal("other-plan"))
			Expect(brokers[1].Services[1].Guid).To(Equal("service-guid2"))
			Expect(brokers[1].Services[1].Plans[0].Name).To(Equal("service-plan2"))
			Expect(brokers[1].Services[1].Plans[0].OrgNames).To(Equal([]string{"org1", "org2"}))
		})
	})
})
