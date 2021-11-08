package did

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Base prefix according to the specification.
// https://w3c.github.io/did-core/#identifier
const prefix = "did:"

// Identifier instance based on the DID specification.
type Identifier struct {
	data *identifierData
}

// Internal identifier state.
type identifierData struct {
	// DID Method
	// https://w3c.github.io/did-core/#method-specific-syntax
	Method string

	// The method-specific-id component of a DID
	// method-specific-id = *idchar *( ":" *idchar )
	ID string

	// method-specific-id may be composed of multiple `:` separated idstrings
	IDStrings []string

	// DID URL
	// did-url = did *( ";" param ) path-abempty [ "?" query ] [ "#" fragment ]
	// did-url may contain multiple params, a path, query, and fragment
	Params []Param

	// DID Path, the portion of a DID reference that follows the first forward
	// slash character.
	// https://w3c.github.io/did-core/#path
	Path string

	// Path may be composed of multiple `/` separated segments
	// path-abempty  = *( "/" segment )
	PathSegments []string

	// DID Query
	// https://w3c.github.io/did-core/#query
	// query = *( pchar / "/" / "?" )
	Query string

	// DID Fragment, the portion of a DID reference that follows the first hash
	// sign character ("#")
	// https://w3c.github.io/did-core/#fragment
	Fragment string

	// Indicates that there are DID controller(s) other than the DID subject.
	// https://w3c.github.io/did-core/#authorization-and-delegation
	Controller string

	// Cryptographic keys associated with the subject.
	VerificationMethods []*PublicKey

	// Enabled authentication mechanisms.
	// https://w3c.github.io/did-core/#authentication
	AuthenticationMethod []string

	// Enabled assertion mechanisms.
	// https://w3c.github.io/did-core/#assertionmethod
	AssertionMethod []string

	// Enabled key agreement mechanisms.
	// https://w3c.github.io/did-core/#keyagreement
	KeyAgreement []string

	// Enabled capability invocation mechanisms.
	// https://w3c.github.io/did-core/#capabilityinvocation
	CapabilityInvocation []string

	// Enabled capability delegation mechanisms.
	// https://w3c.github.io/did-core/#capabilitydelegation
	CapabilityDelegation []string

	// Service endpoints enabled.
	Services []*ServiceEndpoint

	// Time of original creation normalized to UTC 00:00.
	Created *time.Time

	// Time of the latest update normalized to UTC 00:00.
	Updated *time.Time
}

// NewIdentifier provides a helper factory method to generate a free-form identifier
// instance using the provided method and id string.
func NewIdentifier(method string, idString string) (*Identifier, error) {
	if strings.TrimSpace(method) == "" {
		return nil, errors.New("no method specified")
	}
	if strings.TrimSpace(idString) == "" {
		return nil, errors.New("no id string specified")
	}

	now := time.Now().UTC()
	return &Identifier{
		data: &identifierData{
			ID:      idString,
			Method:  method,
			Created: &now,
		},
	}, nil
}

// NewIdentifierWithMode provides a helper factory method to generate new random
// identifier instances using one of the modes described in the "bryk" DID Method
// specification.
func NewIdentifierWithMode(method string, tag string, mode idStringMode) (*Identifier, error) {
	// Get id string based on the selected method
	id := ""
	switch mode {
	case ModeUUID:
		id = randomUUID()
	case ModeHash:
		id = randomHash()
	}

	// Append tag to the id string if provided
	if tag != "" {
		id = fmt.Sprintf("%s:%s", tag, id)
	}

	// Return identifier
	return NewIdentifier(method, id)
}

