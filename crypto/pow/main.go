package pow

import (
	"context"
	"encoding/hex"
	"hash"
	"math"
	"math/big"
)

// Source elements can be subject of a Proof-of-Work validation round.
type Source interface {
	// Returns the current value of the "nonce" element.
	Nonce() int64

	// Set the value of the "nonce" element to a secure initial state.
	ResetNonce()

	// Adjust the value of the "nonce" element for a new processing round.
	IncrementNonce()

	// Returns a deterministic byte representation of the source instance.
	MarshalBinary() ([]byte, error)
}

// Solve the proof-of-work challenge for the source instance, i.e. finds
// an appropriate hash value for it based on the specified difficulty level.
// Since this is a potentially long-running operation it can be canceled at
// any time using the provided context.
func Solve(ctx context.Context, src Source, digest hash.Hash, difficulty uint) <-chan string {
	res := make(chan string)
	target := big.NewInt(1)
	target.Lsh(target, 256-difficulty)
	src.ResetNonce()
	go func(target *big.Int) {
		defer close(res)
		var hashInt big.Int
		var h, data []byte
		var err error
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if src.Nonce() < math.MaxInt64 {
					data, err = src.MarshalBinary()
					if err != nil {
						src.IncrementNonce()
						continue
					}
					digest.Reset()
					if _, err = digest.Write(data); err != nil {
						src.IncrementNonce()
						continue
					}
					h = digest.Sum(nil)
					hashInt.SetBytes(h)
					if hashInt.Cmp(target) == -1 {
						res <- hex.EncodeToString(h)
						return
					}
					src.IncrementNonce()
				}
			}
		}
	}(target)
	return res
}

// Verify the source element satisfies the expected proof-of-work challenge
// based on the specified difficulty target.
func Verify(src Source, digest hash.Hash, difficulty uint) bool {
	data, err := src.MarshalBinary()
	if err != nil {
		return false
	}
	if _, err = digest.Write(data); err != nil {
		return false
	}
	target := big.NewInt(1)
	target.Lsh(target, 256-difficulty)
	var hashInt big.Int
	hashInt.SetBytes(digest.Sum(nil))
	return hashInt.Cmp(target) == -1
}
