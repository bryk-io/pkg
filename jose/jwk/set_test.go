package jwk

import (
	"encoding/json"
	"testing"

	tdd "github.com/stretchr/testify/assert"
	"go.bryk.io/pkg/jose/jwa"
)

func TestSetFindByID(t *testing.T) {
	assert := tdd.New(t)

	set := &Set{
		Keys: []Record{
			{KeyID: "key1", KeyType: "RSA", Alg: "RS256"},
			{KeyID: "key2", KeyType: "EC", Alg: "ES256"},
			{KeyID: "key3", KeyType: "oct", Alg: "HS256"},
		},
	}

	// Find existing key
	key, found := set.FindByID("key2")
	assert.True(found, "should find key2")
	assert.Equal("key2", key.KeyID)
	assert.Equal("EC", key.KeyType)

	// Find non-existing key
	_, found = set.FindByID("nonexistent")
	assert.False(found, "should not find nonexistent key")

	// Empty set
	emptySet := &Set{}
	_, found = emptySet.FindByID("key1")
	assert.False(found, "should not find key in empty set")
}

func TestSetFind(t *testing.T) {
	assert := tdd.New(t)

	set := &Set{
		Keys: []Record{
			{KeyID: "key1", KeyType: "RSA", Alg: "RS256", Use: "sig"},
			{KeyID: "key2", KeyType: "EC", Alg: "ES256", Use: "sig"},
			{KeyID: "key3", KeyType: "oct", Alg: "HS256", Use: "enc"},
		},
	}

	// Find by ID
	key, found := set.Find(ByID("key2"))
	assert.True(found)
	assert.Equal("key2", key.KeyID)

	// Find by multiple selectors
	key, found = set.Find(ByKeyType("RSA"), ByAlg("RS256"))
	assert.True(found)
	assert.Equal("key1", key.KeyID)

	// Find with no match
	_, found = set.Find(ByID("nonexistent"))
	assert.False(found)

	// Find with conflicting selectors
	_, found = set.Find(ByID("key1"), ByKeyType("EC"))
	assert.False(found)
}

func TestSetFilter(t *testing.T) {
	assert := tdd.New(t)

	set := &Set{
		Keys: []Record{
			{KeyID: "key1", KeyType: "RSA", Alg: "RS256", Use: "sig"},
			{KeyID: "key2", KeyType: "EC", Alg: "ES256", Use: "sig"},
			{KeyID: "key3", KeyType: "oct", Alg: "HS256", Use: "enc"},
			{KeyID: "key4", KeyType: "RSA", Alg: "RS384", Use: "sig"},
		},
	}

	// Filter by key type
	rsaKeys := set.Filter(ByKeyType("RSA"))
	assert.Len(rsaKeys, 2)
	assert.Equal("key1", rsaKeys[0].KeyID)
	assert.Equal("key4", rsaKeys[1].KeyID)

	// Filter by use
	sigKeys := set.Filter(ByUse("sig"))
	assert.Len(sigKeys, 3)

	// Filter with multiple selectors
	filtered := set.Filter(ByKeyType("RSA"), ByUse("sig"))
	assert.Len(filtered, 2)

	// No selectors - return all
	all := set.Filter()
	assert.Len(all, 4)

	// No matches
	none := set.Filter(ByAlg("none"))
	assert.Len(none, 0)
}

