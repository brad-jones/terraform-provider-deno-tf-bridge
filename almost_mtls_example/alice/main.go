package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/exec"
	"time"
)

type CertMessage struct {
	Cert string `json:"cert"`
}

type BobResponse struct {
	Port int    `json:"port"`
	Cert string `json:"cert"`
}

func main() {
	// Generate Alice's self-signed certificate
	log.Println("Alice: Generating self-signed TLS certificate...")
	aliceCert, aliceKey, aliceCertPEM, err := generateSelfSignedCert("Alice")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Alice: Generated certificate:\n%s", aliceCertPEM)

	// Start Bob as a child process
	cmd := exec.Command("deno", "run", "-qA", "bob/main.ts")
	cmd.Stderr = os.Stderr

	// Get pipes for stdin/stdout
	bobStdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}
	bobStdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	// Start Bob
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	log.Println("Alice: Started Bob")

	// Send Alice's certificate to Bob via STDIN
	aliceCertMsg := CertMessage{
		Cert: aliceCertPEM,
	}
	aliceCertJSON, err := json.Marshal(aliceCertMsg)
	if err != nil {
		log.Fatal(err)
	}
	aliceCertJSON = append(aliceCertJSON, '\n')

	if _, err := bobStdin.Write(aliceCertJSON); err != nil {
		log.Fatal(err)
	}
	bobStdin.Close()
	log.Println("Alice: Sent certificate to Bob via STDIN")

	// Read Bob's response (port and certificate)
	decoder := json.NewDecoder(bobStdout)
	var bobResp BobResponse
	if err := decoder.Decode(&bobResp); err != nil {
		log.Fatal(err)
	}
	log.Printf("Alice: Bob is listening on port: %d", bobResp.Port)
	log.Printf("Alice: Received Bob's certificate:\n%s", bobResp.Cert)

	// Parse Bob's certificate
	bobCertPool := x509.NewCertPool()
	if !bobCertPool.AppendCertsFromPEM([]byte(bobResp.Cert)) {
		log.Fatal("Alice: Failed to parse Bob's certificate")
	}

	// Create TLS config that trusts Bob's certificate
	// and presents Alice's certificate as client cert
	tlsConfig := &tls.Config{
		RootCAs: bobCertPool,
		Certificates: []tls.Certificate{
			{
				Certificate: [][]byte{aliceCert.Raw},
				PrivateKey:  aliceKey,
			},
		},
		// Don't set ServerName - let it use the IP address from the URL
		// The certificate has 127.0.0.1 in the SAN field
	}

	// Create HTTP client with TLS config
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
		Timeout: 10 * time.Second,
	}

	// Calculate SHA-256 hash of Alice's certificate
	certHash := sha256.Sum256([]byte(aliceCertPEM))
	certHashHex := hex.EncodeToString(certHash[:])
	log.Printf("Alice: Certificate hash: %s", certHashHex)

	// Send HTTPS request to Bob with Alice's cert hash in header
	url := fmt.Sprintf("https://127.0.0.1:%d/message", bobResp.Port)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		log.Fatal(err)
	}

	// Include Alice's certificate hash in header (workaround for Deno's lack of mTLS support)
	req.Header.Set("X-Client-Cert-Hash", certHashHex)
	req.Header.Set("Content-Type", "application/json")

	log.Println("Alice: Sending HTTPS request to Bob...")
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Alice: Failed to connect to Bob: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("Alice: Unexpected status %d: %s", resp.StatusCode, string(body))
	}

	// Read Bob's response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var response map[string]string
	if err := json.Unmarshal(body, &response); err != nil {
		log.Fatal(err)
	}

	log.Printf("Alice: Received Bob's message: %s", response["message"])
	log.Println("Alice: Successfully completed secure communication!")

	// Terminate Bob's process
	log.Println("Alice: Terminating Bob's process")
	if err := cmd.Process.Kill(); err != nil {
		log.Printf("Alice: Failed to kill Bob: %v", err)
	}
}

// generateSelfSignedCert creates a self-signed X.509 certificate
func generateSelfSignedCert(commonName string) (*x509.Certificate, *ecdsa.PrivateKey, string, error) {
	// Generate ECDSA private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to generate private key: %w", err)
	}

	// Create certificate template
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to generate serial number: %w", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(24 * time.Hour) // Valid for 24 hours

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: commonName,
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		DNSNames:              []string{"localhost"},
	}

	// Create self-signed certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to create certificate: %w", err)
	}

	// Parse certificate
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Encode certificate to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	return cert, privateKey, string(certPEM), nil
}
