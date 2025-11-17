package ed25519

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"

	"github.com/btcsuite/btcutil/base58"
	"github.com/tyler-smith/go-bip32"
	"github.com/tyler-smith/go-bip39"
)

type Ed25519Box struct {
	Mnemonic  string   `json:"mnemonic"`
	Bip44Path []uint32 `json:"bip44Path"`
	Pub       string   `json:"publicKey"`
	Prv       string   `json:"privateKey"`
}

func GenerateKeyPair(mnemonic, mnemonicPassword string, bip44DerivePath []uint32) Ed25519Box {
	if mnemonic == "" {
		entropy, _ := bip39.NewEntropy(256)
		mnemonic, _ = bip39.NewMnemonic(entropy)
	}

	seed := bip39.NewSeed(mnemonic, mnemonicPassword)
	masterPrivateKey, _ := bip32.NewMasterKey(seed)

	if len(bip44DerivePath) == 0 {
		bip44DerivePath = []uint32{44, 7337, 0, 0}
	}

	var childKey *bip32.Key = masterPrivateKey
	for _, pathPart := range bip44DerivePath {
		childKey, _ = childKey.NewChildKey(bip32.FirstHardenedChild + pathPart)
	}

	publicKeyObject, privateKeyObject := generateKeyPairFromSeed(childKey.Key)

	pubKeyBytes, _ := x509.MarshalPKIXPublicKey(publicKeyObject)
	privKeyBytes, _ := x509.MarshalPKCS8PrivateKey(privateKeyObject)

	return Ed25519Box{Mnemonic: mnemonic, Bip44Path: bip44DerivePath, Pub: base58.Encode(pubKeyBytes[12:]), Prv: base64.StdEncoding.EncodeToString(privKeyBytes)}
}

func GenerateSignature(privateKeyAsBase64, msg string) string {
	privateKeyAsBytes, _ := base64.StdEncoding.DecodeString(privateKeyAsBase64)

	privKeyInterface, _ := x509.ParsePKCS8PrivateKey(privateKeyAsBytes)
	finalPrivateKey := privKeyInterface.(ed25519.PrivateKey)

	msgAsBytes := []byte(msg)
	signature, _ := finalPrivateKey.Sign(rand.Reader, msgAsBytes, crypto.Hash(0))

	return base64.StdEncoding.EncodeToString(signature)
}

func VerifySignature(stringMessage, base58PubKey, base64Signature string) bool {
	msgAsBytes := []byte(stringMessage)
	publicKeyAsBytesWithNoAsnPrefix := base58.Decode(base58PubKey)

	pubKeyAsBytesWithAsnPrefix := append([]byte{0x30, 0x2a, 0x30, 0x05, 0x06, 0x03, 0x2b, 0x65, 0x70, 0x03, 0x21, 0x00}, publicKeyAsBytesWithNoAsnPrefix...)
	pubKeyInterface, _ := x509.ParsePKIXPublicKey(pubKeyAsBytesWithAsnPrefix)
	finalPubKey := pubKeyInterface.(ed25519.PublicKey)

	signature, _ := base64.StdEncoding.DecodeString(base64Signature)

	return ed25519.Verify(finalPubKey, msgAsBytes, signature)
}

func generateKeyPairFromSeed(seed []byte) (ed25519.PublicKey, ed25519.PrivateKey) {
	privateKey := ed25519.NewKeyFromSeed(seed)
	pubKey, _ := privateKey.Public().(ed25519.PublicKey)
	return pubKey, privateKey
}
