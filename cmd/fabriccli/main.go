// Command fabriccli is an interactive test client for the tokenization chaincode. One instance
// runs per identity (custodian1, investor1, investor2), each in its own container/terminal.
package main

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"

	"raxonplatform/internal/config"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	cfg := config.Load()
	label := os.Getenv("RAXON_IDENTITY_LABEL")
	if label == "" {
		label = "identity"
	}

	conn, err := dialGateway(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dial gateway: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	id, sign, err := loadIdentity(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load identity: %v\n", err)
		os.Exit(1)
	}

	gw, err := client.Connect(id, client.WithSign(sign), client.WithClientConnection(conn))
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect gateway: %v\n", err)
		os.Exit(1)
	}
	defer gw.Close()

	contract := gw.GetNetwork(cfg.ChannelName).GetContract(cfg.FabricChaincodeName)

	fmt.Printf("fabriccli [%s] connected — channel=%s chaincode=%s\n", label, cfg.ChannelName, cfg.FabricChaincodeName)
	printHelp()
	repl(contract)
}

func repl(contract *client.Contract) {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				fmt.Fprintf(os.Stderr, "scan error: %v\n", err)
			}
			return
		}
		fields := strings.Fields(scanner.Text())
		if len(fields) == 0 {
			continue
		}
		if err := dispatch(contract, fields[0], fields[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		}
	}
}

func dispatch(contract *client.Contract, cmd string, args []string) error {
	switch cmd {
	case "help":
		printHelp()
		return nil
	case "mint":
		return submit(contract, "Mint", args, 3)
	case "burn":
		return submit(contract, "Burn", args, 3)
	case "setnav":
		return submit(contract, "SetNAV", args, 3)
	case "query-balance":
		return evaluate(contract, "QueryBalance", args, 2)
	case "whoami":
		return evaluate(contract, "WhoAmI", args, 0)
	case "request-transfer":
		return submit(contract, "CustodianApprovalContract:RequestTransfer", args, 4)
	case "request-encumbrance":
		return submit(contract, "CustodianApprovalContract:RequestEncumbrance", args, 3)
	case "request-unencumbrance":
		return submit(contract, "CustodianApprovalContract:RequestUnencumbrance", args, 3)
	case "approve":
		return submit(contract, "CustodianApprovalContract:ApproveRequest", args, 1)
	case "reject":
		return submit(contract, "CustodianApprovalContract:RejectRequest", args, 2)
	case "query-request":
		return evaluate(contract, "CustodianApprovalContract:QueryRequest", args, 1)
	default:
		return fmt.Errorf("unknown command %q (type 'help')", cmd)
	}
}

func submit(contract *client.Contract, fn string, args []string, want int) error {
	if len(args) != want {
		return fmt.Errorf("%s expects %d argument(s)", fn, want)
	}
	result, err := contract.SubmitTransaction(fn, args...)
	if err != nil {
		return err
	}
	printResult(result)
	return nil
}

func evaluate(contract *client.Contract, fn string, args []string, want int) error {
	if len(args) != want {
		return fmt.Errorf("%s expects %d argument(s)", fn, want)
	}
	result, err := contract.EvaluateTransaction(fn, args...)
	if err != nil {
		return err
	}
	printResult(result)
	return nil
}

func printResult(result []byte) {
	if len(result) == 0 {
		fmt.Println("ok")
		return
	}
	fmt.Println(string(result))
}

func printHelp() {
	fmt.Println(`commands:
  mint <ownerID> <tokenID> <amount>                      (custodian only)
  burn <ownerID> <tokenID> <amount>                       (custodian only)
  setnav <tokenID> <price> <changePct>                    (custodian only)
  query-balance <ownerID> <tokenID>
  whoami                                                   prints this identity's chain ID (use as ownerID for requests)
  request-transfer <ownerID> <toOwnerID> <tokenID> <amount>       (investor only)
  request-encumbrance <ownerID> <tokenID> <amount>                (investor only)
  request-unencumbrance <ownerID> <tokenID> <amount>              (investor only)
  approve <requestID>                                     (custodian only)
  reject <requestID> <reason>                             (custodian only)
  query-request <requestID>
  help`)
}

func dialGateway(cfg config.Config) (*grpc.ClientConn, error) {
	caCert, err := os.ReadFile(cfg.FabricTLSCACertPath)
	if err != nil {
		return nil, fmt.Errorf("read tls ca cert: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse tls ca cert")
	}
	creds := credentials.NewTLS(&tls.Config{RootCAs: pool, ServerName: cfg.FabricGatewayPeer})
	return grpc.NewClient(cfg.FabricPeerEndpoint, grpc.WithTransportCredentials(creds))
}

func loadIdentity(cfg config.Config) (*identity.X509Identity, identity.Sign, error) {
	certPEM, err := os.ReadFile(cfg.FabricCertPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read client cert: %w", err)
	}
	cert, err := identity.CertificateFromPEM(certPEM)
	if err != nil {
		return nil, nil, fmt.Errorf("parse client cert: %w", err)
	}
	id, err := identity.NewX509Identity(cfg.FabricMSPID, cert)
	if err != nil {
		return nil, nil, fmt.Errorf("create x509 identity: %w", err)
	}
	keyPEM, err := os.ReadFile(cfg.FabricKeyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read client key: %w", err)
	}
	privateKey, err := identity.PrivateKeyFromPEM(keyPEM)
	if err != nil {
		return nil, nil, fmt.Errorf("parse client key: %w", err)
	}
	sign, err := identity.NewPrivateKeySign(privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("create signer: %w", err)
	}
	return id, sign, nil
}
