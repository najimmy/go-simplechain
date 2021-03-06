package cmd

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"

	crypto "github.com/libp2p/go-libp2p-crypto"
	peer "github.com/libp2p/go-libp2p-peer"
	b58 "github.com/mr-tron/base58/base58"
	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/core"
	"github.com/nacamp/go-simplechain/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type ConfigAccount struct {
	Address string   `json:"address"`
	Balance *big.Int `json:"balance"`
}

type Consensus struct {
	Name        string   `json:"name"`
	Period      uint64   `json:"period"`
	Round       uint64   `json:"round"`
	TotalMiners uint64   `json:"total_miners"`
	Difficulty  *big.Int `json:"difficulty"`
}

type Config struct {
	HostId          string          `json:"host_id"`
	RpcAddress      string          `json:"rpc_address"`
	DBPath          string          `json:"db_path"`
	MinerAddress    string          `json:"miner_address"`
	MinerPassphrase string          `json:"miner_passphrase"`
	Port            int             `json:"port"`
	Seeds           []string        `json:"seeds"`
	Voters          []ConfigAccount `json:"voters"`
	EnableMining    bool            `json:"enable_mining"`
	Consensus       Consensus       `json:"consensus"`
	NodeKeyPath     string          `json:"node_key_path"`
	KeystoreFile    string          `json:"keystore_file"`
	Coinbase        string          `json:"coinbase"`
	MiningReward    int             `json:"mining_reward"`
}

func MakeVoterAccountsFromConfig(config *Config) (voters []*core.Account) {
	voters = make([]*core.Account, 0)
	for _, voter := range config.Voters {
		account := core.NewAccount()
		copy(account.Address[:], common.FromHex(voter.Address))
		account.Balance = voter.Balance
		voters = append(voters, account)
	}
	return voters
}

func NewConfigFromFile(file string) (config *Config) {
	configFile, err := os.Open(file)
	defer configFile.Close()
	if err != nil {
		log.CLog().WithFields(logrus.Fields{}).Panic(err)
	}
	jsonParser := json.NewDecoder(configFile)
	if err != nil {
		log.CLog().WithFields(logrus.Fields{}).Panic(err)
	}
	config = &Config{}
	err = jsonParser.Decode(config)
	if err != nil {
		log.CLog().WithFields(logrus.Fields{}).Panic(err)
	}
	return config
}

func (c *Config) NodePrivateKey() (key crypto.PrivKey, err error) {
	//os.MkdirAll(instanceDir, 0700)
	nodePrivKeyPath := filepath.Join(c.NodeKeyPath, "node_priv.key")
	if _, err := os.Stat(nodePrivKeyPath); os.IsNotExist(err) {
		//write
		if err := os.MkdirAll(c.NodeKeyPath, 0700); err != nil {
			return nil, err
		}
		priv, pub, err := crypto.GenerateKeyPair(crypto.Secp256k1, 256)
		if err != nil {
			return nil, err
		}

		//private key
		b, err := crypto.MarshalPrivateKey(priv)
		if err != nil {
			return nil, err
		}
		hexStr := hex.EncodeToString(b)
		ioutil.WriteFile(nodePrivKeyPath, []byte(hexStr), 0644)

		//public id
		id, err := peer.IDFromPublicKey(pub)
		if err != nil {
			return nil, err
		}
		ioutil.WriteFile(filepath.Join(c.NodeKeyPath, "node_pub.id"), []byte(b58.Encode([]byte(id))), 0644)

		return priv, nil
	} else {
		// read
		file, err := os.Open(nodePrivKeyPath)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			b, err := hex.DecodeString(scanner.Text())
			if err != nil {
				return nil, err
			}
			priv, err := crypto.UnmarshalPrivateKey(b)
			return priv, err
		}

		if err := scanner.Err(); err != nil {
			return nil, err
		}
		return nil, nil
	}
}

func (c *Config) VerifyConsensus() (err error) {
	if c.Consensus.Name == "dpos" {
		if c.Consensus.Period <= 0 {
			return errors.New("Period must be greater than 0")
		}
		if c.Consensus.Round <= 0 {
			return errors.New("Round must be greater than 0")
		}
		if c.Consensus.TotalMiners <= 0 || c.Consensus.TotalMiners%3 != 0 {
			return errors.New("TotalMiners must be a multiple of three")
		}
		if c.Consensus.TotalMiners > uint64(len(c.Voters)) {
			return errors.New("The number of voters  must be  equal to or greater than TotalMiners")
		}

	}
	return nil
}
