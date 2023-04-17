/*
 * Copyright 2023 github.com/fatima-go
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * @project fatima-core
 * @author jin
 * @date 23. 4. 14. 오후 5:07
 */

package cipher

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
)

// The AES block size in bytes.
const BlockSize = 16
const (
	fatimaCipherKey       = "12345678901234567890123456789012"
	fatimaCipherIv        = "1234567890123456"
	fatimaCipherBlockSize = 16
)

func Aes256Encode(plaintext string) (string, error) {
	bPlaintext := PKCS5Padding([]byte(plaintext), fatimaCipherBlockSize)
	block, err := aes.NewCipher([]byte(fatimaCipherKey))
	if err != nil {
		return "", fmt.Errorf("aes cipher creating error : %s", err.Error())
	}
	ciphertext := make([]byte, len(bPlaintext))
	mode := cipher.NewCBCEncrypter(block, []byte(fatimaCipherIv))
	mode.CryptBlocks(ciphertext, bPlaintext)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func Aes256Decode(cipherText string) (string, error) {
	cipherTextDecoded, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", fmt.Errorf("base64 decode error : %s", err.Error())
	}

	block, err := aes.NewCipher([]byte(fatimaCipherKey))
	if err != nil {
		return "", fmt.Errorf("aes cipher creating error : %s", err.Error())
	}

	mode := cipher.NewCBCDecrypter(block, []byte(fatimaCipherIv))
	plainText := make([]byte, len(cipherTextDecoded))
	mode.CryptBlocks(plainText, cipherTextDecoded)
	trimmedPlainText := trimPKCS5(plainText)
	return string(trimmedPlainText), nil
}

func PKCS5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func trimPKCS5(text []byte) []byte {
	padding := text[len(text)-1]
	return text[:len(text)-int(padding)]
}
