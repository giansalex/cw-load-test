package main

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	"github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/giansalex/cw-load-test/pkg/loadtest"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
)

const (
	CointType = 118
	// Bech32Prefix defines the Bech32 prefix used for Cronos Accounts
	Bech32Prefix = "juno"

	// Bech32PrefixAccAddr defines the Bech32 prefix of an account's address
	Bech32PrefixAccAddr = Bech32Prefix
	// Bech32PrefixAccPub defines the Bech32 prefix of an account's public key
	Bech32PrefixAccPub = Bech32Prefix + cosmostypes.PrefixPublic
	// Bech32PrefixValAddr defines the Bech32 prefix of a validator's operator address
	Bech32PrefixValAddr = Bech32Prefix + cosmostypes.PrefixValidator + cosmostypes.PrefixOperator
	// Bech32PrefixValPub defines the Bech32 prefix of a validator's operator public key
	Bech32PrefixValPub = Bech32Prefix + cosmostypes.PrefixValidator + cosmostypes.PrefixOperator + cosmostypes.PrefixPublic
	// Bech32PrefixConsAddr defines the Bech32 prefix of a consensus node address
	Bech32PrefixConsAddr = Bech32Prefix + cosmostypes.PrefixValidator + cosmostypes.PrefixConsensus
	// Bech32PrefixConsPub defines the Bech32 prefix of a consensus node public key
	Bech32PrefixConsPub = Bech32Prefix + cosmostypes.PrefixValidator + cosmostypes.PrefixConsensus + cosmostypes.PrefixPublic
)

func main() {
	// Create a connection to the gRPC server.
	grpcConn, err := grpc.Dial(
		"143.110.235.84:9090", // Or your gRPC server address.
		grpc.WithInsecure(),   // The Cosmos SDK doesn't support any transport security mechanism.
	)
	if err != nil {
		panic(err)
	}
	defer grpcConn.Close()

	configCro()
	// Broadcast the tx via gRPC. We create a new client for the Protobuf Tx
	// service.
	txClient := tx.NewServiceClient(grpcConn)
	txBytes, err := getTxBytes() /* Fill in with your signed transaction bytes. */
	if err != nil {
		panic(err)
	}

	fmt.Println("Run simulate")
	// We then call the Simulate method on this client.
	grpcRes, err := txClient.Simulate(
		context.Background(),
		&tx.SimulateRequest{
			TxBytes: txBytes,
		},
	)
	if err != nil {
		panic(err)
	}

	fmt.Println(grpcRes.GasInfo) // Prints estimated gas used.
}

func getTxBytes() ([]byte, error) {
	signer := loadtest.NewSignature("uni").RegisterInterfaces(loadtest.RegisterDefaultInterfaces)
	info, err := signer.Recover("office attend puppy cash parrot maid raise journey destroy logic dragon horse logic impulse penalty whip typical april exercise dolphin feed between talent exhaust")
	if err != nil {
		return nil, err
	}

	withdrawAddr, _ := cosmostypes.AccAddressFromBech32(info.GetAddress().String())
	msgTx := distrtypes.NewMsgSetWithdrawAddress(withdrawAddr, withdrawAddr)

	txBuilder := signer.GetTxConfig().NewTxBuilder()
	err = txBuilder.SetMsgs(msgTx)
	if err != nil {
		return nil, err
	}
	fee := cosmostypes.NewCoins(cosmostypes.NewCoin("ujunox", cosmostypes.NewInt(5000)))
	txBuilder.SetFeeAmount(fee)
	txBuilder.SetGasLimit(200000)

	return signer.Sign(46187, 179, txBuilder)
}

func configCro() {
	config := cosmostypes.GetConfig()
	config.SetBech32PrefixForAccount(Bech32PrefixAccAddr, Bech32PrefixAccPub)
	config.SetBech32PrefixForValidator(Bech32PrefixValAddr, Bech32PrefixValPub)
	config.SetBech32PrefixForConsensusNode(Bech32PrefixConsAddr, Bech32PrefixConsPub)
	config.SetCoinType(CointType)

	config.Seal()
}
