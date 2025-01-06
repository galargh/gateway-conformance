package tests

import (
	"strings"
	"testing"

	"github.com/ipfs/gateway-conformance/tooling"
	. "github.com/ipfs/gateway-conformance/tooling/ipns"
	"github.com/ipfs/gateway-conformance/tooling/specs"
	. "github.com/ipfs/gateway-conformance/tooling/test"
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
)

var (
	// See fixtures/ipns_records/README.md for information. These are invalid records, so we cannot load them with MustOpenIPNSRecordWithKey
	ipnsV1                = "k51qzi5uqu5dm4tm0wt8srkg9h9suud4wuiwjimndrkydqm81cqtlb5ak6p7ku"
	ipnsV1V2BrokenValueV1 = "k51qzi5uqu5dlmit2tuwdvnx4sbnyqgmvbxftl0eo3f33wwtb9gr7yozae9kpw"
	ipnsV1V2BrokenSigV2   = "k51qzi5uqu5diamp7qnnvs1p1gzmku3eijkeijs3418j23j077zrkok63xdm8c"

	ipnsV1V2BrokenSigV1     = MustOpenIPNSRecordWithKey("ipns_records/k51qzi5uqu5dilgf7gorsh9vcqqq4myo6jd4zmqkuy9pxyxi5fua3uf7axph4y_v1-v2-broken-signature-v1.ipns-record")
	bodyIPNSV1V2BrokenSigV1 = mustBytesFromRawCID(strings.TrimPrefix(ipnsV1V2BrokenSigV1.Value(), "/ipfs/"))

	ipnsV1V2     = MustOpenIPNSRecordWithKey("ipns_records/k51qzi5uqu5dlkw8pxuw9qmqayfdeh4kfebhmreauqdc6a7c3y7d5i9fi8mk9w_v1-v2.ipns-record")
	bodyIPNSV1V2 = mustBytesFromRawCID(strings.TrimPrefix(ipnsV1V2.Value(), "/ipfs/"))

	ipnsV2     = MustOpenIPNSRecordWithKey("ipns_records/k51qzi5uqu5dit2ku9mutlfgwyz8u730on38kd10m97m36bjt66my99hb6103f_v2.ipns-record")
	bodyIPNSV2 = mustBytesFromRawCID(strings.TrimPrefix(ipnsV2.Value(), "/ipfs/"))
)

func mustBytesFromRawCID(c string) []byte {
	mh, err := multihash.Decode(cid.MustParse(c).Hash())
	if err != nil {
		panic(err)
	}
	return mh.Digest
}

