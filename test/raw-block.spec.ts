import { Fixture } from "../util/fixtures.js";
import { config } from "./config.js";
import { run, TestRequestSuiteDefinition } from "declarative-e2e-test";
import { ok } from "assert";
import { Response } from "supertest";

const IPLD_RAW_TYPE = "application/vnd.ipld.raw";

const test: TestRequestSuiteDefinition = {
  "Test HTTP Gateway Raw Block (application/vnd.ipld.raw) Support": {
    tests: {
      "GET with format=raw param returns a raw block": {
        url: `/ipfs/${Fixture.get("dir").cid}/dir?format=raw`,
        expect: [200, Fixture.get("dir").getString("dir")],
      },
      "GET for application/vnd.ipld.raw returns a raw block": {
        url: `/ipfs/${Fixture.get("dir").cid}/dir`,
        headers: { accept: IPLD_RAW_TYPE },
        expect: [200, Fixture.get("dir").getString("dir")],
      },
      "GET response for application/vnd.ipld.raw has expected response headers":
        {
          url: `/ipfs/${Fixture.get("dir").cid}/dir/ascii.txt`,
          headers: { accept: IPLD_RAW_TYPE },
          expect: [
            200,
            {
              headers: {
                "content-type": IPLD_RAW_TYPE,
                "content-length": Fixture.get("dir")
                  .getLength("dir/ascii.txt")
                  .toString(),
                "content-disposition": new RegExp(
                  `attachment;\\s*filename="${Fixture.get("dir").getCID(
                    "dir/ascii.txt"
                  )}\\.bin`
                ),
                "x-content-type-options": "nosniff",
              },
              body: Fixture.get("dir").getString("dir/ascii.txt"),
            },
          ],
        },
      "GET for application/vnd.ipld.raw with query filename includes Content-Disposition with custom filename":
        {
          url: `/ipfs/${Fixture.get(
            "dir"
          ).cid}/dir/ascii.txt?filename=foobar.bin`,
          headers: { accept: IPLD_RAW_TYPE },
          expect: [
            200,
            {
              headers: {
                "content-disposition": new RegExp(
                  `attachment;\\s*filename="foobar\\.bin"`
                ),
              },
            },
          ],
        },
      "GET response for application/vnd.ipld.raw has expected caching headers":
        {
          url: `/ipfs/${Fixture.get("dir").cid}/dir/ascii.txt`,
          headers: { accept: IPLD_RAW_TYPE },
          expect: [
            200,
            {
              headers: {
                etag: `"${Fixture.get("dir").getCID("dir/ascii.txt")}.raw"`,
                "x-ipfs-path": `/ipfs/${Fixture.get(
                  "dir"
                ).cid}/dir/ascii.txt`,
                "x-ipfs-roots": new RegExp(
                  Fixture.get("dir").cid.toString()
                ),
              },
            },
            function (response: Response) {
              const cachePragmata = (
                response.headers["cache-control"] || ""
              ).split(/\s*,\s*/);
              Object.entries({
                "public pragma": (str: string) =>
                  str.toLowerCase() === "public",
                "immutable pragma": (str: string) =>
                  str.toLowerCase() === "immutable",
                "max-age pragma": (str: string) => {
                  if (!/^max-age=/i.test(str)) return false;
                  if (parseInt(str.replace("max-age=", ""), 10) < 29030400)
                    return false; // at least 29030400
                  return true;
                },
              }).forEach(([label, func]) =>
                ok(cachePragmata.find(func), label)
              );
            },
          ],
        },
    },
  },
};

console.log('Running test: raw-block.spec.ts')
run(test, config);
