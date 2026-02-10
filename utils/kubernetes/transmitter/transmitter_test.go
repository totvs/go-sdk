package transmitter

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/totvs/go-sdk/utils/pipeline"
)

func TestCRDTransmitterTestSuite(t *testing.T) {
	suite.Run(t, new(CRDTransmitterTestSuite))
}

type CRDTransmitterTestSuite struct {
	suite.Suite
}

func (s *CRDTransmitterTestSuite) TestShouldPanicOnNilClient() {
	s.Panics(func() {
		NewCRDTransmitter[client.Object](nil, "spoke1", "name", &stubMapper{})
	})
}

func (s *CRDTransmitterTestSuite) TestShouldPanicOnEmptyClusterName() {
	s.Panics(func() {
		NewCRDTransmitter[client.Object](stubClient{}, "", "name", &stubMapper{})
	})
}

func (s *CRDTransmitterTestSuite) TestShouldPanicOnNilMapper() {
	s.Panics(func() {
		NewCRDTransmitter[client.Object](stubClient{}, "spoke1", "name", nil)
	})
}

// stubClient implements client.Client minimally for panic tests.
// Only needs to be non-nil; methods are not called in these tests.
type stubClient struct {
	client.Client
}

// stubMapper implements CRDMapper[client.Object] for panic tests.
type stubMapper struct{}

func (m *stubMapper) NewObject() client.Object                              { return nil }
func (m *stubMapper) MapToCreate(clusterName string) client.Object          { return nil }
func (m *stubMapper) MapToStatus(obj client.Object, report pipeline.Report) {}
