package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"golang.org/x/crypto/hkdf"
)

const ProtocolVersion = "v1"

// Key management parameters
const (
	// Rekey after this many messages
	MaxMessagesBeforeRekey = 10000
	// Rekey after this duration
	MaxTimeBeforeRekey = 1 * time.Hour
	// Sequence number overflow threshold (2^63)
	SeqOverflowThreshold = uint64(1 << 63)
	// Number of old keys to keep for handling out-of-order messages
	KeyWindowSize = 3
)

// KeyWindow maintains a sliding window of recent keys
type KeyWindow struct {
	keys            [][]byte
	ratchetCounters []uint64
}

func newKeyWindow() *KeyWindow {
	return &KeyWindow{
		keys:            make([][]byte, 0, KeyWindowSize),
		ratchetCounters: make([]uint64, 0, KeyWindowSize),
	}
}

func (kw *KeyWindow) add(key []byte, counter uint64) {
	// Make a copy of the key
	keyCopy := make([]byte, len(key))
	copy(keyCopy, key)

	kw.keys = append(kw.keys, keyCopy)
	kw.ratchetCounters = append(kw.ratchetCounters, counter)

	// Keep only the last KeyWindowSize keys
	if len(kw.keys) > KeyWindowSize {
		kw.keys = kw.keys[1:]
		kw.ratchetCounters = kw.ratchetCounters[1:]
	}
}

func (kw *KeyWindow) get(counter uint64) []byte {
	for i, c := range kw.ratchetCounters {
		if c == counter {
			return kw.keys[i]
		}
	}
	return nil
}

type PublicKeyMessage struct {
	PublicKey string `json:"public_key"`
}

type BobResponse struct {
	PublicKey string `json:"public_key"`
	Port      int    `json:"port"`
}

type MessageResponse struct {
	Ack           bool   `json:"ack"`
	RatchetSeq    uint64 `json:"ratchet_seq"`
	EncryptedData []byte `json:"encrypted_data"`
}

