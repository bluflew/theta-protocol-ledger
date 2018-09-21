package types

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var chainID string = "test_chain"

func TestCoinbaseTxSignable(t *testing.T) {
	chainID := "test_chain_id"
	va1PrivAcc := PrivAccountFromSecret("validator1")

	coinbaseTx := &CoinbaseTx{
		Proposer: NewTxInput(va1PrivAcc.PrivKey.PublicKey(), Coins{{"", 0}}, 1),
		Outputs: []TxOutput{
			TxOutput{
				Address: getTestAddress("validator1"),
				Coins:   Coins{{"", 333}},
			},
			TxOutput{
				Address: getTestAddress("validator1"),
				Coins:   Coins{{"", 444}},
			},
		},
		BlockHeight: 10,
	}
	signBytes := coinbaseTx.SignBytes(chainID)
	signBytesHex := fmt.Sprintf("%X", signBytes)
	expected := "010D746573745F636861696E5F69640A6A0A400A14F5761FEE27AE9FFB47AFF76811D2F7EDACC6FD531200180122002A2212201D37B3B381CBCD292B1EE5588B9BD9AF971FCCB586D6E2BA50821BED2B1E18E612110A0A76616C696461746F7231120310CD0212110A0A76616C696461746F7231120310BC03180A"

	assert.Equal(t, expected, signBytesHex,
		"Got unexpected sign string for CoinbaseTx. Expected:\n%v\nGot:\n%v", expected, signBytesHex)
}

func TestCoinbaseTxJSON(t *testing.T) {
	assert, require := assert.New(t), require.New(t)

	chainID := "test_chain_id"
	va1PrivAcc := PrivAccountFromSecret("validator1")
	va2PrivAcc := PrivAccountFromSecret("validator2")

	// Construct a CoinbaseTx signature
	tx := &CoinbaseTx{
		Proposer: NewTxInput(va1PrivAcc.PrivKey.PublicKey(), Coins{{"", 0}}, 1),
		Outputs: []TxOutput{
			TxOutput{
				Address: va2PrivAcc.PrivKey.PublicKey().Address(),
				Coins:   Coins{{"foo", 8}},
			},
		},
		BlockHeight: 10,
	}
	tx.Proposer.Signature = va1PrivAcc.Sign(tx.SignBytes(chainID))

	// serialize this as json and back
	js, err := json.Marshal(tx)
	require.Nil(err)
	// fmt.Println(string(js))
	var txs Tx
	err = json.Unmarshal(js, &txs)
	require.Nil(err)
	tx2, ok := txs.(*CoinbaseTx)
	require.True(ok)

	// make sure they are the same!
	signBytes := tx.SignBytes(chainID)
	signBytes2 := tx2.SignBytes(chainID)
	assert.Equal(signBytes, signBytes2)
	assert.Equal(tx, tx2)

	// sign this thing
	sig := va1PrivAcc.Sign(signBytes)
	// we handle both raw sig and wrapped sig the same
	tx.SetSignature(va1PrivAcc.PrivKey.PublicKey().Address(), sig)
	tx2.SetSignature(va1PrivAcc.PrivKey.PublicKey().Address(), sig)
	assert.Equal(tx, tx2)

	// let's marshal / unmarshal this with signature
	js, err = json.Marshal(tx)
	require.Nil(err)
	// fmt.Println(string(js))
	err = json.Unmarshal(js, &txs)
	require.Nil(err)
	tx2, ok = txs.(*CoinbaseTx)
	require.True(ok)

	// and make sure the sig is preserved
	assert.Equal(tx, tx2)
	assert.False(tx2.Proposer.Signature.IsEmpty())
}

func TestSlashTxSignable(t *testing.T) {
	va1PrivAcc := PrivAccountFromSecret("validator1")
	slashTx := &SlashTx{
		Proposer:        NewTxInput(va1PrivAcc.PrivKey.PublicKey(), Coins{{"", 0}}, 1),
		SlashedAddress:  getTestAddress("014FAB"),
		ReserveSequence: 1,
		SlashProof:      []byte("2345ABC"),
	}
	signBytes := slashTx.SignBytes(chainID)
	signBytesHex := fmt.Sprintf("%X", signBytes)
	expected := "010A746573745F636861696E32550A400A14F5761FEE27AE9FFB47AFF76811D2F7EDACC6FD531200180122002A2212201D37B3B381CBCD292B1EE5588B9BD9AF971FCCB586D6E2BA50821BED2B1E18E612063031344641421801220732333435414243"

	assert.Equal(t, expected, signBytesHex,
		"Got unexpected sign string for CoinbaseTx. Expected:\n%v\nGot:\n%v", expected, signBytesHex)
}

