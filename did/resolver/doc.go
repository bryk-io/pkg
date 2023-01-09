/*
Package resolver provides a "DIF Universal Resolver" handler and driver implementation.

DID resolution is the process of obtaining a DID document for a given DID. This is
one of four required operations that can be performed on any DID ("Read"; the other
ones being "Create", "Update", and "Deactivate"). The details of these operations
differ depending on the DID method. Building on top of DID resolution, DID URL
dereferencing is the process of retrieving a representation of a resource for a
given DID URL. Software and/or hardware that is able to execute these processes is
called a DID resolver.

More information:
https://w3c-ccg.github.io/did-resolution
*/
package resolver
