package loadtest

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
)

const (
	accounName = "golang"
)

type Signature struct {
	keyBase            keyring.Keyring
	interfacesRegistry codectypes.InterfaceRegistry
	chainID            string
}

// RegisterInterfaces register decoding interface to the decoder by using the provided interface
// registry.
func (signature *Signature) RegisterInterfaces(registry func(registry codectypes.InterfaceRegistry)) *Signature {
	registry(signature.interfacesRegistry)

	return signature
}

func (signature *Signature) Import(armor, pass string) error {
	kb := keyring.NewInMemory()

	err := kb.ImportPrivKey(accounName, armor, pass)
	if err != nil {
		return err
	}

	signature.keyBase = kb

	return nil
}

func (signature *Signature) Recover(paraphrase string) (keyring.Info, error) {
	kb := keyring.NewInMemory()

	hdPath := hd.CreateHDPath(cosmostypes.GetConfig().GetCoinType(), 0, 0)

	info, err := kb.NewAccount(accounName, paraphrase, "", hdPath.String(), hd.Secp256k1)
	if err != nil {
		return nil, err
	}

	signature.keyBase = kb

	return info, nil
}

func (signature *Signature) GetTxConfig() client.TxConfig {
	marshaler := codec.NewProtoCodec(signature.interfacesRegistry)

	return authtx.NewTxConfig(marshaler, authtx.DefaultSignModes)
}

func (signature *Signature) Sign(accNro, sequence uint64, txBuilder client.TxBuilder) ([]byte, error) {

	marshaler := codec.NewProtoCodec(signature.interfacesRegistry)
	txConfig := authtx.NewTxConfig(marshaler, authtx.DefaultSignModes)

	signMode := signing.SignMode_SIGN_MODE_DIRECT
	key, err := signature.keyBase.Key(accounName)
	if err != nil {
		return nil, err
	}
	pubKey := key.GetPubKey()
	signerData := authsigning.SignerData{
		ChainID:       signature.chainID,
		AccountNumber: accNro,
		Sequence:      sequence,
	}

	sigData := signing.SingleSignatureData{
		SignMode:  signMode,
		Signature: nil,
	}
	sig := signing.SignatureV2{
		PubKey:   pubKey,
		Data:     &sigData,
		Sequence: sequence,
	}

	if err := txBuilder.SetSignatures(sig); err != nil {
		return nil, err
	}

	// Generate the bytes to be signed.
	bytesToSign, err := txConfig.SignModeHandler().GetSignBytes(signMode, signerData, txBuilder.GetTx())
	if err != nil {
		return nil, err
	}

	sigBytes, _, err := signature.keyBase.Sign(accounName, bytesToSign)
	if err != nil {
		return nil, err
	}

	// Construct the SignatureV2 struct
	sigData = signing.SingleSignatureData{
		SignMode:  signMode,
		Signature: sigBytes,
	}
	sig = signing.SignatureV2{
		PubKey:   pubKey,
		Data:     &sigData,
		Sequence: sequence,
	}

	err = txBuilder.SetSignatures(sig)
	if err != nil {
		return nil, err
	}

	parsedTx := txBuilder.GetTx()

	return authtx.DefaultTxEncoder()(parsedTx)
}

func (signature *Signature) ParseJson(json string) (cosmostypes.Tx, error) {

	marshaler := codec.NewProtoCodec(signature.interfacesRegistry)

	return authtx.DefaultJSONTxDecoder(marshaler)([]byte(json))
}

// NewDecoder creates a new decoder
func NewSignature(chainID string) *Signature {
	interfaceRegistry := codectypes.NewInterfaceRegistry()

	return &Signature{
		interfacesRegistry: interfaceRegistry,
		chainID:            chainID,
	}
}