func TestSlashTxJSON(t *testing.T) {
	assert, require := assert.New(t), require.New(t)

	chainID := "test_chain_id"
	va1PrivAcc := PrivAccountFromSecret("validator1")

	// Construct a SlashTx signature
	tx := &SlashTx{
		Proposer:        NewTxInput(va1PrivAcc.PrivKey.PublicKey(), Coins{{"", 0}}, 1),
		SlashedAddress:  getTestAddress("014FAB"),
		ReserveSequence: 1,
		SlashProof:      []byte("2345ABC"),
	}

	// serialize this as json and back
	js, err := json.Marshal(tx)
	require.Nil(err)
	// fmt.Println(string(js))
	var txs Tx
	err = json.Unmarshal(js, &txs)
	require.Nil(err)
	tx2, ok := txs.(*SlashTx)
	require.True(ok)

	// make sure they are the same!
	signBytes := tx.SignBytes(chainID)
	signBytes2 := tx2.SignBytes(chainID)
	assert.Equal(signBytes, signBytes2)
	assert.Equal(tx, tx2)

	// sign this thing
	sig := va1PrivAcc.Sign(signBytes)
	// we handle both raw sig and wrapped sig the same
	tx.SetSignature(va1PrivAcc.PrivKey.PublicKey().Address(), sig)
	tx2.SetSignature(va1PrivAcc.PrivKey.PublicKey().Address(), sig)
	assert.Equal(tx, tx2)

	// let's marshal / unmarshal this with signature
	js, err = json.Marshal(tx)
	require.Nil(err)
	// fmt.Println(string(js))
	err = json.Unmarshal(js, &txs)
	require.Nil(err)
	tx2, ok = txs.(*SlashTx)
	require.True(ok)

	// and make sure the sig is preserved
	assert.Equal(tx, tx2)
	assert.False(tx2.Proposer.Signature.IsEmpty())
}

func TestSendTxSignable(t *testing.T) {
	sendTx := &SendTx{
		Gas: 222,
		Fee: Coin{"", 111},
		Inputs: []TxInput{
			TxInput{
				Address:  getTestAddress("input1"),
				Coins:    Coins{{"", 12345}},
				Sequence: 67890,
			},
			TxInput{
				Address:  getTestAddress("input2"),
				Coins:    Coins{{"", 111}},
				Sequence: 222,
			},
		},
		Outputs: []TxOutput{
			TxOutput{
				Address: getTestAddress("output1"),
				Coins:   Coins{{"", 333}},
			},
			TxOutput{
				Address: getTestAddress("output2"),
				Coins:   Coins{{"", 444}},
			},
		},
	}
	signBytes := sendTx.SignBytes(chainID)
	signBytesHex := fmt.Sprintf("%X", signBytes)
	expected := "010A746573745F636861696E125308DE011202106F1A150A06696E70757431120310B96018B2920422002A001A130A06696E707574321202106F18DE0122002A00220E0A076F757470757431120310CD02220E0A076F757470757432120310BC03"

	assert.Equal(t, expected, signBytesHex,
		"Got unexpected sign string for SendTx. Expected:\n%v\nGot:\n%v", expected, signBytesHex)
}

