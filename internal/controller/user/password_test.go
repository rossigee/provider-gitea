package user

import (
	"context"
	"testing"

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	v2 "github.com/rossigee/provider-gitea/apis/user/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newPasswordTestClient(t *testing.T, objs ...client.Object) *externalClient {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	return &externalClient{kube: fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()}
}

func userWith(ns string, p v2.UserParameters) *v2.User {
	return &v2.User{ObjectMeta: metav1.ObjectMeta{Name: "u", Namespace: ns}, Spec: v2.UserSpec{ForProvider: p}}
}

func TestReadPassword_SecretRef(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "pw", Namespace: "gitea-resources"},
		Data:       map[string][]byte{"password": []byte("s3cret")},
	}
	e := newPasswordTestClient(t, secret)
	cr := userWith("gitea-resources", v2.UserParameters{
		PasswordSecretRef: &xpv1.SecretKeySelector{SecretReference: xpv1.SecretReference{Name: "pw", Namespace: "gitea-resources"}, Key: "password"},
	})

	got, err := e.readPassword(context.Background(), cr)
	if err != nil {
		t.Fatalf("readPassword() error = %v", err)
	}
	if got != "s3cret" {
		t.Fatalf("readPassword() = %q, want %q", got, "s3cret")
	}
}

func TestReadPassword_SecretRefPreferredOverPlainText(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "pw", Namespace: "gitea-resources"},
		Data:       map[string][]byte{"password": []byte("from-secret")},
	}
	e := newPasswordTestClient(t, secret)
	cr := userWith("gitea-resources", v2.UserParameters{
		Password:          "from-spec",
		PasswordSecretRef: &xpv1.SecretKeySelector{SecretReference: xpv1.SecretReference{Name: "pw", Namespace: "gitea-resources"}, Key: "password"},
	})

	got, err := e.readPassword(context.Background(), cr)
	if err != nil {
		t.Fatalf("readPassword() error = %v", err)
	}
	if got != "from-secret" {
		t.Fatalf("readPassword() = %q, want the secret value", got)
	}
}

func TestReadPassword_PlainTextFallback(t *testing.T) {
	e := newPasswordTestClient(t)
	cr := userWith("gitea-resources", v2.UserParameters{Password: "legacy"})

	got, err := e.readPassword(context.Background(), cr)
	if err != nil {
		t.Fatalf("readPassword() error = %v", err)
	}
	if got != "legacy" {
		t.Fatalf("readPassword() = %q, want %q", got, "legacy")
	}
}

func TestReadPassword_MissingKey(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "pw", Namespace: "gitea-resources"},
		Data:       map[string][]byte{"other": []byte("x")},
	}
	e := newPasswordTestClient(t, secret)
	cr := userWith("gitea-resources", v2.UserParameters{
		PasswordSecretRef: &xpv1.SecretKeySelector{SecretReference: xpv1.SecretReference{Name: "pw", Namespace: "gitea-resources"}, Key: "password"},
	})

	if _, err := e.readPassword(context.Background(), cr); err == nil {
		t.Fatal("readPassword() expected an error for a missing key")
	}
}

func TestReadPassword_NeitherSet(t *testing.T) {
	e := newPasswordTestClient(t)
	cr := userWith("gitea-resources", v2.UserParameters{})

	if _, err := e.readPassword(context.Background(), cr); err == nil {
		t.Fatal("readPassword() expected an error when neither field is set")
	}
}
