package tx

import (
	"chain/crypto/sha3pool"
	"chain/protocol/bc"
)

func init() {
	bc.TxHashesFunc = TxHashes
}

// TxHashes returns all hashes needed for validation and state updates.
func TxHashes(oldTx *bc.TxData) (hashes *bc.TxHashes, err error) {
	txid, header, entries, err := mapTx(oldTx)
	if err != nil {
		return nil, err
	}

	hashes = new(bc.TxHashes)
	hashes.ID = bc.Hash(txid)

	// OutputIDs
	for _, resultHash := range header.body.Results {
		result := entries[resultHash]
		if _, ok := result.(*output); ok {
			hashes.OutputIDs = append(hashes.OutputIDs, bc.Hash(resultHash))
		}
	}

	var txRefDataHash bc.Hash // xxx calculate this for the tx

	hashes.VMContexts = make([]*bc.VMContext, len(oldTx.Inputs))

	for entryID, ent := range entries {
		switch ent := ent.(type) {
		case *nonce:
			// xxx check time range is within network-defined limits
			trID := ent.body.TimeRange
			trEntry := entries[trID].(*timeRange) // xxx avoid panics here
			iss := struct {
				ID           bc.Hash
				ExpirationMS uint64
			}{bc.Hash(entryID), trEntry.body.MaxTimeMS}
			hashes.Issuances = append(hashes.Issuances, iss)

		case *issuance:
			vmc := newVMContext(bc.Hash(entryID), hashes.ID, txRefDataHash)
			vmc.RefDataHash = bc.Hash(ent.body.Data) // xxx should this be the id of the data entry? or the hash of the data that's _in_ the data entry?
			vmc.NonceID = (*bc.Hash)(&ent.body.Anchor)
			hashes.VMContexts[ent.Ordinal()] = vmc

		case *spend:
			vmc := newVMContext(bc.Hash(entryID), hashes.ID, txRefDataHash)
			vmc.RefDataHash = bc.Hash(ent.body.Data)
			vmc.OutputID = (*bc.Hash)(&ent.body.SpentOutput)
			hashes.VMContexts[ent.Ordinal()] = vmc
		}
	}

	return hashes, nil
}

// populates the common fields of a VMContext for an Entry, regardless of whether
// that Entry is a Spend or an Issuance
func newVMContext(entryID, txid, txRefDataHash bc.Hash) *bc.VMContext {
	vmc := new(bc.VMContext)

	// TxRefDataHash
	vmc.TxRefDataHash = txRefDataHash

	// EntryID
	vmc.EntryID = entryID

	// TxSigHash
	hasher := sha3pool.Get256()
	defer sha3pool.Put256(hasher)
	hasher.Write(entryID[:])
	hasher.Write(txid[:])
	hasher.Read(vmc.TxSigHash[:])

	return vmc
}