func TestOperationSelectors(t *testing.T) {
	assert := tdd.New(t)

	set := &Set{
		Keys: []Record{
			{KeyID: "key1", KeyType: "RSA", Alg: "RS256", Use: "sig", KeyOps: []string{"sign", "verify"}},
			{KeyID: "key2", KeyType: "EC", Alg: "ES256", Use: "sig", KeyOps: []string{"verify"}},
			{KeyID: "key3", KeyType: "oct", Alg: "HS256", Use: "enc", KeyOps: []string{"encrypt", "decrypt"}},
			{KeyID: "key4", KeyType: "RSA", Alg: "RS256", Use: "enc", KeyOps: []string{"wrapKey", "unwrapKey"}},
			{KeyID: "key5", KeyType: "RSA", Alg: "RS256", Use: "", KeyOps: []string{}},
		},
	}

	// CanSign - keys with Use="sig" or no use/key_ops, and with sign op or no key_ops
	// key1 has Use="sig" and "sign" op -> matches
	// key2 has Use="sig" but only "verify" op, no "sign" -> does NOT match
	// key5 has no Use/key_ops -> matches (permissive)
	signKeys := set.Filter(CanSign())
	assert.Len(signKeys, 2) // key1, key5

	// CanVerify - keys with Use="sig" or no use/key_ops, and with verify op or no key_ops
	// key1 has Use="sig" and "verify" op -> matches
	// key2 has Use="sig" and "verify" op -> matches
	// key5 has no Use/key_ops -> matches
	verifyKeys := set.Filter(CanVerify())
	assert.Len(verifyKeys, 3) // key1, key2, key5

	// CanEncrypt - keys with Use="enc" or no use/key_ops, and with encrypt/wrapKey op or no key_ops
	// key3 has Use="enc" and "encrypt" op -> matches
	// key4 has Use="enc" and "wrapKey" op -> matches (wrapKey is for key encryption)
	// key5 has no Use/key_ops -> matches
	encryptKeys := set.Filter(CanEncrypt())
	assert.Len(encryptKeys, 3) // key3, key4, key5

	// CanDecrypt - keys with Use="enc" or no use/key_ops, and with decrypt/unwrapKey op or no key_ops
	// key3 has Use="enc" and "decrypt" op -> matches
	// key4 has Use="enc" and "unwrapKey" op -> matches
	// key5 has no Use/key_ops -> matches
	decryptKeys := set.Filter(CanDecrypt())
	assert.Len(decryptKeys, 3) // key3, key4, key5
}

func TestSetAdd(t *testing.T) {
	assert := tdd.New(t)

	set := &Set{}

	// Add valid key
	key, _ := New(jwa.RS256)
	key.SetID("key1")
	err := set.Add(key.Export(false), true)
	assert.Nil(err)
	assert.Equal(1, set.Len())

	// Add key without ID (should work)
	key2, _ := New(jwa.ES256)
	err = set.Add(key2.Export(false), true)
	assert.Nil(err)
	assert.Equal(2, set.Len())

	// Add duplicate ID with check
	key3, _ := New(jwa.HS256)
	key3.SetID("key1")
	err = set.Add(key3.Export(false), true)
	assert.NotNil(err)
	assert.Contains(err.Error(), "already exists")
	assert.Equal(2, set.Len())

	// Add duplicate ID without check
	err = set.Add(key3.Export(false), false)
	assert.Nil(err)
	assert.Equal(3, set.Len())

	// Add invalid key
	invalidKey := Record{KeyID: "key4"} // missing kty
	err = set.Add(invalidKey, true)
	assert.NotNil(err)
	assert.Contains(err.Error(), "invalid key")
}

func TestSetRemove(t *testing.T) {
	assert := tdd.New(t)

	set := &Set{
		Keys: []Record{
			{KeyID: "key1", KeyType: "RSA"},
			{KeyID: "key2", KeyType: "EC"},
			{KeyID: "key3", KeyType: "oct"},
		},
	}

	// Remove existing key
	removed := set.Remove("key2")
	assert.True(removed)
	assert.Equal(2, set.Len())

	// Remove non-existing key
	removed = set.Remove("nonexistent")
	assert.False(removed)
	assert.Equal(2, set.Len())

	// Verify remaining keys
	_, found := set.FindByID("key1")
	assert.True(found)
	_, found = set.FindByID("key3")
	assert.True(found)
	_, found = set.FindByID("key2")
	assert.False(found)
}

func TestSetLenAndIsEmpty(t *testing.T) {
	assert := tdd.New(t)

	// Empty set
	set := &Set{}
	assert.Equal(0, set.Len())
	assert.True(set.IsEmpty())

	// Non-empty set
	set.Keys = append(set.Keys, Record{KeyID: "key1", KeyType: "RSA"})
	assert.Equal(1, set.Len())
	assert.False(set.IsEmpty())
}

