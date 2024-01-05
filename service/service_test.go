package service

import (
	"crypto/tls"
	"testing"
	"time"

	"github.com/enbility/eebus-go/logging"
	"github.com/enbility/eebus-go/logging/mocks"
	"github.com/enbility/eebus-go/spine/model"
	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestServiceSuite(t *testing.T) {
	suite.Run(t, new(ServiceSuite))
}

type ServiceSuite struct {
	suite.Suite

	config *Configuration

	sut *EEBUSService

	serviceHandler *MockEEBUSServiceHandler
	conHub         *MockConnectionsHub
	logging        *mocks.Logging
}

func (s *ServiceSuite) SetupSuite()   {}
func (s *ServiceSuite) TearDownTest() {}

func (s *ServiceSuite) BeforeTest(suiteName, testName string) {
	ctrl := gomock.NewController(s.T())

	s.serviceHandler = NewMockEEBUSServiceHandler(ctrl)

	s.conHub = NewMockConnectionsHub(ctrl)

	s.logging = mocks.NewLogging(s.T())

	certificate := tls.Certificate{}
	s.config, _ = NewConfiguration(
		"vendor", "brand", "model", "serial", model.DeviceTypeTypeEnergyManagementSystem,
		[]model.EntityTypeType{model.EntityTypeTypeCEM}, 4729, certificate, 230.0, time.Second*4)

	s.sut = NewEEBUSService(s.config, s.serviceHandler)
}

func (s *ServiceSuite) Test_EEBUSHandler() {
	testSki := "test"

	entry := &MdnsEntry{
		Ski: testSki,
	}

	entries := []*MdnsEntry{entry}
	s.serviceHandler.EXPECT().VisibleRemoteServicesUpdated(gomock.Any(), gomock.Any())
	s.sut.VisibleMDNSRecordsUpdated(entries)

	s.serviceHandler.EXPECT().RemoteSKIConnected(gomock.Any(), gomock.Any())
	s.sut.RemoteSKIConnected(testSki)

	s.serviceHandler.EXPECT().RemoteSKIDisconnected(gomock.Any(), gomock.Any())
	s.sut.RemoteSKIDisconnected(testSki)

	s.serviceHandler.EXPECT().ServiceShipIDUpdate(gomock.Any(), gomock.Any())
	s.sut.ServiceShipIDUpdate(testSki, "shipid")

	s.serviceHandler.EXPECT().ServicePairingDetailUpdate(gomock.Any(), gomock.Any())
	detail := &ConnectionStateDetail{}
	s.sut.ServicePairingDetailUpdate(testSki, detail)

	s.serviceHandler.EXPECT().AllowWaitingForTrust(gomock.Any()).Return(true)
	result := s.sut.AllowWaitingForTrust(testSki)
	assert.Equal(s.T(), true, result)

}

func (s *ServiceSuite) Test_ConnectionsHub() {
	testSki := "test"

	s.sut.connectionsHub = s.conHub

	s.conHub.EXPECT().PairingDetailForSki(gomock.Any())
	s.sut.PairingDetailForSki(testSki)

	s.conHub.EXPECT().StartBrowseMdnsSearch()
	s.sut.StartBrowseMdnsEntries()

	s.conHub.EXPECT().StopBrowseMdnsSearch()
	s.sut.StopBrowseMdnsEntries()

	s.conHub.EXPECT().ServiceForSKI(gomock.Any())
	details := s.sut.RemoteServiceForSKI(testSki)
	assert.Nil(s.T(), details)

	s.conHub.EXPECT().RegisterRemoteSKI(gomock.Any(), gomock.Any())
	s.sut.RegisterRemoteSKI(testSki, true)

	s.conHub.EXPECT().InitiatePairingWithSKI(gomock.Any())
	s.sut.InitiatePairingWithSKI(testSki)

	s.conHub.EXPECT().CancelPairingWithSKI(gomock.Any())
	s.sut.CancelPairingWithSKI(testSki)

	s.conHub.EXPECT().DisconnectSKI(gomock.Any(), gomock.Any())
	s.sut.DisconnectSKI(testSki, "reason")
}

func (s *ServiceSuite) Test_SetLogging() {
	s.sut.SetLogging(nil)
	assert.Equal(s.T(), &logging.NoLogging{}, logging.Log)

	s.sut.SetLogging(s.logging)
	assert.Equal(s.T(), s.logging, logging.Log)

	s.sut.SetLogging(&logging.NoLogging{})
	assert.Equal(s.T(), &logging.NoLogging{}, logging.Log)
}

func (s *ServiceSuite) Test_Setup() {

	err := s.sut.Setup()
	assert.NotNil(s.T(), err)

	s.config.certificate, err = CreateCertificate("unit", "org", "de", "cn")
	assert.Nil(s.T(), err)

	err = s.sut.Setup()
	assert.Nil(s.T(), err)

	s.sut.connectionsHub = s.conHub
	s.conHub.EXPECT().Start()
	s.sut.Start()

	time.Sleep(time.Millisecond * 200)

	s.conHub.EXPECT().Shutdown()
	s.sut.Shutdown()

	device := s.sut.LocalDevice()
	assert.NotNil(s.T(), device)
}