func TestSendTxJSON(t *testing.T) {
	assert, require := assert.New(t), require.New(t)

	chainID := "test_chain_id"
	test1PrivAcc := PrivAccountFromSecret("sendtx1")
	test2PrivAcc := PrivAccountFromSecret("sendtx2")

	// Construct a SendTx signature
	tx := &SendTx{
		Gas: 1,
		Fee: Coin{"foo", 2},
		Inputs: []TxInput{
			NewTxInput(test1PrivAcc.PrivKey.PublicKey(), Coins{{"foo", 10}}, 1),
		},
		Outputs: []TxOutput{
			TxOutput{
				Address: test2PrivAcc.PrivKey.PublicKey().Address(),
				Coins:   Coins{{"foo", 8}},
			},
		},
	}

	// serialize this as json and back
	js, err := json.Marshal(tx)
	require.Nil(err)
	// fmt.Println(string(js))
	var txs Tx
	err = json.Unmarshal(js, &txs)
	require.Nil(err)
	tx2, ok := txs.(*SendTx)
	require.True(ok)

	// make sure they are the same!
	signBytes := tx.SignBytes(chainID)
	signBytes2 := tx2.SignBytes(chainID)
	assert.Equal(signBytes, signBytes2)
	assert.Equal(tx, tx2)

	// sign this thing
	sig := test1PrivAcc.Sign(signBytes)
	// we handle both raw sig and wrapped sig the same
	tx.SetSignature(test1PrivAcc.PrivKey.PublicKey().Address(), sig)
	tx2.SetSignature(test1PrivAcc.PrivKey.PublicKey().Address(), sig)
	assert.Equal(tx, tx2)

	// let's marshal / unmarshal this with signature
	js, err = json.Marshal(tx)
	require.Nil(err)
	// fmt.Println(string(js))
	err = json.Unmarshal(js, &txs)
	require.Nil(err)
	tx2, ok = txs.(*SendTx)
	require.True(ok)

	// and make sure the sig is preserved
	assert.Equal(tx, tx2)
	assert.False(tx2.Inputs[0].Signature.IsEmpty())
}

func TestReserveFundTxSignable(t *testing.T) {
	reserveFundTx := &ReserveFundTx{
		Gas: 222,
		Fee: Coin{"", 111},
		Source: TxInput{
			Address:  getTestAddress("input1"),
			Coins:    Coins{{"", 12345}},
			Sequence: 67890,
		},
		Collateral:  Coins{{"", 22897}},
		ResourceIds: [][]byte{[]byte("rid00123")},
		Duration:    uint64(999),
	}

	signBytes := reserveFundTx.SignBytes(chainID)
	signBytesHex := fmt.Sprintf("%X", signBytes)
	expected := "010A746573745F636861696E1A3108DE011202106F1A150A06696E70757431120310B96018B2920422002A00220410F1B2012A08726964303031323330E707"

	assert.Equal(t, expected, signBytesHex,
		"Got unexpected sign string for ReserveFundTx. Expected:\n%v\nGot:\n%v", expected, signBytesHex)
}

func TestReserveFundTxJSON(t *testing.T) {
	assert, require := assert.New(t), require.New(t)

	chainID := "test_chain_id"
	test1PrivAcc := PrivAccountFromSecret("reservefundtx")

	// Construct a ReserveFundTx signature
	tx := &ReserveFundTx{
		Gas:         222,
		Fee:         Coin{"", 111},
		Source:      NewTxInput(test1PrivAcc.PrivKey.PublicKey(), Coins{{"", 10}}, 1),
		Collateral:  Coins{{"", 22897}},
		ResourceIds: [][]byte{[]byte("rid00123")},
		Duration:    uint64(999),
	}

	// serialize this as json and back
	js, err := json.Marshal(tx)
	require.Nil(err)
	// fmt.Println(string(js))
	var txs Tx
	err = json.Unmarshal(js, &txs)
	require.Nil(err)
	tx2, ok := txs.(*ReserveFundTx)
	require.True(ok)

	// make sure they are the same!
	signBytes := tx.SignBytes(chainID)
	signBytes2 := tx2.SignBytes(chainID)
	assert.Equal(signBytes, signBytes2)
	assert.Equal(tx, tx2)

	// sign this thing
	sig := test1PrivAcc.Sign(signBytes)
	// we handle both raw sig and wrapped sig the same
	tx.SetSignature(test1PrivAcc.PrivKey.PublicKey().Address(), sig)
	tx2.SetSignature(test1PrivAcc.PrivKey.PublicKey().Address(), sig)

	assert.Equal(tx, tx2)

	// let's marshal / unmarshal this with signature
	js, err = json.Marshal(tx)
	require.Nil(err)
	// fmt.Println(string(js))
	err = json.Unmarshal(js, &txs)
	require.Nil(err)
	tx2, ok = txs.(*ReserveFundTx)
	require.True(ok)

	// and make sure the sig is preserved
	assert.Equal(tx, tx2)
	assert.False(tx2.Source.Signature.IsEmpty())
}
func TestReleaseFundTxSignable(t *testing.T) {
	releaseFundTx := &ReleaseFundTx{
		Gas: 222,
		Fee: Coin{"", 111},
		Source: TxInput{
			Address:  getTestAddress("input1"),
			Coins:    Coins{{"", 12345}},
			Sequence: 67890,
		},
		ReserveSequence: 12,
	}

	signBytes := releaseFundTx.SignBytes(chainID)
	signBytesHex := fmt.Sprintf("%X", signBytes)
	expected := "010A746573745F636861696E222008DE011202106F1A150A06696E70757431120310B96018B2920422002A00200C"

	assert.Equal(t, expected, signBytesHex,
		"Got unexpected sign string for ReleaseFundTx. Expected:\n%v\nGot:\n%v", expected, signBytesHex)
}