// FromDocument restores an identifier instance from a previously generated DID Document.
func FromDocument(doc *Document) (*Identifier, error) {
	id, err := Parse(doc.Subject)
	if err != nil {
		return nil, err
	}

	// Restore public keys
	for _, k := range doc.VerificationMethod {
		rk := &PublicKey{}
		*rk = k
		id.data.VerificationMethods = append(id.data.VerificationMethods, rk)
	}

	// Restore service endpoints
	for _, s := range doc.Services {
		rs := &ServiceEndpoint{}
		*rs = s
		id.data.Services = append(id.data.Services, rs)
	}

	// Restore verification relationships
	id.data.Controller = doc.Controller
	id.data.AuthenticationMethod = append(id.data.AuthenticationMethod, doc.Authentication...)
	id.data.AssertionMethod = append(id.data.AssertionMethod, doc.AssertionMethod...)
	id.data.KeyAgreement = append(id.data.KeyAgreement, doc.KeyAgreement...)
	id.data.CapabilityInvocation = append(id.data.CapabilityInvocation, doc.CapabilityInvocation...)
	id.data.CapabilityDelegation = append(id.data.CapabilityDelegation, doc.CapabilityDelegation...)
	return id, nil
}

// IsURL returns true if a DID has a Path, a Query or a Fragment
// https://w3c.github.io/did-core/#did-url-syntax
func (d *Identifier) IsURL() bool {
	dd := d.data
	return (len(dd.Params) > 0 || dd.Path != "" || len(dd.PathSegments) > 0 || dd.Query != "" || dd.Fragment != "")
}

// GetReference returns a valid DID with the provided fragment appended.
func (d *Identifier) GetReference(fragment string) string {
	return fmt.Sprintf("%s#%s", d.DID(), fragment)
}

// Method returns the method segment of the identifier instance.
func (d *Identifier) Method() string {
	return strings.ToLower(d.data.Method)
}

// Path returns the path segment of the identifier instance.
func (d *Identifier) Path() string {
	return d.path()
}

// Fragment returns the fragment segment of the identifier instance.
func (d *Identifier) Fragment() string {
	if d.data.Fragment == "" {
		return ""
	}
	return fmt.Sprintf("#%s", d.data.Fragment)
}

// RawQuery returns the query portion of the identifier instance as a string.
func (d *Identifier) RawQuery() string {
	return d.data.Query
}

// Query returns the URL-decoded contents of the query segment of the identifier instance.
func (d *Identifier) Query() (url.Values, error) {
	if d.data.Query == "" {
		return nil, errors.New("no query values")
	}
	q, err := url.ParseQuery(d.data.Query)
	if err != nil {
		return nil, wrap(err, "failed to parse query segment")
	}
	return q, nil
}

// DID returns the DID segment of the identifier instance.
func (d *Identifier) DID() string {
	return fmt.Sprintf("%s%s:%s", prefix, d.data.Method, d.idString())
}

// Subject returns the specific ID segment of the identifier instance.
func (d *Identifier) Subject() string {
	return d.idString()
}

// Verify search for common errors in the identifier structure.
func (d *Identifier) Verify(c IDStringVerifier) error {
	// Method is required
	if d.data.Method == "" {
		return errors.New("no method specified")
	}

	// Specific ID string is required
	if d.idString() == "" {
		return errors.New("no id string specified")
	}

	// Custom verification of the specific id string
	if c != nil {
		if err := c(d.idString()); err != nil {
			return err
		}
	}

	return nil
}

// String encodes a DID instance into a valid DID string.
func (d *Identifier) String() string {
	// base identifier structure verification
	if err := d.Verify(nil); err != nil {
		return ""
	}

	var buf strings.Builder

	// write base did segment
	buf.WriteString(d.DID())

	// write params
	buf.WriteString(d.params())

	// write path
	buf.WriteString(d.path())

	if d.data.Query != "" {
		// write a leading ? and then Query
		buf.WriteByte('?')
		buf.WriteString(d.data.Query)
	}

	if d.data.Fragment != "" {
		// write a leading # and then the fragment value
		buf.WriteByte('#')
		buf.WriteString(d.data.Fragment)
	}

	return buf.String()
}

