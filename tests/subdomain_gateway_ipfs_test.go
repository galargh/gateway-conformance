package tests

import (
	"testing"

	"github.com/ipfs/gateway-conformance/tooling"
	"github.com/ipfs/gateway-conformance/tooling/car"
	. "github.com/ipfs/gateway-conformance/tooling/check"
	"github.com/ipfs/gateway-conformance/tooling/specs"
	. "github.com/ipfs/gateway-conformance/tooling/test"
	. "github.com/ipfs/gateway-conformance/tooling/tmpl"
)

func TestUnixFSDirectoryListingOnSubdomainGateway(t *testing.T) {
	tooling.LogTestGroup(t, GroupUnixFS)

	fixture := car.MustOpenUnixfsCar("dir_listing/fixtures.car")
	root := fixture.MustGetNode()
	file := fixture.MustGetNode("ą", "ę", "file-źł.txt")

	tests := SugarTests{}

	// run against origins explicitly passed via --subdomain-url
	u := SubdomainGatewayURL()

	tests = append(tests, SugarTests{
		{
			Name: "backlink on root CID should be hidden (TODO: cleanup Kubo-specifics)",
			Request: Request().
				Header("Host", Fmt("{{cid}}.ipfs.{{host}}", root.Cid(), u.Host)).
				Path("/"),
			Response: Expect().
				BodyWithHint("backlink on root CID should be hidden",
					And(
						Contains("Index of"),
						Not(Contains(`<a href="/">..</a>`)),
					)),
		},
		{
			Name: "redirect dir listing to URL with trailing slash",
			Request: Request().
				Header("Host", Fmt("{{cid}}.ipfs.{{host}}", root.Cid(), u.Host)).
				Path("/ą/ę"),
			Response: Expect().
				Status(301).
				Headers(
					Header("Location").Equals(`/%c4%85/%c4%99/`),
				),
		},
		{
			Name: "Regular dir listing HTML (TODO: cleanup Kubo-specifics)",
			Request: Request().
				Header("Host", Fmt("{{cid}}.ipfs.{{host}}", root.Cid(), u.Host)).
				Path("/ą/ę/"),
			Response: Expect().
				Headers(
					Header("Etag").Contains(`"DirIndex-`),
				).BodyWithHint(`
					- backlink on subdirectory should point at parent directory (TODO:  kubo-specific)
					- breadcrumbs should leverage path-based router mounted on the parent domain (TODO:  kubo-specific)
					- name column should be a link to content root mounted at subdomain origin
					`,
				And(
					Contains("Index of"),
					Contains(
						`<a href="/%C4%85/%C4%99/..">..</a>`,
					),
					Contains(
						`/ipfs/<a href="//{{host}}/ipfs/{{cid}}">{{cid}}</a>/<a href="//{{host}}/ipfs/{{cid}}/%C4%85">ą</a>/<a href="//{{host}}/ipfs/{{cid}}/%C4%85/%C4%99">ę</a>`,
						u.Host, // We don't have a subdomain here which prevents issues with normalization and cidv0
						root.Cid(),
					),
					Contains(
						`<a href="/%C4%85/%C4%99/file-%C5%BA%C5%82.txt">file-źł.txt</a>`,
					),
					Contains(
						`<a class="ipfs-hash" translate="no" href="//{{host}}/ipfs/{{cid}}?filename=file-%25C5%25BA%25C5%2582.txt">`,
						u.Host, // We don't have a subdomain here which prevents issues with normalization and cidv0
						file.Cid(),
					),
				)),
		},
	}...)

	RunWithSpecs(t, tests, specs.SubdomainGatewayIPFS)
}

