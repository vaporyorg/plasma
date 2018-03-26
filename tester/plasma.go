package tester

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/kyokan/plasma/chain"
	"github.com/kyokan/plasma/contracts/gen/contracts"
	"github.com/kyokan/plasma/util"
)

func StartExit(
	plasma *contracts.Plasma,
	privateKeyECDSA *ecdsa.PrivateKey,
	address string,
	txs []chain.Transaction,
	merkle util.MerkleTree,
	txindex int,
) {
	auth := createAuth(privateKeyECDSA)
	opts := createCallOpts(address)

	blocknum, err := plasma.CurrentChildBlock(opts)

	if err != nil {
		panic(err)
	}

	oindex := new(big.Int).SetUint64(0)

	bytes, err := rlp.EncodeToBytes(&txs[txindex])

	if err != nil {
		panic(err)
	}

	proof := createMerkleProof(merkle, txindex)

	//    1
	//  1   2
	// 1 2 3 4
	fmt.Println("**** proof")
	fmt.Println(hex.EncodeToString(proof))

	tx, err := plasma.StartExit(
		auth,
		new(big.Int).SetInt64(blocknum.Int64()-1),
		new(big.Int).SetInt64(int64(txindex)),
		oindex,
		bytes,
		proof,
	)

	if err != nil {
		panic(err)
	}

	fmt.Printf("Start Exit pending: 0x%x\n", tx.Hash())
}

func SubmitBlock(
	plasma *contracts.Plasma,
	privateKeyECDSA *ecdsa.PrivateKey,
	address string,
	txs []chain.Transaction,
	merkle util.MerkleTree,
) {
	auth := createAuth(privateKeyECDSA)

	var root [32]byte
	copy(root[:], merkle.Root.Hash[:32])
	tx, err := plasma.SubmitBlock(auth, root)

	if err != nil {
		log.Fatalf("Failed to submit block: %v", err)
	}

	fmt.Printf("Submit block pending: 0x%x\n", tx.Hash())
}

func Deposit(
	plasma *contracts.Plasma,
	privateKeyECDSA *ecdsa.PrivateKey,
	address string,
	value int,
) {
	auth := createAuth(privateKeyECDSA)
	auth.Value = new(big.Int).SetInt64(int64(value))

	t := createTestTransaction(address, value)
	bytes, err := rlp.EncodeToBytes(&t)

	if err != nil {
		panic(err)
	}

	tx, err := plasma.Deposit(auth, bytes)

	if err != nil {
		log.Fatalf("Failed to deposit: %v", err)
	}

	fmt.Printf("Deposit pending: 0x%x\n", tx.Hash())
}

func CreateTransactions(address string) []chain.Transaction {
	t1 := createTestTransaction(address, 100)
	t2 := createTestTransaction(address, 200)
	t3 := createTestTransaction(address, 300)
	return []chain.Transaction{t1, t2, t3}
}

func createAuth(privateKeyECDSA *ecdsa.PrivateKey) *bind.TransactOpts {
	auth := bind.NewKeyedTransactor(privateKeyECDSA)
	auth.GasPrice = new(big.Int).SetUint64(1)
	auth.GasLimit = uint64(4712388)
	return auth
}

func createCallOpts(address string) *bind.CallOpts {
	return &bind.CallOpts{
		From:    common.HexToAddress(address),
		Context: context.Background(),
	}
}

func createMerkleProof(merkle util.MerkleTree, index int) []byte {
	proofs := findProofs(&merkle.Root, [][]byte{}, 1)

	if index >= len(proofs) {
		panic("Transaction index must be within set of proofs")
	}

	return proofs[index]
}

// Note that this tree will be left heavy and always have a right node, even if it has a hash of zero bytes.
// TODO: we could optimize this with an index.
func findProofs(node *util.MerkleNode, curr [][]byte, depth int) [][]byte {
	if node.Left == nil && node.Right == nil {
		if depth == 16 {
			// Reverse it.
			var copyCurr []byte

			for i := len(curr) - 1; i >= 0; i-- {
				copyCurr = append(copyCurr, curr[i]...)
			}

			return [][]byte{copyCurr}
		}

		return [][]byte{}
	}

	var left [][]byte
	var right [][]byte

	if node.Left != nil {
		left = findProofs(node.Left, append(curr, node.Right.Hash), depth+1)
	}

	if node.Right != nil {
		right = findProofs(node.Right, append(curr, node.Left.Hash), depth+1)
	}

	return append(left, right...)
}

func createMerkleRoot(merkle util.MerkleTree) [32]byte {
	var res [32]byte
	hash := merkle.Root.Hash

	for i := 0; i < Min(len(res), len(hash)); i++ {
		res[i] = hash[i]
	}

	return res
}

func CreateMerkleTree(accepted []chain.Transaction) util.MerkleTree {
	hashables := make([]util.RLPHashable, len(accepted))

	for i := range accepted {
		txPtr := &accepted[i]
		hashables[i] = util.RLPHashable(txPtr)
	}

	merkle := util.TreeFromRLPItems(hashables)
	return merkle
}

func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func createTestTransaction(address string, amount int) chain.Transaction {
	return chain.Transaction{
		Input0: chain.ZeroInput(),
		Input1: chain.ZeroInput(),
		Sig0:   []byte{},
		Sig1:   []byte{},
		Output0: &chain.Output{
			NewOwner: common.HexToAddress(address),
			Amount:   new(big.Int).SetInt64(int64(amount)),
		},
		Output1: chain.ZeroOutput(),
		Fee:     new(big.Int),
		BlkNum:  uint64(0),
		TxIdx:   0,
	}
}
