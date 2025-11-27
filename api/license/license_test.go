package license

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	lic := New()
	assert.Equal(t, 36, len(lic.ID))
	assert.Equal(t, "free", lic.Plan)
	assert.Equal(t, "", lic.Customer)
	assert.Equal(t, "", lic.Signature)
	assert.Equal(t, "1", lic.Version)
}

func TestNewFree(t *testing.T) {
	lic := NewFree()
	assert.Equal(t, "00000000-0000-0000-0000-000000000000", lic.ID)
	assert.Equal(t, "free", lic.Plan)
	assert.Equal(t, "anonymous", lic.Customer)
	assert.Equal(t, "1996-08-24T00:00:00Z", lic.CreatedAt.Format(time.RFC3339))
	assert.Equal(t, "2099-12-31T23:59:59Z", lic.ExpiredAt.Format(time.RFC3339))
	assert.Equal(t, "", lic.Signature)
	assert.Equal(t, "1", lic.Version)
	assert.Equal(t,
		`{"id":"00000000-0000-0000-0000-000000000000","plan":"free","customer":"anonymous","expired_at":"2099-12-31T23:59:59Z","created_at":"1996-08-24T00:00:00Z","version":"1","signature":""}`,
		lic.String())
}

func TestParseLicense(t *testing.T) {
	valid := `{"id":"id","plan":"free","customer":"test","expired_at":"2099-12-31T00:00:00Z","created_at":"1996-08-24T00:00:00Z","version":"1","signature":""}`
	lic, err := ParseLicense(valid)
	assert.NoError(t, err)
	assert.Equal(t, "id", lic.ID)
	assert.Equal(t, "free", lic.Plan)
	assert.Equal(t, "1", lic.Version)
	assert.Equal(t, "test", lic.Customer)
	assert.Equal(t, "", lic.Signature)
	assert.Equal(t, "1996-08-24T00:00:00Z", lic.CreatedAt.Format(time.RFC3339))
	assert.Equal(t, "2099-12-31T00:00:00Z", lic.ExpiredAt.Format(time.RFC3339))

	malformed := ""
	lic, err = ParseLicense(malformed)
	assert.Nil(t, lic)
	assert.EqualError(t, err, "failed to parse license: unexpected end of JSON input")
}

func TestSign(t *testing.T) {
	PublicKey = "3eed19da2c0c83e467c3fe11d758ee3678e15e4b4f6caba2d368b6aa9245e09d"
	PrivateKey = "d44d89fab9acb2a530656c482196afc2c0a757b1029dc41a383149c1940701603eed19da2c0c83e467c3fe11d758ee3678e15e4b4f6caba2d368b6aa9245e09d"
	lic := NewFree()
	err := lic.Sign()
	assert.NoError(t, err)

	err = lic.Validate()
	assert.NoError(t, err)
}

func TestValidate(t *testing.T) {
	publicKey := `af7de7c014e75770cd22290e44929bd9053011a5065502748ffdc87ce69b9ce2`

	tests := []struct {
		scenario  string
		publicKey string
		license   string
		error     error
	}{
		{
			scenario:  "valid",
			publicKey: publicKey,
			license:   `{"created_at":"1996-08-24T00:00:00Z","customer":"test","expired_at":"2099-12-31T00:00:00Z","id":"id","plan":"free","signature":"fe961db0c6c18a0e46ce4972b5361fa2ed615a6a4cbb90aa48b204d800f8801a67ec035a73123ddf1d540af9483f93a0a5ce6cbf041ebaedf88ffdd1ba905404","version":"1"}`,
			error:     nil,
		},
		{
			scenario:  "invalid signature",
			publicKey: publicKey,
			license:   `{"created_at":"1996-08-24T00:00:00Z","customer":"test","expired_at":"2099-12-31T00:00:00Z","id":"id","plan":"free","signature":"","version":"1"}`,
			error:     errors.New("signature is invalid"),
		},
		{
			scenario:  "invalid signature",
			publicKey: publicKey,
			license:   `{"created_at":"1996-08-24T00:00:00Z","customer":"test","expired_at":"2099-12-31T00:00:00Z","id":"id","plan":"free","signature":"a","version":"1"}`,
			error:     errors.New("signature is invalid: encoding/hex: odd length hex string"),
		},
		{
			scenario:  "expired",
			publicKey: publicKey,
			license:   `{"created_at":"1996-08-24T00:00:00Z","customer":"test","expired_at":"2025-01-01T00:00:00Z","id":"id","plan":"free","signature":"a6664418f7774f1099d54b897e2e2b3eac696de7c927d24097b295eb0f43c94bb80a0e9e34c2d0d4b0c3d429ac5fb6645b2f0d10f6334f06a6a60e0c50e7df00","version":"1"}`,
			error:     errors.New("license is expired"),
		},
		{
			scenario:  "missing public key",
			publicKey: "",
			license:   `{"created_at":"1996-08-24T00:00:00Z","customer":"test","expired_at":"2025-01-01T00:00:00Z","id":"id","plan":"free","signature":"a6664418f7774f1099d54b897e2e2b3eac696de7c927d24097b295eb0f43c94bb80a0e9e34c2d0d4b0c3d429ac5fb6645b2f0d10f6334f06a6a60e0c50e7df00","version":"1"}`,
			error:     errors.New("public key is missing"),
		},
		{
			scenario:  "invalid public key",
			publicKey: "test",
			license:   `{"created_at":"1996-08-24T00:00:00Z","customer":"test","expired_at":"2025-01-01T00:00:00Z","id":"id","plan":"free","signature":"a6664418f7774f1099d54b897e2e2b3eac696de7c927d24097b295eb0f43c94bb80a0e9e34c2d0d4b0c3d429ac5fb6645b2f0d10f6334f06a6a60e0c50e7df00","version":"1"}`,
			error:     errors.New("public key is invalid: encoding/hex: invalid byte: U+0074 't'"),
		},
	}

	for _, test := range tests {
		lic, err := ParseLicense(test.license)
		assert.NoError(t, err)

		PublicKey = test.publicKey

		validateErr := lic.Validate()
		if test.error == nil {
			assert.Nil(t, validateErr)
		} else {
			assert.EqualError(t, validateErr, test.error.Error())
		}
	}
}