func TestReleaseFundTxJSON(t *testing.T) {
	assert, require := assert.New(t), require.New(t)

	chainID := "test_chain_id"
	test1PrivAcc := PrivAccountFromSecret("releasefundtx")

	// Construct a ReserveFundTx signature
	tx := &ReserveFundTx{
		Gas:         222,
		Fee:         Coin{"", 111},
		Source:      NewTxInput(test1PrivAcc.PrivKey.PublicKey(), Coins{{"", 10}}, 1),
		Collateral:  Coins{{"", 22897}},
		ResourceIds: [][]byte{[]byte("rid00123")},
		Duration:    uint64(999),
	}

	// serialize this as json and back
	js, err := json.Marshal(tx)
	require.Nil(err)
	// fmt.Println(string(js))
	var txs Tx
	err = json.Unmarshal(js, &txs)
	require.Nil(err)
	tx2, ok := txs.(*ReserveFundTx)
	require.True(ok)

	// make sure they are the same!
	signBytes := tx.SignBytes(chainID)
	signBytes2 := tx2.SignBytes(chainID)
	assert.Equal(signBytes, signBytes2)
	assert.Equal(tx, tx2)

	// sign this thing
	sig := test1PrivAcc.Sign(signBytes)
	// we handle both raw sig and wrapped sig the same
	tx.SetSignature(test1PrivAcc.PrivKey.PublicKey().Address(), sig)
	tx2.SetSignature(test1PrivAcc.PrivKey.PublicKey().Address(), sig)

	assert.Equal(tx, tx2)

	// let's marshal / unmarshal this with signature
	js, err = json.Marshal(tx)
	require.Nil(err)
	// fmt.Println(string(js))
	err = json.Unmarshal(js, &txs)
	require.Nil(err)
	tx2, ok = txs.(*ReserveFundTx)
	require.True(ok)

	// and make sure the sig is preserved
	assert.Equal(tx, tx2)
	assert.False(tx2.Source.Signature.IsEmpty())
}

func TestServicePaymentTxSourceSignable(t *testing.T) {
	servicePaymentTx := &ServicePaymentTx{
		Gas: 222,
		Fee: Coin{"", 111},
		Source: TxInput{
			Address:  getTestAddress("source"),
			Coins:    Coins{{"", 12345}},
			Sequence: 67890,
		},
		Target: TxInput{
			Address:  getTestAddress("target"),
			Coins:    Coins{{"", 0}},
			Sequence: 22341,
		},
		PaymentSequence: 3,
		ReserveSequence: 12,
		ResourceId:      []byte("rid00123"),
	}

	signBytes := servicePaymentTx.SourceSignBytes(chainID)
	signBytesHex := fmt.Sprintf("%X", signBytes)
	expected := "010A746573745F636861696E2A3112001A110A06736F75726365120310B96022002A00220C0A0674617267657422002A002803300C3A087269643030313233"

	assert.Equal(t, expected, signBytesHex,
		"Got unexpected sign string for ServicePaymentTx. Expected:\n%v\nGot:\n%v", expected, signBytesHex)
}