func TestSetClear(t *testing.T) {
	assert := tdd.New(t)

	set := &Set{
		Keys: []Record{
			{KeyID: "key1", KeyType: "RSA"},
			{KeyID: "key2", KeyType: "EC"},
		},
	}

	set.Clear()
	assert.Equal(0, set.Len())
	assert.True(set.IsEmpty())
}

func TestSetMerge(t *testing.T) {
	assert := tdd.New(t)

	key1, _ := New(jwa.RS256)
	key1.SetID("key1")
	key2, _ := New(jwa.ES256)
	key2.SetID("key2")
	key3, _ := New(jwa.HS256)
	key3.SetID("key1")
	key4, _ := New(jwa.RS256)
	key4.SetID("key4")

	set1 := &Set{
		Keys: []Record{
			key1.Export(true),
			key2.Export(true),
		},
	}

	set2 := &Set{
		Keys: []Record{
			key3.Export(true),
			key1.Export(true), // Duplicate ID
		},
	}

	// Merge without duplicate check
	err := set1.Merge(set2, false)
	assert.Nil(err)
	assert.Equal(4, set1.Len())

	// Merge with duplicate check
	set3 := &Set{
		Keys: []Record{
			key4.Export(true),
		},
	}
	err = set1.Merge(set3, true)
	assert.Nil(err)
	assert.Equal(5, set1.Len())

	// Merge with duplicate - should error
	set4 := &Set{
		Keys: []Record{
			{KeyID: "key1", KeyType: "EC"},
		},
	}
	err = set1.Merge(set4, true)
	assert.NotNil(err)
	assert.Contains(err.Error(), "skipped duplicate keys")
	// Note: keys with duplicate IDs are skipped, but valid keys are still added
	assert.Equal(5, set1.Len())
}

func TestSetClone(t *testing.T) {
	assert := tdd.New(t)

	original := &Set{
		Keys: []Record{
			{
				KeyID:            "key1",
				KeyType:          "RSA",
				KeyOps:           []string{"sign", "verify"},
				Alg:              "RS256",
				Use:              "sig",
				N:                "abc123",
				E:                "AQAB",
				D:                "private",
				P:                "prime1",
				Q:                "prime2",
				CertificateChain: []string{"cert1", "cert2"},
			},
		},
	}

	clone := original.Clone()

	// Verify clone has same data
	assert.Equal(original.Len(), clone.Len())
	assert.Equal(original.Keys[0].KeyID, clone.Keys[0].KeyID)
	assert.Equal(original.Keys[0].N, clone.Keys[0].N)

	// Modify clone - should not affect original
	clone.Keys[0].KeyID = "modified-id"
	clone.Keys[0].KeyOps[0] = "modified-op"
	clone.Keys[0].CertificateChain[0] = "modified-cert"
	assert.Equal("key1", original.Keys[0].KeyID)
	assert.Equal("sign", original.Keys[0].KeyOps[0])
	assert.Equal("cert1", original.Keys[0].CertificateChain[0])

	// Clone nil set
	assert.Nil((*Set)(nil).Clone())
}

func TestSetValidate(t *testing.T) {
	assert := tdd.New(t)

	key1, _ := New(jwa.RS256)
	key1.SetID("key1")
	key2, _ := New(jwa.ES256)
	key2.SetID("key2")

	// Valid set
	validSet := &Set{
		Keys: []Record{
			key1.Export(true),
			key2.Export(true),
		},
	}
	err := validSet.Validate()
	assert.Nil(err)

	// Nil set
	var nilSet *Set
	err = nilSet.Validate()
	assert.NotNil(err)
	assert.Contains(err.Error(), "nil")

	// Duplicate KeyIDs
	dupSet := &Set{
		Keys: []Record{
			key1.Export(true),
			key1.Export(true),
		},
	}
	err = dupSet.Validate()
	assert.NotNil(err)
	assert.Contains(err.Error(), "duplicate KeyID")

	// Invalid key (missing kty)
	invalidSet := &Set{
		Keys: []Record{
			{KeyID: "key1"},
		},
	}
	err = invalidSet.Validate()
	assert.NotNil(err)
	assert.Contains(err.Error(), "kty")
}

