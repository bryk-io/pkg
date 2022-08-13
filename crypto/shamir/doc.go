/*
Package shamir provides a simple implementation for the Shamir's Secret Sharing algorithm.

Shamir's Secret Sharing is a cryptographic algorithm created by Adi Shamir. It is a form of
secret sharing, where a secret is divided into several unique parts (shares). To reconstruct
the original secret, a minimum number (threshold) of parts is required. In the threshold scheme
this number is less than the total number of parts. Otherwise all participants are needed to
reconstruct the original secret.

Use 'Split' to obtain the shares of a given secret.

	secret := []byte("super-secure-secret")
	shares, err := Split(secret, 5, 3)

Use 'Combine' to restore the original secret from a list of shares.

	secret, err := Combine(shares)

More information:
https://cs.jhu.edu/~sdoshi/crypto/papers/shamirturing.pdf

Based on the original implementation by Hashicorp:
https://www.hashicorp.com/
*/
package shamir
