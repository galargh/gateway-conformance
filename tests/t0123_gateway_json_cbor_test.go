package tests

import (
	"fmt"
	"testing"

	"github.com/ipfs/gateway-conformance/tooling/car"
	. "github.com/ipfs/gateway-conformance/tooling/check"
	"github.com/ipfs/gateway-conformance/tooling/test"
	. "github.com/ipfs/gateway-conformance/tooling/test"
)

func TestGatewayJsonCbor(t *testing.T) {
	fixture := car.MustOpenUnixfsCar("t0123-gateway-json-cbor.car")

	fileJSON := fixture.MustGetNode("ą", "ę", "t.json")
	fileJSONCID := fileJSON.Cid()
	fileJSONData := fileJSON.RawData()

	tests := SugarTests{
		{
			Name: "GET UnixFS file with JSON bytes is returned with application/json Content-Type (1)",
			Hint: `
			## Quick regression check for JSON stored on UnixFS:
			## it has nothing to do with DAG-JSON and JSON codecs,
			## but a lot of JSON data is stored on UnixFS and is requested with or without various hints
			## and we want to avoid surprises like https://github.com/protocol/bifrost-infra/issues/2290
			`,
			Request: Request().
				Path("ipfs/%s", fileJSONCID).
				Headers(
					Header("Accept", "application/json"),
				),
			Response: Expect().
				Status(200).
				Headers(
					Header("Content-Type").
						Equals("application/json"),
				).
				Body(fileJSONData),
		},
		{
			Name: "GET UnixFS file with JSON bytes is returned with application/json Content-Type (2)",
			Hint: `
			## Quick regression check for JSON stored on UnixFS:
			## it has nothing to do with DAG-JSON and JSON codecs,
			## but a lot of JSON data is stored on UnixFS and is requested with or without various hints
			## and we want to avoid surprises like https://github.com/protocol/bifrost-infra/issues/2290
			`,
			Request: Request().
				Path("ipfs/%s", fileJSONCID).
				Headers(
					Header("Accept", "application/json"),
				),
			Response: Expect().
				Status(200).
				Headers(
					Header("Content-Type").
						Equals("application/json"),
				).
				Body(fileJSONData),
		},
	}

	test.Run(t, tests)
}

