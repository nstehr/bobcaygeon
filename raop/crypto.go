package raop

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"log"
	"net"
	"strings"
)

// this stuff is based on another go airtunes server: https://github.com/joelgibson/go-airplay/blob/master/airplay/auth.go

//from Shairport: https://github.com/abrasive/shairport/
const privateKey string = ``

// these three functions are from the above git repo
func base64pad(s string) string {
	for len(s)%4 != 0 {
		s += "="
	}
	return s
}

func base64unpad(s string) string {
	if idx := strings.Index(s, "="); idx >= 0 {
		s = s[:idx]
	}
	return s
}

func aeskeyFromRsa(rsaaeskey64 string) (key []byte, err error) {
	s64 := base64pad(rsaaeskey64)
	s, err := base64.StdEncoding.DecodeString(s64)
	if err != nil {
		return
	}
	privKey, err := getPrivateKey()
	if err != nil {
		return nil, err
	}
	return rsa.DecryptOAEP(sha1.New(), nil, privKey, s, nil)
}

// the challenge response is the following
// 1. the base64 decoded data passed in as the challenge
// 2. the local connection IP address is added
// 3. the local connection's interface's mac address is added in (same that is used for the bonjour broadcast)
// 4. padding 0s are added if less than 32 bytes
// 5. the payload is signed with the private key
// 6. the signed data is base64 encoded
func generateChallengeResponse(challenge string, macAddr net.HardwareAddr, ipAddr string) (string, error) {

	log.Printf(fmt.Sprintf("building challenge for %s (ip: %s, mac: %s)", challenge, ipAddr, macAddr.String()))

	// the incoming challenge will be unpadded, need to pad to
	a := base64pad(challenge)
	decodedChallenge, err := base64.StdEncoding.DecodeString(a)
	if err != nil {
		return "", err
	}
	if len(decodedChallenge) != 16 {
		return "", fmt.Errorf("Incorrect challenge received")
	}

	b := net.ParseIP(ipAddr)
	// ParseIP will always return a 16 byte array, so if we have a
	// ipv4 address we need to get the last 4 bytes only
	if b.To4() != nil {
		b = b[len(b)-4:]
	}

	decodedChallenge = append(decodedChallenge, b...)
	decodedChallenge = append(decodedChallenge, macAddr...)

	for len(decodedChallenge) < 32 {
		decodedChallenge = append(decodedChallenge, 0)
	}

	log.Println(hex.EncodeToString(decodedChallenge))

	rsaPrivKey, err := getPrivateKey()
	if err != nil {
		return "", err
	}

	signedResponse, err := rsa.SignPKCS1v15(nil, rsaPrivKey, crypto.Hash(0), decodedChallenge)
	if err != nil {
		return "", err
	}

	signedResponse64 := base64.StdEncoding.EncodeToString(signedResponse)

	if len(signedResponse64) != len(challenge) {
		signedResponse64 = base64unpad(signedResponse64)
	}

	log.Println(fmt.Sprintf("Generated challenge response: %s", signedResponse64))
	return signedResponse64, nil
}

func getPrivateKey() (*rsa.PrivateKey, error) {
	pemBlock, _ := pem.Decode([]byte(privateKey))
	key, err := x509.ParsePKCS1PrivateKey(pemBlock.Bytes)
	if err != nil {
		return nil, err
	}
	return key, nil
}
