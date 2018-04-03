package tester

import (
	"context"
	"crypto/ecdsa"
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

func CurrentChildBlock(
	plasma *contracts.Plasma,
	address string,
) *big.Int {
	opts := createCallOpts(address)

	blocknum, err := plasma.CurrentChildBlock(opts)

	if err != nil {
		panic(err)
	}

	return blocknum
}

func LastExitId(
	plasma *contracts.Plasma,
	address string,
) *big.Int {
	opts := createCallOpts(address)
	exitId, err := plasma.LastExitId(opts)

	if err != nil {
		panic(err)
	}

	return exitId
}

func Finalize(
	plasma *contracts.Plasma,
	privateKeyECDSA *ecdsa.PrivateKey,
) {
	auth := createAuth(privateKeyECDSA)
	tx, err := plasma.Finalize(auth)

	if err != nil {
		panic(err)
	}

	fmt.Printf("Finalize pending: 0x%x\n", tx.Hash())
}

func ChallengeExit(
	plasma *contracts.Plasma,
	privateKeyECDSA *ecdsa.PrivateKey,
	address string,
	txs []chain.Transaction,
	merkle util.MerkleTree,
	blocknum *big.Int,
	txindex *big.Int,
	exitId *big.Int,
) {
	auth := createAuth(privateKeyECDSA)
	bytes, err := rlp.EncodeToBytes(&txs[txindex.Int64()])

	if err != nil {
		panic(err)
	}

	// This must be a tx and it's okay if it's the same block, but could be another.
	// Weird to do down cast but lets try it.
	proof := createMerkleProof(merkle, txindex)

	tx, err := plasma.ChallengeExit(
		auth,
		exitId,
		blocknum,
		txindex,
		bytes,
		proof,
	)

	if err != nil {
		panic(err)
	}

	fmt.Printf("Challenge Exit pending: 0x%x\n", tx.Hash())
}

func StartExit(
	plasma *contracts.Plasma,
	privateKeyECDSA *ecdsa.PrivateKey,
	address string,
	txs []chain.Transaction,
	merkle util.MerkleTree,
	blocknum *big.Int,
	txindex *big.Int,
) {
	auth := createAuth(privateKeyECDSA)
	oindex := new(big.Int).SetUint64(0)
	bytes, err := rlp.EncodeToBytes(&txs[txindex.Int64()])

	if err != nil {
		panic(err)
	}

	proof := createMerkleProof(merkle, txindex)

	tx, err := plasma.StartExit(
		auth,
		blocknum,
		txindex,
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
	t *chain.Transaction,
) {
	auth := createAuth(privateKeyECDSA)
	auth.Value = new(big.Int).SetInt64(int64(value))

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

func createMerkleProof(merkle util.MerkleTree, index *big.Int) []byte {
	proofs := findProofs(&merkle.Root, [][]byte{}, 1)

	if index.Int64() >= int64(len(proofs)) {
		panic("Transaction index must be within set of proofs")
	}

	return proofs[index.Int64()]
}

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

func CreateMerkleTree(accepted []chain.Transaction) util.MerkleTree {
	hashables := make([]util.RLPHashable, len(accepted))

	for i := range accepted {
		txPtr := &accepted[i]
		hashables[i] = util.RLPHashable(txPtr)
	}

	merkle := util.TreeFromRLPItems(hashables)
	return merkle
}