// ## Reading UnixFS (data encoded with dag-pb codec) as DAG-CBOR and DAG-JSON
// ## (returns representation defined in https://ipld.io/specs/codecs/dag-pb/spec/#logical-format)
func TestDAgPbConversion(t *testing.T) {
	fixture := car.MustOpenUnixfsCar("t0123-gateway-json-cbor.car")

	dir := fixture.MustGetRoot()
	file := fixture.MustGetNode("ą", "ę", "file-źł.txt")

	dirCID := dir.Cid()
	fileCID := file.Cid()
	fileData := file.RawData()

	table := []struct {
		Name        string
		Format      string
		Disposition string
	}{
		{"DAG-JSON", "json", "inline"},
		{"DAG-CBOR", "cbor", "attachment"},
	}

	for _, row := range table {
		// ipfs dag get --output-codec dag-$format $FILE_CID > ipfs_dag_get_output
		formatedFile := file.Formatted("dag-" + row.Format)
		formatedDir := dir.Formatted("dag-" + row.Format)

		tests := SugarTests{
			/**
				test_expect_success "GET UnixFS file as $name with format=dag-$format converts to the expected Content-Type" '
				curl -sD headers "http://127.0.0.1:$GWAY_PORT/ipfs/$FILE_CID?format=dag-$format" > curl_output 2>&1 &&
				ipfs dag get --output-codec dag-$format $FILE_CID > ipfs_dag_get_output 2>&1 &&
				test_cmp ipfs_dag_get_output curl_output &&
				test_should_contain "Content-Type: application/vnd.ipld.dag-$format" headers &&
				test_should_contain "Content-Disposition: ${disposition}\; filename=\"${FILE_CID}.${format}\"" headers &&
				test_should_not_contain "Content-Type: application/$format" headers
			'
			*/
			{
				Name: fmt.Sprintf("GET UnixFS file as %s with format=dag-%s converts to the expected Content-Type", row.Name, row.Format),
				Request: Request().
					Path("ipfs/%s", fileCID).
					Query("format", "dag-"+row.Format),
				Response: Expect().
					Status(200).
					Headers(
						Header("Content-Type").
							Equals("application/vnd.ipld.dag-%s", row.Format),
						Header("Content-Disposition").
							Contains("%s; filename=\"%s.%s\"", row.Disposition, fileCID, row.Format),
						Header("Content-Type").
							Not().Contains("application/%s", row.Format),
					).Body(
					formatedFile,
				),
			},
			/**
			test_expect_success "GET UnixFS directory as $name with format=dag-$format converts to the expected Content-Type" '
			curl -sD headers "http://127.0.0.1:$GWAY_PORT/ipfs/$DIR_CID?format=dag-$format" > curl_output 2>&1 &&
			ipfs dag get --output-codec dag-$format $DIR_CID > ipfs_dag_get_output 2>&1 &&
			test_cmp ipfs_dag_get_output curl_output &&
			test_should_contain "Content-Type: application/vnd.ipld.dag-$format" headers &&
			test_should_contain "Content-Disposition: ${disposition}\; filename=\"${DIR_CID}.${format}\"" headers &&
			test_should_not_contain "Content-Type: application/$format" headers
			'
			*/
			{
				Name: fmt.Sprintf("GET UnixFS directory as %s with format=dag-%s converts to the expected Content-Type", row.Name, row.Format),
				Request: Request().
					Path("ipfs/%s?format=dag-%s", dirCID, row.Format),
				Response: Expect().
					Status(200).
					Headers(
						Header("Content-Type").
							Equals("application/vnd.ipld.dag-%s", row.Format),
						Header("Content-Disposition").
							Contains("%s; filename=\"%s.%s\"", row.Disposition, dirCID, row.Format),
						Header("Content-Type").
							Not().Contains("application/%s", row.Format),
					).Body(
					formatedDir,
				),
			},
			/**
			test_expect_success "GET UnixFS as $name with 'Accept: application/vnd.ipld.dag-$format' converts to the expected Content-Type" '
			curl -sD - -H "Accept: application/vnd.ipld.dag-$format" "http://127.0.0.1:$GWAY_PORT/ipfs/$FILE_CID" > curl_output 2>&1 &&
			test_should_contain "Content-Disposition: ${disposition}\; filename=\"${FILE_CID}.${format}\"" curl_output &&
			test_should_contain "Content-Type: application/vnd.ipld.dag-$format" curl_output &&
			test_should_not_contain "Content-Type: application/$format" curl_output
			'
			*/
			{
				Name: fmt.Sprintf("GET UnixFS as %s with 'Accept: application/vnd.ipld.dag-%s' converts to the expected Content-Type", row.Name, row.Format),
				Request: Request().
					Path("ipfs/%s", fileCID).
					Headers(
						Header("Accept", "application/vnd.ipld.dag-%s", row.Format),
					),
				Response: Expect().
					Status(200).
					Headers(
						Header("Content-Disposition").
							Contains("%s; filename=\"%s.%s\"", row.Disposition, fileCID, row.Format),
						Header("Content-Type").
							Equals("application/vnd.ipld.dag-%s", row.Format),
						Header("Content-Type").
							Not().Contains("application/%s", row.Format),
					),
			},
			/**
			test_expect_success "GET UnixFS as $name with 'Accept: foo, application/vnd.ipld.dag-$format,bar' converts to the expected Content-Type" '
			curl -sD - -H "Accept: foo, application/vnd.ipld.dag-$format,text/plain" "http://127.0.0.1:$GWAY_PORT/ipfs/$FILE_CID" > curl_output 2>&1 &&
			test_should_contain "Content-Type: application/vnd.ipld.dag-$format" curl_output
			'
			*/
			{
				Name: fmt.Sprintf("GET UnixFS as %s with 'Accept: foo, application/vnd.ipld.dag-%s,bar' converts to the expected Content-Type", row.Name, row.Format),
				Request: Request().
					Path("ipfs/%s", fileCID).
					Headers(
						Header("Accept", "foo, application/vnd.ipld.dag-%s,bar", row.Format),
					),
				Response: Expect().
					Status(200).
					Headers(
						Header("Content-Type").
							Equals("application/vnd.ipld.dag-%s", row.Format),
					),
			},
			/**
			test_expect_success "GET UnixFS with format=$format (not dag-$format) is no-op (no conversion)" '
			curl -sD headers "http://127.0.0.1:$GWAY_PORT/ipfs/$FILE_CID?format=$format" > curl_output 2>&1 &&
			ipfs cat $FILE_CID > cat_output &&
			test_cmp cat_output curl_output &&
			test_should_contain "Content-Type: text/plain" headers &&
			test_should_not_contain "Content-Type: application/$format" headers &&
			test_should_not_contain "Content-Type: application/vnd.ipld.dag-$format" headers
			'
			*/
			{
				Name: fmt.Sprintf("GET UnixFS with format=%s (not dag-%s) is no-op (no conversion)", row.Format, row.Format),
				Request: Request().
					Path("ipfs/%s?format=%s", fileCID, row.Format),
				Response: Expect().
					Status(200).
					Headers(
						// NOTE: kubo gateway returns "text/plain; charset=utf-8" for example
						Header("Content-Type").
							Contains("text/plain"),
						Header("Content-Type").
							Not().Contains("application/%s", row.Format),
						Header("Content-Type").
							Not().Contains("application/vnd.ipld.dag-%s", row.Format),
					).Body(
					fileData,
				),
			},
			/**
			test_expect_success "GET UnixFS with 'Accept: application/$format' (not dag-$format) is no-op (no conversion)" '
			curl -sD headers -H "Accept: application/$format" "http://127.0.0.1:$GWAY_PORT/ipfs/$FILE_CID" > curl_output 2>&1 &&
			ipfs cat $FILE_CID > cat_output &&
			test_cmp cat_output curl_output &&
			test_should_contain "Content-Type: text/plain" headers &&
			test_should_not_contain "Content-Type: application/$format" headers &&
			test_should_not_contain "Content-Type: application/vnd.ipld.dag-$format" headers
			'
			*/
			{
				Name: fmt.Sprintf("GET UnixFS with 'Accept: application/%s' (not dag-%s) is no-op (no conversion)", row.Format, row.Format),
				Request: Request().
					Path("ipfs/%s", fileCID).
					Headers(
						Header("Accept", "application/%s", row.Format),
					),
				Response: Expect().
					Status(200).
					Headers(
						// NOTE: kubo gateway returns "text/plain; charset=utf-8" for example
						Header("Content-Type").
							Contains("text/plain"),
						Header("Content-Type").
							Not().Contains("application/%s", row.Format),
						Header("Content-Type").
							Not().Contains("application/vnd.ipld.dag-%s", row.Format),
					).Body(
					fileData,
				),
			},
		}

		test.Run(t, tests)
	}
}

