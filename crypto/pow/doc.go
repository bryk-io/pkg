/*
Package pow provides an implementation of the Proof-of-Work algorithm.

The current Proof-of-Work algorithms are based on the original HashCash proposal.
HashCash was originally designed as a mechanism to throttle systematic abuse of
un-metered internet resources such as email, and anonymous re-mailers in May 1997.
The algorithm is based on utilizing a cost-function that must be efficiently verifiable,
but parameterizable expensive to compute.

Formally, our implementation provides a non-interactive, publicly auditable, trapdoor-free
cost function with unbounded probabilistic cost.

- Non-interactive: The client chose it’s own challenge or random start value

- Publicly Auditable: The produced result can be efficiently verified by any third
party without access to any trapdoor or secret information.

- Trapdoor-free:  The server has no advantage in producing (minting) correct and verifiable
solutions (tokens) to the challenge (cost-function).

- Unbounded Probabilistic Cost: The challenge (cost-function) can in theory take forever
to compute, though the probability of taking significantly longer than expected decreases
rapidly towards zero.

Usage

The package can be used as a library to either produce (mint) correct results to a Proof-of-Work
challenge, or to verify (audit) the correctness of previously generated values (tokens).
Both operations utilize a 'Source' element as target and two main configuration options to
dynamically adjust the algorithm settings.

The provided digest parameter specify the hashing mechanism to use when creating and validating
PoW solutions. The difficulty level is specified as an unsigned integer that determines the
number of bits that must be zeroed when producing the hash to be considered a valid solution to
the challenge.

	// Start a new PoW round using SHA256 and a difficulty level of 16 bits
	res := Solve(ctx, src, sha256.New(), 16)
	log.Printf("solution found: %x", <-res)

	// The solution will be similar to:
	// 0000ff54fb17895b926a1c52efa92d0c86636194612cbbd527d8c931024e5fc6

More information:
http://www.hashcash.org/hashcash.pdf

*/
package pow