// Document returns the DID document for the identifier instance. If 'safe'
// is true, the returned document remove any private key material present,
// making the document safe to be published and shared.
func (d *Identifier) Document(safe bool) *Document {
	doc := &Document{
		Context: []interface{}{
			defaultContext,
			securityContext,
			ed25519Context,
			x25519Context,
		},
		Subject:              d.String(),
		Controller:           d.data.Controller,
		VerificationMethod:   d.VerificationMethods(),
		Services:             d.Services(),
		Authentication:       d.data.AuthenticationMethod,
		AssertionMethod:      d.data.AssertionMethod,
		KeyAgreement:         d.data.KeyAgreement,
		CapabilityInvocation: d.data.CapabilityInvocation,
		CapabilityDelegation: d.data.CapabilityDelegation,
	}

	// Remove private keys on safe representations.
	if safe {
		for i := range doc.VerificationMethod {
			doc.VerificationMethod[i].Private = nil
		}
	}
	return doc
}

// Controller returns the DID currently set as controller for the identifier
// instance.
func (d *Identifier) Controller() string {
	return d.data.Controller
}

// SetController updates the DID set as controller for the identifier instance.
func (d *Identifier) SetController(did string) error {
	if _, err := Parse(did); err != nil {
		return err
	}
	d.data.Controller = did
	return nil
}

// AddNewVerificationMethod generates and registers a new cryptographic key for
// the identifier instance.
func (d *Identifier) AddNewVerificationMethod(id string, kt KeyType) error {
	if !strings.HasPrefix(id, prefix) {
		id = d.GetReference(id)
	}
	for _, k := range d.data.VerificationMethods {
		if k.ID == id {
			return errors.New("duplicated key identifier")
		}
	}
	pk, err := newCryptoKey(kt)
	if err != nil {
		return err
	}
	pk.Controller = d.DID()
	pk.ID = id
	d.data.VerificationMethods = append(d.data.VerificationMethods, pk)
	d.update()
	return nil
}

// AddVerificationMethod attach an existing cryptographic key to the identifier.
func (d *Identifier) AddVerificationMethod(id string, private []byte, kt KeyType) error {
	if !strings.HasPrefix(id, prefix) {
		id = d.GetReference(id)
	}
	for _, k := range d.data.VerificationMethods {
		if k.ID == id {
			return errors.New("duplicated key identifier")
		}
	}
	pk, err := loadExistingKey(private, kt)
	if err != nil {
		return err
	}
	pk.Controller = d.DID()
	pk.ID = id
	d.data.VerificationMethods = append(d.data.VerificationMethods, pk)
	d.update()
	return nil
}

// RemoveVerificationMethod will permanently eliminate a registered key from the
// instance. An error will be produced if the key you're trying to remove is the
// only enabled authentication key.
func (d *Identifier) RemoveVerificationMethod(id string) error {
	if !strings.HasPrefix(id, prefix) {
		id = d.GetReference(id)
	}
	for i, k := range d.data.VerificationMethods {
		if k.ID == id {
			if len(d.data.AuthenticationMethod) == 1 && d.data.AuthenticationMethod[0] == id {
				return errors.New("can't remove only authentication key")
			}

			d.data.VerificationMethods = append(d.data.VerificationMethods[:i], d.data.VerificationMethods[i+1:]...)
			d.update()
			return nil
		}
	}
	return errors.New("invalid key identifier")
}

// VerificationMethod retrieve a key based on it's id (fragment value), "nil"
// is returned if the identifier is invalid.
func (d *Identifier) VerificationMethod(id string) *PublicKey {
	if !strings.HasPrefix(id, prefix) {
		id = d.GetReference(id)
	}
	for _, k := range d.data.VerificationMethods {
		if k.ID == id {
			return k
		}
	}
	return nil
}

// AddService set a new service endpoint for the identifier instance.
func (d *Identifier) AddService(se *ServiceEndpoint) error {
	// Set proper service identifier
	if !strings.Contains(se.ID, d.DID()) {
		se.ID = d.GetReference(se.ID)
	}

	// Verify the service is not already registered
	check := false
	for _, s := range d.data.Services {
		if s.ID == se.ID {
			check = true
			break
		}
	}
	if check {
		return errors.New("duplicated service ID")
	}
	d.data.Services = append(d.data.Services, se)
	d.update()
	return nil
}

