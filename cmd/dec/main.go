package main

import (
	"fmt"

	"github.com/giansalex/cw-load-test/pkg/loadtest"

	"github.com/calvinlauyh/cosmosutils"
)

func main() {
	decoder := cosmosutils.NewDecoder()
	// Register only the interfaces of your interest.
	decoder.RegisterInterfaces(loadtest.RegisterDefaultInterfaces)

	anyBase64EncodedTx := "Cp4BCpsBCiMvY29zbW9zLnN0YWtpbmcudjFiZXRhMS5Nc2dEZWxlZ2F0ZRJ0CitqdW5vMTVhdTZ2MnQ2dmVuNXkzZmF5cW52eHdjM3prbTNqajJzOG15ajQ1EjJqdW5vdmFsb3BlcjE5NHY4dXdlZTJmdnMyczhmYTVrN2owM2t0d2M4N2g1eW0zOWpmdhoRCgV1anVubxIIMTU5NDcyMDkSWApQCkYKHy9jb3Ntb3MuY3J5cHRvLnNlY3AyNTZrMS5QdWJLZXkSIwohA4s8NfCB784sQrvWKiKf/PYUdOS5br72TlH+TcyMuVwAEgQKAggBGFYSBBCA4gkaQMO1d+tsKjN4l0AI+51+HTcA55LjSxlSbK+dKBquVPjvXDYCnw2X/1NvN1S4TAG2HdEjV03i+hsIptUNpV0CCg4="
	tx, err := decoder.DecodeBase64(anyBase64EncodedTx)
	if err != nil {
		panic(err)
	}

	sal, err := tx.MarshalToJSON()
	// Handle the error and work with tx
	fmt.Println(string(sal))
}
