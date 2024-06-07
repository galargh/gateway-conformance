package tests

import (
	"testing"

	"github.com/ipfs/gateway-conformance/tooling"
	"github.com/ipfs/gateway-conformance/tooling/car"
	. "github.com/ipfs/gateway-conformance/tooling/check"
	"github.com/ipfs/gateway-conformance/tooling/dnslink"
	"github.com/ipfs/gateway-conformance/tooling/specs"
	. "github.com/ipfs/gateway-conformance/tooling/test"
	. "github.com/ipfs/gateway-conformance/tooling/tmpl"
)

func TestRedirectsFileSupport(t *testing.T) {
	tooling.LogSpecs(t, "https://specs.ipfs.tech/http-gateways/web-redirects-file/")
	fixture := car.MustOpenUnixfsCar("redirects_file/redirects.car")
	redirectDir := fixture.MustGetNode("examples")
	redirectDirCID := redirectDir.Base32Cid()

	custom404 := fixture.MustGetNode("examples", "404.html")
	custom410 := fixture.MustGetNode("examples", "410.html")
	custom451 := fixture.MustGetNode("examples", "451.html")

	tests := SugarTests{}

	// Redirects require origin isolation (https://specs.ipfs.tech/http-gateways/web-redirects-file/)
	// This means we only run these tests against origins explicitly passed via --subdomain-url
	u := SubdomainGatewayURL()

	dirCIDInSubdomain := Fmt("{{cid}}.ipfs.{{host}}", redirectDirCID, u.Host)

	tests = append(tests, SugarTests{
		{
			Name: "request for {cid}.ipfs.example.com/redirect-one redirects with default of 301, per _redirects file",
			Request: Request().
				Header("Host", dirCIDInSubdomain).
				Path("/redirect-one"),
			Response: Expect().
				Status(301).
				Headers(
					Header("Location").Equals("/one.html"),
				),
		},
		{
			Name: "request for {cid}.ipfs.example.com/301-redirect-one redirects with 301, per _redirects file",
			Request: Request().
				Header("Host", dirCIDInSubdomain).
				Path("/301-redirect-one"),
			Response: Expect().
				Status(301).
				Headers(
					Header("Location").Equals("/one.html"),
				),
		},
		{
			Name: "request for {cid}.ipfs.example.com/302-redirect-two redirects with 302, per _redirects file",
			Request: Request().
				Header("Host", dirCIDInSubdomain).
				Path("/302-redirect-two"),
			Response: Expect().
				Status(302).
				Headers(
					Header("Location").Equals("/two.html"),
				),
		},
		{
			Name: "request for {cid}.ipfs.example.com/200-index returns 200, per _redirects file",
			Request: Request().
				Header("Host", dirCIDInSubdomain).
				Path("/200-index"),
			Response: Expect().
				Status(200).
				Body(Contains("my index")),
		},
		{
			Name: "request for {cid}.ipfs.example.com/posts/:year/:month/:day/:title redirects with 301 and placeholders, per _redirects file",
			Request: Request().
				Header("Host", dirCIDInSubdomain).
				Path("/posts/2022/01/01/hello-world"),
			Response: Expect().
				Status(301).
				Headers(
					Header("Location").Equals("/articles/2022/01/01/hello-world"),
				),
		},
		{
			Name: "request for {cid}.ipfs.example.com/splat/one.html redirects with 301 and splat placeholder, per _redirects file",
			Request: Request().
				Header("Host", dirCIDInSubdomain).
				Path("/splat/one.html"),
			Response: Expect().
				Status(301).
				Headers(
					Header("Location").Equals("/redirected-splat/one.html"),
				),
		},
		{
			Name: "request for {cid}.ipfs.example.com/not-found/has-no-redirects-entry returns custom 404, per _redirects file",
			Request: Request().
				Header("Host", dirCIDInSubdomain).
				Path("/not-found/has-no-redirects-entry"),
			Response: Expect().
				Status(404).
				Headers(
					Header("Cache-Control").Equals("public, max-age=29030400, immutable"),
					Header("Etag").Equals(`"{{etag}}"`, custom404.Cid().String()),
				).
				Body(Contains(custom404.ReadFile())),
		},
		{
			Name: "request for {cid}.ipfs.example.com/gone/has-no-redirects-entry returns custom 410, per _redirects file",
			Request: Request().
				Header("Host", dirCIDInSubdomain).
				Path("/gone/has-no-redirects-entry"),
			Response: Expect().
				Status(410).
				Headers(
					Header("Cache-Control").Equals("public, max-age=29030400, immutable"),
					Header("Etag").Equals(`"{{etag}}"`, custom410.Cid().String()),
				).
				Body(Contains(custom410.ReadFile())),
		},
		{
			Name: "request for {cid}.ipfs.example.com/unavail/has-no-redirects-entry returns custom 451, per _redirects file",
			Request: Request().
				Header("Host", dirCIDInSubdomain).
				Path("/unavail/has-no-redirects-entry"),
			Response: Expect().
				Status(451).
				Headers(
					Header("Cache-Control").Equals("public, max-age=29030400, immutable"),
					Header("Etag").Equals(`"{{etag}}"`, custom451.Cid().String()),
				).
				Body(Contains(custom451.ReadFile())),
		},
		{
			Name: "request for {cid}.ipfs.example.com/catch-all returns 200, per _redirects file",
			Request: Request().
				Header("Host", dirCIDInSubdomain).
				Path("/catch-all"),
			Response: Expect().
				Status(200).
				Body(Contains("my index")),
		},
	}...)

	// # Invalid file, containing forced redirect
	invalidRedirectsDirCID := fixture.MustGetNode("forced").Base32Cid()
	invalidDirSubdomain := Fmt("{{cid}}.ipfs.{{host}}", invalidRedirectsDirCID, u.Host)

	tooLargeRedirectsDirCID := fixture.MustGetNode("too-large").Base32Cid()
	tooLargeDirSubdomain := Fmt("{{cid}}.ipfs.{{host}}", tooLargeRedirectsDirCID, u.Host)

	tests = append(tests, SugarTests{
		{
			Name: "invalid file: request for $INVALID_REDIRECTS_DIR_HOSTNAME/not-found returns error about invalid redirects file",
			Hint: `if accessing a path that doesn't exist, read _redirects and fail parsing, and return error`,
			Request: Request().
				Header("Host", invalidDirSubdomain).
				Path("/not-found"),
			Response: Expect().
				Status(500).
				Body(
					And(
						Contains("could not parse _redirects:"),
						Contains(`forced redirects (or "shadowing") are not supported`),
					),
				).Spec("https://specs.ipfs.tech/http-gateways/web-redirects-file/#no-forced-redirects"),
			Spec: "https://specs.ipfs.tech/http-gateways/web-redirects-file/#error-handling",
		},
		{
			Name: "invalid file: request for $TOO_LARGE_REDIRECTS_DIR_HOSTNAME/not-found returns error about too large redirects file",
			Hint: `if accessing a path that doesn't exist and _redirects file is too large, return error`,
			Request: Request().
				Header("Host", tooLargeDirSubdomain).
				Path("/not-found"),
			Response: Expect().
				Status(500).
				Body(
					And(
						Contains("could not parse _redirects:"),
						Contains("redirects file size cannot exceed"),
					),
				),
			Spec: "https://specs.ipfs.tech/http-gateways/web-redirects-file/#max-file-size",
		},
	}...)

	// # With CRLF line terminator
	newlineRedirectsDirCID := fixture.MustGetNode("newlines").Base32Cid()
	newlineHost := Fmt("{{cid}}.ipfs.{{host}}", newlineRedirectsDirCID, u.Host)

	// # Good codes
	goodRedirectDirCID := fixture.MustGetNode("good-codes").Base32Cid()
	goodRedirectDirHost := Fmt("{{cid}}.ipfs.{{host}}", goodRedirectDirCID, u.Host)

	// # Bad codes
	badRedirectDirCID := fixture.MustGetNode("bad-codes").Base32Cid()
	badRedirectDirHost := Fmt("{{cid}}.ipfs.{{host}}", badRedirectDirCID, u.Host)

	tests = append(tests, SugarTests{
		{
			Name: "newline: request for $NEWLINE_REDIRECTS_DIR_HOSTNAME/redirect-one redirects with default of 301, per _redirects file",
			Request: Request().
				Header("Host", newlineHost).
				Path("/redirect-one"),
			Response: Expect().
				Status(301).
				Headers(
					Header("Location").Equals("/one.html"),
				),
		},
		{
			Name: "good codes: request for $GOOD_REDIRECTS_DIR_HOSTNAME/redirect-one redirects with default of 301, per _redirects file",
			Request: Request().
				Header("Host", goodRedirectDirHost).
				Path("/a301"),
			Response: Expect().
				Status(301).
				Headers(
					Header("Location").Equals("/b301"),
				),
		},
		{
			Name: "bad codes: request for $BAD_REDIRECTS_DIR_HOSTNAME/found.html doesn't return error about bad code",
			Request: Request().
				Header("Host", badRedirectDirHost).
				Path("/found.html"),
			Response: Expect().
				Status(200).
				Body(
					And(
						Contains("my found"),
						Not(Contains("unsupported redirect status")),
					),
				),
		},
	}...)

	RunWithSpecs(t, tests, specs.SubdomainGatewayIPFS, specs.RedirectsFile)
}

