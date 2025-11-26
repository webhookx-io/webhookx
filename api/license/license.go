package license

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	uuid "github.com/satori/go.uuid"
)

var (
	// PublicKey is the public key used for verifying license
	PublicKey = ""
	// PrivateKey is the private key used for signing license
	PrivateKey = ""
)

var (
	ErrSignatureInvalid = errors.New("signature is invalid")
	ErrPublicKeyMissing = errors.New("public key is missing")
	ErrPublicKeyInvalid = errors.New("public key is invalid")
	ErrLicenseExpired   = errors.New("license is expired")
)

type License struct {
	license   string
	ID        string    `json:"id"`
	Plan      string    `json:"plan"`
	Customer  string    `json:"customer"`
	ExpiredAt time.Time `json:"expired_at"`
	CreatedAt time.Time `json:"created_at"`
	Version   string    `json:"version"`
	Signature string    `json:"signature"`
}

// New returns a default license
func New() *License {
	t := time.Now().UTC().Truncate(time.Second)
	license := License{
		ID:        uuid.NewV4().String(),
		Plan:      "free",
		ExpiredAt: t.AddDate(1, 0, 0),
		CreatedAt: t,
		Version:   "1",
	}
	return &license
}

// NewFree returns a free license
func NewFree() *License {
	license := License{
		ID:        "00000000-0000-0000-0000-000000000000",
		Customer:  "anonymous",
		Plan:      "free",
		ExpiredAt: time.Date(2099, 12, 31, 0, 0, 0, 0, time.UTC),
		CreatedAt: time.Date(1996, 8, 24, 0, 0, 0, 0, time.UTC),
		Version:   "1",
	}
	return &license
}

// Sign signs the license
func (l *License) Sign() error {
	props, err := ParseProperties(l.String())
	if err != nil {
		return err
	}
	err = l.SignProperties(props)
	if err != nil {
		return err
	}
	licenseJSON, err := props.JSON()
	if err != nil {
		return err
	}
	l.license = licenseJSON
	l.Signature = props.GetString("signature")
	return nil
}

func (l *License) SignProperties(props Properties) error {
	props.Delete("signature")
	message, err := props.JSON()
	if err != nil {
		return err
	}
	signer, err := NewSigner(PrivateKey)
	if err != nil {
		return err
	}
	signature := signer.Sign(message)
	props.Set("signature", signature)
	return nil
}

// Validate validates license
func (l *License) Validate() error {
	props, err := ParseProperties(l.license)
	if err != nil {
		return err
	}

	signature, err := hex.DecodeString(props.GetString("signature"))
	if err != nil {
		return fmt.Errorf("%w: %s", ErrSignatureInvalid, err.Error())
	}

	if PublicKey == "" {
		return ErrPublicKeyMissing
	}
	verifier, err := NewVerifier(PublicKey)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrPublicKeyInvalid, err.Error())
	}

	props.Delete("signature")
	message, err := props.JSON()
	if err != nil {
		return err
	}

	if ok := verifier.Verify([]byte(message), signature); !ok {
		return ErrSignatureInvalid
	}

	if l.Expired() {
		return ErrLicenseExpired
	}

	return nil
}

func (l *License) Expired() bool {
	return time.Now().After(l.ExpiredAt)
}

func (l *License) String() string {
	bytes, err := json.Marshal(l)
	if err != nil {
		panic(err)
	}
	return string(bytes)
}

func (l *License) GetLicense() string {
	return l.license
}

type Properties map[string]interface{}

func (props Properties) JSON() (string, error) {
	b, err := json.Marshal(props)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (props Properties) GetString(key string) string {
	if val, ok := props[key]; ok {
		switch v := val.(type) {
		case string:
			return v
		case fmt.Stringer:
			return v.String()
		default:
			return fmt.Sprintf("%v", v)
		}
	}
	return ""
}

func (props Properties) Set(key string, value string) {
	props[key] = value
}

func (props Properties) Delete(key string) {
	delete(props, key)
}

func ParseProperties(str string) (Properties, error) {
	var props Properties
	err := json.Unmarshal([]byte(str), &props)
	if err != nil {
		return nil, err
	}
	return props, nil
}

func ParseLicense(licenseJSON string) (*License, error) {
	license := License{
		license: licenseJSON,
	}
	if err := json.Unmarshal([]byte(licenseJSON), &license); err != nil {
		return nil, fmt.Errorf("failed to parse license: %w", err)
	}
	return &license, nil
}
