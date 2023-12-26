# Decentralized Identifiers

Decentralized Identifiers (DIDs) are a new type of identifier for verifiable,
"self-sovereign" digital identity. DIDs are fully under the control of the DID
subject, independent from any centralized registry, identity provider, or
certificate authority. DIDs are URLs that relate a DID subject to means for
trustworthy interactions with that subject. DIDs resolve to DID Documents —
simple documents that describe how to use that specific DID. Each DID Document
may contain at least three things: proof purposes, verification methods, and
service endpoints. Proof purposes are combined with verification methods to
provide mechanisms for proving things. For example, a DID Document can specify
that a particular verification method, such as a cryptographic public key, can
be used to verify a proof that was created for the purpose of authentication.
Service endpoints enable trusted interactions with the DID controller.

A DID instance has the following format.

```shell
 did:METHOD:SPECIFIC_ID
```

For example.

```shell
 did:bryk:4d81bd52-2edb-4703-b8fc-b26d514a9c56
```

Once resolved, the DID must provide it's corresponding DID Document.

```json
{
  "@context": [
    "https://www.w3.org/ns/did/v1",
    "https://w3id.org/security/v1"
  ],
  "id": "did:bryk:4d81bd52-2edb-4703-b8fc-b26d514a9c56",
  "created": "2019-03-11T01:42:34+08:00",
  "updated": "2019-06-12T05:09:44Z",
  "publicKey": [
    {
      "id": "did:bryk:4d81bd52-2edb-4703-b8fc-b26d514a9c56#master",
      "type": "Ed25519VerificationKey2018",
      "controller": "did:bryk:4d81bd52-2edb-4703-b8fc-b26d514a9c56",
      "publicKeyHex": "be4db03c2f809aa79ea3055a2da8ddfd807fecd073356e337561cd0640251d9f"
    },
    {
      "id": "did:bryk:4d81bd52-2edb-4703-b8fc-b26d514a9c56#code-sign",
      "type": "Ed25519VerificationKey2018",
      "controller": "did:bryk:4d81bd52-2edb-4703-b8fc-b26d514a9c56",
      "publicKeyHex": "e7cc93d399e467a39fca74e32795b1ab1110a7dc94e8623830cd069c1cac72b8"
    }
  ],
  "authentication": [
    "did:bryk:4d81bd52-2edb-4703-b8fc-b26d514a9c56#master"
  ],
  "proof": {
    "@context": [
      "https://w3id.org/security/v1"
    ],
    "type": "Ed25519Signature2018",
    "creator": "did:bryk:4d81bd52-2edb-4703-b8fc-b26d514a9c56#master",
    "created": "2019-06-12T05:09:47Z",
    "domain": "did.bryk.io",
    "nonce": "57d8c5c54022c1e3635eb95b4dae7524",
    "proofValue": "rmK26r4PFXcOkYLG99rDu8o0wx3i5Gys/Ti6AmiUmd01NvWrW2oo9g/6SPScN2m9Z0u2p+kWMw70rqXBgM8LCQ=="
  }
}
```

## Usage

A new identifier instance can be created either randomly from scratch or by parsing
an existing DID Document. Once created, a DID instance can be used to perform management
tasks using the available functions on the instance's interface. The resulting (updated)
DID Document can be easily obtained from the instance for storage or publish.

```go
// Create a new identifier instance
id, err := NewIdentifierWithMode("bryk", "c137", ModeUUID)
if err != nil {
  panic(err)
}

// Add a new key and enable it as authentication mechanism
_= id.AddNewKey("master", KeyTypeEd, EncodingBase58)
_ = id.AddAuthenticationMethod("master")

// Obtain the DID document of the instance and encode it in JSON format
doc := id.Document(true)
js, _ := json.MarshalIndent(doc, "", "  ")
fmt.Printf("%s", js)
```

## Sign and Verify

A DID instance can be used to produce and verify digitally signed messages. The
signature can be produced as either a JSON-LD document or a raw binary value.

```go
// Get master key
masterKey := id.Key("master")
msg := []byte("original message to sign")

// Get binary message signature
signatureBin, _  := masterKey.Sign(msg)
if !masterKey.Verify(msg, signatureBin) {
  panic("failed to verify binary signature")
}

// Get a JSON-LD message signature
signatureJSON, _ := masterKey.ProduceSignatureLD(msg, "example.com")
if !masterKey.VerifySignatureLD(msg, signatureJSON) {
  panic("failed to verify JSON-LD signature")
}
```

JSON-LD signatures produce a document similar to the following.

```json
 {
   "@context": [
     "https://w3id.org/security/v1"
   ],
   "type": "Ed25519Signature2018",
   "creator": "did:bryk:e4a79533-7466-48a7-b5d4-19caa4679ada#master",
   "created": "2019-08-04T19:20:30Z",
   "domain": "example.com",
   "nonce": "6aa89eea31100f0b6e635fb856c98336",
   "signatureValue": "9coFFyo3Vgq+HJg5yj+QRyub9/5A2sGUfc8ermPV9LEgmV+/Q79jX84ktKo8ZPo0T9MT5TCb/STNGeKBXqbZCw=="
 }
```

More information: <https://w3c-ccg.github.io/did-spec/>
