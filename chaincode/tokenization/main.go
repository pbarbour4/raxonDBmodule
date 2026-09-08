package main

import (
	"log"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

func main() {
	chaincode, err := contractapi.NewChaincode(&TokenContract{}, &CustodianApprovalContract{})
	if err != nil {
		log.Panicf("error creating tokenization chaincode: %v", err)
	}
	if err := chaincode.Start(); err != nil {
		log.Panicf("error starting tokenization chaincode: %v", err)
	}
}
