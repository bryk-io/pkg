/*
Package wkc provides a "DID Well Known Configuration" implementation.

Making it possible to connect existing systems and Decentralized Identifiers
(DIDs) is an important undertaking that can aid in bootstrapping adoption and
usefulness of DIDs. One such form of connection is the ability of a DID
controller to prove they are the same entity that controls an Internet domain.

The DID Configuration resource provides proof of a bi-directional relationship
between the controller of an Internet domain and a DID via cryptographically
verifiable signatures that are linked to a DID's key material.

The DID Configuration resource MUST exist at the domain root, in the IETF 8615
Well-Known Resource directory, as follows: `/.well-known/did-configuration`.

# Usage

First register a new verification method with a new or existing DID. This key
will be used to generate and validate JWT tokens.

	id, _ := did.NewIdentifier("bryk", uuid.NewString())
	err := RegisterKey(id, "did-jwt-wkc")
	if err != nil {
		panic(err)
	}

Generate as many domain links as required

	dom1, err := GenerateDomainLink(id, "did-jwt-wkc", "acme.com")
	if err != nil {
		panic(err)
	}

Generate the "well known configuration" block. Configuration is commonly exposed as
a JSON document.

	conf := new(Configuration)
	conf.Entries = []*DomainLink{dom1}
	js, _ := json.MarshalIndent(conf, "", "  ")
	fmt.Printf("%s\n", js)

More information:
https://identity.foundation/specs/did-configuration/
*/
package wkc
