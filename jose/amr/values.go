package amr

// Authentication Method Reference values as described by the RFC-8176
// specification. Used on the 'amr' registered claim.
// https://tools.ietf.org/html/rfc8176
const (
	// FACE = Biometric authentication using facial recognition.
	FACE = "face"

	// FPT = Biometric authentication using a fingerprint.
	FPT = "fpt"

	// IRIS = Biometric authentication using an iris scan.
	IRIS = "iris"

	// RETINA = Biometric authentication using a retina scan.
	RETINA = "retina"

	// VBM = Biometric authentication using a voice-print.
	VBM = "vbm"

	// GEO = Use of geolocation information for authentication.
	GEO = "geo"

	// HWK = Proof-of-Possession (PoP) of a hardware-secured key.
	HWK = "hwk"

	// SWK = Proof-of-Possession (PoP) of a software-secured key.
	SWK = "swk"

	// KBA = Knowledge-based authentication, as specified by NIST-800-63-2.
	// https://nvlpubs.nist.gov/nistpubs/SpecialPublications/NIST.SP.800-63-2.pdf
	KBA = "kba"

	// MCA = Multiple-channel authentication. The authentication involves
	// communication over more than one distinct communication channel.
	// For instance, a multiple-channel authentication might involve both
	// entering information into a workstation's browser and providing
	// information on a telephone call to a pre-registered number.
	MCA = "mca"

	// MFA = Multiple-factor authentication. When this is present, specific
	// authentication methods used may also be included.
	MFA = "mfa"

	// OTP = One-time password. One-time password specifications that this
	// authentication method applies to include Hash-Based and Time-Based.
	OTP = "otp"

	// PIN = Personal Identification Number (PIN) or pattern (not restricted
	// to containing only numbers) that a user enters to unlock a key on the
	// device. This mechanism should have a way to deter an attacker from
	// obtaining the PIN by trying repeated guesses.
	PIN = "pin"

	// PWD = Password-based authentication.
	PWD = "pwd"

	// RBA = Risk-based authentication.
	RBA = "rba"

	// SMS = Confirmation using SMS message to the user at a registered number.
	SMS = "sms"

	// TEL = Confirmation by telephone call to the user at a registered number.
	// This authentication technique is sometimes also referred to as "call
	// back".
	TEL = "tel"

	// USER = User presence test. Evidence that the end user is present and
	// interacting with the device. This is sometimes also referred to as
	// "test of user presence".
	USER = "user"

	// WIA = Windows integrated authentication.
	WIA = "wia"

	// SC = Smart card.
	SC = "sc"
)