// # Requesting CID with plain json (0x0200) and cbor (0x51) codecs
// # (note these are not UnixFS, not DAG-* variants, just raw block identified by a CID with a special codec)
func TestPlainCodec(t *testing.T) {
	table := []struct {
		Name        string
		Format      string
		Disposition string
		Checker     func(value []byte) Check[[]byte]
	}{
		{"plain JSON codec", "json", "inline", IsJSONEqual},
		{"plain CBOR codec", "cbor", "attachment", IsEqualBytes},
	}

	for _, row := range table {
		plain := car.MustOpenUnixfsCar(fmt.Sprintf("t0123/plain.%s.car", row.Format)).MustGetRoot()
		plainOrDag := car.MustOpenUnixfsCar(fmt.Sprintf("t0123/plain-that-can-be-dag.%s.car", row.Format)).MustGetRoot()
		formatted := plainOrDag.Formatted("dag-" + row.Format)

		plainCID := plain.Cid()
		plainOrDagCID := plainOrDag.Cid()

		tests := SugarTests{
			/**
			# no explicit format, just codec in CID
			test_expect_success "GET $name without Accept or format= has expected $format Content-Type and body as-is" '
			CID=$(echo "{ \"test\": \"plain json\" }" | ipfs dag put --input-codec json --store-codec $format) &&
			curl -sD headers "http://127.0.0.1:$GWAY_PORT/ipfs/$CID" > curl_output 2>&1 &&
			ipfs block get $CID > ipfs_block_output 2>&1 &&
			test_cmp ipfs_block_output curl_output &&
			test_should_contain "Content-Disposition: ${disposition}\; filename=\"${CID}.${format}\"" headers &&
			test_should_contain "Content-Type: application/$format" headers
			'
			*/
			{
				Name: fmt.Sprintf(`GET %s without Accept or format= has expected "%s" Content-Type and body as-is`, row.Name, row.Format),
				Hint: `
				No explicit format, just codec in CID
				`,
				Request: Request().
					Path("ipfs/%s", plainCID),
				Response: Expect().
					Status(200).
					Headers(
						Header("Content-Disposition").
							Contains(fmt.Sprintf("%s; filename=\"%s.%s\"", row.Disposition, plainCID, row.Format)),
						Header("Content-Type").
							Contains(fmt.Sprintf("application/%s", row.Format)),
					).Body(
					plain.RawData(),
				),
			},
			/**
			# explicit format still gives correct output, just codec in CID
			test_expect_success "GET $name with ?format= has expected $format Content-Type and body as-is" '
			CID=$(echo "{ \"test\": \"plain json\" }" | ipfs dag put --input-codec json --store-codec $format) &&
			curl -sD headers "http://127.0.0.1:$GWAY_PORT/ipfs/$CID?format=$format" > curl_output 2>&1 &&
			ipfs block get $CID > ipfs_block_output 2>&1 &&
			test_cmp ipfs_block_output curl_output &&
			test_should_contain "Content-Disposition: ${disposition}\; filename=\"${CID}.${format}\"" headers &&
			test_should_contain "Content-Type: application/$format" headers
			'
			*/
			{
				Name: fmt.Sprintf("GET %s with ?format= has expected %s Content-Type and body as-is", row.Name, row.Format),
				Hint: `
				Explicit format still gives correct output, just codec in CID
				`,
				Request: Request().
					Path("ipfs/%s", plainCID).
					Query("format", row.Format),
				Response: Expect().
					Status(200).
					Headers(
						Header("Content-Disposition").
							Contains("%s; filename=\"%s.%s\"", row.Disposition, plainCID, row.Format),
						Header("Content-Type").
							Contains("application/%s", row.Format),
					).Body(
					plain.RawData(),
				),
			},
			/**
			# explicit format still gives correct output, just codec in CID
			test_expect_success "GET $name with Accept has expected $format Content-Type and body as-is" '
			CID=$(echo "{ \"test\": \"plain json\" }" | ipfs dag put --input-codec json --store-codec $format) &&
			curl -sD headers -H "Accept: application/$format" "http://127.0.0.1:$GWAY_PORT/ipfs/$CID" > curl_output 2>&1 &&
			ipfs block get $CID > ipfs_block_output 2>&1 &&
			test_cmp ipfs_block_output curl_output &&
			test_should_contain "Content-Disposition: ${disposition}\; filename=\"${CID}.${format}\"" headers &&
			test_should_contain "Content-Type: application/$format" headers
			'
			*/
			{
				Name: fmt.Sprintf("GET %s with Accept has expected %s Content-Type and body as-is", row.Name, row.Format),
				Hint: `
				Explicit format still gives correct output, just codec in CID
				`,
				Request: Request().
					Path("ipfs/%s", plainCID).
					Header("Accept", fmt.Sprintf("application/%s", row.Format)),
				Response: Expect().
					Status(200).
					Headers(
						Header("Content-Disposition").
							Contains("%s; filename=\"%s.%s\"", row.Disposition, plainCID, row.Format),
						Header("Content-Type").
							Contains("application/%s", row.Format),
					).Body(
					plain.RawData(),
				),
			},
			/**
			# explicit dag-* format passed, attempt to parse as dag* variant
			## Note: this works only for simple JSON that can be upgraded to  DAG-JSON.
			test_expect_success "GET $name with format=dag-$format interprets $format as dag-* variant and produces expected Content-Type and body" '
			CID=$(echo "{ \"test\": \"plain-json-that-can-also-be-dag-json\" }" | ipfs dag put --input-codec json --store-codec $format) &&
			curl -sD headers "http://127.0.0.1:$GWAY_PORT/ipfs/$CID?format=dag-$format" > curl_output_param 2>&1 &&
			ipfs dag get --output-codec dag-$format $CID > ipfs_dag_get_output 2>&1 &&
			test_cmp ipfs_dag_get_output curl_output_param &&
			test_should_contain "Content-Disposition: ${disposition}\; filename=\"${CID}.${format}\"" headers &&
			test_should_contain "Content-Type: application/vnd.ipld.dag-$format" headers &&
			curl -s -H "Accept: application/vnd.ipld.dag-$format" "http://127.0.0.1:$GWAY_PORT/ipfs/$CID" > curl_output_accept 2>&1 &&
			test_cmp curl_output_param curl_output_accept
			'
			*/
			{
				Name: fmt.Sprintf("GET %s with format=dag-%s interprets %s as dag-* variant and produces expected Content-Type and body", row.Name, row.Format, row.Format),
				Hint: `
				Explicit dag-* format passed, attempt to parse as dag* variant
				Note: this works only for simple JSON that can be upgraded to  DAG-JSON.
				`,
				Request: Request().
					Path("ipfs/%s", plainOrDagCID).
					Query("format", fmt.Sprintf("dag-%s", row.Format)),
				Response: Expect().
					Status(200).
					Headers(
						Header("Content-Disposition").
							Contains("%s; filename=\"%s.%s\"", row.Disposition, plainOrDagCID, row.Format),
						Header("Content-Type").
							Contains("application/vnd.ipld.dag-%s", row.Format),
					).Body(
					row.Checker(formatted),
				),
			},
		}

		test.Run(t, tests)
	}
}

