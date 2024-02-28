package main

import (
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/enbility/eebus-go/api"
	"github.com/enbility/eebus-go/service"
	shipapi "github.com/enbility/ship-go/api"
	"github.com/enbility/ship-go/cert"
	"github.com/enbility/spine-go/model"
)

var remoteSki string

type evse struct {
	myService *service.Service
}

func (h *evse) run() {
	var err error
	var certificate tls.Certificate

	if len(os.Args) == 5 {
		remoteSki = os.Args[2]

		certificate, err = tls.LoadX509KeyPair(os.Args[3], os.Args[4])
		if err != nil {
			usage()
			log.Fatal(err)
		}
	} else {
		certificate, err = cert.CreateCertificate("Demo", "Demo", "DE", "Demo-Unit-02")
		if err != nil {
			log.Fatal(err)
		}

		pemdata := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: certificate.Certificate[0],
		})
		fmt.Println(string(pemdata))

		//--------------------------- ADDED CODE ---------------------------
		//write the certificate to a file
		crtFile, err := os.Create("../../eeprom/evse.crt")
		if err != nil {
			log.Fatal(err)
		}

		defer crtFile.Close()
		_, err = crtFile.Write(pemdata)
		if err != nil {
			log.Fatal(err)
		}
		//--------------------------- ADDED CODE ---------------------------

		b, err := x509.MarshalECPrivateKey(certificate.PrivateKey.(*ecdsa.PrivateKey))
		if err != nil {
			log.Fatal(err)
		}
		pemdata = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: b})
		fmt.Println(string(pemdata))

		//--------------------------- ADDED CODE ---------------------------
		//write the private key to a file
		keyFile, err := os.Create("../../eeprom/evse.key")
		if err != nil {
			log.Fatal(err)
		}

		defer keyFile.Close()
		_, err = keyFile.Write(pemdata)
		if err != nil {
			log.Fatal(err)
		}
		//--------------------------- ADDED CODE ---------------------------
	}

	port, err := strconv.Atoi(os.Args[1])
	if err != nil {
		usage()
		log.Fatal(err)
	}

	configuration, err := api.NewConfiguration(
		"Demo", "Demo", "EVSE", "234567890",
		model.DeviceTypeTypeChargingStation,
		[]model.EntityTypeType{model.EntityTypeTypeEVSE},
		port, certificate, 230, time.Second*4)
	if err != nil {
		log.Fatal(err)
	}
	configuration.SetAlternateIdentifier("Demo-EVSE-234567890")

	h.myService = service.NewService(configuration, h)
	h.myService.SetLogging(h)

	if err = h.myService.Setup(); err != nil {
		fmt.Println(err)
		return
	}

	if len(remoteSki) == 0 {
		os.Exit(0)
	}

	h.myService.RegisterRemoteSKI(remoteSki, true)

	h.myService.Start()
	// defer h.myService.Shutdown()
}

// EEBUSServiceHandler

func (h *evse) RemoteSKIConnected(service api.ServiceInterface, ski string) {}

func (h *evse) RemoteSKIDisconnected(service api.ServiceInterface, ski string) {}

func (h *evse) VisibleRemoteServicesUpdated(service api.ServiceInterface, entries []shipapi.RemoteService) {
}

func (h *evse) ServiceShipIDUpdate(ski string, shipdID string) {}

func (h *evse) ServicePairingDetailUpdate(ski string, detail *shipapi.ConnectionStateDetail) {
	if ski == remoteSki && detail.State() == shipapi.ConnectionStateRemoteDeniedTrust {
		fmt.Println("The remote service denied trust. Exiting.")
		h.myService.RegisterRemoteSKI(ski, false)
		h.myService.CancelPairingWithSKI(ski)
		h.myService.Shutdown()
		os.Exit(0)
	}
}

func (h *evse) AllowWaitingForTrust(ski string) bool {
	return ski == remoteSki
}

// main app
func usage() {
	fmt.Println("First Run:")
	fmt.Println("  go run /cmd/evse/main.go <serverport>")
	fmt.Println()
	fmt.Println("General Usage:")
	fmt.Println("  go run /cmd/evse/main.go <serverport> <hems-ski> <crtfile> <keyfile>")
}

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}

	h := evse{}
	h.run()

	// Clean exit to make sure mdns shutdown is invoked
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	// User exit
}

// Logging interface

func (h *evse) Trace(args ...interface{}) {
	h.print("TRACE", args...)
}

func (h *evse) Tracef(format string, args ...interface{}) {
	h.printFormat("TRACE", format, args...)
}

func (h *evse) Debug(args ...interface{}) {
	h.print("DEBUG", args...)
}

func (h *evse) Debugf(format string, args ...interface{}) {
	h.printFormat("DEBUG", format, args...)
}

func (h *evse) Info(args ...interface{}) {
	h.print("INFO ", args...)
}

func (h *evse) Infof(format string, args ...interface{}) {
	h.printFormat("INFO ", format, args...)
}

func (h *evse) Error(args ...interface{}) {
	h.print("ERROR", args...)
}

func (h *evse) Errorf(format string, args ...interface{}) {
	h.printFormat("ERROR", format, args...)
}

func (h *evse) currentTimestamp() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func (h *evse) print(msgType string, args ...interface{}) {
	value := fmt.Sprintln(args...)
	fmt.Printf("%s %s %s", h.currentTimestamp(), msgType, value)
}

func (h *evse) printFormat(msgType, format string, args ...interface{}) {
	value := fmt.Sprintf(format, args...)
	fmt.Println(h.currentTimestamp(), msgType, value)
}
