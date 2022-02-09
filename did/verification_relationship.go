package did

import (
	"errors"
)

// VerificationRelationship specifies the semantics and intended usage
// of a specific verification mechanism enabled on a DID instance.
type VerificationRelationship int

const (
	// AuthenticationVM is used to express a verification relationship which
	// an entity can use to prove it is the DID subject or acting on behalf
	// of the DID Subject as a DID Controller.
	// https://w3c.github.io/did-core/#authentication
	AuthenticationVM VerificationRelationship = iota

	// AssertionVM is used to express a verification relationship which indicates
	// that a  verification method can be used to assert a statement on behalf of
	// the DID subject.
	// https://w3c.github.io/did-core/#assertionmethod
	AssertionVM

	// KeyAgreementVM is used to express a verification relationship which an
	// entity can use to engage in key agreement protocols on behalf of the DID
	// subject.
	// https://w3c.github.io/did-core/#keyagreement
	KeyAgreementVM

	// CapabilityInvocationVM is used to express a verification relationship which
	// an entity can use to invoke capabilities as the DID subject or on behalf of
	// the DID subject. A capability is an expression of an action that the DID
	// subject is authorized to take.
	// https://w3c.github.io/did-core/#capabilityinvocation
	CapabilityInvocationVM

	// CapabilityDelegationVM Used to express a verification relationship which an
	// entity can use to grant capabilities as the DID subject or on behalf of the
	// DID subject to other capability invokers.
	// https://w3c.github.io/did-core/#capabilitydelegation
	CapabilityDelegationVM
)

// AddVerificationRelationship appends reference as a valid verification mechanism
// for the DID instance. Verification methods can be used to authenticate or
// authorize interactions with the DID subject or associated parties.
// https://w3c.github.io/did-core/#verification-methods
func (d *Identifier) AddVerificationRelationship(reference string, vm VerificationRelationship) error {
	// Reference must be a valid DID
	if _, err := Parse(reference); err != nil {
		return err
	}

	// Add verification method
	switch vm {
	case AuthenticationVM:
		for _, k := range d.data.AuthenticationMethod {
			if k == reference {
				return errors.New("already registered authentication method")
			}
		}
		d.data.AuthenticationMethod = append(d.data.AuthenticationMethod, reference)
	case AssertionVM:
		for _, k := range d.data.AssertionMethod {
			if k == reference {
				return errors.New("already registered assertion method")
			}
		}
		d.data.AssertionMethod = append(d.data.AssertionMethod, reference)
	case KeyAgreementVM:
		for _, k := range d.data.KeyAgreement {
			if k == reference {
				return errors.New("already registered key agreement method")
			}
		}
		d.data.KeyAgreement = append(d.data.KeyAgreement, reference)
	case CapabilityInvocationVM:
		for _, k := range d.data.CapabilityInvocation {
			if k == reference {
				return errors.New("already registered capability invocation method")
			}
		}
		d.data.CapabilityInvocation = append(d.data.CapabilityInvocation, reference)
	case CapabilityDelegationVM:
		for _, k := range d.data.CapabilityDelegation {
			if k == reference {
				return errors.New("already registered capability delegation method")
			}
		}
		d.data.CapabilityDelegation = append(d.data.CapabilityDelegation, reference)
	}

	// Register update
	d.update()
	return nil
}

// RemoveVerificationRelationship updates the DID instance by removing a previously
// enable verification method reference.
// https://w3c.github.io/did-core/#verification-methods
func (d *Identifier) RemoveVerificationRelationship(reference string, vm VerificationRelationship) error {
	// Reference must be a valid DID
	if _, err := Parse(reference); err != nil {
		return err
	}

	// Remove verification method
	switch vm {
	case AuthenticationVM:
		for i, k := range d.data.AuthenticationMethod {
			if k == reference {
				d.data.AuthenticationMethod = append(d.data.AuthenticationMethod[:i], d.data.AuthenticationMethod[i+1:]...)
				break
			}
		}
	case AssertionVM:
		for i, k := range d.data.AssertionMethod {
			if k == reference {
				d.data.AssertionMethod = append(d.data.AssertionMethod[:i], d.data.AssertionMethod[i+1:]...)
				break
			}
		}
	case KeyAgreementVM:
		for i, k := range d.data.KeyAgreement {
			if k == reference {
				d.data.KeyAgreement = append(d.data.KeyAgreement[:i], d.data.KeyAgreement[i+1:]...)
				break
			}
		}
	case CapabilityInvocationVM:
		for i, k := range d.data.CapabilityInvocation {
			if k == reference {
				d.data.CapabilityInvocation = append(d.data.CapabilityInvocation[:i], d.data.CapabilityInvocation[i+1:]...)
				break
			}
		}
	case CapabilityDelegationVM:
		for i, k := range d.data.CapabilityDelegation {
			if k == reference {
				d.data.CapabilityDelegation = append(d.data.CapabilityDelegation[:i], d.data.CapabilityDelegation[i+1:]...)
				break
			}
		}
	}

	// Register update
	d.update()
	return nil
}

// GetVerificationRelationship return the references currently enabled as verification
// methods, of the specified type, for the identifier instance.
// https://w3c.github.io/did-core/#verification-methods
func (d *Identifier) GetVerificationRelationship(vm VerificationRelationship) []string {
	switch vm {
	case AuthenticationVM:
		return d.data.AuthenticationMethod
	case AssertionVM:
		return d.data.AssertionMethod
	case KeyAgreementVM:
		return d.data.KeyAgreement
	case CapabilityInvocationVM:
		return d.data.CapabilityInvocation
	case CapabilityDelegationVM:
		return d.data.CapabilityDelegation
	default:
		return []string{}
	}
}