func main() {
	// Generate Alice's key pair
	aliceKeyPair, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Alice: Generated key pair")

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

	// Send Alice's public key to Bob via STDIN
	alicePublicKeyMsg := PublicKeyMessage{
		PublicKey: base64.StdEncoding.EncodeToString(aliceKeyPair.PublicKey().Bytes()),
	}
	alicePublicKeyJSON, err := json.Marshal(alicePublicKeyMsg)
	if err != nil {
		log.Fatal(err)
	}
	alicePublicKeyJSON = append(alicePublicKeyJSON, '\n')

	if _, err := bobStdin.Write(alicePublicKeyJSON); err != nil {
		log.Fatal(err)
	}
	bobStdin.Close()
	log.Printf("Alice: Sent public key to Bob: %s", alicePublicKeyMsg.PublicKey)

	// Read Bob's response (public key and port)
	decoder := json.NewDecoder(bobStdout)
	var bobResp BobResponse
	if err := decoder.Decode(&bobResp); err != nil {
		log.Fatal(err)
	}
	log.Printf("Alice: Received Bob's public key: %s", bobResp.PublicKey)
	log.Printf("Alice: Bob is listening on port: %d", bobResp.Port)

	// Import Bob's public key
	bobPublicKeyBytes, err := base64.StdEncoding.DecodeString(bobResp.PublicKey)
	if err != nil {
		log.Fatal(err)
	}
	bobPublicKey, err := ecdh.X25519().NewPublicKey(bobPublicKeyBytes)
	if err != nil {
		log.Fatal(err)
	}

	// Derive shared secret
	sharedSecret, err := aliceKeyPair.ECDH(bobPublicKey)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Alice: Derived shared secret: %s", base64.StdEncoding.EncodeToString(sharedSecret))

	// Derive separate keys for each direction
	aliceToBobKey := deriveDirectionalKey(sharedSecret, "alice-to-bob")
	bobToAliceKey := deriveDirectionalKey(sharedSecret, "bob-to-alice")
	log.Printf("Alice: Derived alice-to-bob key: %s", base64.StdEncoding.EncodeToString(aliceToBobKey))
	log.Printf("Alice: Derived bob-to-alice key: %s", base64.StdEncoding.EncodeToString(bobToAliceKey))

	// Initialize sequence counters and key windows
	var sendSeq, recvSeq uint64 = 1, 0
	var ratchetCounter uint64 = 0
	var pendingRatchet bool = false
	var messageCount uint64 = 0
	lastRekeyTime := time.Now()

	// Key windows for handling out-of-order messages
	aliceToBobWindow := newKeyWindow()
	bobToAliceWindow := newKeyWindow()

	// Store initial keys in windows
	aliceToBobWindow.add(aliceToBobKey, ratchetCounter)
	bobToAliceWindow.add(bobToAliceKey, ratchetCounter)

	// Encrypt a message for Bob
	message := "Hello Bob! This is Alice sending you a secret message."
	aad := []byte(fmt.Sprintf("%s:seq:%d:alice-to-bob:/message", ProtocolVersion, sendSeq))
	encryptedMessage, err := encryptMessage(aliceToBobKey, []byte(message), sendSeq, aad)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Alice: Encrypted message (seq=%d): %s", sendSeq, base64.StdEncoding.EncodeToString(encryptedMessage))
	pendingRatchet = true
	sendSeq++
	messageCount++

	// Send HTTP POST request to Bob
	url := fmt.Sprintf("http://127.0.0.1:%d/message", bobResp.Port)
	resp, err := http.Post(url, "application/octet-stream", bytes.NewReader(encryptedMessage))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// Read Bob's response (JSON with ack and encrypted data)
	var msgResp MessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&msgResp); err != nil {
		log.Fatal(err)
	}
	log.Printf("Alice: Received response with ack=%v, ratchet_seq=%d", msgResp.Ack, msgResp.RatchetSeq)

	// Decrypt Bob's response with replay protection and key window fallback
	aadResp := []byte(fmt.Sprintf("%s:seq:%d:bob-to-alice:/message", ProtocolVersion, recvSeq+1))
	decryptedResponse, responseSeq, err := decryptMessage(bobToAliceKey, msgResp.EncryptedData, aadResp)
	if err != nil {
		// Try previous keys in the window (handles message loss scenarios)
		log.Printf("Alice: Decryption failed with current key, trying key window...")
		decrypted := false
		for i := len(bobToAliceWindow.keys) - 1; i >= 0; i-- {
			oldKey := bobToAliceWindow.keys[i]
			decryptedResponse, responseSeq, err = decryptMessage(oldKey, msgResp.EncryptedData, aadResp)
			if err == nil {
				log.Printf("Alice: Successfully decrypted with key from window (ratchet_counter=%d)", bobToAliceWindow.ratchetCounters[i])
				decrypted = true
				break
			}
		}
		if !decrypted {
			log.Fatalf("Alice: Failed to decrypt message with current key or key window: %v", err)
		}
	}
	if responseSeq <= recvSeq {
		log.Fatalf("Alice: Replay attack detected! Received seq=%d, expected > %d", responseSeq, recvSeq)
	}
	recvSeq = responseSeq
	log.Printf("Alice: Decrypted Bob's response (seq=%d): %s", responseSeq, string(decryptedResponse))

	// Ratchet keys ONLY after receiving acknowledgment
	if msgResp.Ack && pendingRatchet {
		ratchetCounter++
		// Store old keys in window before ratcheting
		aliceToBobWindow.add(aliceToBobKey, ratchetCounter-1)
		bobToAliceWindow.add(bobToAliceKey, ratchetCounter-1)

		aliceToBobKey = ratchetKey(aliceToBobKey, ratchetCounter)
		bobToAliceKey = ratchetKey(bobToAliceKey, ratchetCounter)
		log.Printf("Alice: Ratcheted alice-to-bob key (counter=%d): %s", ratchetCounter, base64.StdEncoding.EncodeToString(aliceToBobKey))
		log.Printf("Alice: Ratcheted bob-to-alice key (counter=%d): %s", ratchetCounter, base64.StdEncoding.EncodeToString(bobToAliceKey))
		pendingRatchet = false
	} else if !msgResp.Ack {
		log.Println("Alice: Message not acknowledged, keys NOT ratcheted (synchronization preserved)")
	}

	// Check if we need to perform full rekey
	timeSinceRekey := time.Since(lastRekeyTime)
	needRekey := false
	rekeyReason := ""

	if sendSeq >= SeqOverflowThreshold || recvSeq >= SeqOverflowThreshold {
		needRekey = true
		rekeyReason = "sequence number overflow threshold reached"
	} else if messageCount >= MaxMessagesBeforeRekey {
		needRekey = true
		rekeyReason = fmt.Sprintf("message count limit reached (%d)", MaxMessagesBeforeRekey)
	} else if timeSinceRekey >= MaxTimeBeforeRekey {
		needRekey = true
		rekeyReason = fmt.Sprintf("time limit reached (%v)", MaxTimeBeforeRekey)
	}

	if needRekey {
		log.Printf("Alice: Initiating full rekey: %s", rekeyReason)
		// In production, this would trigger a new key exchange
		// For this example, we'll just log it
		log.Printf("Alice: Rekey would reset sequence numbers and derive new keys from fresh ECDH exchange")
		log.Printf("Alice: Current state - sendSeq=%d, recvSeq=%d, messageCount=%d, time=%v",
			sendSeq, recvSeq, messageCount, timeSinceRekey)
	}

	// Kill Bob's process since he runs indefinitely
	log.Println("Alice: Terminating Bob's process")
	if err := cmd.Process.Kill(); err != nil {
		log.Printf("Alice: Failed to kill Bob: %v", err)
	}
}

