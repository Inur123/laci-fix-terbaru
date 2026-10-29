package security

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
)

const backupMagic = "LACIBKP1"
const backupChunkSize = 1 << 20

func (c *Crypto) backupAEAD() (cipher.AEAD, error) {
	mac := hmac.New(sha256.New, c.key)
	_, _ = mac.Write([]byte("laci-backup-encryption-v1"))
	block, err := aes.NewCipher(mac.Sum(nil))
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func (c *Crypto) EncryptBackup(ctx context.Context, dst io.Writer, src io.Reader) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	aead, err := c.backupAEAD()
	if err != nil {
		return err
	}
	var header [16]byte
	copy(header[:8], backupMagic)
	if _, err := rand.Read(header[8:]); err != nil {
		return err
	}
	if err := writeBackupBytes(dst, header[:]); err != nil {
		return err
	}
	plain := make([]byte, backupChunkSize+aead.Overhead())
	var counter uint32
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, readErr := io.ReadFull(src, plain[:backupChunkSize])
		if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
			return readErr
		}
		if n > 0 {
			if err := sealBackupRecord(dst, aead, header, counter, plain[:n]); err != nil {
				return err
			}
			counter++
			if counter == 0 {
				return errors.New("backup exceeds maximum record count")
			}
		}
		if readErr != nil {
			return sealBackupRecord(dst, aead, header, counter, nil)
		}
	}
}

func sealBackupRecord(dst io.Writer, aead cipher.AEAD, header [16]byte, counter uint32, plain []byte) error {
	var nonce [12]byte
	copy(nonce[:8], header[8:])
	binary.BigEndian.PutUint32(nonce[8:], counter)
	var aad [24]byte
	copy(aad[:16], header[:])
	binary.BigEndian.PutUint32(aad[16:20], counter)
	binary.BigEndian.PutUint32(aad[20:], uint32(len(plain)))
	if err := writeBackupBytes(dst, aad[20:]); err != nil {
		return err
	}
	return writeBackupBytes(dst, aead.Seal(plain[:0], nonce[:], plain, aad[:]))
}

func (c *Crypto) DecryptBackup(ctx context.Context, dst io.Writer, src io.Reader) error {
	aead, err := c.backupAEAD()
	if err != nil {
		return err
	}
	var header [16]byte
	if _, err := io.ReadFull(src, header[:]); err != nil || string(header[:8]) != backupMagic {
		return errors.New("invalid encrypted backup header")
	}
	sealedBuffer := make([]byte, backupChunkSize+aead.Overhead())
	var counter uint32
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		var length [4]byte
		if _, err := io.ReadFull(src, length[:]); err != nil {
			return errors.New("truncated encrypted backup")
		}
		n := binary.BigEndian.Uint32(length[:])
		if n > backupChunkSize {
			return errors.New("invalid encrypted backup record")
		}
		sealed := sealedBuffer[:int(n)+aead.Overhead()]
		if _, err := io.ReadFull(src, sealed); err != nil {
			return errors.New("truncated encrypted backup")
		}
		var nonce [12]byte
		copy(nonce[:8], header[8:])
		binary.BigEndian.PutUint32(nonce[8:], counter)
		var aad [24]byte
		copy(aad[:16], header[:])
		binary.BigEndian.PutUint32(aad[16:20], counter)
		copy(aad[20:], length[:])
		plain, err := aead.Open(sealed[:0], nonce[:], sealed, aad[:])
		if err != nil {
			return errors.New("encrypted backup authentication failed")
		}
		if n == 0 {
			var trailing [1]byte
			if _, err := io.ReadFull(src, trailing[:]); err != io.EOF {
				return errors.New("trailing encrypted backup data")
			}
			return nil
		}
		if err := writeBackupBytes(dst, plain); err != nil {
			return err
		}
		counter++
		if counter == 0 {
			return errors.New("backup exceeds maximum record count")
		}
	}
}

func writeBackupBytes(dst io.Writer, value []byte) error {
	n, err := dst.Write(value)
	if err != nil {
		return err
	}
	if n != len(value) {
		return io.ErrShortWrite
	}
	return nil
}
