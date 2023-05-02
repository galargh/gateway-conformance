package check

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ipfs/boxo/ipld/car/v2/blockstore"
	"github.com/ipfs/go-cid"
)

type CheckIsCarFile struct {
	blockCIDs         []cid.Cid
	rootCIDs          []cid.Cid
	blocksWithContent map[cid.Cid][]byte
}

func IsCar() *CheckIsCarFile {
	return &CheckIsCarFile{
		blockCIDs:         []cid.Cid{},
		blocksWithContent: map[cid.Cid][]byte{},
		rootCIDs:          []cid.Cid{},
	}
}

func decoded(cidStr string) cid.Cid {
	cid, err := cid.Decode(cidStr)
	if err != nil {
		panic(fmt.Errorf("invalid CID: %w", err))
	}
	return cid
}

func (c CheckIsCarFile) HasBlock(cidStr string) CheckIsCarFile {
	c.blockCIDs = append(c.blockCIDs, decoded(cidStr))
	return c
}

func (c CheckIsCarFile) HasRoot(cidStr string) CheckIsCarFile {
	c.rootCIDs = append(c.rootCIDs, decoded(cidStr))
	return c
}

func (c CheckIsCarFile) HasBlockWithContent(cidStr string, content []byte) CheckIsCarFile {
	// Clone previous map to prevent side-effects
	current := c.blocksWithContent
	c.blocksWithContent = make(map[cid.Cid][]byte, len(c.blocksWithContent)+1)
	for k, v := range current {
		c.blocksWithContent[k] = v
	}

	// Update with new value
	c.blocksWithContent[decoded(cidStr)] = content

	return c
}

func (c *CheckIsCarFile) Check(carContent []byte) CheckOutput {
	reader := bytes.NewReader(carContent)
	bs, err := blockstore.NewReadOnly(reader, nil)

	if err != nil {
		return CheckOutput{
			Success: false,
			Reason:  fmt.Sprintf("failed to open car file: %v", err),
		}
	}
	defer bs.Close()

	for _, blockCID := range c.blockCIDs {
		has, err := bs.Has(context.Background(), blockCID)
		if err != nil {
			return CheckOutput{
				Success: false,
				Reason:  fmt.Sprintf("failed to check for block '%s': %v", blockCID, err),
			}
		}
		if !has {
			return CheckOutput{
				Success: false,
				Reason:  fmt.Sprintf("block '%s' not found in car file", blockCID),
			}
		}
	}

	for blockCID, expectedContent := range c.blocksWithContent {
		blockData, err := bs.Get(context.Background(), blockCID)
		if err != nil {
			return CheckOutput{
				Success: false,
				Reason:  fmt.Sprintf("failed to get block '%s': %v", blockCID, err),
			}
		}

		b1 := blockData.RawData()
		b2 := expectedContent

		// diff the bytes:
		if !bytes.Equal(b1, b2) {
			return CheckOutput{
				Success: false,
				Reason:  fmt.Sprintf("block '%s' with expected content not found in car file.", blockCID),
			}
		}
	}

	if len(c.rootCIDs) > 0 {
		roots, err := bs.Roots()
		if err != nil {
			return CheckOutput{
				Success: false,
				Reason:  fmt.Sprintf("failed to get roots: %v", err),
			}
		}

		for _, rootCID := range c.rootCIDs {
			// check that rootCID is in roots:
			found := false
			for _, root := range roots {
				if root.Equals(rootCID) {
					found = true
					break
				}
			}

			if !found {
				return CheckOutput{
					Success: false,
					Reason:  fmt.Sprintf("root '%s' not found in car file", rootCID),
				}
			}
		}
	}

	return CheckOutput{
		Success: true,
	}
}