func TestParseSet(t *testing.T) {
	assert := tdd.New(t)

	// Valid JSON
	jsonData := `{"keys":[{"kty":"RSA","kid":"key1","alg":"RS256"},{"kty":"EC","kid":"key2","alg":"ES256"}]}`
	set, err := ParseSet([]byte(jsonData))
	assert.Nil(err)
	assert.Equal(2, set.Len())
	assert.Equal("key1", set.Keys[0].KeyID)

	// Invalid JSON
	_, err = ParseSet([]byte("invalid json"))
	assert.NotNil(err)
	assert.Contains(err.Error(), "failed to parse")

	// Valid JSON but invalid set (duplicate IDs)
	invalidJSON := `{"keys":[{"kty":"RSA","kid":"key1"},{"kty":"EC","kid":"key1"}]}`
	_, err = ParseSet([]byte(invalidJSON))
	assert.NotNil(err)
	assert.Contains(err.Error(), "duplicate")
}

func TestRecordValidate(t *testing.T) {
	assert := tdd.New(t)

	// Valid record
	k1, _ := New(jwa.RS256)
	valid := k1.Export(false)
	err := valid.Validate()
	assert.Nil(err)

	// Missing kty
	noKty := Record{KeyID: "key1"}
	err = noKty.Validate()
	assert.NotNil(err)
	assert.Contains(err.Error(), "kty")

	// Invalid key type
	badKty := Record{KeyType: "INVALID", Alg: "RS256"}
	err = badKty.Validate()
	assert.NotNil(err)
	assert.Contains(err.Error(), "key type")

	// Invalid use
	badUse := Record{KeyType: "RSA", Use: "invalid"}
	err = badUse.Validate()
	assert.NotNil(err)
	assert.Contains(err.Error(), "use")

	// Invalid key_ops value
	badKeyOps := Record{KeyType: "RSA", KeyOps: []string{"invalid"}}
	err = badKeyOps.Validate()
	assert.NotNil(err)
	assert.Contains(err.Error(), "key_ops")

	// Duplicate key_ops
	dupKeyOps := Record{KeyType: "RSA", KeyOps: []string{"sign", "sign"}}
	err = dupKeyOps.Validate()
	assert.NotNil(err)
	assert.Contains(err.Error(), "duplicate")

	// Inconsistent use and key_ops - sig with encrypt op
	inconsistent := Record{KeyType: "RSA", Use: "sig", KeyOps: []string{"encrypt"}}
	err = inconsistent.Validate()
	assert.NotNil(err)
	assert.Contains(err.Error(), "inconsistent")

	// Inconsistent use and key_ops - enc with sign op
	inconsistent2 := Record{KeyType: "RSA", Use: "enc", KeyOps: []string{"sign"}}
	err = inconsistent2.Validate()
	assert.NotNil(err)
	assert.Contains(err.Error(), "inconsistent")

	// Algorithm mismatch
	wrongAlg := Record{KeyType: "RSA", Alg: "ES256"}
	err = wrongAlg.Validate()
	assert.NotNil(err)
	assert.Contains(err.Error(), "requires key type")

	// Valid algorithm for key type
	ec, _ := New(jwa.ES256)
	validEC := ec.Export(false)
	err = validEC.Validate()
	assert.Nil(err)

	// Valid oct key with HS algorithm
	oct, _ := New(jwa.HS256)
	validOct := oct.Export(true)
	err = validOct.Validate()
	assert.Nil(err)
}

