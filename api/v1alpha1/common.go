/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"context"
	"crypto/rand"
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"math/big"
	mrand "math/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

const (
	lowerCharSet   = "abcdedfghijklmnopqrst"
	upperCharSet   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	specialCharSet = "!@#$%&*"
	numberSet      = "0123456789"
	allCharSet     = lowerCharSet + upperCharSet + specialCharSet + numberSet
)

type AdminConnectionRef struct {
	// +kubebuilder:validation:Optional
	// +nullable
	Namespace string `json:"namespace,omitEmpty"`
	Name      string `json:"name"`
}

type SecretKeySource struct {
	SecretKeyRef v1.SecretKeySelector `json:"secretKeyRef"`
}

// GetSecretRefValue returns the value of a secret in the supplied namespace
func GetSecretRefValue(ctx context.Context, client client.Client, namespace string, secretSelector *v1.SecretKeySelector) (string, error) {

	// Fetch the Secret instance
	secret, err := GetSecret(ctx, client, namespace, secretSelector)
	if err != nil {
		return "", err
	}
	if data, ok := secret.Data[secretSelector.Key]; ok {
		return string(data), nil
	}
	return "", fmt.Errorf("key %s not found in secret %s", secretSelector.Key, secretSelector.Name)

}

func GetSecret(ctx context.Context, client client.Client, namespace string, secretSelector *v1.SecretKeySelector) (*v1.Secret, error) {
	var namespacedName types.NamespacedName

	namespacedName.Name = secretSelector.Name
	namespacedName.Namespace = namespace

	// Fetch the Secret instance
	secret := &v1.Secret{}
	err := client.Get(ctx, namespacedName, secret)
	if err != nil {
		return nil, err
	}
	return secret, nil
}

// Escape Not all statements can be prepared with parameters (usernames/passwords).
// For escaping MySQL strings.
// See: https://stackoverflow.com/questions/31647406/mysql-real-escape-string-equivalent-for-golang
func Escape(sql string) string {
	dest := make([]byte, 0, 2*len(sql))
	var escape byte
	for i := 0; i < len(sql); i++ {
		c := sql[i]

		escape = 0

		switch c {
		case 0: /* Must be escaped for 'mysql' */
			escape = '0'
			break
		case '\n': /* Must be escaped for logs */
			escape = 'n'
			break
		case '\r':
			escape = 'r'
			break
		case '\\':
			escape = '\\'
			break
		case '\'':
			escape = '\''
			break
		case '"': /* Better safe than sorry */
			escape = '"'
			break
		case '\032': //十进制26,八进制32,十六进制1a, /* This gives problems on Win32 */
			escape = 'Z'
		}

		if escape != 0 {
			dest = append(dest, '\\', escape)
		} else {
			dest = append(dest, c)
		}
	}

	return string(dest)
}

// GeneratePassword Formulated from: https://golangbyexample.com/generate-random-password-golang/
func GeneratePassword(passwordLength, minSpecialChar, minNum, minUpperCase int) string {
	var password strings.Builder

	//Set special character
	for i := 0; i < minSpecialChar; i++ {
		random, _ := rand.Int(rand.Reader, big.NewInt(int64(len(specialCharSet))))
		password.WriteString(string(specialCharSet[random.Int64()]))
	}

	//Set numeric
	for i := 0; i < minNum; i++ {
		random, _ := rand.Int(rand.Reader, big.NewInt(int64(len(numberSet))))
		password.WriteString(string(numberSet[random.Int64()]))
	}

	//Set uppercase
	for i := 0; i < minUpperCase; i++ {
		random, _ := rand.Int(rand.Reader, big.NewInt(int64(len(upperCharSet))))
		password.WriteString(string(upperCharSet[random.Int64()]))
	}

	remainingLength := passwordLength - minSpecialChar - minNum - minUpperCase
	for i := 0; i < remainingLength; i++ {
		random, _ := rand.Int(rand.Reader, big.NewInt(int64(len(allCharSet))))
		password.WriteString(string(allCharSet[random.Int64()]))
	}
	inRune := []rune(password.String())
	mrand.Shuffle(len(inRune), func(i, j int) {
		inRune[i], inRune[j] = inRune[j], inRune[i]
	})
	return string(inRune)
}
