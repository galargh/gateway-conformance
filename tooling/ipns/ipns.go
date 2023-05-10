package ipns

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"

	"github.com/ipfs/gateway-conformance/tooling/fixtures"
)

/**
 * Extracts the public key from the path of an IPNS record.
 * The path is expected to be in the format of:
 * some/path/then/[pubkey](_anything)?.ipns-record
 */
func extractPubkeyFromPath(path string) (string, error) {
	filename := filepath.Base(path)
	r := regexp.MustCompile(`^(.+?)(_.*|)\.ipns-record$`)
	matches := r.FindStringSubmatch(filename)

	if len(matches) < 2 {
		return "", fmt.Errorf("failed to extract pubkey from path: %s", path)
	}

	return matches[1], nil
}

func OpenIPNSRecordWithKey(absPath string) (*IpnsRecord, error) {
	// name is [pubkey](_anything)?.ipns-record
	pubkey, err := extractPubkeyFromPath(absPath)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}

	r, err := UnmarshalIpnsRecord(data, pubkey)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func MustOpenIPNSRecordWithKey(file string) *IpnsRecord {
	fixturePath := path.Join(fixtures.Dir(), file)
	
	ipnsRecord, err := OpenIPNSRecordWithKey(fixturePath)
	if err != nil {
		panic(err)
	}

	return ipnsRecord
}