func TestRedirectsFileSupportWithDNSLink(t *testing.T) {
	tooling.LogTestGroup(t, GroupDNSLink)
	dnsLinks := dnslink.MustOpenDNSLink("redirects_file/dnslink.yml")
	dnsLink := dnsLinks.MustGet("redirects-examples")

	tests := SugarTests{
		{
			Name: "request for //{dnslink} redirects with default of 301, per _redirects file",
			Request: Request().
				Header("Host", dnsLink).
				Path("/redirect-one"),
			Response: Expect().
				Status(301).
				Headers(
					Header("Location", "/one.html"),
				),
		},
		{
			Name: "request for //{dnslink}/en/has-no-redirects-entry returns custom 404, per _redirects file",
			Hint: `ensure custom 404 works and has the same cache headers as regular /ipns/ paths`,
			Request: Request().
				Header("Host", dnsLink).
				Path("/not-found/has-no-redirects-entry"),
			Response: Expect().
				Status(404).
				Headers(
					Header("Etag", `"Qmd9GD7Bauh6N2ZLfNnYS3b7QVAijbud83b8GE8LPMNBBP"`),
					Header("Cache-Control").Not().Contains("public, max-age=29030400, immutable"),
					Header("Cache-Control").Not().Contains("immutable"),
					Header("Date").Exists(),
				).
				Body(
					Contains("my 404"),
				),
		},
	}

	// TODO:
	RunWithSpecs(t, tests, specs.DNSLinkGateway, specs.RedirectsFile)
}

