package discovery

import (
	"io"
	"log"
	"net"
	"time"

	"golang.org/x/exp/constraints"
	"golang.org/x/exp/slices"

	"github.com/google/uuid"
	ssdp "github.com/koron/go-ssdp"
)

var (
	serviceType    = "game:hearts"
	serverName     = "HeartsServer/1.0"
	serverUniqueId = uuid.NewString()
	cacheMaxAge, _ = time.ParseDuration("30m")
)

// Advertise the HeartsService via SSDP at the given hostLocation.
// Close() the returned advertiser when done.
func AdvertiseService(hostLocation string) (io.Closer, error) {
	//ssdp.Logger = log.New(os.Stderr, "[SSDP] ", log.LstdFlags)
	ad, err := ssdp.Advertise(serviceType, serverUniqueId, hostLocation, serverName, int(cacheMaxAge.Seconds()))
	if err != nil {
		return nil, err
	}
	return &advertiser{ad: ad}, nil
}

type advertiser struct {
	ad *ssdp.Advertiser
}

func (a advertiser) Close() error {
	a.ad.Bye()
	return a.ad.Close()
}

// Find any HeartsService providers on the current LAN via SSDP.
// Returns a list of host addresses.
func FindService(waitTime time.Duration) ([]string, error) {
	//ssdp.Logger = log.New(os.Stderr, "[SSDP] ", log.LstdFlags)
	//listenOnlyToEn0()
	servers, err := ssdp.Search(serviceType, int(waitTime.Seconds()), "")
	if err != nil {
		return nil, err
	}
	var locs []string
	for _, svr := range servers {
		// fmt.Printf("Found server %s/%s/%s\n", svr.Server, svr.Type, svr.Location)
		if svr.Type != serviceType {
			continue
		}
		locs = append(locs, svr.Location)
	}
	return uniq(locs), nil
}

func uniq[E constraints.Ordered](s []E) []E {
	slices.Sort(s)
	return slices.Compact(s)
}

func listenOnlyToEn0() {
	en0, err := net.InterfaceByName("en0")
	if err != nil {
		log.Printf("Can't find interface 'en0'. SSDP listening on all interfaces")
		return
	}
	ssdp.Interfaces = []net.Interface{*en0}
}
