package crypto

import (
	"encoding/base64"
	"strings"
	"sync"
	"testing"
)

// validKeyBase64 is a base64-encoded 32-byte key for testing.
var validKeyBase64 = base64.StdEncoding.EncodeToString(make([]byte, 32))

func TestNewMessageEncryptor(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		wantErr   bool
		errSubstr string
	}{
		{
			name: "valid 32-byte key",
			key:  validKeyBase64,
		},
		{
			name:      "invalid base64",
			key:       "not-valid-base64!!!",
			wantErr:   true,
			errSubstr: "decode encryption key",
		},
		{
			name:      "key too short (16 bytes)",
			key:       base64.StdEncoding.EncodeToString(make([]byte, 16)),
			wantErr:   true,
			errSubstr: "exactly 32 bytes",
		},
		{
			name:      "key too long (64 bytes)",
			key:       base64.StdEncoding.EncodeToString(make([]byte, 64)),
			wantErr:   true,
			errSubstr: "exactly 32 bytes",
		},
		{
			name:      "empty key",
			key:       "",
			wantErr:   true,
			errSubstr: "exactly 32 bytes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc, err := NewMessageEncryptor(tt.key)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewMessageEncryptor() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errSubstr != "" {
				if !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errSubstr)
				}
			}
			if !tt.wantErr && enc == nil {
				t.Error("expected non-nil encryptor")
			}
		})
	}
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	enc, err := NewMessageEncryptor(validKeyBase64)
	if err != nil {
		t.Fatalf("NewMessageEncryptor() error = %v", err)
	}

	tests := []struct {
		name      string
		plaintext string
	}{
		{"short text", "hello"},
		{"unicode", "Привет мир 🌍"},
		{"long text", strings.Repeat("a", 10000)},
		{"special characters", `<script>alert("xss")</script>`},
		{"single character", "x"},
		{"whitespace only", "   \t\n  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ciphertext, iv, encErr := enc.Encrypt(tt.plaintext)
			if encErr != nil {
				t.Fatalf("Encrypt() error = %v", encErr)
			}
			if ciphertext == "" {
				t.Fatal("ciphertext should not be empty for non-empty plaintext")
			}
			if iv == "" {
				t.Fatal("iv should not be empty for non-empty plaintext")
			}

			got, decErr := enc.Decrypt(ciphertext, iv)
			if decErr != nil {
				t.Fatalf("Decrypt() error = %v", decErr)
			}
			if got != tt.plaintext {
				t.Errorf("Decrypt() = %q, want %q", got, tt.plaintext)
			}
		})
	}
}

func TestEncrypt_EmptyPlaintext(t *testing.T) {
	enc, err := NewMessageEncryptor(validKeyBase64)
	if err != nil {
		t.Fatalf("NewMessageEncryptor() error = %v", err)
	}

	ciphertext, iv, encErr := enc.Encrypt("")
	if encErr != nil {
		t.Fatalf("Encrypt() error = %v", encErr)
	}
	if ciphertext != "" {
		t.Errorf("ciphertext = %q, want empty", ciphertext)
	}
	if iv != "" {
		t.Errorf("iv = %q, want empty", iv)
	}
}

