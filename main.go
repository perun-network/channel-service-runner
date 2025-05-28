package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/nervosnetwork/ckb-sdk-go/v2/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"perun.network/channel-service/deployment"
	"perun.network/channel-service/rpc/proto"
	"perun.network/channel-service/service"
	"perun.network/channel-service/wallet"
	"perun.network/perun-ckb-backend/wallet/address"
	"perun.network/perun-ckb-backend/wallet/external"
	"polycry.pt/poly-go/sortedkv/leveldb"
)

func SetLogFile(path string) {
	logFile, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("error opening file: %v", err)
	}
	log.SetOutput(logFile)
}

func setupWalletServiceClient(url string) proto.WalletServiceClient {
	conn, err := grpc.NewClient(url, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to wallet service: %v", err)
	}

	client := proto.NewWalletServiceClient(conn)
	if client == nil {
		log.Fatalf("Failed to create wallet service client")
	}

	go func() {
		for {
			if conn.GetState() == connectivity.TransientFailure {
				log.Println("WalletServiceClient: Connection lost. Reconnecting...")
				for {
					if conn.GetState() != connectivity.TransientFailure {
						log.Println("WalletServiceClient: Reconnection successful!")
						break
					}
					time.Sleep(1 * time.Second)
					conn, err = grpc.NewClient(url, grpc.WithTransportCredentials(insecure.NewCredentials()))
					if err != nil {
						log.Printf("WalletServiceClient:Error reconnecting: %v\n", err)
					} else {
						client = proto.NewWalletServiceClient(conn)
					}
				}
			}
			time.Sleep(1 * time.Second)
		}
	}()
	return client
}

func main() {
	go Start()
	select {}
}

type Config struct {
	Host              string `json:"host"`
	WSSURL            string `json:"ws_url"`
	Network           string `json:"network"`
	TestnetRPCNodeURL string `json:"testnet_rpc_node_url"`
	DevnetRPCNodeURL  string `json:"devnet_rpc_node_url"`
	PublicKey         string `json:"public_key"`
	SUDTOwnerLockArg  string `json:"sudt_owner_lock_arg"`
	Database          string `json:"database"`
	Logfile           string `json:"logfile"`
}

func Start() {
	log.Println("Start---------")
	// port := flag.String("port", "50051", "The port the gRPC server will listen on")
	configPath := flag.String("config", "config.json", "path to config json file")
	flag.Parse()
	file, err := os.Open(*configPath)
	if err != nil {
		log.Fatalf("Failed to open config file: %v", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}
	var config Config

	if err := json.Unmarshal(data, &config); err != nil {
		log.Fatalf("Failed to parse config file: %v", err)
	}
	SetLogFile(config.Logfile)

	systemScriptsPath := flag.String("system_scripts", "./testnet/default_scripts.json", "path to system scripts json file")
	flag.Parse()
	systemScriptsFile, err := os.Open(*systemScriptsPath)
	if err != nil {
		log.Fatalf("Failed to open system scripts file: %v", err)
	}
	defer systemScriptsFile.Close()

	data, err = io.ReadAll(systemScriptsFile)
	if err != nil {
		log.Fatalf("Failed to read system scripts file: %v", err)
	}
	var systemScripts deployment.SystemScripts
	if err := json.Unmarshal(data, &systemScripts); err != nil {
		log.Fatalf("Failed to parse system scripts file: %v", err)
	}

	deployedScriptsPath := flag.String("migration_data", "./testnet/contracts_cell_deps.json", "path to data on on-chain deployed scripts")
	flag.Parse()
	migrationDataFile, err := os.Open(*deployedScriptsPath)
	if err != nil {
		log.Fatalf("Failed to open migration data file: %v", err)
	}
	defer migrationDataFile.Close()
	data, err = io.ReadAll(migrationDataFile)
	if err != nil {
		log.Fatalf("Failed to read migration data file: %v", err)
	}
	var migrationData deployment.Migration
	if err := json.Unmarshal(data, &migrationData); err != nil {
		log.Fatalf("Failed to parse migration data file: %v", err)
	}
	var network types.Network
	var rpcNodeUrl string
	if config.Network == "testnet" {
		network = types.NetworkTest
		rpcNodeUrl = config.TestnetRPCNodeURL
	} else if config.Network == "devnet" {
		network = types.NetworkTest
		rpcNodeUrl = config.DevnetRPCNodeURL
	} else if config.Network == "mainnet" {
		log.Fatal("Mainnet is not supported yet")
		panic("invalid network type")
	} else {
		log.Fatalf("Invalid network type: %s", config.Network)
		panic("Invalid network type")
	}

	deploy, _, err := GetDeployment(config, systemScripts, migrationData, network)
	if err != nil {
		log.Fatalf("Failed to get deployment: %v", err)
	}

	pubKey, err := GetPubKey(config.PublicKey)
	if err != nil {
		log.Fatalf("Failed to get public key: %v", err)
	}
	participant, err := address.NewDefaultParticipant(&pubKey)
	if err != nil {
		log.Fatalf("Failed to create participant: %v", err)
	}

	db, err := leveldb.LoadDatabase(config.Database)
	if err != nil {
		log.Fatalf("Failed to load database: %v", err)
	}

	listener, err := net.Listen("tcp", config.Host)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", config.Host, err)
	}

	walletServiceClient := setupWalletServiceClient(config.WSSURL)

	channelService, err := service.NewChannelService(walletServiceClient, network, rpcNodeUrl, deploy, nil, db)
	if err != nil {
		log.Fatalf("Failed to create channel service: %v", err)
	}

	channelServiceServer := grpc.NewServer()
	proto.RegisterChannelServiceServer(channelServiceServer, channelService)

	go func() {
		err = channelServiceServer.Serve(listener)
		if err != nil {
			log.Fatalf("Failed to serve channel service: %v", err)
		}
	}()

	_, err = channelService.InitializeUser(*participant, walletServiceClient, external.NewWallet(wallet.NewExternalClient(walletServiceClient)))
	if err != nil {
		log.Fatalf("Failed to initialize user: %v", err)
	}
}