func TestServicePaymentTxTargetSignable(t *testing.T) {
	servicePaymentTx := &ServicePaymentTx{
		Gas: 222,
		Fee: Coin{"", 111},
		Source: TxInput{
			Address:  getTestAddress("source"),
			Coins:    Coins{{"", 12345}},
			Sequence: 67890,
		},
		Target: TxInput{
			Address:  getTestAddress("target"),
			Coins:    Coins{{"", 0}},
			Sequence: 22341,
		},
		PaymentSequence: 3,
		ReserveSequence: 12,
		ResourceId:      []byte("rid00123"),
	}

	signBytes := servicePaymentTx.TargetSignBytes(chainID)
	signBytesHex := fmt.Sprintf("%X", signBytes)
	expected := "010A746573745F636861696E2A4008DE011202106F1A150A06736F75726365120310B96018B2920422002A0022120A06746172676574120018C5AE0122002A002803300C3A087269643030313233"

	assert.Equal(t, expected, signBytesHex,
		"Got unexpected sign string for ServicePaymentTx. Expected:\n%v\nGot:\n%v", expected, signBytesHex)
}

func TestServicePaymentTxJSON(t *testing.T) {
	assert, require := assert.New(t), require.New(t)

	chainID := "test_chain_id"
	sourcePrivAcc := PrivAccountFromSecret("servicepaymenttxsource")
	targetPrivAcc := PrivAccountFromSecret("servicepaymenttxtarget")

	// Construct a ReserveFundTx signature
	tx := &ServicePaymentTx{
		Gas:             222,
		Fee:             Coin{"", 111},
		Source:          NewTxInput(sourcePrivAcc.PrivKey.PublicKey(), Coins{{"foo", 10000}}, 1),
		Target:          NewTxInput(targetPrivAcc.PrivKey.PublicKey(), Coins{{"foo", 0}}, 1),
		PaymentSequence: 3,
		ReserveSequence: 12,
		ResourceId:      []byte("rid00123"),
	}

	// serialize this as json and back
	js, err := json.Marshal(tx)
	require.Nil(err)
	// fmt.Println(string(js))
	var txs Tx
	err = json.Unmarshal(js, &txs)
	require.Nil(err)
	tx2, ok := txs.(*ServicePaymentTx)
	require.True(ok)

	// make sure they are the same!
	sourceSignBytes := tx.SourceSignBytes(chainID)
	sourceSignBytes2 := tx2.SourceSignBytes(chainID)
	assert.Equal(sourceSignBytes, sourceSignBytes2)

	targetSignBytes := tx.TargetSignBytes(chainID)
	targetSignBytes2 := tx2.TargetSignBytes(chainID)
	assert.Equal(targetSignBytes, targetSignBytes2)
}

func TestSplitContractTxSignable(t *testing.T) {
	split := Split{
		Address:    []byte("splitaddr1"),
		Percentage: 30,
	}
	splitContractTx := &SplitContractTx{
		Gas:        222,
		Fee:        Coin{"", 111},
		ResourceId: []byte("rid00123"),
		Initiator: TxInput{
			Address:  getTestAddress("source"),
			Coins:    Coins{{"", 12345}},
			Sequence: 67890,
		},
		Splits:   []Split{split},
		Duration: 99,
	}

	signBytes := splitContractTx.SignBytes(chainID)
	signBytesHex := fmt.Sprintf("%X", signBytes)
	expected := "010A746573745F636861696E3A3A08DE011202106F1A08726964303031323322150A06736F75726365120310B96018B2920422002A002A0E0A0A73706C69746164647231101E3063"

	assert.Equal(t, expected, signBytesHex,
		"Got unexpected sign string for SplitContractTx. Expected:\n%v\nGot:\n%v", expected, signBytesHex)
}

