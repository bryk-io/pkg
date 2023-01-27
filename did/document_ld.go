package did

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/piprate/json-gold/ld"
	"go.bryk.io/pkg/errors"
)

const (
	defaultContext  = "https://www.w3.org/ns/did/v1"
	securityContext = "https://w3id.org/security/v1"
	ed25519Context  = "https://w3id.org/security/suites/ed25519-2020/v1"
	x25519Context   = "https://w3id.org/security/suites/x25519-2020/v1"
	extV1Context    = "https://did-ns.aidtech.network/v1"
)

// https://did-ns.aidtech.network/v1/
var extV1 = `{
  "@context": {
    "id": "@id",
    "type": "@type",
    "@protected": true,
    "extensions": {
      "@id": "https://did-ns.aidtech.network/v1#extension",
      "@container": "@set",
      "@context": {
        "id": {
          "@id": "https://did-ns.aidtech.network/v1#extension-id"
        },
        "version": {
          "@id": "https://did-ns.aidtech.network/v1#extension-version"
        },
        "data": {
          "@id": "https://did-ns.aidtech.network/v1#extension-data",
          "@context": {
            "asset": {
              "@id": "https://did-ns.aidtech.network/v1#algo-connect-asset"
            },
            "address": {
              "@id": "https://did-ns.aidtech.network/v1#algo-connect-address"
            },
            "network": {
              "@id": "https://did-ns.aidtech.network/v1#algo-connect-network"
            }
          }
        }
      }
    }
  }
}`

// https://www.w3.org/ns/did/v1
var didV1 = `{
  "@context": {
    "@protected": true,
    "id": "@id",
    "type": "@type",
    "alsoKnownAs": {
      "@id": "https://www.w3.org/ns/activitystreams#alsoKnownAs",
      "@type": "@id"
    },
    "assertionMethod": {
      "@id": "https://w3id.org/security#assertionMethod",
      "@type": "@id",
      "@container": "@set"
    },
    "authentication": {
      "@id": "https://w3id.org/security#authenticationMethod",
      "@type": "@id",
      "@container": "@set"
    },
    "capabilityDelegation": {
      "@id": "https://w3id.org/security#capabilityDelegationMethod",
      "@type": "@id",
      "@container": "@set"
    },
    "capabilityInvocation": {
      "@id": "https://w3id.org/security#capabilityInvocationMethod",
      "@type": "@id",
      "@container": "@set"
    },
    "controller": {
      "@id": "https://w3id.org/security#controller",
      "@type": "@id"
    },
    "keyAgreement": {
      "@id": "https://w3id.org/security#keyAgreementMethod",
      "@type": "@id",
      "@container": "@set"
    },
    "service": {
      "@id": "https://www.w3.org/ns/did#service",
      "@type": "@id",
      "@context": {
        "@protected": true,
        "id": "@id",
        "type": "@type",
        "serviceEndpoint": {
          "@id": "https://www.w3.org/ns/did#serviceEndpoint",
          "@type": "@id"
        }
      }
    },
    "verificationMethod": {
      "@id": "https://w3id.org/security#verificationMethod",
      "@type": "@id"
    }
  }
}`

