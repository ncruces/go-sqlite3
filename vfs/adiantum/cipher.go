package adiantum

import "lukechampine.com/adiantum/hbsh"

// HBSHCreator creates an [hbsh.HBSH] cipher,
// given key material.
type HBSHCreator interface {
	// KDF maps a secret (text) to a key of the appropriate size.
	KDF(text string) (key []byte)

	// HBSH creates an HBSH cipher given an appropriate key.
	HBSH(key []byte) *hbsh.HBSH
}