func TestDecrypt_EmptyInputs(t *testing.T) {
	enc, err := NewMessageEncryptor(validKeyBase64)
	if err != nil {
		t.Fatalf("NewMessageEncryptor() error = %v", err)
	}

	tests := []struct {
		name       string
		ciphertext string
		iv         string
		want       string
	}{
		{"both empty", "", "", ""},
		{"empty ciphertext", "", "abc", ""},
		{"empty iv", "abc", "", "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, decErr := enc.Decrypt(tt.ciphertext, tt.iv)
			if decErr != nil {
				t.Fatalf("Decrypt() error = %v", decErr)
			}
			if got != tt.want {
				t.Errorf("Decrypt() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEncrypt_UniqueNonce(t *testing.T) {
	enc, err := NewMessageEncryptor(validKeyBase64)
	if err != nil {
		t.Fatalf("NewMessageEncryptor() error = %v", err)
	}

	const iterations = 100
	ivs := make(map[string]struct{}, iterations)

	for range iterations {
		_, iv, encErr := enc.Encrypt("same plaintext")
		if encErr != nil {
			t.Fatalf("Encrypt() error = %v", encErr)
		}
		if _, exists := ivs[iv]; exists {
			t.Fatal("duplicate IV detected — nonce must be unique per encryption")
		}
		ivs[iv] = struct{}{}
	}
}

func TestDecrypt_CorruptedCiphertext(t *testing.T) {
	enc, err := NewMessageEncryptor(validKeyBase64)
	if err != nil {
		t.Fatalf("NewMessageEncryptor() error = %v", err)
	}

	ciphertext, iv, encErr := enc.Encrypt("secret message")
	if encErr != nil {
		t.Fatalf("Encrypt() error = %v", encErr)
	}

	tests := []struct {
		name       string
		ciphertext string
		iv         string
		errSubstr  string
	}{
		{
			name:       "corrupted ciphertext",
			ciphertext: base64.StdEncoding.EncodeToString([]byte("garbage data here")),
			iv:         iv,
			errSubstr:  "decryption failed",
		},
		{
			name:       "wrong IV",
			ciphertext: ciphertext,
			iv:         base64.StdEncoding.EncodeToString(make([]byte, 12)),
			errSubstr:  "decryption failed",
		},
		{
			name:       "invalid base64 ciphertext",
			ciphertext: "not-valid-base64!!!",
			iv:         iv,
			errSubstr:  "decode ciphertext",
		},
		{
			name:       "invalid base64 IV",
			ciphertext: ciphertext,
			iv:         "not-valid-base64!!!",
			errSubstr:  "decode IV",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, decErr := enc.Decrypt(tt.ciphertext, tt.iv)
			if decErr == nil {
				t.Fatal("expected error for corrupted input")
			}
			if !strings.Contains(decErr.Error(), tt.errSubstr) {
				t.Errorf("error %q does not contain %q", decErr.Error(), tt.errSubstr)
			}
		})
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	key1 := base64.StdEncoding.EncodeToString(make([]byte, 32))

	key2Bytes := make([]byte, 32)
	key2Bytes[0] = 0xFF // different key
	key2 := base64.StdEncoding.EncodeToString(key2Bytes)

	enc1, err := NewMessageEncryptor(key1)
	if err != nil {
		t.Fatalf("NewMessageEncryptor(key1) error = %v", err)
	}
	enc2, err := NewMessageEncryptor(key2)
	if err != nil {
		t.Fatalf("NewMessageEncryptor(key2) error = %v", err)
	}

	ciphertext, iv, encErr := enc1.Encrypt("secret")
	if encErr != nil {
		t.Fatalf("Encrypt() error = %v", encErr)
	}

	_, decErr := enc2.Decrypt(ciphertext, iv)
	if decErr == nil {
		t.Fatal("expected error when decrypting with wrong key")
	}
}

func TestEncryptDecrypt_Concurrent(t *testing.T) {
	enc, err := NewMessageEncryptor(validKeyBase64)
	if err != nil {
		t.Fatalf("NewMessageEncryptor() error = %v", err)
	}

	var wg sync.WaitGroup
	const goroutines = 50

	for i := range goroutines {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			plaintext := strings.Repeat("x", id+1)
			ciphertext, iv, encErr := enc.Encrypt(plaintext)
			if encErr != nil {
				t.Errorf("goroutine %d: Encrypt() error = %v", id, encErr)
				return
			}

			got, decErr := enc.Decrypt(ciphertext, iv)
			if decErr != nil {
				t.Errorf("goroutine %d: Decrypt() error = %v", id, decErr)
				return
			}

			if got != plaintext {
				t.Errorf("goroutine %d: Decrypt() = %q, want %q", id, got, plaintext)
			}
		}(i)
	}

	wg.Wait()
}
