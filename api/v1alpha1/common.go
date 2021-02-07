package v1alpha1

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type AdminConnectionRef struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

type SecretKeySource struct {
	SecretKeyRef v1.SecretKeySelector `json:"secretKeyRef"`
}

// GetSecretRefValue returns the value of a secret in the supplied namespace
func GetSecretRefValue(ctx context.Context, client client.Client, namespace string, secretSelector *v1.SecretKeySelector) (string, error) {

	var namespacedName types.NamespacedName

	namespacedName.Name = secretSelector.Name
	namespacedName.Namespace = namespace

	// Fetch the Stack instance
	secret := &v1.Secret{}
	err := client.Get(ctx, namespacedName, secret)
	if err != nil {
		return "", err
	}
	if data, ok := secret.Data[secretSelector.Key]; ok {
		return string(data), nil
	}
	return "", fmt.Errorf("key %s not found in secret %s", secretSelector.Key, secretSelector.Name)

}

// Not all statements can be prepared with parameters (usernames/passwords).
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