// https://w3id.org/security/v1
var securityV1 = `{
  "@context": {
    "id": "@id",
    "type": "@type",

    "dc": "http://purl.org/dc/terms/",
    "sec": "https://w3id.org/security#",
    "xsd": "http://www.w3.org/2001/XMLSchema#",

    "EcdsaKoblitzSignature2016": "sec:EcdsaKoblitzSignature2016",
    "Ed25519Signature2018": "sec:Ed25519Signature2018",
    "EncryptedMessage": "sec:EncryptedMessage",
    "GraphSignature2012": "sec:GraphSignature2012",
    "LinkedDataSignature2015": "sec:LinkedDataSignature2015",
    "LinkedDataSignature2016": "sec:LinkedDataSignature2016",
    "CryptographicKey": "sec:Key",

    "authenticationTag": "sec:authenticationTag",
    "canonicalizationAlgorithm": "sec:canonicalizationAlgorithm",
    "cipherAlgorithm": "sec:cipherAlgorithm",
    "cipherData": "sec:cipherData",
    "cipherKey": "sec:cipherKey",
    "created": {"@id": "dc:created", "@type": "xsd:dateTime"},
    "creator": {"@id": "dc:creator", "@type": "@id"},
    "digestAlgorithm": "sec:digestAlgorithm",
    "digestValue": "sec:digestValue",
    "domain": "sec:domain",
    "encryptionKey": "sec:encryptionKey",
    "expiration": {"@id": "sec:expiration", "@type": "xsd:dateTime"},
    "expires": {"@id": "sec:expiration", "@type": "xsd:dateTime"},
    "initializationVector": "sec:initializationVector",
    "iterationCount": "sec:iterationCount",
    "nonce": "sec:nonce",
    "normalizationAlgorithm": "sec:normalizationAlgorithm",
    "owner": {"@id": "sec:owner", "@type": "@id"},
    "password": "sec:password",
    "privateKey": {"@id": "sec:privateKey", "@type": "@id"},
    "privateKeyPem": "sec:privateKeyPem",
    "publicKey": {"@id": "sec:publicKey", "@type": "@id"},
    "publicKeyBase58": "sec:publicKeyBase58",
    "publicKeyPem": "sec:publicKeyPem",
    "publicKeyWif": "sec:publicKeyWif",
    "publicKeyService": {"@id": "sec:publicKeyService", "@type": "@id"},
    "revoked": {"@id": "sec:revoked", "@type": "xsd:dateTime"},
    "salt": "sec:salt",
    "signature": "sec:signature",
    "signatureAlgorithm": "sec:signingAlgorithm",
    "signatureValue": "sec:signatureValue"
  }
}`

// https://w3id.org/security/suites/ed25519-2020/v1
var ed255192020V1 = `{
  "@context": {
    "id": "@id",
    "type": "@type",
    "@protected": true,
    "proof": {
      "@id": "https://w3id.org/security#proof",
      "@type": "@id",
      "@container": "@graph"
    },
    "Ed25519VerificationKey2020": {
      "@id": "https://w3id.org/security#Ed25519VerificationKey2020",
      "@context": {
        "@protected": true,
        "id": "@id",
        "type": "@type",
        "controller": {
          "@id": "https://w3id.org/security#controller",
          "@type": "@id"
        },
        "revoked": {
          "@id": "https://w3id.org/security#revoked",
          "@type": "http://www.w3.org/2001/XMLSchema#dateTime"
        },
        "publicKeyMultibase": {
          "@id": "https://w3id.org/security#publicKeyMultibase",
          "@type": "https://w3id.org/security#multibase"
        }
      }
    },
    "Ed25519Signature2020": {
      "@id": "https://w3id.org/security#Ed25519Signature2020",
      "@context": {
        "@protected": true,
        "id": "@id",
        "type": "@type",
        "challenge": "https://w3id.org/security#challenge",
        "created": {
          "@id": "http://purl.org/dc/terms/created",
          "@type": "http://www.w3.org/2001/XMLSchema#dateTime"
        },
        "domain": "https://w3id.org/security#domain",
        "expires": {
          "@id": "https://w3id.org/security#expiration",
          "@type": "http://www.w3.org/2001/XMLSchema#dateTime"
        },
        "nonce": "https://w3id.org/security#nonce",
        "proofPurpose": {
          "@id": "https://w3id.org/security#proofPurpose",
          "@type": "@vocab",
          "@context": {
            "@protected": true,
            "id": "@id",
            "type": "@type",
            "assertionMethod": {
              "@id": "https://w3id.org/security#assertionMethod",
              "@type": "@id",
              "@container": "@set"
            },
            "authentication": {
              "@id": "https://w3id.org/security#authenticationMethod",
              "@type": "@id",
              "@container": "@set"
            },
            "capabilityInvocation": {
              "@id": "https://w3id.org/security#capabilityInvocationMethod",
              "@type": "@id",
              "@container": "@set"
            },
            "capabilityDelegation": {
              "@id": "https://w3id.org/security#capabilityDelegationMethod",
              "@type": "@id",
              "@container": "@set"
            },
            "keyAgreement": {
              "@id": "https://w3id.org/security#keyAgreementMethod",
              "@type": "@id",
              "@container": "@set"
            }
          }
        },
        "proofValue": {
          "@id": "https://w3id.org/security#proofValue",
          "@type": "https://w3id.org/security#multibase"
        },
        "verificationMethod": {
          "@id": "https://w3id.org/security#verificationMethod",
          "@type": "@id"
        }
      }
    }
  }
}`