// encryptMessage encrypts plaintext using AES-GCM with the shared secret
// Returns seqNum + nonce + ciphertext concatenated
func encryptMessage(sharedSecret, plaintext []byte, seqNum uint64, aad []byte) ([]byte, error) {
	block, err := aes.NewCipher(sharedSecret)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	// Encrypt with AAD (Additional Authenticated Data)
	ciphertext := aesGCM.Seal(nil, nonce, plaintext, aad)

	// Prepend sequence number and nonce to ciphertext
	seqBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(seqBytes, seqNum)
	result := make([]byte, 0, 8+len(nonce)+len(ciphertext))
	result = append(result, seqBytes...)
	result = append(result, nonce...)
	result = append(result, ciphertext...)

	return result, nil
}

// decryptMessage decrypts a message encrypted with encryptMessage
// Expects seqNum + nonce + ciphertext concatenated
// Returns plaintext and sequence number
func decryptMessage(sharedSecret, data []byte, aad []byte) ([]byte, uint64, error) {
	block, err := aes.NewCipher(sharedSecret)
	if err != nil {
		return nil, 0, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, 0, err
	}

	nonceSize := aesGCM.NonceSize()
	if len(data) < 8+nonceSize {
		return nil, 0, fmt.Errorf("ciphertext too short")
	}

	// Extract sequence number
	seqNum := binary.BigEndian.Uint64(data[:8])

	// Extract nonce and ciphertext
	nonce := data[8 : 8+nonceSize]
	ciphertext := data[8+nonceSize:]

	// Decrypt with AAD verification
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, 0, err
	}

	return plaintext, seqNum, nil
}

// deriveDirectionalKey derives a direction-specific key from the shared secret
// This ensures Alice's sending key is different from Bob's sending key
func deriveDirectionalKey(sharedSecret []byte, direction string) []byte {
	salt := []byte("denobridge-directional-key")
	info := []byte(direction)

	hkdfReader := hkdf.New(sha256.New, sharedSecret, salt, info)
	directionalKey := make([]byte, 32) // AES-256 requires 32 bytes
	if _, err := io.ReadFull(hkdfReader, directionalKey); err != nil {
		log.Fatalf("Failed to derive directional key: %v", err)
	}

	return directionalKey
}

// ratchetKey derives a new key from the current key using HKDF
// This provides forward secrecy - compromise of current key doesn't expose past messages
func ratchetKey(currentKey []byte, counter uint64) []byte {
	salt := []byte("denobridge-key-ratchet")
	info := []byte(fmt.Sprintf("ratchet:%d", counter))

	hkdfReader := hkdf.New(sha256.New, currentKey, salt, info)
	newKey := make([]byte, 32) // AES-256 requires 32 bytes
	if _, err := io.ReadFull(hkdfReader, newKey); err != nil {
		log.Fatalf("Failed to ratchet key: %v", err)
	}

	return newKey
}
