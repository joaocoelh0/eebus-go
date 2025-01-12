package service

import (
	"crypto/x509"
	"errors"
	"fmt"
	"sync"

	"github.com/enbility/eebus-go/api"
	shipapi "github.com/enbility/ship-go/api"
	"github.com/enbility/ship-go/cert"
	"github.com/enbility/ship-go/hub"
	"github.com/enbility/ship-go/logging"
	"github.com/enbility/ship-go/mdns"
	spineapi "github.com/enbility/spine-go/api"
	"github.com/enbility/spine-go/model"
	"github.com/enbility/spine-go/spine"
)

// A service is the central element of an EEBUS service
// including its websocket server and a zeroconf service.
type Service struct {
	configuration *api.Configuration

	// The local service details
	localService *shipapi.ServiceDetails

	// Connection Registrations
	connectionsHub shipapi.HubInterface

	// The SPINE specific device definition
	spineLocalDevice spineapi.DeviceLocalInterface

	serviceHandler api.ServiceReaderInterface

	startOnce sync.Once
}

// creates a new EEBUS service
func NewService(configuration *api.Configuration, serviceHandler api.ServiceReaderInterface) *Service {
	return &Service{
		configuration:  configuration,
		serviceHandler: serviceHandler,
	}
}

var _ api.ServiceInterface = (*Service)(nil)

// Starts the service by initializeing mDNS and the server.
func (s *Service) Setup() error {
	sd := s.configuration

	if len(sd.Certificate().Certificate) == 0 {
		return errors.New("missing certificate")
	}

	leaf, err := x509.ParseCertificate(sd.Certificate().Certificate[0])
	if err != nil {
		return err
	}

	ski, err := cert.SkiFromCertificate(leaf)
	if err != nil {
		return err
	}

	// Initialize the local service
	// The ShipID is defined in SHIP Spec 3. as
	//   Each SHIP node has a globally unique SHIP ID. The SHIP ID is used to uniquely identify a SHIP node,
	//   e.g. in its service discovery. This ID is present in the mDNS/DNS-SD local service discovery;
	// In SHIP 13.4.6.2 the accessMethods.id is defined as
	//   The originator's unique ID
	// I assume those two to mean the same.
	// TODO: clarify
	s.localService = shipapi.NewServiceDetails(ski)
	s.localService.SetShipID(sd.Identifier())
	s.localService.SetDeviceType(string(sd.DeviceType()))
	s.localService.SetRegisterAutoAccept(sd.RegisterAutoAccept())

	logging.Log().Info("Local SKI: ", ski)

	vendor := sd.VendorCode()
	if vendor == "" {
		vendor = sd.DeviceBrand()
	}

	serial := sd.DeviceSerialNumber()
	if serial != "" {
		serial = fmt.Sprintf("-%s", serial)
	}

	// Create the SPINE device address, according to Protocol Specification 7.1.1.2
	deviceAdress := fmt.Sprintf("d:_i:%s_%s%s", vendor, sd.DeviceModel(), serial)

	// Create the local SPINE device
	s.spineLocalDevice = spine.NewDeviceLocal(
		sd.DeviceBrand(),
		sd.DeviceModel(),
		sd.DeviceSerialNumber(),
		sd.Identifier(),
		deviceAdress,
		sd.DeviceType(),
		sd.FeatureSet(),
		sd.HeartbeatTimeout(),
	)

	// Create the device entities and add it to the SPINE device
	for _, entityType := range sd.EntityTypes() {
		entityAddressId := model.AddressEntityType(len(s.spineLocalDevice.Entities()))
		entityAddress := []model.AddressEntityType{entityAddressId}
		entity := spine.NewEntityLocal(s.spineLocalDevice, entityType, entityAddress)
		s.spineLocalDevice.AddEntity(entity)
	}

	// setup mDNS
	mdns := mdns.NewMDNS(
		s.localService.SKI(),
		sd.DeviceBrand(),
		sd.DeviceModel(),
		string(sd.DeviceType()),
		sd.Identifier(),
		sd.MdnsServiceName(),
		sd.Port(),
		sd.Interfaces(),
		sd.MdnsProviderSelection(),
	)

	// Setup connections hub with mDNS and websocket connection handling
	s.connectionsHub = hub.NewHub(s, mdns, s.configuration.Port(), s.configuration.Certificate(), s.localService)

	return nil
}

// Starts the service
func (s *Service) Start() {
	s.startOnce.Do(func() {
		s.connectionsHub.Start()
	})
}

// Shutdown all services and stop the server.
func (s *Service) Shutdown() {
	// Shut down all running connections
	s.connectionsHub.Shutdown()
}

func (s *Service) Configuration() *api.Configuration {
	return s.configuration
}

func (s *Service) LocalService() *shipapi.ServiceDetails {
	return s.localService
}

func (s *Service) LocalDevice() spineapi.DeviceLocalInterface {
	return s.spineLocalDevice
}

// Sets a custom logging implementation
// By default NoLogging is used, so no logs are printed
func (s *Service) SetLogging(logger logging.LoggingInterface) {
	if logger == nil {
		return
	}
	logging.SetLogging(logger)
}

// Get the current pairing details for a given SKI
func (s *Service) PairingDetailForSki(ski string) *shipapi.ConnectionStateDetail {
	return s.connectionsHub.PairingDetailForSki(ski)
}

// Returns the Service detail of a given remote SKI
func (s *Service) RemoteServiceForSKI(ski string) *shipapi.ServiceDetails {
	return s.connectionsHub.ServiceForSKI(ski)
}

// Sets the SKI as being paired or not
// and connect it if paired and not currently being connected
func (s *Service) RegisterRemoteSKI(ski string, enable bool) {
	s.connectionsHub.RegisterRemoteSKI(ski, enable)
}

// Close a connection to a remote SKI
func (s *Service) DisconnectSKI(ski string, reason string) {
	s.connectionsHub.DisconnectSKI(ski, reason)
}

// Triggers the pairing process for a SKI
func (s *Service) InitiateOrApprovePairingWithSKI(ski string) {
	s.connectionsHub.InitiateOrApprovePairingWithSKI(ski)
}

// Cancels the pairing process for a SKI
func (s *Service) CancelPairingWithSKI(ski string) {
	s.connectionsHub.CancelPairingWithSKI(ski)
}