func TestGatewayIPNSPath(t *testing.T) {
	tests := SugarTests{
		{
			Name: "GET for /ipns/name with V1-only signature MUST fail with 5XX",
			Hint: `
			Legacy V1 IPNS records are considered insecure. A gateway should
			never return data when IPNS Record is missing V2 signature, EVEN
			when V1 signature matches the payload.
			More details in IPIP-428.
			`,
			Request: Request().
				Path("/ipns/{{name}}", ipnsV1),
			Response: Expect().
				StatusBetween(500, 599),
		},
		{
			Name: "GET for /ipns/name with valid V1+V2 signatures with V1-vs-V2 value mismatch MUST fail with 5XX",
			Hint: `
			Legacy V1 signatures in IPNS records are considered insecure and
			got replaced with V2 that signs entire CBOR in the data field.
			Producing records with both V1 and V2 signatures is valid for
			backward-compatibility, but validation logic requires V1 (legacy
			protobuf fields) and V2 (CBOR in data field) to match. This means
			that even when both signatures are valid, if V1 and V2 values do
			not match, the IPNS record should not be considered valid, as it
			could allow signature reuse attacks against V1 users.
			More details in IPIP-428.
			`,
			Request: Request().
				Path("/ipns/{{name}}", ipnsV1V2BrokenValueV1),
			Response: Expect().
				StatusMatch("5xx"),
		},
		{
			Name: "GET for /ipns/name with valid V2 and broken V1 signature succeeds",
			Hint: `
			Legacy V1 signatures in IPNS records are considered insecure and
			got replaced with V2 that signs entire CBOR in the data field.
			Integrity of the record is protected by SignatureV2, V1 can be
			ignored as long V1 values match V2 ones in CBOR.
			More details in IPIP-428.
			`,
			Request: Request().
				Path("/ipns/{{name}}", ipnsV1V2BrokenSigV1.Key()),
			Response: Expect().
				Status(200).
				Body(bodyIPNSV1V2BrokenSigV1),
		},
		{
			Name: "GET for /ipns/name with valid V1+V2 signatures succeeds",
			Hint: `
			Records with legacy V1 signatures should not impact V2 verification.
			The payload should match the content path from IPNS Record's Value field.
			More details in IPIP-428.
			`,
			Request: Request().
				Path("/ipns/{{name}}", ipnsV1V2.Key()),
			Response: Expect().
				Body(bodyIPNSV1V2),
		},
		{
			Name: "GET for /ipns/name with valid V2-only signature succeeds",
			Hint: `
			Legacy V1 signatures in IPNS records are considered insecure and
			got replaced with V2 that signs entire CBOR in the data field.
			Gateway MUST correctly resolve IPNS records without V1 fields.
			More details in IPIP-428.
			`,
			Request: Request().
				Path("/ipns/{{name}}", ipnsV2.Key()),
			Response: Expect().
				Body(bodyIPNSV2),
		},
		{
			Name: "GET for /ipns/name with valid V1 and broken V2 signature MUST fail with 5XX",
			Hint: `
			Legacy V1 IPNS records are considered insecure. A gateway should
			never return data when IPNS Record is missing a valid V2 signature,
			EVEN when V1 signature is valid.
			More details in IPIP-428.
			`,
			Request: Request().
				Path("/ipns/{{name}}", ipnsV1V2BrokenSigV2),
			Response: Expect().
				StatusMatch("5xx"),
		},
	}

	RunWithSpecs(t, tests, specs.PathGatewayIPNS)
}

func TestRedirectCanonicalIPNS(t *testing.T) {
	tooling.LogTestGroup(t, GroupIPNS)

	tests := SugarTests{
		{
			Name: "GET for /ipns/{b58-multihash-of-ed25519-key} redirects to /ipns/{cidv1-libp2p-key-base36}",
			Hint: `
			CIDv1 in case-insensitive encoding ensures it works in contexts
			such as authority component of URL. Base36 ensures ED25519
			libp2p-key fits in a single DNS label, making the IPNS name
			compatible with subdomain gateways.
			`,
			Request: Request().
				Path("/ipns/12D3KooWRBy97UB99e3J6hiPesre1MZeuNQvfan4gBziswrRJsNK/root2/"),
			Response: Expect().
				Status(302).
				Headers(
					Header("Location").Equals("/ipns/k51qzi5uqu5dlvj2baxnqndepeb86cbk3ng7n3i46uzyxzyqj2xjonzllnv0v8/root2/"),
				),
		},
		{
			Name: "GET for /ipns/{cidv0-like-b58-multihash-of-rsa-key} redirects to /ipns/{cidv1-libp2p-key-base36}",
			Hint: `
			CIDv1 in case-insensitive encoding ensures it works in contexts
			such as authority component of URL.
			`,
			Request: Request().
				Path("/ipns/QmcJM7PRfkSbcM5cf1QugM5R37TLRKyJGgBEhXjLTB8uA2/root2/"),
			Response: Expect().
				Status(302).
				Headers(
					Header("Location").Equals("/ipns/k2k4r8ol4m8kkcqz509c1rcjwunebj02gcnm5excpx842u736nja8ger/root2/"),
				),
		},
	}

	RunWithSpecs(t, tests, specs.PathGatewayIPNS)
}
