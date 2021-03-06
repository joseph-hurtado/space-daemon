package textile

import (
	"context"
	"fmt"
	"os"
	"time"

	connmgr "github.com/libp2p/go-libp2p-connmgr"

	"github.com/FleekHQ/space-daemon/config"
	"github.com/FleekHQ/space-daemon/log"
	"github.com/textileio/textile/v2/cmd"
	"github.com/textileio/textile/v2/core"
)

var IpfsAddr string
var MaxThreadsConn int
var MinThreadsConn int

type TextileBuckd struct {
	textile   *core.Textile
	IsRunning bool
	Ready     chan bool
	cfg       config.Config
}

func NewBuckd(cfg config.Config) *TextileBuckd {
	return &TextileBuckd{
		Ready: make(chan bool),
		cfg:   cfg,
	}
}

func (tb *TextileBuckd) Start(ctx context.Context) error {
	IpfsAddr = tb.cfg.GetString(config.Ipfsaddr, "/ip4/127.0.0.1/tcp/5001")
	MinThreadsConn = tb.cfg.GetInt(config.MinThreadsConnection, 50)
	MaxThreadsConn = tb.cfg.GetInt(config.MaxThreadsConnection, 100)

	addrAPI := cmd.AddrFromStr(tb.cfg.GetString(config.BuckdApiMaAddr, "/ip4/127.0.0.1/tcp/3006"))
	addrAPIProxy := cmd.AddrFromStr(tb.cfg.GetString(config.BuckdApiProxyMaAddr, "/ip4/127.0.0.1/tcp/3007"))
	addrThreadsHost := cmd.AddrFromStr(tb.cfg.GetString(config.BuckdThreadsHostMaAddr, "/ip4/0.0.0.0/tcp/4006"))

	addrIpfsAPI := cmd.AddrFromStr(IpfsAddr)

	gatewayPort := tb.cfg.GetInt(config.BuckdGatewayPort, 8006)
	addrGatewayHost := cmd.AddrFromStr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", gatewayPort))
	addrGatewayURL := fmt.Sprintf("http://127.0.0.1:%d", gatewayPort)

	buckdPath := tb.cfg.GetString(config.BuckdPath, "")
	if buckdPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		buckdPath = homeDir + "/.buckd"
		log.Debug("No Buckd Path provided. Using default.", "path:"+buckdPath)
	}

	textile, err := core.NewTextile(ctx, core.Config{
		RepoPath:           buckdPath + "/repo",
		CollectionRepoPath: buckdPath + "/collections",
		AddrAPI:            addrAPI,
		AddrAPIProxy:       addrAPIProxy,
		AddrThreadsHost:    addrThreadsHost,
		AddrIPFSAPI:        addrIpfsAPI,
		AddrGatewayHost:    addrGatewayHost,
		AddrGatewayURL:     addrGatewayURL,
		//AddrPowergateAPI: addrPowergateApi,
		//UseSubdomains:    config.Viper.GetBool("gateway.subdomains"),
		//DNSDomain:        dnsDomain,
		//DNSZoneID:        dnsZoneID,
		//DNSToken:         dnsToken,
		ThreadsConnManager: connmgr.NewConnManager(MinThreadsConn, MaxThreadsConn, time.Second*20),
		Debug:              false,
	})
	if err != nil {
		return err
	}

	textile.Bootstrap()

	log.Info("Welcome to bucket", fmt.Sprintf("peerID:%s", textile.HostID().String()))

	log.Info("Sleeping for 5s to wait for buckd grpc ports to listen ...")
	time.Sleep(5 * time.Second)

	tb.textile = textile
	tb.IsRunning = true
	tb.Ready <- true
	return nil
}

func (tb *TextileBuckd) WaitForReady() chan bool {
	return tb.Ready
}

func (tb *TextileBuckd) Stop() error {
	tb.IsRunning = false
	err := tb.textile.Close(true)
	if err != nil {
		return err
	}
	return nil
}

func (tb *TextileBuckd) Shutdown() error {
	close(tb.Ready)
	return tb.Stop()
}
