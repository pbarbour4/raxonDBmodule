package pipeline

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"raxonplatform/internal/config"
	"raxonplatform/internal/contracts"
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
	events, err := network.ChaincodeEvents(ctx, p.cfg.FabricChaincodeName)
	if err != nil {
		return fmt.Errorf("open chaincode event stream: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case evt, ok := <-events:
			if !ok {
				return errors.New("chaincode event stream closed")
			}
			envelope, err := toEventEnvelope(channelName, evt)
			if err != nil {
				log.Printf("decode event payload failed tx_id=%s: %v", evt.TransactionID, err)
				continue
			}
			if err := p.repo.SaveEvent(ctx, envelope); err != nil {
				log.Printf("persist event failed event_id=%s: %v", envelope.EventID, err)
				continue
			}
			if err := p.materialize(ctx, envelope); err != nil {
				log.Printf("materialize event failed event_id=%s: %v", envelope.EventID, err)
			}
		}
	}
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

func toEventEnvelope(channelName string, evt *client.ChaincodeEvent) (contracts.EventEnvelope, error) {
	var payload map[string]any
	if len(evt.Payload) > 0 {
		if err := json.Unmarshal(evt.Payload, &payload); err != nil {
			return contracts.EventEnvelope{}, fmt.Errorf("unmarshal event payload: %w", err)
		}
	}
	return contracts.EventEnvelope{
		EventID:            fmt.Sprintf("%s-%d", evt.TransactionID, evt.BlockNumber),
		ChannelName:        channelName,
		BlockHeight:        int64(evt.BlockNumber),
		TxID:               evt.TransactionID,
		ChaincodeEventName: evt.EventName,
		EventType:          evt.EventName,
		Payload:            payload,
		OccurredAt:         time.Now().UTC(),
		IngestedAt:         time.Now().UTC(),
	}, nil
}
