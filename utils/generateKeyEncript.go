package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Venqis-NolaTech/campaing-app-core-go/pkg/config"
	"golang.org/x/crypto/scrypt"
)

var masterKey = config.GetString("chat.key")
var masterIv = config.GetString("chat.iv")

func GenerateKeyEncript() (string, error) {
	password := "some password"
	ivBuffer := make([]byte, 16)
	_, err := rand.Read(ivBuffer)
	if err != nil {
		return "", err
	}
	salt := make([]byte, 32)
	_, err = rand.Read(salt)
	if err != nil {
		return "", err
	}
	keyBuffer, err := scrypt.Key([]byte(password), salt, 16384, 8, 1, 32)
	if err != nil {
		return "", err
	}
	key := hex.EncodeToString(keyBuffer)
	iv := hex.EncodeToString(ivBuffer)
	encriptionData, err := makePublicEncryptUtil(map[string]string{
		"key": key,
		"iv":  iv,
	})
	if err != nil {
		return "", err
	}
	encriptionDataBase64, err := toBase64(encriptionData)
	if err != nil {
		return "", err
	}
	return encriptionDataBase64, nil
}

func GenerateRandomKeyAndIV() (string, string, error) {
	// Generar clave de 32 bytes
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return "", "", err
	}

	// Generar IV de 16 bytes
	iv := make([]byte, 16)
	_, err = rand.Read(iv)
	if err != nil {
		return "", "", err
	}

	// Convertir a hexadecimal
	keyHex := hex.EncodeToString(key)
	ivHex := hex.EncodeToString(iv)

	return keyHex, ivHex, nil
}

func makePublicEncryptUtil(data any) (string, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	bufferData := []byte(jsonData)

	// Convertir la clave hexadecimal a bytes
	keyBytes, err := hex.DecodeString(masterKey)
	if err != nil {
		fmt.Printf("Error decodificando key: %v\n", err)
		return "", err
	}

	// Convertir el IV hexadecimal a bytes
	ivBytes, err := hex.DecodeString(masterIv)
	if err != nil {
		fmt.Printf("Error decodificando IV: %v\n", err)
		return "", err
	}

	// Hacer padding del input para que sea múltiplo de 16 bytes
	paddedData := pkcs7Padding(bufferData, 16)

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		fmt.Printf("Error creando cipher: %v\n", err)
		return "", err
	}

	cipher := cipher.NewCBCEncrypter(block, ivBytes)
	encrypted := make([]byte, len(paddedData))
	cipher.CryptBlocks(encrypted, paddedData)

	return hex.EncodeToString(encrypted), nil
}

func makePublicDecryptUtil(data string) (string, string, error) {
	paddedData, err := hex.DecodeString(data)
	if err != nil {
		return "", "", err
	}

	keyBytes, err := hex.DecodeString(masterKey)
	if err != nil {
		return "", "", err
	}

	ivBytes, err := hex.DecodeString(masterIv)
	if err != nil {
		fmt.Printf("Error creando cipher: %v\n", err)
		return "", "", err
	}
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		fmt.Printf("Error creando cipher: %v\n", err)
		return "", "", err
	}
	cipher := cipher.NewCBCDecrypter(block, ivBytes)
	decrypted := make([]byte, len(paddedData))
	cipher.CryptBlocks(decrypted, paddedData)

	unpaddedData := pkcs7Unpadding(decrypted)

	//return key and iv

	unpaddedDataString := string(unpaddedData)

	//unpaddedDataString is a json string
	var dataJSON map[string]string
	err = json.Unmarshal([]byte(unpaddedDataString), &dataJSON)
	if err != nil {
		return "", "", err
	}

	return dataJSON["key"], dataJSON["iv"], nil
}

func pkcs7Unpadding(data []byte) []byte {
	padding := int(data[len(data)-1])
	return data[:len(data)-padding]
}

// Función para hacer padding PKCS7
func pkcs7Padding(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padtext := make([]byte, padding)
	for i := range padtext {
		padtext[i] = byte(padding)
	}
	return append(data, padtext...)
}

func EncryptMessage(message string, encriptionData string) (string, error) {
	encriptionData, err := fromBase64(encriptionData)
	if err != nil {
		return "", err
	}
	key, iv, err := makePublicDecryptUtil(encriptionData)
	if err != nil {
		return "", err
	}
	if message == "" {
		return "", errors.New("message is empty")
	}
	bufferData := []byte(message)

	// Aplicar padding PKCS7
	paddedData := pkcs7Padding(bufferData, 16)

	keyBytes, err := hex.DecodeString(key)
	if err != nil {
		return "", err
	}
	ivBytes, err := hex.DecodeString(iv)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}
	cipher := cipher.NewCBCEncrypter(block, ivBytes)
	encrypted := make([]byte, len(paddedData))
	cipher.CryptBlocks(encrypted, paddedData)

	encryptedBase64, err := toBase64(hex.EncodeToString(encrypted))
	if err != nil {
		return "", err
	}
	return encryptedBase64, nil
}

func DecryptMessage(message string, encriptionData string) (string, error) {
	encriptionData, err := fromBase64(encriptionData)
	if err != nil {
		return "", err
	}
	key, iv, err := makePublicDecryptUtil(encriptionData)
	if err != nil {
		return "", err
	}
	if message == "" {
		return "", errors.New("message is empty")
	}
	messageBase64, err := fromBase64(message)
	if err != nil {
		return "", err
	}
	encryptedBuffer, err := hex.DecodeString(messageBase64)
	if err != nil {
		return "", err
	}

	// Validar que el buffer tenga el tamaño correcto para AES (múltiplo de 16 bytes)
	if len(encryptedBuffer) == 0 {
		return "", err
	}
	if len(encryptedBuffer)%16 != 0 {
		return "", err
	}

	keyBytes, err := hex.DecodeString(key)
	if err != nil {
		return "", err
	}
	ivBytes, err := hex.DecodeString(iv)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}
	decipher := cipher.NewCBCDecrypter(block, ivBytes)
	decrypted := make([]byte, len(encryptedBuffer))
	decipher.CryptBlocks(decrypted, encryptedBuffer)

	// Remover padding PKCS7
	unpaddedData := pkcs7Unpadding(decrypted)

	return string(unpaddedData), nil
}

func toBase64(message string) (string, error) {
	decodedBytes, err := hex.DecodeString(message)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(decodedBytes), nil
}

func fromBase64(message string) (string, error) {
	decodedBytes, err := base64.StdEncoding.DecodeString(message)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(decodedBytes), nil
}

/*


	//test from encripted

	message := "Hello World"

	message, err = utils.EncryptMessage(message, room.EncriptionData)
	if err != nil {
		fmt.Println("error", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New(utils.ERRORS["INTERNAL_SERVER_ERROR"]))
	}

	fmt.Println("message encrypted", message)

	message, err = utils.DecryptMessage(message, room.EncriptionData)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New(utils.ERRORS["INTERNAL_SERVER_ERROR"]))
	}

	fmt.Println("message decrypted", message)

*/
