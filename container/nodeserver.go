package container

import (
	"time"

	"github.com/nacamp/go-simplechain/account"
	"github.com/nacamp/go-simplechain/cmd"
	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/consensus/dpos"
	"github.com/nacamp/go-simplechain/consensus/poa"
	"github.com/nacamp/go-simplechain/consensus/pow"
	"github.com/nacamp/go-simplechain/core"
	"github.com/nacamp/go-simplechain/core/service"
	"github.com/nacamp/go-simplechain/log"
	"github.com/nacamp/go-simplechain/net"
	"github.com/nacamp/go-simplechain/rpc"
	"github.com/nacamp/go-simplechain/storage"
	"github.com/sirupsen/logrus"
)

type NodeServer struct {
	consensus  core.Consensus
	bc         *core.BlockChain
	rpcServer  *rpc.RpcServer
	wallet     *account.Wallet
	bcService  *service.BlockChainService
	db         storage.Storage
	streamPool *net.PeerStreamPool
	node       *net.Node
	config     *cmd.Config
}

func NewNodeServer(config *cmd.Config) *NodeServer {
	var err error
	err = config.VerifyConsensus()
	if err != nil {
		log.CLog().WithFields(logrus.Fields{}).Panic(err)
	}
	ns := NodeServer{config: config}

	ns.streamPool = net.NewPeerStreamPool()

	if config.DBPath == "" {
		ns.db, _ = storage.NewMemoryStorage()
	} else {
		ns.db, _ = storage.NewLevelDBStorage(config.DBPath)
	}
	ns.bc = core.NewBlockChain(ns.db, common.HexToAddress(config.Coinbase), uint64(config.MiningReward))

	ns.wallet = account.NewWallet(config.KeystoreFile)
	ns.wallet.Load()

	privKey, err := config.NodePrivateKey()
	if err != nil {
		log.CLog().WithFields(logrus.Fields{
			"Msg": err,
		}).Panic("NodePrivateKey")
	}
	ns.node = net.NewNode(config.Port, privKey, ns.streamPool)

	if config.Consensus.Name == "dpos" {
		ns.consensus = dpos.NewDpos(ns.streamPool, config.Consensus.Period, config.Consensus.Round, config.Consensus.TotalMiners)
	} else if config.Consensus.Name == "poa" {
		ns.consensus = poa.NewPoa(ns.streamPool, config.Consensus.Period)
	} else {
		ns.consensus = pow.NewPow(ns.streamPool, config.Consensus.Difficulty)
	}

	if config.EnableMining {
		log.CLog().WithFields(logrus.Fields{
			"Address":   config.MinerAddress,
			"Consensus": config.Consensus,
		}).Info("Miner Info")
		err := ns.wallet.TimedUnlock(common.HexToAddress(config.MinerAddress), config.MinerPassphrase, time.Duration(0))
		if err != nil {
			log.CLog().Fatal(err)
		}
		if config.Consensus.Name == "dpos" {
			//? Setup is not suitable to exist in consensus package because setup have wallet(not core package)
			ns.consensus.(*dpos.Dpos).SetupMining(common.HexToAddress(config.MinerAddress), ns.wallet)
		} else if config.Consensus.Name == "poa" {
			ns.consensus.(*poa.Poa).SetupMining(common.HexToAddress(config.MinerAddress), ns.wallet)
		} else {
			ns.consensus.(*pow.Pow).SetupMining(common.HexToAddress(config.MinerAddress), ns.wallet)
		}
	}
	ns.bc.Setup(ns.consensus, cmd.MakeVoterAccountsFromConfig(config))

	ns.bcService = service.NewBlockChainService(ns.bc, ns.streamPool)
	ns.streamPool.AddHandler(ns.bcService)

	ns.rpcServer = rpc.NewRpcServer(config.RpcAddress)
	rpcService := &rpc.RpcService{}
	rpcService.Setup(ns.rpcServer, config, ns.bc, ns.wallet)
	ns.node.Setup(common.HashToHex(ns.bc.GenesisBlock.Hash()))
	return &ns
}

func (ns *NodeServer) Start() {
	ns.node.Start(ns.config.Seeds[0])
	ns.consensus.Start()
	ns.bcService.Start()
	ns.rpcServer.Start()
}
