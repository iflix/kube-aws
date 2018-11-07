package config

import (
	"io/ioutil"
	"os"
	"testing"

	"fmt"

	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/coreos/coreos-cloudinit/config/validate"
	"github.com/kubernetes-incubator/kube-aws/model"
	"github.com/kubernetes-incubator/kube-aws/test/helper"
	"github.com/stretchr/testify/assert"
)

var numEncryption int

type dummyEncryptService struct{}

func (d *dummyEncryptService) Encrypt(input *kms.EncryptInput) (*kms.EncryptOutput, error) {
	output := kms.EncryptOutput{
		CiphertextBlob: []byte(fmt.Sprintf("%s%d", string(input.Plaintext), numEncryption)),
	}
	numEncryption += 1
	return &output, nil
}

func TestDummyEncryptService(t *testing.T) {
	encService := &dummyEncryptService{}
	plaintext := []byte("mysecretinformation")

	first, err := encService.Encrypt(&kms.EncryptInput{
		Plaintext: plaintext,
	})

	if err != nil {
		t.Errorf("failed to encrypt data %plaintext : %v", plaintext, err)
	}

	second, err := encService.Encrypt(&kms.EncryptInput{
		Plaintext: plaintext,
	})

	if err != nil {
		t.Errorf("failed to encrypt data %plaintext : %v", plaintext, err)
	}

	if first == second {
		t.Errorf("dummy encrypt service should produce different ciphertext for each encryption but it didnt: first = %v, second = %v", first, second)
	}
}

func TestCloudConfigTemplating(t *testing.T) {
	cluster, err := ClusterFromBytes([]byte(singleAzConfigYaml))
	if err != nil {
		t.Fatalf("Unable to load cluster config: %v", err)
	}

	cfg, err := cluster.Config()
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	opts := CredentialsOptions{
		GenerateCA: true,
		KIAM:       true,
	}

	var compactAssets *CompactAssets

	cachedEncryptor := CachedEncryptor{
		bytesEncryptionService: bytesEncryptionService{kmsKeyARN: cfg.KMSKeyARN, kmsSvc: &dummyEncryptService{}},
	}

	helper.WithTempDir(func(dir string) {
		_, err = cluster.NewAssetsOnDisk(dir, opts)
		if err != nil {
			t.Fatalf("Error generating default assets: %v", err)
		}

		encryptedAssets, err := ReadOrEncryptAssets(dir, true, true, true, cachedEncryptor)
		if err != nil {
			t.Fatalf("failed to compress assets: %v", err)
		}

		compactAssets, err = encryptedAssets.Compact()
		if err != nil {
			t.Fatalf("failed to compress assets: %v", err)
		}
	})

	if compactAssets == nil {
		t.Fatal("compactAssets is unexpectedly nil")
		t.FailNow()
	}

	cfg.AssetsConfig = compactAssets

	for _, cloudTemplate := range []struct {
		Name     string
		Template []byte
	}{
		{
			Name:     "CloudConfigController",
			Template: CloudConfigController,
		},
	} {
		tmpfile, _ := ioutil.TempFile("", "ud")
		tmpfile.Write(cloudTemplate.Template)
		tmpfile.Close()
		defer os.Remove(tmpfile.Name())

		udata, err := model.NewUserData(tmpfile.Name(), cfg)
		if !assert.NoError(t, err, "Error loading template %s", cloudTemplate.Name) {
			continue
		}
		content, err := udata.Parts[model.USERDATA_S3].Template()
		if !assert.NoError(t, err, "Can't render template %s", cloudTemplate.Name) {
			continue
		}

		report, err := validate.Validate([]byte(content))
		if !assert.NoError(t, err, "cloud-config %s could not be parsed", cloudTemplate.Name) {
			for _, entry := range report.Entries() {
				t.Errorf("%s: %+v", cloudTemplate.Name, entry)
			}
			continue
		}
	}
}
