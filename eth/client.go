package eth

import (
	"context"
	"log"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/kyokan/plasma/util"
	plasma_common "github.com/kyokan/plasma/common"
)

const depositFilter = "0xe1fffcc4923d04b559f4d29a8bfc6cda04eb5b0d3c460751c2402c5c5cc9109c"
const depositDescription = `[{"anonymous":false,"inputs":[{"indexed":false,"name":"sender","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Deposit","type":"event"}]`

const GWEI = 1000000000

var nonce int64 = 0

type DepositEvent struct {
	Sender common.Address
	Value  *big.Int
}

type clientState struct {
	typedClient *ethclient.Client
	rpcClient   *rpc.Client
	useGeth     bool
}


func NewClient(url string) (plasma_common.Client, error) {
	c, err := rpc.Dial(url)

	if err != nil {
		return nil, err
	}
	state := clientState{typedClient: ethclient.NewClient(c), rpcClient: c}
	return &state, nil
}

func (c *clientState) GetBalance(addr common.Address) (*big.Int, error) {
	log.Printf("Attempting to get balance for %s", util.AddressToHex(&addr))

	return c.typedClient.BalanceAt(context.Background(), addr, nil)
}

func (c *clientState) SignData(addr *common.Address, data []byte) ([]byte, error) {
	encoded := hexutil.Encode(data)
	hexAddress := util.AddressToHex(addr)
	log.Printf("Attempting to sign data (%s) on behalf of %s", encoded, hexAddress)
	var res string
	err := c.rpcClient.Call(&res, "eth_sign", hexAddress, encoded)
	log.Printf("Received signature (%s) on behalf of %s.", res, hexAddress)

	if err != nil {
		return nil, err
	}

	return hexutil.Decode(res)
}

// Can be used by plasma client to send a sign transaction request to a remote geth node.
// TODO: needs to be tested.
func (c *clientState) NewGethTransactor(keyAddr common.Address) *bind.TransactOpts {
	gweiPrice := big.NewInt(10)
	nonce++

	return &bind.TransactOpts{
		From: keyAddr,
		Signer: func(signer types.Signer, address common.Address, tx *types.Transaction) (*types.Transaction, error) {
			data := signer.Hash(tx).Bytes()
			signature, err := c.SignData(&address, data)
			if err != nil {
				return nil, err
			}
			return tx.WithSignature(signer, signature)
		},
		// Nonce:    big.NewInt(nonce),
		GasPrice: gweiPrice.Mul(gweiPrice, big.NewInt(GWEI)),
	}
}

func (c *clientState) SubscribeDeposits(address common.Address, resChan chan<- DepositEvent) error {
	query := ethereum.FilterQuery{
		FromBlock: nil,
		ToBlock:   nil,
		Topics:    [][]common.Hash{{common.HexToHash(depositFilter)}},
		Addresses: []common.Address{address},
	}

	ch := make(chan types.Log)
	_, err := c.typedClient.SubscribeFilterLogs(context.TODO(), query, ch)

	if err != nil {
		return err
	}

	log.Printf("Watching for deposits on address %s.", util.AddressToHex(&address))

	depositAbi, err := abi.JSON(strings.NewReader(depositDescription))

	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case event := <-ch:
				parseDepositEvent(&depositAbi, resChan, &event)
			}
		}
	}()

	return nil
}

func parseDepositEvent(depositAbi *abi.ABI, resChan chan<- DepositEvent, raw *types.Log) {
	event := DepositEvent{}
	err := depositAbi.Unpack(&event, "Deposit", raw.Data)

	if err != nil {
		log.Print("Failed to unpack deposit: ", err)
		return
	}

	log.Printf("Received %s wei deposit from %s.", event.Value.String(), util.AddressToHex(&event.Sender))
	resChan <- event
}