// https://w3id.org/security/suites/x25519-2020/v1
var x255192020V1 = `{
  "@context": {
    "id": "@id",
    "type": "@type",
    "@protected": true,
    "X25519KeyAgreementKey2020": {
      "@id": "https://w3id.org/security#X25519KeyAgreementKey2020",
      "@context": {
        "@protected": true,
        "id": "@id",
        "type": "@type",
        "controller": {
          "@id": "https://w3id.org/security#controller",
          "@type": "@id"
        },
        "revoked": {
          "@id": "https://w3id.org/security#revoked",
          "@type": "http://www.w3.org/2001/XMLSchema#dateTime"
        },
        "publicKeyMultibase": {
          "@id": "https://w3id.org/security#publicKeyMultibase",
          "@type": "https://w3id.org/security#multibase"
        }
      }
    }
  }
}`

// Local LD document loader for offline processing.
var loaderLD *offlineLoader

// Main LD processor instance.
var processorLD *ld.JsonLdProcessor

type offlineLoader struct {
	list map[string]*ld.RemoteDocument
}

func (ol *offlineLoader) init() {
	ol.list = make(map[string]*ld.RemoteDocument)
	base, _ := ld.DocumentFromReader(bytes.NewReader([]byte(didV1)))
	ol.list[defaultContext] = &ld.RemoteDocument{
		DocumentURL: defaultContext,
		ContextURL:  defaultContext,
		Document:    base,
	}
	security, _ := ld.DocumentFromReader(bytes.NewReader([]byte(securityV1)))
	ol.list[securityContext] = &ld.RemoteDocument{
		DocumentURL: securityContext,
		ContextURL:  securityContext,
		Document:    security,
	}
	ed255192020, _ := ld.DocumentFromReader(bytes.NewReader([]byte(ed255192020V1)))
	ol.list[ed25519Context] = &ld.RemoteDocument{
		DocumentURL: ed25519Context,
		ContextURL:  ed25519Context,
		Document:    ed255192020,
	}
	x255192020, _ := ld.DocumentFromReader(bytes.NewReader([]byte(x255192020V1)))
	ol.list[x25519Context] = &ld.RemoteDocument{
		DocumentURL: x25519Context,
		ContextURL:  x25519Context,
		Document:    x255192020,
	}
	extCtx, _ := ld.DocumentFromReader(bytes.NewReader([]byte(extV1)))
	ol.list[extV1Context] = &ld.RemoteDocument{
		DocumentURL: extV1Context,
		ContextURL:  extV1Context,
		Document:    extCtx,
	}
}

func (ol *offlineLoader) LoadDocument(u string) (*ld.RemoteDocument, error) {
	doc, ok := ol.list[u]
	if !ok {
		return nil, fmt.Errorf("missing doc: %s", u)
	}
	return doc, nil
}

// Produces an RDF dataset on the JSON-LD document, the algorithm used is "URDNA2015"
// and the format "application/n-quads".
// https://json-ld.github.io/normalization/spec
func normalize(v interface{}) ([]byte, error) {
	// Intermediate generic representation
	js, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	doc := make(map[string]interface{})
	if err = json.Unmarshal(js, &doc); err != nil {
		return nil, err
	}

	// Setup processor if required
	if processorLD == nil {
		processorLD = ld.NewJsonLdProcessor()
	}

	// Normalize document
	n, err := processorLD.Normalize(doc, ldOptions())
	if err != nil {
		return nil, err
	}
	nd, ok := n.(string)
	if !ok {
		return nil, errors.New("invalid normalized document")
	}
	return []byte(nd), nil
}

// Returns an expanded JSON-LD document.
// http://www.w3.org/TR/json-ld-api/#expansion-algorithm
func expand(v interface{}) ([]byte, error) {
	// Intermediate generic representation
	js, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	doc := make(map[string]interface{})
	if err = json.Unmarshal(js, &doc); err != nil {
		return nil, err
	}

	// Setup processor if required
	if processorLD == nil {
		processorLD = ld.NewJsonLdProcessor()
	}

	// Expand document
	expanded, err := processorLD.Expand(doc, ldOptions())
	if err != nil {
		return nil, err
	}

	// Return encoding of expanded result
	return json.MarshalIndent(expanded, "", "  ")
}

// Return the LD processor configuration options.
func ldOptions() *ld.JsonLdOptions {
	if loaderLD == nil {
		loaderLD = &offlineLoader{}
		loaderLD.init()
	}
	options := ld.NewJsonLdOptions("")
	options.ProcessingMode = ld.JsonLd_1_1
	options.Format = "application/n-quads"
	options.Algorithm = "URDNA2015"
	options.DocumentLoader = loaderLD
	return options
}
