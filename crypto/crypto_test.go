package crypto

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/btcsuite/btcd/btcec"
	"github.com/nacamp/go-simplechain/common"
	"golang.org/x/crypto/sha3"

	"github.com/stretchr/testify/assert"
)

func TestHash(t *testing.T) {
	assert.Equal(t, hex.EncodeToString(Sha3b256([]byte("dummy test"))), "6151d993d53d37941297e3f3e31a26a7cdc1d5fb3efc4a5a25132cdd38e05b15", "test sha3-256")
}

func TestMakeAddress(t *testing.T) {
	priv, address := CreateAddress()
	/*
		address: c6d40a9bf9fe9d90019511a2147dc0958657da97463ca59d2594d5536dcdfd30ed93707d
		priv:    0x8a21cd44e684dd2d8d9205b0bfb69339435c7bd016ebc21fddaddffd0d47ed63

		address: d182458d4f299f73f496b7025912b0688653dbef74bc98638cd73e7e9ca01f8e9d416e44
		priv:  	 0xd7573bb27684e1911b5e8bfb3a553f860ce873562e64016fec0974a6163a5cff

		address: fdf75c884f7f1d1537177a3a35e783236739a426ee649fa3e2d8aed598b4f29e838170e2
		priv:    0x47661aa6cccada84454842404ec0cca83760254191232f1d4cc11653d397ac2e
	*/
	fmt.Println("address: ", common.AddressToHex(address))
	fmt.Println("priv: ", common.ToHex((*btcec.PrivateKey)(priv).Serialize()))
	assert.Equal(t, CreateAddressFromPrivateKey(ByteToPrivateKey(common.FromHex("0xd7573bb27684e1911b5e8bfb3a553f860ce873562e64016fec0974a6163a5cff"))),
		common.HexToAddress("0xd182458d4f299f73f496b7025912b0688653dbef74bc98638cd73e7e9ca01f8e9d416e44"), "")
}

func TestCreateAndEcrecover(t *testing.T) {
	priv, address := CreateAddress()
	fmt.Println(common.AddressToHex(address))
	assert.Equal(t, CreateAddressFromPrivateKey(priv), address, "")

	hash := make([]byte, 32)
	hasher := sha3.New256()
	k := []byte("data...")
	hasher.Write(k)
	hash = hasher.Sum(k[:0])
	// fmt.Printf("\n%#v\n", hash)

	signed, _ := Sign(hash, priv)
	pub, err := Ecrecover(hash, signed)
	assert.NoError(t, err)
	assert.Equal(t, address, CreateAddressFromPublicKeyByte(pub))

	hasher.Write([]byte("."))
	hash = hasher.Sum(nil)
	pub, _ = Ecrecover(hash, signed)
	assert.NotEqual(t, address, CreateAddressFromPublicKeyByte(pub))
}

func TestValidateAddress(t *testing.T) {
	_, address := CreateAddress()
	assert.True(t, ValidateAddress(address))

	address[0] += 0x01
	assert.False(t, ValidateAddress(address))
}

func TestGcm(t *testing.T) {
	key := []byte{0x0, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf}
	plainData := []byte("This is plaintext")
	_, cipherData, _ := GcmEncrypt(plainData, key, true)
	plainData2, _ := GcmDecrypt(nil, cipherData, key)
	assert.Equal(t, plainData, plainData2)
}

func TestHashPassword(t *testing.T) {

	hash1, _ := HashPassword("This is plaintext")
	hash2, _ := HashPassword("This is plaintext")
	assert.Equal(t, hash1, hash2)
	assert.Equal(t, 32, len(hash1))
}
