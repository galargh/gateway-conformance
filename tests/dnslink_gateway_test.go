package tests

import (
	"net/url"
	"testing"

	"github.com/ipfs/gateway-conformance/tooling"
	"github.com/ipfs/gateway-conformance/tooling/car"
	. "github.com/ipfs/gateway-conformance/tooling/check"
	"github.com/ipfs/gateway-conformance/tooling/dnslink"
	"github.com/ipfs/gateway-conformance/tooling/helpers"
	"github.com/ipfs/gateway-conformance/tooling/specs"
	. "github.com/ipfs/gateway-conformance/tooling/test"
	"github.com/ipfs/gateway-conformance/tooling/tmpl"
)

func TestDNSLinkGatewayUnixFSDirectoryListing(t *testing.T) {
	tooling.LogTestGroup(t, GroupDNSLink)

	fixture := car.MustOpenUnixfsCar("dir_listing/fixtures.car")
	file := fixture.MustGetNode("ą", "ę", "file-źł.txt")

	// DNSLink domain and fixture we will be using for Host headerthis test
	dnsLinkHostname := "dnslink-website.example.org"
	dnsLinks := dnslink.MustOpenDNSLink("dir_listing/dnslink.yml")
	dnsLink := dnsLinks.MustGet("dir-listing-website")

	tests := SugarTests{}

	// Sent requests to endpoint defined by --gateway-url
	u, err := url.Parse(GatewayURL)
	if err != nil {
		t.Fatal(err)
	}

	// Host header should use dnslink domain with the same scheme as --gateway-url
	hostHdr := tmpl.Fmt("{{scheme}}://{{dnslink}}", u.Scheme, dnsLink)

	tests = append(tests, SugarTests{
		{
			Name: "Backlink on root CID should be hidden (TODO: cleanup Kubo-specifics)",
			Request: Request().
				Header("Host", hostHdr).
				URL(GatewayURL),
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
				Header("Host", hostHdr).
				URL(GatewayURL + "/ą/ę"),
			Response: Expect().
				Status(301).
				Headers(
					Header("Location").Equals(`/%c4%85/%c4%99/`),
				),
		},
		{
			Name: "Regular dir listing (TODO: cleanup Kubo-specifics)",
			Request: Request().
				Header("Host", hostHdr).
				URL(GatewayURL + "/ą/ę"),
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
						Contains(`/ipns/<a href="//{{hostname}}/">{{hostname}}</a>/<a href="//{{hostname}}/%C4%85">ą</a>/<a href="//{{hostname}}/%C4%85/%C4%99">ę</a>`, dnsLinkHostname),
						Contains(`<a href="/%C4%85/%C4%99/file-%C5%BA%C5%82.txt">file-źł.txt</a>`),
						Contains(`<a class="ipfs-hash" translate="no" href="https://cid.ipfs.tech/#{{cid}}" target="_blank" rel="noreferrer noopener">`, file.Cid()),
					),
				),
		},
	}...)

	RunWithSpecs(t, helpers.UnwrapSubdomainTests(t, tests), specs.DNSLinkGateway)
}
