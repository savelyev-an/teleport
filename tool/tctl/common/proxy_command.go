package common

import (
	"context"
	"fmt"
	"github.com/gravitational/kingpin"
	"github.com/gravitational/teleport/lib/auth"
	"github.com/gravitational/teleport/lib/service"
	"github.com/gravitational/trace"
)

// ProxyCommand returns information about connected proxies
type ProxyCommand struct {
	config *service.Config
	lsCmd  *kingpin.CmdClause
}

// Initialize creates the proxy command and subcommands
func (p *ProxyCommand) Initialize(app *kingpin.Application, config *service.Config) {
	p.config = config

	auth := app.Command("proxy", "Operations with information for cluster proxies").Hidden()
	p.lsCmd = auth.Command("ls", "List connected auth servers")
}

// ListProxies prints currently connected proxies
func (p *ProxyCommand) ListProxies(ctx context.Context, clusterAPI auth.ClientI) error {
	proxies, err := clusterAPI.GetProxies()
	if err != nil {
		return trace.Wrap(err)
	}

	for _, proxy := range proxies {
		fmt.Printf("%s\n", proxy.GetName())
		fmt.Printf("%s\n", proxy.GetAddr())
		fmt.Printf("%s\n", proxy.GetHostname())

		fmt.Println()
	}
	return nil
}

// TryRun runs the proxy command
func (p *ProxyCommand) TryRun(ctx context.Context, cmd string, client auth.ClientI) (match bool, err error) {
	switch cmd {
	case p.lsCmd.FullCommand():
		err = p.ListProxies(ctx, client)
		return false, nil
	}
	return true, trace.Wrap(err)
}
