package tdp

import (
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/alternative-storage/torus"
	"github.com/alternative-storage/torus/distributor/protocols"
)

const defaultPort = "40000"

func init() {
	protocols.RegisterRPCListener("tdp", tdpRPCListener)
	protocols.RegisterRPCDialer("tdp", tdpRPCDialer)
}

func tdpRPCListener(url *url.URL, handler protocols.RPC, gmd torus.GlobalMetadata) (protocols.RPCServer, error) {
	if strings.Contains(url.Host, ":") {
		return Serve(url.Host, handler, gmd.BlockSize)
	}
	return Serve(net.JoinHostPort(url.Host, defaultPort), handler, gmd.BlockSize)
}

func tdpRPCDialer(url *url.URL, timeout time.Duration, gmd torus.GlobalMetadata) (protocols.RPC, error) {
	if strings.Contains(url.Host, ":") {
		return Dial(url.Host, timeout, gmd.BlockSize)
	}
	return Dial(net.JoinHostPort(url.Host, defaultPort), timeout, gmd.BlockSize)
}
