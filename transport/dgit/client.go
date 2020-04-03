package dgit

import (
	"context"
	"fmt"
	"path"

	logging "github.com/ipfs/go-log"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/dgit/tupelo/clientbuilder"
	"github.com/quorumcontrol/dgit/tupelo/repotree"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	tupelo "github.com/quorumcontrol/tupelo-go-sdk/gossip/client"
	"github.com/go-git/go-git/v5/plumbing/transport"
	gitclient "github.com/go-git/go-git/v5/plumbing/transport/client"
	"github.com/go-git/go-git/v5/plumbing/transport/server"
)

var log = logging.Logger("dgit.client")

type Client struct {
	transport.Transport

	ctx            context.Context
	tupelo         *tupelo.Client
	nodestore      nodestore.DagStore
	server         transport.Transport
	DefaultStorage string
}

const protocol = "dgit"

func Protocol() string {
	return protocol
}

func Default() (*Client, error) {
	client, ok := gitclient.Protocols[protocol]
	if !ok {
		return nil, fmt.Errorf("no client registered for '%s'", protocol)
	}

	asClient, ok := client.(*Client)
	if !ok {
		return nil, fmt.Errorf("%s registered %T, but is not a dgit.Client", protocol, client)
	}

	return asClient, nil
}

func NewClient(ctx context.Context, basePath string) (*Client, error) {
	var err error
	c := &Client{
		ctx:            ctx,
		DefaultStorage: "siaskynet",
	}
	dir := path.Join(basePath, protocol)
	c.tupelo, c.nodestore, err = clientbuilder.Build(ctx, dir)
	return c, err
}

// FIXME: this probably shouldn't be here
func NewLocalClient(ctx context.Context) (*Client, error) {
	var err error
	c := &Client{
		ctx:            ctx,
		DefaultStorage: "chaintree",
	}
	c.tupelo, c.nodestore, err = clientbuilder.BuildLocal(ctx)
	return c, err
}

// FIXME: this probably shouldn't be here
func (c *Client) CreateRepoTree(ctx context.Context, endpoint *transport.Endpoint, auth transport.AuthMethod, storage string) (*consensus.SignedChainTree, error) {
	if storage == "" {
		storage = c.DefaultStorage
	}
	return repotree.Create(ctx, &repotree.RepoTreeOptions{
		Name:              endpoint.Host + endpoint.Path,
		ObjectStorageType: storage,
		Client:            c.tupelo,
		NodeStore:         c.nodestore,
		Ownership:         []string{auth.String()},
	})
}

func (c *Client) FindRepoTree(ctx context.Context, repo string) (*consensus.SignedChainTree, error) {
	return repotree.Find(ctx, repo, c.tupelo)
}

func (c *Client) RegisterAsDefault() {
	gitclient.InstallProtocol(protocol, c)
}

func (c *Client) NewUploadPackSession(ep *transport.Endpoint, auth transport.AuthMethod) (transport.UploadPackSession, error) {
	loader := NewChainTreeLoader(c.ctx, c.tupelo, c.nodestore, auth)
	return server.NewServer(loader).NewUploadPackSession(ep, auth)
}

func (c *Client) NewReceivePackSession(ep *transport.Endpoint, auth transport.AuthMethod) (transport.ReceivePackSession, error) {
	loader := NewChainTreeLoader(c.ctx, c.tupelo, c.nodestore, auth)
	return server.NewServer(loader).NewReceivePackSession(ep, auth)
}
