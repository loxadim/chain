package account

import (
	"context"
	"testing"
	"time"

	"chain/database/pg/pgtest"
	"chain/protocol/bc"
	"chain/protocol/prottest"
)

const sampleAccountUTXOs = `
	INSERT INTO account_utxos
	(tx_hash, index, output_id, unspent_id, asset_id, amount, account_id, control_program_index,
     control_program, confirmed_in) VALUES (
		decode('270b725a94429496a178c56b390a89d03f801fe2ee992d90cf4fdf7d7855318e', 'hex'),
		0,
		decode('9886ae2dc24b6d868c68768038c43801e905a62f1a9b826ca0dc357f00c30117', 'hex'),
		decode('a31a96f07fc4e24c5d1aebd08e882f00324e64c66e9d7081fbbd1019a033e6c5', 'hex'),
		decode('df1df9d4f66437ab5be715e4d1faeb29d24c80a6dc8276d6a630f05c5f1f7693', 'hex'),
		1000, 'accEXAMPLE', 1, '\x6a'::bytea, 1);
`

func TestCancelReservation(t *testing.T) {
	ctx := context.Background()
	c := prottest.NewChain(t)
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)

	_, err := db.Exec(ctx, sampleAccountUTXOs)
	if err != nil {
		t.Fatal(err)
	}

	var h bc.Hash
	var assetID bc.AssetID
	err = h.UnmarshalText([]byte("270b725a94429496a178c56b390a89d03f801fe2ee992d90cf4fdf7d7855318e"))
	if err != nil {
		t.Fatal(err)
	}
	err = assetID.UnmarshalText([]byte("df1df9d4f66437ab5be715e4d1faeb29d24c80a6dc8276d6a630f05c5f1f7693"))
	if err != nil {
		t.Fatal(err)
	}
	outputid := bc.ComputeOutputID(h, 0)
	unspentid := bc.ComputeUnspentID(outputid, bc.Hash{}) // assuming all-zero hash of the output - this test ignores it.

	// Fake the output in the state tree.
	_, s := c.State()
	err = s.Tree.Insert(unspentid.Bytes(), []byte{0xc0, 0x01, 0xca, 0xfe})
	if err != nil {
		t.Error(err)
	}

	utxoDB := newReserver(db, c, nil)
	res, err := utxoDB.ReserveUTXO(ctx, outputid, nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	// Verify that the UTXO is reserved.
	_, err = utxoDB.ReserveUTXO(ctx, outputid, nil, time.Now())
	if err != ErrReserved {
		t.Fatalf("got=%s want=%s", err, ErrReserved)
	}

	// Cancel the reservation.
	err = utxoDB.Cancel(ctx, res.ID)
	if err != nil {
		t.Fatal(err)
	}

	// Reserving again should succeed.
	_, err = utxoDB.ReserveUTXO(ctx, outputid, nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
}
