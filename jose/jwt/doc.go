/*
Package jwt provides a JSON Web Token (JWT) implementation as specified by RFC-7519.

JSON Web Token (JWT) is a compact, URL-safe means of representing claims to be
transferred between two parties.  The claims in a JWT are encoded as a JSON
object that is used as the payload of a JSON Web Signature (JWS) structure or
as the plaintext of a JSON Web Encryption (JWE) structure, enabling the claims
to be digitally signed or integrity protected with a Message Authentication Code
(MAC) and/or encrypted. JWTs are always represented using the JWS Compact
Serialization or the JWE Compact Serialization.

In its compact form, a JSON Web Token consist of three parts separated by dots.

	Header.Payload.Signature

The signature is used to verify the message wasn't changed along the way (i.e., integrity),
and in the case of tokens signed with a private key, it can also verify that the sender
of the JWT is who it says it is (i.e., provenance).

# Claims

The second part of the token is the payload, which contains the claims. Claims are
statements about an entity (typically, the JWT holder) and additional data. There
are three types of claims: registered, public, and private.

Registered claims are a set of predefined claims which are not mandatory but recommended,
to provide a set of useful, interoperable claims.

Public claims can be defined at will by those using JWTs. But to avoid collisions they
should be defined in the IANA JSON Web Token Registry or be defined as a URI that
contains a collision resistant namespace.

Private claims are the custom claims created to share information between parties that
agree on using them and are neither registered nor public claims.

More information:
https://tools.ietf.org/html/rfc7519
*/
package jwt
