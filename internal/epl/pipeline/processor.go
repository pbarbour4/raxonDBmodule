package pipeline

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"raxonplatform/internal/config"
	"raxonplatform/internal/db/repository"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const eventReconnectDelay = 5 * time.Second

type Processor struct {
	repo *repository.Repository
	cfg  config.Config
}

func NewProcessor(repo *repository.Repository, cfg config.Config) *Processor {
	return &Processor{repo: repo, cfg: cfg}
}

func (p *Processor) Start(ctx context.Context, channelName string) error {
	log.Printf("epl processor ready channel=%s", channelName)

	for {
		if err := p.subscribeAndProcess(ctx, channelName); err != nil {
			log.Printf("fabric event subscription error, retrying: %v", err)
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(eventReconnectDelay):
		}
	}
}

// subscribeAndProcess connects to the Fabric Gateway and consumes chaincode events until the
// stream ends or the context is cancelled.
func (p *Processor) subscribeAndProcess(ctx context.Context, channelName string) error {
	conn, err := p.dialGateway()
	if err != nil {
		return fmt.Errorf("dial fabric gateway: %w", err)
	}
	defer conn.Close()

	id, sign, err := p.loadIdentity()
	if err != nil {
		return fmt.Errorf("load fabric identity: %w", err)
	}

	gw, err := client.Connect(id, client.WithSign(sign), client.WithClientConnection(conn))
	if err != nil {
		return fmt.Errorf("connect fabric gateway: %w", err)
	}
	defer gw.Close()

	network := gw.GetNetwork(channelName)
	checkpoint, err := p.repo.GetCheckpointHeight(ctx, channelName)
	if err != nil {
		return fmt.Errorf("read channel checkpoint: %w", err)
	}
	options := make([]client.BlockEventsOption, 0, 1)
	if checkpoint >= 0 {
		options = append(options, client.WithStartBlock(uint64(checkpoint+1)))
	}
	blocks, err := network.BlockEvents(ctx, options...)
	if err != nil {
		return fmt.Errorf("open block event stream: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case block, ok := <-blocks:
			if !ok {
				return errors.New("block event stream closed")
			}
			decoded, err := decodeBlock(channelName, p.cfg.FabricChaincodeName, block)
			if err != nil {
				return fmt.Errorf("decode committed block: %w", err)
			}
			if err := p.processBlock(ctx, channelName, decoded); err != nil {
				return err
			}
		}
	}
}

func (p *Processor) processBlock(ctx context.Context, channelName string, block decodedBlock) error {
	if err := p.repo.SaveBlock(ctx, channelName, int64(block.height), block.hash, block.previousHash, len(block.transactions), block.timestamp); err != nil {
		return err
	}
	for _, transaction := range block.transactions {
		if err := p.repo.SaveTransaction(ctx, channelName, transaction.txID, int64(block.height), transaction.index, transaction.validationCode, transaction.chaincode, transaction.function, transaction.args, transaction.timestamp); err != nil {
			return err
		}
		if transaction.validationCode != 0 {
			if err := p.repo.SaveInvalidTransaction(ctx, transaction.txID, int64(block.height), fmt.Sprintf("validation code %d", transaction.validationCode)); err != nil {
				return err
			}
			continue
		}
		for _, envelope := range transaction.events {
			eventID, shouldProcess, err := p.repo.SaveEvent(ctx, envelope)
			if err != nil {
				return fmt.Errorf("persist event tx_id=%s: %w", envelope.TxID, err)
			}
			envelope.EventID = eventID
			if !shouldProcess {
				continue
			}
			if err := p.materialize(ctx, envelope); err != nil {
				return fmt.Errorf("materialize event tx_id=%s: %w", envelope.TxID, err)
			}
			if err := p.repo.MarkEventProcessed(ctx, envelope); err != nil {
				return err
			}
		}
	}
	if err := p.repo.UpdateCheckpoint(ctx, channelName, int64(block.height), block.hash); err != nil {
		return err
	}
	return nil
}

func (p *Processor) dialGateway() (*grpc.ClientConn, error) {
	caCert, err := os.ReadFile(p.cfg.FabricTLSCACertPath)
	if err != nil {
		return nil, fmt.Errorf("read tls ca cert: %w", err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caCert) {
		return nil, errors.New("failed to parse tls ca cert")
	}

	transportCreds := credentials.NewTLS(&tls.Config{
		RootCAs:    pool,
		ServerName: p.cfg.FabricGatewayPeer,
	})

	return grpc.NewClient(p.cfg.FabricPeerEndpoint, grpc.WithTransportCredentials(transportCreds))
}

func (p *Processor) loadIdentity() (*identity.X509Identity, identity.Sign, error) {
	certPEM, err := os.ReadFile(p.cfg.FabricCertPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read client cert: %w", err)
	}
	cert, err := identity.CertificateFromPEM(certPEM)
	if err != nil {
		return nil, nil, fmt.Errorf("parse client cert: %w", err)
	}
	id, err := identity.NewX509Identity(p.cfg.FabricMSPID, cert)
	if err != nil {
		return nil, nil, fmt.Errorf("create x509 identity: %w", err)
	}

	keyPEM, err := os.ReadFile(p.cfg.FabricKeyPath)
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