func TestRecordClone(t *testing.T) {
	assert := tdd.New(t)

	original := Record{
		KeyID:            "key1",
		KeyType:          "RSA",
		KeyOps:           []string{"sign", "verify"},
		Alg:              "RS256",
		Use:              "sig",
		N:                "modulus",
		E:                "exponent",
		D:                "private",
		P:                "prime1",
		Q:                "prime2",
		DP:               "dp",
		DQ:               "dq",
		Qi:               "qi",
		K:                "symmetric",
		CertificateChain: []string{"cert1", "cert2"},
	}

	clone := original.Clone()

	// Verify all fields copied
	assert.Equal(original.KeyID, clone.KeyID)
	assert.Equal(original.KeyType, clone.KeyType)
	assert.Equal(original.Alg, clone.Alg)
	assert.Equal(original.Use, clone.Use)
	assert.Equal(original.N, clone.N)
	assert.Equal(original.D, clone.D)

	// Modify clone slices - should not affect original
	clone.KeyOps[0] = "modified"
	clone.CertificateChain[0] = "modified"
	assert.Equal("sign", original.KeyOps[0])
	assert.Equal("cert1", original.CertificateChain[0])
}

func TestRecordHasPrivateKey(t *testing.T) {
	assert := tdd.New(t)

	// RSA with private key
	rsaPrivate := Record{KeyType: "RSA", D: "private"}
	assert.True(rsaPrivate.HasPrivateKey())

	// RSA without private key
	rsaPublic := Record{KeyType: "RSA", N: "modulus", E: "AQAB"}
	assert.False(rsaPublic.HasPrivateKey())

	// EC with private key
	ecPrivate := Record{KeyType: "EC", D: "private"}
	assert.True(ecPrivate.HasPrivateKey())

	// EC without private key
	ecPublic := Record{KeyType: "EC", X: "x", Y: "y"}
	assert.False(ecPublic.HasPrivateKey())

	// Symmetric key
	octKey := Record{KeyType: "oct", K: "secret"}
	assert.True(octKey.HasPrivateKey())

	// Empty oct key
	octEmpty := Record{KeyType: "oct"}
	assert.False(octEmpty.HasPrivateKey())

	// Unknown key type
	unknown := Record{KeyType: "UNKNOWN"}
	assert.False(unknown.HasPrivateKey())
}

func TestSetGetKey(t *testing.T) {
	assert := tdd.New(t)

	set := &Set{
		Keys: []Record{
			{KeyID: "key1", KeyType: "RSA"},
			{KeyID: "key2", KeyType: "EC"},
		},
	}

	// Valid index
	key, ok := set.GetKey(0)
	assert.True(ok)
	assert.Equal("key1", key.KeyID)

	key, ok = set.GetKey(1)
	assert.True(ok)
	assert.Equal("key2", key.KeyID)

	// Negative index
	_, ok = set.GetKey(-1)
	assert.False(ok)

	// Out of bounds
	_, ok = set.GetKey(2)
	assert.False(ok)
}

func TestSetKeyIDs(t *testing.T) {
	assert := tdd.New(t)

	set := &Set{
		Keys: []Record{
			{KeyID: "key1", KeyType: "RSA"},
			{KeyID: "", KeyType: "EC"}, // No ID
			{KeyID: "key3", KeyType: "oct"},
		},
	}

	ids := set.KeyIDs()
	assert.Len(ids, 2)
	assert.Contains(ids, "key1")
	assert.Contains(ids, "key3")
	assert.NotContains(ids, "")
}

func TestSetFirst(t *testing.T) {
	assert := tdd.New(t)

	// Non-empty set
	set := &Set{
		Keys: []Record{
			{KeyID: "key1", KeyType: "RSA"},
			{KeyID: "key2", KeyType: "EC"},
		},
	}
	key, ok := set.First()
	assert.True(ok)
	assert.Equal("key1", key.KeyID)

	// Empty set
	emptySet := &Set{}
	_, ok = emptySet.First()
	assert.False(ok)
}

