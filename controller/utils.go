package controller

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"math/big"
	"reflect"
	"sort"

	"github.com/crewjam/saml/samlsp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	AnnotationLastAppliedConfiguration = "operator.coroot.com/last-applied-configuration"
	RandomStringCharset                = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

func RandomString(length int) string {
	res := make([]byte, length)
	for i := range res {
		randomIndex, _ := rand.Int(rand.Reader, big.NewInt(int64(len(RandomStringCharset))))
		res[i] = RandomStringCharset[randomIndex.Int64()]
	}
	return string(res)
}

type LastAppliedConfiguration struct {
	Annotations map[string]string `json:"annotations,omitempty"`
	Spec        json.RawMessage   `json:"spec,omitempty"`
}

func MergeSpecs[T any](obj client.Object, currentSpec *T, targetSpec T, targetAnnotations map[string]string) error {
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}

	current, err := json.Marshal(currentSpec)
	if err != nil {
		return fmt.Errorf("failed to marshal current: %w", err)
	}
	target, err := json.Marshal(targetSpec)
	if err != nil {
		return fmt.Errorf("failed to marshal target: %w", err)
	}

	var original []byte
	lastApplied := []byte(annotations[AnnotationLastAppliedConfiguration])
	var cfg LastAppliedConfiguration
	if err = json.Unmarshal(lastApplied, &cfg); err != nil {
		original = lastApplied
	} else {
		original = cfg.Spec
		for k := range cfg.Annotations {
			delete(annotations, k)
		}
	}
	for k, v := range targetAnnotations {
		annotations[k] = v
	}
	cfg.Annotations = targetAnnotations
	cfg.Spec = target
	lastApplied, err = json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal last applied: %w", err)
	}
	annotations[AnnotationLastAppliedConfiguration] = string(lastApplied)
	obj.SetAnnotations(annotations)

	patchMeta, err := strategicpatch.NewPatchMetaFromStruct(currentSpec)
	if err != nil {
		return fmt.Errorf("failed to produce patch meta from struct: %w", err)
	}
	patch, err := strategicpatch.CreateThreeWayMergePatch(original, target, current, patchMeta, true)
	if err != nil {
		return fmt.Errorf("failed to create three way merge patch: %w", err)
	}

	if string(patch) == "{}" {
		return nil
	}

	merged, err := strategicpatch.StrategicMergePatchUsingLookupPatchMeta(current, patch, patchMeta)
	if err != nil {
		return fmt.Errorf("failed to apply patch: %w", err)
	}

	valueOfCurrent := reflect.Indirect(reflect.ValueOf(currentSpec))
	into := reflect.New(valueOfCurrent.Type())
	if err = json.Unmarshal(merged, into.Interface()); err != nil {
		return fmt.Errorf("failed to unmarshal merged data: %w", err)
	}
	if !valueOfCurrent.CanSet() {
		return fmt.Errorf("unable to set unmarshalled value into current object")
	}
	valueOfCurrent.Set(reflect.Indirect(into))
	return nil
}

func envVarFromSecret(name string, secret *corev1.SecretKeySelector, plainTextValue string) corev1.EnvVar {
	if secret == nil {
		return corev1.EnvVar{Name: name, Value: plainTextValue}
	}
	return corev1.EnvVar{Name: name, ValueFrom: &corev1.EnvVarSource{SecretKeyRef: secret}}
}

func ValidateSamlIdentityProviderMetadata(metadata string) error {
	_, err := samlsp.ParseMetadata([]byte(metadata))
	return err
}

type ConfigEnvs map[string]*corev1.SecretKeySelector

func (ce ConfigEnvs) Add(selector *corev1.SecretKeySelector) string {
	hash := crc32.ChecksumIEEE([]byte(selector.Name + "/" + selector.Key))
	name := fmt.Sprintf("COROOT_CONFIG_SECRET_%X", hash)
	ce[name] = selector
	return fmt.Sprintf(`${%s}`, name)
}

func (ce ConfigEnvs) List() []corev1.EnvVar {
	envs := make([]corev1.EnvVar, 0, len(ce))
	for name, selector := range ce {
		envs = append(envs, corev1.EnvVar{Name: name, ValueFrom: &corev1.EnvVarSource{SecretKeyRef: selector}})
	}
	sort.Slice(envs, func(i, j int) bool { return envs[i].Name < envs[j].Name })
	return envs
}

func mergeEnvVars(defaults []corev1.EnvVar, overrides []corev1.EnvVar) []corev1.EnvVar {
	res := make([]corev1.EnvVar, 0, len(defaults)+len(overrides))
	res = append(res, defaults...)
	for _, o := range overrides {
		replaced := false
		for i := range res {
			if res[i].Name == o.Name {
				res[i] = o
				replaced = true
				break
			}
		}
		if !replaced {
			res = append(res, o)
		}
	}
	return res
}