// ## Pathing, traversal over DAG-JSON and DAG-CBOR
func TestPathing(t *testing.T) {
	dagJSONTraversal := car.MustOpenUnixfsCar("t0123/dag-json-traversal.car").MustGetRoot()
	dagCBORTraversal := car.MustOpenUnixfsCar("t0123/dag-cbor-traversal.car").MustGetRoot()

	dagJSONTraversalCID := dagJSONTraversal.Cid()
	dagCBORTraversalCID := dagCBORTraversal.Cid()

	tests := SugarTests{
		/**
		  test_expect_success "GET DAG-JSON traversal returns 501 if there is path remainder" '
		  curl -sD - "http://127.0.0.1:$GWAY_PORT/ipfs/$DAG_JSON_TRAVERSAL_CID/foo?format=dag-json" > curl_output 2>&1 &&
		  test_should_contain "501 Not Implemented" curl_output &&
		  test_should_contain "reading IPLD Kinds other than Links (CBOR Tag 42) is not implemented" curl_output
		  '
		*/
		{
			Name: "GET DAG-JSON traversal returns 501 if there is path remainder",
			Request: Request().
				Path("ipfs/%s/foo", dagJSONTraversalCID).
				Query("format", "dag-json"),
			Response: Expect().
				Status(501).
				Body(Contains("reading IPLD Kinds other than Links (CBOR Tag 42) is not implemented")),
		},
		/**
		  test_expect_success "GET DAG-JSON traverses multiple links" '
		  curl -s "http://127.0.0.1:$GWAY_PORT/ipfs/$DAG_JSON_TRAVERSAL_CID/foo/link/bar?format=dag-json" > curl_output 2>&1 &&
		  jq --sort-keys . curl_output > actual &&
		  echo "{ \"hello\": \"this is not a link\" }" | jq --sort-keys . > expected &&
		  test_cmp expected actual
		  '
		*/
		{
			Name: "GET DAG-JSON traverses multiple links",
			Request: Request().
				Path("ipfs/%s/foo/link/bar", dagJSONTraversalCID).
				Query("format", "dag-json"),
			Response: Expect().
				Status(200).
				Body(
					// TODO: I like that this text is readable and easy to understand.
					// 		 but we might prefer matching abstract values, something like "IsJSONEqual(someFixture.formatedAsJSON))"
					IsJSONEqual([]byte(`{"hello": "this is not a link"}`)),
				),
		},
		/**
		  test_expect_success "GET DAG-CBOR traversal returns 501 if there is path remainder" '
		  curl -sD - "http://127.0.0.1:$GWAY_PORT/ipfs/$DAG_CBOR_TRAVERSAL_CID/foo?format=dag-cbor" > curl_output 2>&1 &&
		  test_should_contain "501 Not Implemented" curl_output &&
		  test_should_contain "reading IPLD Kinds other than Links (CBOR Tag 42) is not implemented" curl_output
		  '
		*/
		{
			Name: "GET DAG-CBOR traversal returns 501 if there is path remainder",
			Request: Request().
				Path("ipfs/%s/foo", dagCBORTraversalCID).
				Query("format", "dag-cbor"),
			Response: Expect().
				Status(501).
				Body(Contains("reading IPLD Kinds other than Links (CBOR Tag 42) is not implemented")),
		},
		/**
		  test_expect_success "GET DAG-CBOR traverses multiple links" '
		  curl -s "http://127.0.0.1:$GWAY_PORT/ipfs/$DAG_CBOR_TRAVERSAL_CID/foo/link/bar?format=dag-json" > curl_output 2>&1 &&
		  jq --sort-keys . curl_output > actual &&
		  echo "{ \"hello\": \"this is not a link\" }" | jq --sort-keys . > expected &&
		  test_cmp expected actual
		  '
		*/
		{
			Name: "GET DAG-CBOR traverses multiple links",
			Request: Request().
				Path("ipfs/%s/foo/link/bar", dagCBORTraversalCID).
				Query("format", "dag-json"),
			Response: Expect().
				Status(200).
				Body(
					// TODO: I like that this text is readable and easy to understand.
					// 		 but we might prefer matching abstract values, something like "IsJSONEqual(someFixture.formatedAsJSON))"
					IsJSONEqual([]byte(`{"hello": "this is not a link"}`)),
				),
		},
	}

	test.Run(t, tests)
}

