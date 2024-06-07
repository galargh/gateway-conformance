package tests

import (
	"testing"

	"github.com/ipfs/gateway-conformance/tooling"
	"github.com/ipfs/gateway-conformance/tooling/car"
	. "github.com/ipfs/gateway-conformance/tooling/check"
	"github.com/ipfs/gateway-conformance/tooling/dnslink"
	"github.com/ipfs/gateway-conformance/tooling/specs"
	. "github.com/ipfs/gateway-conformance/tooling/test"
)

func TestDNSLinkGatewayUnixFSDirectoryListing(t *testing.T) {
	tooling.LogTestGroup(t, GroupDNSLink)

	fixture := car.MustOpenUnixfsCar("dir_listing/fixtures.car")
	file := fixture.MustGetNode("ą", "ę", "file-źł.txt")

	dnsLinks := dnslink.MustOpenDNSLink("dir_listing/dnslink.yml")
	dnsLink := dnsLinks.MustGet("dir-listing-website")

	tests := SugarTests{
		{
			Name: "Backlink on root CID should be hidden (TODO: cleanup Kubo-specifics)",
			Request: Request().
				Path("/").
				Header("Host", dnsLink),
			Response: Expect().
				Body(
					And(
						Contains("Index of"),
						Not(Contains(`<a href="/">..</a>`)),
					),
				),
		},
		{
			Name: "Redirect dir listing to URL with trailing slash",
			Request: Request().
				Path("/ą/ę").
				Header("Host", dnsLink),
			Response: Expect().
				Status(301).
				Headers(
					Header("Location").Equals(`/%c4%85/%c4%99/`),
				),
		},
		{
			Name: "Regular dir listing (TODO: cleanup Kubo-specifics)",
			Request: Request().
				Path("/ą/ę/").
				Header("Host", dnsLink),
			Response: Expect().
				Headers(
					Header("Etag").Contains(`"DirIndex-`),
				).
				BodyWithHint(`
					- backlink on subdirectory should point at parent directory (TODO:  kubo-specific)
					- breadcrumbs should point at content root mounted at dnslink origin (TODO:  kubo-specific)
					- name column should be a link to content root mounted at dnslink origin
					- hash column should be a CID link to cid.ipfs.tech
					  DNSLink websites don't have public gateway mounted by default
					  See: https://github.com/ipfs/dir-index-html/issues/42 (TODO: class and other attrs are kubo-specific)
					`,
					And(
						Contains("Index of"),
						Contains(`<a href="/%C4%85/%C4%99/..">..</a>`),
						Contains(`/ipns/<a href="//{{hostname}}/">{{hostname}}</a>/<a href="//{{hostname}}/%C4%85">ą</a>/<a href="//{{hostname}}/%C4%85/%C4%99">ę</a>`, dnsLink),
						Contains(`<a href="/%C4%85/%C4%99/file-%C5%BA%C5%82.txt">file-źł.txt</a>`),
						Contains(`<a class="ipfs-hash" translate="no" href="https://cid.ipfs.tech/#{{cid}}" target="_blank" rel="noreferrer noopener">`, file.Cid()),
					),
				),
		},
	}

	RunWithSpecs(t, tests, specs.DNSLinkGateway)
}