func TestRedirectsFileWithIfNoneMatchHeader(t *testing.T) {
	fixture := car.MustOpenUnixfsCar("redirects_file/redirects-spa.car")

	dnsLinks := dnslink.MustOpenDNSLink("redirects_file/dnslink.yml")
	dnsLink := dnsLinks.MustGet("redirects-spa")

	u := SubdomainGatewayURL()

	dnslinkAtSubdomainGw := Fmt("{{dnslink}}.ipns.{{host}}", dnslink.InlineDNS(dnsLink), u.Host)

	var etag string

	RunWithSpecs(t, SugarTests{
		{
			Name: "request for //{dnslink}.ipns.{subdomain-gateway}/missing-page returns body of index.html as per _redirects",
			Request: Request().
				Path("/missing-page").
				Headers(
					Header("Host", dnslinkAtSubdomainGw),
					Header("Accept", "text/html"),
				),
			Response: Expect().
				Status(200).
				Headers(
					Header("Etag").
						Checks(func(v string) bool {
							etag = v
							return v != ""
						}),
				).
				Body(fixture.MustGetRawData("index.html")),
		},
	}, specs.SubdomainGatewayIPNS, specs.RedirectsFile)

	RunWithSpecs(t, SugarTests{
		{
			Name: "request for //{dnslink}.ipns.{subdomain-gateway}/missing-page with If-None-Match returns 304",
			Request: Request().
				Path("/missing-page").
				Headers(
					Header("Host", dnslinkAtSubdomainGw),
					Header("Accept", "text/html"),
					Header("If-None-Match", etag),
				),
			Response: Expect().
				Status(304),
		},
	}, specs.SubdomainGatewayIPNS, specs.RedirectsFile)

	RunWithSpecs(t, SugarTests{
		{
			Name: "request for //{dnslink}/missing-page returns body of index.html as per _redirects",
			Request: Request().
				Path("/missing-page").
				Headers(
					Header("Host", dnsLink),
					Header("Accept", "text/html"),
				),
			Response: Expect().
				Status(200).
				Headers(
					Header("Etag").
						Checks(func(v string) bool {
							etag = v
							return v != ""
						}),
				).
				Body(fixture.MustGetRawData("index.html")),
		},
	}, specs.DNSLinkGateway, specs.RedirectsFile)

	RunWithSpecs(t, SugarTests{
		{
			Name: "request for //{dnslink}/missing-page with If-None-Match returns 304",
			Request: Request().
				Path("/missing-page").
				Headers(
					Header("Host", dnsLink),
					Header("Accept", "text/html"),
					Header("If-None-Match", etag),
				),
			Response: Expect().
				Status(304),
		},
	}, specs.DNSLinkGateway, specs.RedirectsFile)
}