// ## NATIVE TESTS for DAG-JSON (0x0129) and DAG-CBOR (0x71):
// ## DAG- regression tests for core behaviors when native DAG-(CBOR|JSON) is requested
func TestNativeDag(t *testing.T) {
	missingCID := car.RandomCID()

	table := []struct {
		Name        string
		Format      string
		Disposition string
		Checker     func(value []byte) Check[[]byte]
	}{
		{"plain JSON codec", "json", "inline", IsJSONEqual},
		{"plain CBOR codec", "cbor", "attachment", IsEqualBytes},
	}

	for _, row := range table {
		dagTraversal := car.MustOpenUnixfsCar(fmt.Sprintf("t0123/dag-%s-traversal.car", row.Format)).MustGetRoot()
		dagTraversalCID := dagTraversal.Cid()
		formatted := dagTraversal.Formatted("dag-" + row.Format)

		tests := SugarTests{
			/**
			  # GET without explicit format and Accept: text/html returns raw block

			  test_expect_success "GET $name from /ipfs without explicit format returns the same payload as the raw block" '
			  ipfs block get "/ipfs/$CID" > expected &&
			  curl -sX GET "http://127.0.0.1:$GWAY_PORT/ipfs/$CID" -o curl_output &&
			  test_cmp expected curl_output
			  '
			*/
			{
				Name: fmt.Sprintf("GET %s from /ipfs without explicit format returns the same payload as the raw block", row.Name),
				Hint: `GET without explicit format and Accept: text/html returns raw block`,
				Request: Request().
					Path("ipfs/%s", dagTraversalCID),
				Response: Expect().
					Status(200).
					Body(
						row.Checker(formatted),
					),
			},
			/**
			  # GET dag-cbor block via Accept and ?format and ensure both are the same as `ipfs block get` output

			  test_expect_success "GET $name from /ipfs with format=dag-$format returns the same payload as the raw block" '
			  ipfs block get "/ipfs/$CID" > expected &&
			  curl -sX GET "http://127.0.0.1:$GWAY_PORT/ipfs/$CID?format=dag-$format" -o curl_ipfs_dag_param_output &&
			  test_cmp expected curl_ipfs_dag_param_output
			  '
			*/
			{
				Name: fmt.Sprintf("GET %s from /ipfs with format=dag-%s returns the same payload as the raw block", row.Name, row.Format),
				Hint: `GET dag-cbor block via Accept and ?format and ensure both are the same as ipfs block get output`,
				Request: Request().
					Path("ipfs/%s", dagTraversalCID).
					Query("format", fmt.Sprintf("dag-%s", row.Format)),
				Response: Expect().
					Status(200).
					Body(
						row.Checker(formatted),
					),
			},
			/**
			  test_expect_success "GET $name from /ipfs for application/$format returns the same payload as format=dag-$format" '
			  curl -sX GET "http://127.0.0.1:$GWAY_PORT/ipfs/$CID?format=dag-$format" -o expected &&
			  curl -sX GET "http://127.0.0.1:$GWAY_PORT/ipfs/$CID?format=$format" -o plain_output &&
			  test_cmp expected plain_output
			  '
			*/
			// TODO(lidel): Note we disable this test, we check the payloads above.
			/**
			  test_expect_success "GET $name from /ipfs with application/vnd.ipld.dag-$format returns the same payload as the raw block" '
			  ipfs block get "/ipfs/$CID" > expected_block &&
			  curl -sX GET -H "Accept: application/vnd.ipld.dag-$format" "http://127.0.0.1:$GWAY_PORT/ipfs/$CID" -o curl_ipfs_dag_block_accept_output &&
			  test_cmp expected_block curl_ipfs_dag_block_accept_output
			  '
			*/
			{
				Name: fmt.Sprintf("GET %s from /ipfs with application/vnd.ipld.dag-%s returns the same payload as the raw block", row.Name, row.Format),
				Request: Request().
					Path("ipfs/%s", dagTraversalCID).
					Header("Accept", fmt.Sprintf("application/vnd.ipld.dag-%s", row.Format)),
				Response: Expect().
					Status(200).
					Body(
						row.Checker(formatted),
					),
			},
			/**
			  # Make sure DAG-* can be requested as plain JSON or CBOR and response has plain Content-Type for interop purposes

			  test_expect_success "GET $name with format=$format returns same payload as format=dag-$format but with plain Content-Type" '
			  curl -s "http://127.0.0.1:$GWAY_PORT/ipfs/$CID?format=dag-$format" -o expected &&
			  curl -sD plain_headers "http://127.0.0.1:$GWAY_PORT/ipfs/$CID?format=$format" -o plain_output &&
			  test_should_contain "Content-Type: application/$format" plain_headers &&
			  test_cmp expected plain_output
			  '
			*/
			{
				Name: fmt.Sprintf("GET %s with format=%s returns same payload as format=dag-%s but with plain Content-Type", row.Name, row.Format, row.Format),
				Hint: `Make sure DAG-* can be requested as plain JSON or CBOR and response has plain Content-Type for interop purposes`,
				Request: Request().
					Path("ipfs/%s", dagTraversalCID).
					Query("format", row.Format),
				Response: Expect().
					Status(200).
					Header(Header("Content-Type", "application/%s", row.Format)).
					Body(
						row.Checker(formatted),
					),
			},
			/**
			  test_expect_success "GET $name with Accept: application/$format returns same payload as application/vnd.ipld.dag-$format but with plain Content-Type" '
			  curl -s -H "Accept: application/vnd.ipld.dag-$format" "http://127.0.0.1:$GWAY_PORT/ipfs/$CID" > expected &&
			  curl -sD plain_headers -H "Accept: application/$format" "http://127.0.0.1:$GWAY_PORT/ipfs/$CID" > plain_output &&
			  test_should_contain "Content-Type: application/$format" plain_headers &&
			  test_cmp expected plain_output
			  '
			*/
			{
				Name: fmt.Sprintf("GET %s with Accept: application/%s returns same payload as application/vnd.ipld.dag-%s but with plain Content-Type", row.Name, row.Format, row.Format),
				Request: Request().
					Path("ipfs/%s", dagTraversalCID).
					Header("Accept", "application/%s", row.Format),
				Response: Expect().
					Status(200).
					Header(Header("Content-Type", "application/%s", row.Format)).
					Body(
						row.Checker(formatted),
					),
			},
			/**
			  # Make sure expected HTTP headers are returned with the dag- block

			  test_expect_success "GET response for application/vnd.ipld.dag-$format has expected Content-Type" '
			  curl -svX GET -H "Accept: application/vnd.ipld.dag-$format" "http://127.0.0.1:$GWAY_PORT/ipfs/$CID" >/dev/null 2>curl_output &&
			  test_should_contain "< Content-Type: application/vnd.ipld.dag-$format" curl_output
			  '
			  test_expect_success "GET response for application/vnd.ipld.dag-$format includes Content-Length" '
			  BYTES=$(ipfs block get $CID | wc --bytes)
			  test_should_contain "< Content-Length: $BYTES" curl_output
			  '
			  test_expect_success "GET response for application/vnd.ipld.dag-$format includes Content-Disposition" '
			  test_should_contain "< Content-Disposition: ${disposition}\; filename=\"${CID}.${format}\"" curl_output
			  '
			  test_expect_success "GET response for application/vnd.ipld.dag-$format includes nosniff hint" '
			  test_should_contain "< X-Content-Type-Options: nosniff" curl_output
			  '
			*/
			{
				Name: fmt.Sprintf("GET response for application/vnd.ipld.dag-%s has expected Content-Type", row.Format),
				Hint: `Make sure expected HTTP headers are returned with the dag- block`,
				Request: Request().
					Path("ipfs/%s", dagTraversalCID).
					Header("Accept", fmt.Sprintf("application/vnd.ipld.dag-%s", row.Format)),
				Response: Expect().
					Headers(
						Header("Content-Type").Hint("expected Content-Type").Equals("application/vnd.ipld.dag-%s", row.Format),
						Header("Content-Length").Hint("includes Content-Length").Equals("%d", len(dagTraversal.RawData())),
						Header("Content-Disposition").Hint("includes Content-Disposition").Contains("%s; filename=\"%s.%s\"", row.Disposition, dagTraversalCID, row.Format),
						Header("X-Content-Type-Options").Hint("includes nosniff hint").Contains("nosniff"),
					),
			},
			/**
			  test_expect_success "GET for application/vnd.ipld.dag-$format with query filename includes Content-Disposition with custom filename" '
			  curl -svX GET -H "Accept: application/vnd.ipld.dag-$format" "http://127.0.0.1:$GWAY_PORT/ipfs/$CID?filename=foobar.$format" >/dev/null 2>curl_output_filename &&
			  test_should_contain "< Content-Disposition: ${disposition}\; filename=\"foobar.$format\"" curl_output_filename
			  '
			*/
			{
				Name: fmt.Sprintf("GET for application/vnd.ipld.dag-%s with query filename includes Content-Disposition with custom filename", row.Format),
				Request: Request().
					Path("ipfs/%s", dagTraversalCID).
					Query("filename", fmt.Sprintf("foobar.%s", row.Format)).
					Header("Accept", fmt.Sprintf("application/vnd.ipld.dag-%s", row.Format)),
				Response: Expect().
					Headers(
						Header("Content-Disposition").
							Hint("includes Content-Disposition").
							Contains("%s; filename=\"foobar.%s\"", row.Disposition, row.Format),
					),
			},
			/**
			  test_expect_success "GET for application/vnd.ipld.dag-$format with ?download=true forces Content-Disposition: attachment" '
			  curl -svX GET -H "Accept: application/vnd.ipld.dag-$format" "http://127.0.0.1:$GWAY_PORT/ipfs/$CID?filename=foobar.$format&download=true" >/dev/null 2>curl_output_filename &&
			  test_should_contain "< Content-Disposition: attachment" curl_output_filename
			  '
			*/
			{
				Name: fmt.Sprintf("GET for application/vnd.ipld.dag-%s with ?download=true forces Content-Disposition: attachment", row.Format),
				Request: Request().
					Path("ipfs/%s", dagTraversalCID).
					Query("filename", fmt.Sprintf("foobar.%s", row.Format)).
					Query("download", "true").
					Header("Accept", fmt.Sprintf("application/vnd.ipld.dag-%s", row.Format)),
				Response: Expect().
					Headers(
						Header("Content-Disposition").
							Hint("includes Content-Disposition").
							Contains("attachment; filename=\"foobar.%s\"", row.Format),
					),
			},
			/**
			  # Cache control HTTP headers
			  # (basic checks, detailed behavior is tested in  t0116-gateway-cache.sh)

			  test_expect_success "GET response for application/vnd.ipld.dag-$format includes Etag" '
			  test_should_contain "< Etag: \"${CID}.dag-$format\"" curl_output
			  '
			  test_expect_success "GET response for application/vnd.ipld.dag-$format includes X-Ipfs-Path and X-Ipfs-Roots" '
			  test_should_contain "< X-Ipfs-Path" curl_output &&
			  test_should_contain "< X-Ipfs-Roots" curl_output
			  '
			  test_expect_success "GET response for application/vnd.ipld.dag-$format includes Cache-Control" '
			  test_should_contain "< Cache-Control: public, max-age=29030400, immutable" curl_output
			  '
			*/
			{
				Name: fmt.Sprintf("Cache control HTTP headers (%s)", row.Format),
				Hint: `(basic checks, detailed behavior is tested in t0116-gateway-cache.sh)`,
				Request: Request().
					Path("ipfs/%s", dagTraversalCID).
					Header("Accept", fmt.Sprintf("application/vnd.ipld.dag-%s", row.Format)),
				Response: Expect().
					Headers(
						Header("Etag").Hint("includes Etag").Contains("%s.dag-%s", dagTraversalCID, row.Format),
						Header("X-Ipfs-Path").Hint("includes X-Ipfs-Path").Exists(),
						Header("X-Ipfs-Roots").Hint("includes X-Ipfs-Roots").Exists(),
						Header("Cache-Control").Hint("includes Cache-Control").Contains("public, max-age=29030400, immutable"),
					),
			},
			/**
			  # HTTP HEAD behavior
			  test_expect_success "HEAD $name with no explicit format returns HTTP 200" '
			  curl -I "http://127.0.0.1:$GWAY_PORT/ipfs/$CID" -o output &&
			  test_should_contain "HTTP/1.1 200 OK" output &&
			  test_should_contain "Content-Type: application/vnd.ipld.dag-$format" output &&
			  test_should_contain "Content-Length: " output
			  '
			*/
			{
				Name: fmt.Sprintf("HEAD %s with no explicit format returns HTTP 200", row.Name),
				Request: Request().
					Path("ipfs/%s", dagTraversalCID).
					Method("HEAD"),
				Response: Expect().
					Status(200).
					Headers(
						Header("Content-Type").Hint("includes Content-Type").Contains("application/vnd.ipld.dag-%s", row.Format),
						Header("Content-Length").Hint("includes Content-Length").Exists(),
					),
			},
			/**
			  test_expect_success "HEAD $name with an explicit DAG-JSON format returns HTTP 200" '
			  curl -I "http://127.0.0.1:$GWAY_PORT/ipfs/$CID?format=dag-json" -o output &&
			  test_should_contain "HTTP/1.1 200 OK" output &&
			  test_should_contain "Etag: \"$CID.dag-json\"" output &&
			  test_should_contain "Content-Type: application/vnd.ipld.dag-json" output &&
			  test_should_contain "Content-Length: " output
			  '
			*/
			{
				Name: fmt.Sprintf("HEAD %s with an explicit DAG-JSON format returns HTTP 200", row.Name),
				Request: Request().
					Path("ipfs/%s", dagTraversalCID).
					Query("format", "dag-json").
					Method("HEAD"),
				Response: Expect().
					Status(200).
					Headers(
						Header("Etag").Hint("includes Etag").Contains("%s.dag-json", dagTraversalCID),
						Header("Content-Type").Hint("includes Content-Type").Contains("application/vnd.ipld.dag-json"),
						Header("Content-Length").Hint("includes Content-Length").Exists(),
					),
			},
			/**
			  test_expect_success "HEAD $name with only-if-cached for missing block returns HTTP 412 Precondition Failed" '
			  MISSING_CID=$(echo "{\"t\": \"$(date +%s)\"}" | ipfs dag put --store-codec=dag-${format}) &&
			  ipfs block rm -f -q $MISSING_CID &&
			  curl -I -H "Cache-Control: only-if-cached" "http://127.0.0.1:$GWAY_PORT/ipfs/$MISSING_CID" -o output &&
			  test_should_contain "HTTP/1.1 412 Precondition Failed" output
			  '
			*/
			{
				Name: fmt.Sprintf("HEAD %s with only-if-cached for missing block returns HTTP 412 Precondition Failed", row.Name),
				Request: Request().
					Path("ipfs/%s", missingCID).
					Header("Cache-Control", "only-if-cached").
					Method("HEAD"),
				Response: Expect().
					Status(412),
			},
			/**
			  # IPNS behavior (should be same as immutable /ipfs, but with different caching headers)
			  # To keep tests small we only confirm payload is the same, and then only test delta around caching headers.

			  test_expect_success "Prepare IPNS with dag-$format" '
			  IPNS_ID=$(ipfs key gen --ipns-base=base36 --type=ed25519 ${format}_test_key | head -n1 | tr -d "\n") &&
			  ipfs name publish --key ${format}_test_key --allow-offline -Q "/ipfs/$CID" > name_publish_out &&
			  test_check_peerid "${IPNS_ID}" &&
			  ipfs name resolve "${IPNS_ID}" > output &&
			  printf "/ipfs/%s\n" "$CID" > expected &&
			  test_cmp expected output
			  '
			*/
			// TODO: IPNS
			/**
			  test_expect_success "GET $name from /ipns without explicit format returns the same payload as /ipfs" '
			  curl -sX GET "http://127.0.0.1:$GWAY_PORT/ipfs/$CID" -o ipfs_output &&
			  curl -sX GET "http://127.0.0.1:$GWAY_PORT/ipns/$IPNS_ID" -o ipns_output &&
			  test_cmp ipfs_output ipns_output
			  '
			*/
			// TODO: IPNS
			/**
			  test_expect_success "GET $name from /ipns without explicit format returns the same payload as /ipfs" '
			  curl -sX GET "http://127.0.0.1:$GWAY_PORT/ipfs/$CID?format=dag-$format" -o ipfs_output &&
			  curl -sX GET "http://127.0.0.1:$GWAY_PORT/ipns/$IPNS_ID?format=dag-$format" -o ipns_output &&
			  test_cmp ipfs_output ipns_output
			  '
			*/
			/**
			  test_expect_success "GET $name from /ipns with explicit application/vnd.ipld.dag-$format has expected headers" '
			  curl -svX GET -H "Accept: application/vnd.ipld.dag-$format" "http://127.0.0.1:$GWAY_PORT/ipns/$IPNS_ID" >/dev/null 2>curl_output &&
			  test_should_not_contain "Cache-Control" curl_output &&
			  test_should_contain "< Content-Type: application/vnd.ipld.dag-$format" curl_output &&
			  test_should_contain "< Etag: \"${CID}.dag-$format\"" curl_output &&
			  test_should_contain "< X-Ipfs-Path" curl_output &&
			  test_should_contain "< X-Ipfs-Roots" curl_output
			  '
			*/
			// TODO: IPNS
			/**
			  # When Accept header includes text/html and no explicit format is requested for DAG-(CBOR|JSON)
			  # The gateway returns generated HTML index (see dag-index-html) for web browsers (similar to dir-index-html returned for unixfs dirs)
			  # As this is generated, we don't return immutable Cache-Control, even on /ipfs (same as for dir-index-html)

			  test_expect_success "GET $name on /ipfs with Accept: text/html returns HTML (dag-index-html)" '
			  curl -sD - -H "Accept: text/html" "http://127.0.0.1:$GWAY_PORT/ipfs/$CID" > curl_output 2>&1 &&
			  test_should_not_contain "Content-Disposition" curl_output &&
			  test_should_not_contain "Cache-Control" curl_output &&
			  test_should_contain "Etag: \"DagIndex-" curl_output &&
			  test_should_contain "Content-Type: text/html" curl_output &&
			  test_should_contain "</html>" curl_output
			  '
			*/
			// TODO: IPNS
			/**
			  test_expect_success "GET $name on /ipns with Accept: text/html returns HTML (dag-index-html)" '
			  curl -sD - -H "Accept: text/html" "http://127.0.0.1:$GWAY_PORT/ipns/$IPNS_ID" > curl_output 2>&1 &&
			  test_should_not_contain "Content-Disposition" curl_output &&
			  test_should_not_contain "Cache-Control" curl_output &&
			  test_should_contain "Etag: \"DagIndex-" curl_output &&
			  test_should_contain "Content-Type: text/html" curl_output &&
			  test_should_contain "</html>" curl_output
			  '
			*/
		}

		test.Run(t, tests)
	}
}