// RemoveService will eliminate a previously registered service endpoint for the instance.
func (d *Identifier) RemoveService(name string) error {
	// Set proper service identifier
	if !strings.Contains(name, d.DID()) {
		name = d.GetReference(name)
	}

	// Verify the service is registered
	index := 0
	check := false
	for i, s := range d.data.Services {
		if s.ID == name {
			index = i
			check = true
			break
		}
	}
	if !check {
		return errors.New("service is not registered")
	}

	d.data.Services = append(d.data.Services[:index], d.data.Services[index+1:]...)
	d.update()
	return nil
}

// Service retrieve a service endpoint based on it's id, "nil" is returned if
// the identifier is invalid.
func (d *Identifier) Service(id string) *ServiceEndpoint {
	if !strings.HasPrefix(id, prefix) {
		id = d.GetReference(id)
	}
	for _, s := range d.data.Services {
		if s.ID == id {
			return s
		}
	}
	return nil
}

// GetProof generates a cryptographically verifiable proof of integrity for
// the identifier's document.
// https://w3c.github.io/did-core//#proof-optional
func (d *Identifier) GetProof(keyID, domain string) (*ProofLD, error) {
	// Retrieve key
	pk := d.VerificationMethod(keyID)
	if pk == nil {
		return nil, errors.New("invalid key identifier")
	}

	// Use normalized DID document as base input
	data, err := d.Document(true).NormalizedLD()
	if err != nil {
		return nil, wrap(err, "failed to normalize DID document")
	}

	// Generate proof instance
	return pk.ProduceProof(data, "authentication", domain)
}

// VerificationMethods returns the registered verification methods on
// the identifier instance.
func (d *Identifier) VerificationMethods() []PublicKey {
	keys := make([]PublicKey, len(d.data.VerificationMethods))
	for i, k := range d.data.VerificationMethods {
		keys[i] = *k
	}
	return keys
}

// Services returns the registered service endpoints on the identifier.
func (d *Identifier) Services() []ServiceEndpoint {
	srv := make([]ServiceEndpoint, len(d.data.Services))
	for i, s := range d.data.Services {
		srv[i] = *s
	}
	return srv
}

// Created returns the creation date for the instance, will return an error if no
// date is currently set.
func (d *Identifier) Created() (time.Time, error) {
	if d.data.Created != nil {
		return *d.data.Created, nil
	}
	return time.Now(), errors.New("no creation date set")
}

// Updated returns the date of the last update for the instance, will return an error
// if no date is currently set.
func (d *Identifier) Updated() (time.Time, error) {
	if d.data.Updated != nil {
		return *d.data.Updated, nil
	}
	return time.Now(), errors.New("no update date set")
}

// Returns the "specific-idstring" portion of the identifier instance.
func (d *Identifier) idString() string {
	if d.data.ID != "" {
		return d.data.ID
	} else if len(d.data.IDStrings) > 0 {
		return strings.Join(d.data.IDStrings, ":")
	}
	return ""
}

// Returns the "path" portion of the identifier instance.
func (d *Identifier) path() string {
	p := ""
	if d.data.Path != "" {
		p = "/" + d.data.Path
	} else if len(d.data.PathSegments) > 0 {
		p = "/" + strings.Join(d.data.PathSegments[:], "/")
	}
	return p
}

// Returns the "params" portion of the identifier instance.
func (d *Identifier) params() string {
	if len(d.data.Params) == 0 {
		return ""
	}

	// write a leading ; for each param
	var buf strings.Builder
	for _, p := range d.data.Params {
		if param := p.String(); param != "" {
			buf.WriteByte(';')
			buf.WriteString(param)
		}
	}
	return buf.String()
}

// Adjust the timestamp for last update on the identifier instance.
func (d *Identifier) update() {
	t := time.Now().UTC()
	d.data.Updated = &t
}