func TestSetSelectByOperation(t *testing.T) {
	assert := tdd.New(t)

	set := &Set{
		Keys: []Record{
			{KeyID: "key1", KeyType: "RSA", Use: "sig", KeyOps: []string{"sign", "verify"}},
			{KeyID: "key2", KeyType: "EC", Use: "sig", KeyOps: []string{"verify"}},
			{KeyID: "key3", KeyType: "oct", Use: "enc", KeyOps: []string{"encrypt", "decrypt"}},
			{KeyID: "key4", KeyType: "RSA", Use: "sig"}, // No key_ops - inferred as sig capable
			{KeyID: "key5", KeyType: "EC"},              // No use or key_ops - inferred as capable of any
		},
	}

	// Sign operation - keys with explicit sign op, or no key_ops (inferred)
	signKeys := set.SelectByOperation("sign")
	assert.Len(signKeys, 3) // key1 (explicit), key4 (inferred from use="sig"), key5 (inferred from empty)
	assert.Equal("key1", signKeys[0].KeyID)

	// Verify operation - keys with explicit verify op, or no key_ops with sig use/empty
	verifyKeys := set.SelectByOperation("verify")
	assert.Len(verifyKeys, 4) // key1, key2, key4, key5

	// Encrypt operation - keys with explicit encrypt op, or no key_ops with enc use/empty
	encryptKeys := set.SelectByOperation("encrypt")
	assert.Len(encryptKeys, 2) // key3 (explicit), key5 (inferred from empty)
	assert.Equal("key3", encryptKeys[0].KeyID)
}

func TestMatchKid(t *testing.T) {
	assert := tdd.New(t)

	set := &Set{
		Keys: []Record{
			{KeyID: "key1", KeyType: "RSA"},
			{KeyID: "key2", KeyType: "EC"},
		},
	}

	key, found := set.Find(MatchKid("key2"))
	assert.True(found)
	assert.Equal("key2", key.KeyID)
}

func TestSetJSONRoundTrip(t *testing.T) {
	assert := tdd.New(t)

	original := &Set{
		Keys: []Record{
			{KeyID: "key1", KeyType: "RSA", Alg: "RS256", Use: "sig", KeyOps: []string{"sign", "verify"}},
			{KeyID: "key2", KeyType: "EC", Alg: "ES256", Use: "sig", KeyOps: []string{"verify"}},
		},
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(original)
	assert.Nil(err)

	// Unmarshal back
	parsed, err := ParseSet(jsonData)
	assert.Nil(err)
	assert.Equal(original.Len(), parsed.Len())

	// Verify keys match
	for i := range original.Keys {
		assert.Equal(original.Keys[i].KeyID, parsed.Keys[i].KeyID)
		assert.Equal(original.Keys[i].KeyType, parsed.Keys[i].KeyType)
		assert.Equal(original.Keys[i].Alg, parsed.Keys[i].Alg)
	}
}

func TestEdgeCases(t *testing.T) {
	assert := tdd.New(t)

	// Empty record validation
	empty := Record{}
	err := empty.Validate()
	assert.NotNil(err)
	assert.Contains(err.Error(), "kty")

	// Valid use values
	sigUse := Record{KeyType: "RSA", Use: "sig"}
	assert.Nil(sigUse.Validate())

	encUse := Record{KeyType: "RSA", Use: "enc"}
	assert.Nil(encUse.Validate())

	// Valid key_ops combinations
	sigOps := Record{KeyType: "RSA", KeyOps: []string{"sign", "verify"}}
	assert.Nil(sigOps.Validate())

	encOps := Record{KeyType: "RSA", Use: "enc", KeyOps: []string{"encrypt", "decrypt", "wrapKey", "unwrapKey"}}
	assert.Nil(encOps.Validate())

	// Consistent use and key_ops
	consistent := Record{KeyType: "RSA", Use: "sig", KeyOps: []string{"sign", "verify"}}
	assert.Nil(consistent.Validate())

	// Algorithm with only 1 character (edge case)
	shortAlg := Record{KeyType: "RSA", Alg: "X"}
	err = shortAlg.Validate()
	assert.NotNil(err)
	assert.Contains(err.Error(), "invalid algorithm")
}
