package coreapi

import (
	"context"
	"fmt"
	gopath "path"

	"github.com/ipfs/go-ipfs/namesys/resolve"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-fetcher"
	ipld "github.com/ipfs/go-ipld-format"
	ipfspath "github.com/ipfs/go-path"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
	path "github.com/ipfs/interface-go-ipfs-core/path"
	ipldprime "github.com/ipld/go-ipld-prime"
)

// ResolveNode resolves the path `p` using Unixfs resolver, gets and returns the
// resolved Node.
func (api *CoreAPI) ResolveNode(ctx context.Context, p path.Path) (ipld.Node, error) {
	rp, err := api.ResolvePath(ctx, p)
	if err != nil {
		return nil, err
	}

	node, err := api.dag.Get(ctx, rp.Cid())
	if err != nil {
		return nil, err
	}
	return node, nil
}

// ResolvePath resolves the path `p` using Unixfs resolver, returns the
// resolved path.
func (api *CoreAPI) ResolvePath(ctx context.Context, p path.Path) (path.Resolved, error) {
	if _, ok := p.(path.Resolved); ok {
		return p.(path.Resolved), nil
	}
	if err := p.IsValid(); err != nil {
		return nil, err
	}

	ipath := ipfspath.Path(p.String())
	ipath, err := resolve.ResolveIPNS(ctx, api.namesys, ipath)
	if err == resolve.ErrNoNamesys {
		return nil, coreiface.ErrOffline
	} else if err != nil {
		return nil, err
	}

	if ipath.Segments()[0] != "ipfs" && ipath.Segments()[0] != "ipld" {
		return nil, fmt.Errorf("unsupported path namespace: %s", p.Namespace())
	}
	resolver := *api.resolver
	if ipath.Segments()[0] == "ipld" {
		type reifiable interface {
			WithReifier(nr ipldprime.NodeReifier) fetcher.Factory
		}
		if bsf, ok := resolver.FetcherFactory.(reifiable); ok {
			resolver.FetcherFactory = bsf.WithReifier(nil)
		}
	}

	node, rest, err := resolver.ResolveToLastNode(ctx, ipath)
	if err != nil {
		return nil, err
	}

	root, err := cid.Parse(ipath.Segments()[1])
	if err != nil {
		return nil, err
	}

	return path.NewResolvedPath(ipath, node, root, gopath.Join(rest...)), nil
}