func TestSplitContractTxJSON(t *testing.T) {
	assert, require := assert.New(t), require.New(t)

	chainID := "test_chain_id"
	test1PrivAcc := PrivAccountFromSecret("splitcontracttx")

	// Construct a SplitContractTx signature
	split := Split{
		Address:    []byte("splitaddr1"),
		Percentage: 30,
	}
	tx := &SplitContractTx{
		Gas:        222,
		Fee:        Coin{"", 111},
		ResourceId: []byte("rid00123"),
		Initiator:  NewTxInput(test1PrivAcc.PrivKey.PublicKey(), Coins{{"", 10}}, 1),
		Splits:     []Split{split},
		Duration:   99,
	}

	// serialize this as json and back
	js, err := json.Marshal(tx)
	require.Nil(err)
	// fmt.Println(string(js))
	var txs Tx
	err = json.Unmarshal(js, &txs)
	require.Nil(err)
	tx2, ok := txs.(*SplitContractTx)
	require.True(ok)

	// make sure they are the same!
	signBytes := tx.SignBytes(chainID)
	signBytes2 := tx2.SignBytes(chainID)
	assert.Equal(signBytes, signBytes2)
	assert.Equal(tx, tx2)

	// sign this thing
	sig := test1PrivAcc.Sign(signBytes)
	// we handle both raw sig and wrapped sig the same
	tx.SetSignature(test1PrivAcc.PrivKey.PublicKey().Address(), sig)
	tx2.SetSignature(test1PrivAcc.PrivKey.PublicKey().Address(), sig)

	assert.Equal(tx, tx2)

	// let's marshal / unmarshal this with signature
	js, err = json.Marshal(tx)
	require.Nil(err)
	// fmt.Println(string(js))
	err = json.Unmarshal(js, &txs)
	require.Nil(err)
	tx2, ok = txs.(*SplitContractTx)
	require.True(ok)

	// and make sure the sig is preserved
	assert.Equal(tx, tx2)
	assert.False(tx2.Initiator.Signature.IsEmpty())
}

// func TestUpdateValidatorsTxSignable(t *testing.T) {
// 	updateValidatorsTx := &UpdateValidatorsTx{
// 		Validators: []*abci.Validator{},
// 		Proposer: TxInput{
// 			Address:  []byte("validator1"),
// 			Coins:    Coins{{"", 12345}},
// 			Sequence: 67890,
// 		},
// 	}

// 	signBytes := updateValidatorsTx.SignBytes(chainID)
// 	signBytesHex := fmt.Sprintf("%X", signBytes)
// 	expected := "010A746573745F636861696E"

// 	assert.Equal(t, expected, signBytesHex,
// 		"Got unexpected sign string for UpdateValidatorsTx. Expected:\n%v\nGot:\n%v", expected, signBytesHex)
// }

// func TestUpdateValidatorsTxJSON(t *testing.T) {
// 	assert, require := assert.New(t), require.New(t)

// 	chainID := "test_chain_id"
// 	test1PrivAcc := PrivAccountFromSecret("updatevalidatorstx")

// 	// Construct a UpdateValidatorsTx signature
// 	tx := &UpdateValidatorsTx{
// 		Validators: []*abci.Validator{},
// 		Proposer:   NewTxInput(test1PrivAcc.PubKey, Coins{{"", 10}}, 1),
// 	}

// 	// serialize this as json and back
// 	js, err := json.Marshal(TxS{tx})
// 	require.Nil(err)
// 	// fmt.Println(string(js))
// 	txs := TxS{}
// 	err = json.Unmarshal(js, &txs)
// 	require.Nil(err)
// 	tx2, ok := txs.Tx.(*UpdateValidatorsTx)
// 	require.True(ok)

// 	// make sure they are the same!
// 	signBytes := tx.SignBytes(chainID)
// 	signBytes2 := tx2.SignBytes(chainID)
// 	assert.Equal(signBytes, signBytes2)
// 	assert.Equal(tx, tx2)

// 	// sign this thing
// 	sig := test1PrivAcc.Sign(signBytes)
// 	// we handle both raw sig and wrapped sig the same
// 	tx.SetSignature(test1PrivAcc.PubKey.Address(), sig)
// 	tx2.SetSignature(test1PrivAcc.PubKey.Address(), sig)

// 	assert.Equal(tx, tx2)

// 	// let's marshal / unmarshal this with signature
// 	js, err = json.Marshal(TxS{tx})
// 	require.Nil(err)
// 	// fmt.Println(string(js))
// 	err = json.Unmarshal(js, &txs)
// 	require.Nil(err)
// 	tx2, ok = txs.Tx.(*UpdateValidatorsTx)
// 	require.True(ok)

// 	// and make sure the sig is preserved
// 	assert.Equal(tx, tx2)
// 	assert.False(tx2.Proposer.Signature.Empty())
// }