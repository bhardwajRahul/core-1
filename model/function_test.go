package model

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/staticbackendhq/core/config"
)

func TestFunctionSecretsEncryptDecrypt(t *testing.T) {
	config.Current.AppSecret = "12345678901234567890123456789012"

	encrypted, err := EncryptFunctionSecrets("apiKey=first+value&dash-key=dash+value")
	if err != nil {
		t.Fatal(err)
	}
	if len(encrypted) == 0 {
		t.Fatal("expected encrypted secrets")
	}

	got, err := ExecData{Secrets: encrypted}.GetSecrets()
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]string{
		"apiKey":   "first value",
		"dash-key": "dash value",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v got %v", want, got)
	}
}

func TestFunctionSecretsEmptyInput(t *testing.T) {
	config.Current.AppSecret = "12345678901234567890123456789012"

	encrypted, err := EncryptFunctionSecrets("")
	if err != nil {
		t.Fatal(err)
	}
	if len(encrypted) != 0 {
		t.Fatal("expected empty encrypted secrets")
	}

	got, err := ExecData{}.GetSecrets()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no secrets got %v", got)
	}
}

func TestFunctionSecretsInvalidCiphertext(t *testing.T) {
	config.Current.AppSecret = "12345678901234567890123456789012"

	if _, err := (ExecData{Secrets: []byte("bad")}).GetSecrets(); err == nil {
		t.Fatal("expected invalid ciphertext error")
	}
}

func TestFunctionSecretsRequireValidAppSecret(t *testing.T) {
	tests := []struct {
		name      string
		appSecret string
		wantBytes string
	}{
		{
			name:      "missing",
			appSecret: "",
			wantBytes: "got 0 bytes",
		},
		{
			name:      "invalid length",
			appSecret: "too-short",
			wantBytes: "got 9 bytes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.Current.AppSecret = tt.appSecret

			_, err := EncryptFunctionSecrets("apiKey=first")
			if !errors.Is(err, ErrInvalidFunctionSecretKey) {
				t.Fatalf("expected invalid APP_SECRET error got %v", err)
			}
			if !strings.Contains(err.Error(), tt.wantBytes) {
				t.Fatalf("expected key length in error got %v", err)
			}

			_, err = (ExecData{Secrets: []byte("encrypted")}).GetSecrets()
			if !errors.Is(err, ErrInvalidFunctionSecretKey) {
				t.Fatalf("expected invalid APP_SECRET error got %v", err)
			}
		})
	}
}