func TestGatewaySubdomains(t *testing.T) {
	tooling.LogTestGroup(t, GroupSubdomains)

	fixture := car.MustOpenUnixfsCar("subdomain_gateway/fixtures.car")

	CIDVal := string(fixture.MustGetRawData("hello-CIDv1")) // hello
	DirCID := fixture.MustGetCid("testdirlisting")
	CIDv1 := fixture.MustGetCid("hello-CIDv1")
	CIDv0 := fixture.MustGetCid("hello-CIDv0")
	CIDv0to1 := fixture.MustGetCid("hello-CIDv0to1")
	CIDv1_TOO_LONG := fixture.MustGetCid("hello-CIDv1_TOO_LONG")
	CIDWikipedia := "QmXoypizjW3WknFiJnKLwHCnL72vedxjQkDDP1mXWo6uco"

	dirWithPercentEncodedFilename := car.MustOpenUnixfsCar("path_gateway_unixfs/dir-with-percent-encoded-filename.car")
	dirWithPercentEncodedFilenameCID := dirWithPercentEncodedFilename.MustGetCid()

	tests := SugarTests{}

	// run against origins explicitly passed via --subdomain-url
	u := SubdomainGatewayURL()

	tests = append(tests, SugarTests{
		{
			Name: "request for example.com/ipfs/{cid} redirects to {cid}.ipfs.example.com",
			Hint: `
					path requests to gateways with subdomain support should not
					return payload directly, but redirect to URL with proper
					origin isolation
				`,
			Request: Request().
				Header("Host", u.Host).
				Path("/ipfs/{{cid}}/", CIDv1),
			Response: Expect().
				Status(301).
				Headers(
					Header("Location").
						Hint("request for example.com/ipfs/{CIDv1} returns Location HTTP header for subdomain redirect in browsers").
						Contains("{{scheme}}://{{cid}}.ipfs.{{host}}/", u.Scheme, CIDv1, u.Host),
				),
		},
		{
			Name: "request for example.com/ipfs/{CIDv1}/{filename with percent encoding} redirects to subdomain",
			Hint: "the path remainder MUST be preserved",
			Request: Request().
				Header("Host", u.Host).
				Path("/ipfs/{{cid}}/Portugal%252C+España=Peninsula%20Ibérica.txt", dirWithPercentEncodedFilenameCID),
			Response: Expect().
				Status(301).
				Headers(
					Header("Location").Equals("{{scheme}}://{{cid}}.ipfs.{{host}}/Portugal%252C+Espa%C3%B1a=Peninsula%20Ib%C3%A9rica.txt", u.Scheme, dirWithPercentEncodedFilenameCID, u.Host),
				),
		},
		{
			Name: "request for example.com/ipfs/{DirCID}/ redirects to subdomain",
			Hint: `
					path requests to gateways with subdomain support should not
					return payload directly, but redirect to URL with proper
					origin isolation
				`,
			Request: Request().
				Header("Host", u.Host).
				Path("/ipfs/{{cid}}/", DirCID),
			Response: Expect().
				Status(301).
				Headers(
					Header("Location").
						Hint("request for example.com/ipfs/{DirCID} returns Location HTTP header for subdomain redirect in browsers").
						Contains("{{scheme}}://{{cid}}.ipfs.{{host}}/", u.Scheme, DirCID, u.Host),
				),
		},
		{
			Name: "request for example.com/ipfs/{CIDv0} redirects to {CIDv1}.ipfs.example.com",
			Request: Request().
				Header("Host", u.Host).
				Path("/ipfs/{{cid}}/", CIDv0),
			Response: Expect().
				Status(301).
				Headers(
					Header("Location").
						Hint("request for example.com/ipfs/{CIDv0to1} returns Location HTTP header for subdomain redirect in browsers").
						Contains("{{scheme}}://{{cid}}.ipfs.{{host}}/", u.Scheme, CIDv0to1, u.Host),
				),
		},
		{
			Name: "request for {CID}.ipfs.example.com should return expected payload",
			Request: Request().
				Header("Host", Fmt("{{cid}}.ipfs.{{host}}", CIDv1, u.Host)).
				Path("/"),
			Response: Expect().
				Status(200).
				Body(Contains(CIDVal)),
		},
		{
			Name: "request for {CID}.ipfs.example.com/ipfs/{CID} should return HTTP 404",
			Hint: "ensure /ipfs/ namespace is not mounted on subdomain",
			Request: Request().
				Header("Host", Fmt("{{cid}}.ipfs.{{host}}", CIDv1, u.Host)).
				Path("/ipfs/{{cid}}/", CIDv1),
			Response: Expect().
				Status(404),
		},
		{
			Name: "request for {CID}.ipfs.example.com/ipfs/file.txt should return data from a file in CID content root",
			Hint: "ensure requests to /ipfs/* are not blocked, if content root has such subdirectory",
			Request: Request().
				Header("Host", Fmt("{{cid}}.ipfs.{{host}}", DirCID, u.Host)).
				Path("/ipfs/file.txt"),
			Response: Expect().
				Status(200).
				Body(Contains("I am a txt file")),
		},
		{
			Name: "valid file and subdirectory paths in directory listing at {cid}.ipfs.example.com",
			Hint: "{CID}.ipfs.example.com (Directory Listing)",
			Request: Request().
				Header("Host", Fmt("{{cid}}.ipfs.{{host}}", DirCID, u.Host)).
				Path("/"),
			Response: Expect().
				Status(200).
				Body(And(
					// TODO: implement html expectations https://github.com/ipfs/gateway-conformance/issues/21
					Contains(`<a href="/hello">hello</a>`),
					Contains(`<a href="/ipfs">ipfs</a>`),
				)),
		},
		{
			Name: "valid parent directory path in directory listing at {cid}.ipfs.example.com/sub/dir",
			Hint: "{CID}.ipfs.example.com/ipfs/ipns/ (if exists) should produce a valid directory listing",
			Request: Request().
				Header("Host", Fmt("{{cid}}.ipfs.{{host}}", DirCID, u.Host)).
				Path("/ipfs/ipns/"),
			Response: Expect().
				Status(200).
				Body(And(
					// TODO: implement html expectations https://github.com/ipfs/gateway-conformance/issues/21
					Contains(`<a href="/ipfs/ipns/..">..</a>`),
					Contains(`<a href="/ipfs/ipns/bar">bar</a>`),
				)),
		},
		{
			Name: "request for deep path resource at {cid}.ipfs.example.com/sub/dir/file",
			Hint: "{CID}.ipfs.example.com/ipfs/ipns/bar (if exists) should return expected file",
			Request: Request().
				Header("Host", Fmt("{{cid}}.ipfs.{{host}}", DirCID, u.Host)).
				Path("/ipfs/ipns/bar"),
			Response: Expect().
				Status(200).
				Body(Contains("text-file-content")),
		},
		{
			Name: "valid breadcrumb links in the header of directory listing at {cid}.ipfs.example.com/sub/dir (TODO: cleanup Kubo-specifics)",
			Hint: `
			Note 1: we test for sneaky subdir names  {cid}.ipfs.example.com/ipfs/ipns/ :^)
			Note 2: example.com/ipfs/.. present in HTML will be redirected to subdomain, so this is expected behavior
			`,
			Request: Request().
				Header("Host", Fmt("{{cid}}.ipfs.{{host}}", DirCID, u.Host)).
				Path("/ipfs/ipns/"),
			Response: Expect().
				Status(200).
				Body(
					And(
						Contains("Index of"),
						Contains(`/ipfs/<a href="//{{host}}/ipfs/{{cid}}">{{cid}}</a>/<a href="//{{host}}/ipfs/{{cid}}/ipfs">ipfs</a>/<a href="//{{host}}/ipfs/{{cid}}/ipfs/ipns">ipns</a>`,
							u.Host, DirCID),
					),
				),
		},
		{
			Name: "request for example.com/ipfs/{InvalidCID} produces useful error before redirect",
			Hint: "error message should include original CID (and it should be case-sensitive, as we can't assume everyone uses base32)",
			Request: Request().
				Header("Host", u.Host).
				Path("/ipfs/QmInvalidCID"),
			Response: Expect().
				Body(Contains(`invalid path "/ipfs/QmInvalidCID"`)),
		},
		{
			Name: "request for example.com/ipfs/{CID} with X-Forwarded-Proto: https produces redirect to HTTPS URL",
			Hint: "Support X-Forwarded-Proto",
			Request: Request().
				Header("Host", u.Host).
				Header("X-Forwarded-Proto", "https").
				Path("/ipfs/{{cid}}/", CIDv1),
			Response: Expect().
				Status(301).
				Headers(
					Header("Location").Equals("https://{{cid}}.ipfs.{{host}}/", CIDv1, u.Host),
				),
		},
		{
			Name: "request for example.com/ipfs/{CID} with X-Forwarded-Proto: http produces redirect to HTTP URL",
			Hint: "Support X-Forwarded-Proto",
			Request: Request().
				Header("Host", u.Host).
				Header("X-Forwarded-Proto", "http").
				Path("/ipfs/{{cid}}/", CIDv1),
			Response: Expect().
				Status(301).
				Headers(
					Header("Location").Equals("http://{{cid}}.ipfs.{{host}}/", CIDv1, u.Host),
				),
		},
		{
			Name: "request for example.com/ipfs/?uri=ipfs%3A%2F%2F.. produces redirect to /ipfs/.. content path",
			Hint: "Support ipfs:// in https://developer.mozilla.org/en-US/docs/Web/API/Navigator/registerProtocolHandler",
			Request: Request().
				Header("Host", u.Host).
				Path("/ipfs/").
				Query("uri", "ipfs://{{host}}/wiki/Diego_Maradona.html", CIDWikipedia),
			Response: Expect().
				Status(301).
				Headers(
					Header("Location").Equals("/ipfs/{{cid}}/wiki/Diego_Maradona.html", CIDWikipedia),
				),
		},
		{
			Name: "request for a too long CID at example.com/ipfs/{CIDv1} returns human readable error",
			Hint: "router should not redirect to hostnames that could fail due to DNS limits",
			Request: Request().
				Header("Host", u.Host).
				Path("/ipfs/{{cid}}/", CIDv1_TOO_LONG),
			Response: Expect().
				Status(400).
				Body(Contains("CID incompatible with DNS label length limit of 63")),
		},
		{
			Name: "request for a too long CID at {CIDv1}.ipfs.example.com returns expected payload",
			Hint: "direct request should also fail (provides the same UX as router and avoids confusion)",
			Request: Request().
				Header("Host", Fmt("{{cid}}.ipfs.{{host}}", CIDv1_TOO_LONG, u.Host)).
				Path("/"),
			Response: Expect().
				Status(400).
				Body(Contains("CID incompatible with DNS label length limit of 63")),
		},
		// ## ============================================================================
		// ## Test support for X-Forwarded-Host
		// ## ============================================================================
		{
			Name: "request for fake.domain.com/ipfs/{CID} doesn't match the example.com gateway",
			Hint: "when there is no Host match, request is processed as a path gateway",
			Request: Request().
				Header("Host", "fake.domain.com").
				Path("/ipfs/{{cid}}", CIDv1),
			Response: Expect().
				Status(200),
		},
		{
			Name: "request for fake.domain.com/ipfs/{CID} with X-Forwarded-Host: example.com match the example.com gateway",
			Hint: "X-Forwarded-Host overrides Host, request should be processed as a subdomain gateway",
			Request: Request().
				Header("Host", "fake.domain.com").
				Header("X-Forwarded-Host", u.Host).
				Path("/ipfs/{{cid}}", CIDv1),
			Response: Expect().
				Status(301).
				Headers(
					Header("Location").Equals("{{scheme}}://{{cid}}.ipfs.{{host}}/", u.Scheme, CIDv1, u.Host),
				),
		},
		{
			Name: "request for fake.domain.com/ipfs/{CID} with X-Forwarded-Host: example.com and X-Forwarded-Proto: https match the example.com gateway, redirect with https",
			Request: Request().
				Header("Host", "fake.domain.com").
				Path("/ipfs/{{cid}}", CIDv1).
				Header("X-Forwarded-Host", u.Host).
				Header("X-Forwarded-Proto", "https"),
			Response: Expect().
				Status(301).
				Headers(
					Header("Location").Equals("https://{{cid}}.ipfs.{{host}}/", CIDv1, u.Host),
				),
		},
		{
			Name: "request for fake.domain.com/ipfs/{CID} with X-Forwarded-Host: example.com and X-Forwarded-Proto: http match the example.com gateway, redirect with http",
			Request: Request().
				Header("Host", "fake.domain.com").
				Path("/ipfs/{{cid}}", CIDv1).
				Header("X-Forwarded-Host", u.Host).
				Header("X-Forwarded-Proto", "http"),
			Response: Expect().
				Status(301).
				Headers(
					Header("Location").Equals("http://{{cid}}.ipfs.{{host}}/", CIDv1, u.Host),
				),
		},
	}...)

	RunWithSpecs(t, tests, specs.SubdomainGatewayIPFS)
}
