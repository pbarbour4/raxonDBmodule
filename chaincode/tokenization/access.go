package main

import (
	"fmt"

	"github.com/hyperledger/fabric-chaincode-go/v2/pkg/cid"
	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

const roleAttribute = "role"

// requireCustodian returns an error unless the invoking identity's CA attribute role=custodian.
func requireCustodian(ctx contractapi.TransactionContextInterface) error {
	role, ok, err := cid.GetAttributeValue(ctx.GetStub(), roleAttribute)
	if err != nil {
		return fmt.Errorf("read role attribute: %w", err)
	}
	if !ok || role != "custodian" {
		return fmt.Errorf("caller is not authorized as custodian")
	}
	return nil
}

// requireInvestorOwner returns an error unless the caller has role=investor and is the owner
// of the balance being acted on, so investors cannot move funds belonging to other owners.
func requireInvestorOwner(ctx contractapi.TransactionContextInterface, ownerID string) error {
	role, ok, err := cid.GetAttributeValue(ctx.GetStub(), roleAttribute)
	if err != nil {
		return fmt.Errorf("read role attribute: %w", err)
	}
	if !ok || role != "investor" {
		return fmt.Errorf("caller is not authorized as investor")
	}
	callerID, err := cid.GetID(ctx.GetStub())
	if err != nil {
		return fmt.Errorf("read caller id: %w", err)
	}
	if callerID != ownerID {
		return fmt.Errorf("caller is not the owner of this balance")
	}
	return nil
}
