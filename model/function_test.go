package model

import (
	"reflect"
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
