package util

type Hash []byte

type Hashable interface {
	Hash() Hash
}

type RLPHashable interface {
	RLPHash() Hash
}
