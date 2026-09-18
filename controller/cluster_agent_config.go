package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	corootv1 "github.io/coroot/operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"
)

const clusterAgentConfigPath = "/config/config.yaml"

func (r *CorootReconciler) clusterAgentConfigMap(ctx context.Context, cr *corootv1.Coroot, configEnvs ConfigEnvs) (*corev1.ConfigMap, string) {
	spec := cr.Spec.ClusterAgent
	if spec.AWS == nil && spec.GCP == nil && spec.OCI == nil && len(spec.Databases) == 0 {
		return nil, ""
	}
	logger := log.FromContext(ctx)

	type credentials struct {
		Username string `json:"username,omitempty"`
		Password string `json:"password,omitempty"`
	}
	type database struct {
		Type        string            `json:"type"`
		Host        string            `json:"host,omitempty"`
		Port        string            `json:"port,omitempty"`
		RDS         string            `json:"rds,omitempty"`
		Elasticache string            `json:"elasticache,omitempty"`
		CloudSQL    string            `json:"cloudsql,omitempty"`
		Memorystore string            `json:"memorystore,omitempty"`
		OCIDB       string            `json:"ocidb,omitempty"`
		OCICache    string            `json:"ocicache,omitempty"`
		Credentials *credentials      `json:"credentials,omitempty"`
		Params      map[string]string `json:"params,omitempty"`
	}
	type aws struct {
		Region                string            `json:"region,omitempty"`
		AccessKeyID           string            `json:"accessKeyId,omitempty"`
		SecretAccessKey       string            `json:"secretAccessKey,omitempty"`
		RDSTagFilters         map[string]string `json:"rdsTagFilters,omitempty"`
		ElasticacheTagFilters map[string]string `json:"elasticacheTagFilters,omitempty"`
	}
	type gcp struct {
		ProjectID               string            `json:"projectId,omitempty"`
		Region                  string            `json:"region,omitempty"`
		CredentialsJSON         string            `json:"credentialsJson,omitempty"`
		CloudSQLLabelFilters    map[string]string `json:"cloudsqlLabelFilters,omitempty"`
		MemorystoreLabelFilters map[string]string `json:"memorystoreLabelFilters,omitempty"`
	}

	type oci struct {
		CompartmentIDs  []string          `json:"compartmentIds,omitempty"`
		Region          string            `json:"region,omitempty"`
		TenancyID       string            `json:"tenancyId,omitempty"`
		UserID          string            `json:"userId,omitempty"`
		Fingerprint     string            `json:"fingerprint,omitempty"`
		PrivateKey      string            `json:"privateKey,omitempty"`
		DBTagFilters    map[string]string `json:"dbTagFilters,omitempty"`
		CacheTagFilters map[string]string `json:"cacheTagFilters,omitempty"`
	}
	type config struct {
		AWS       *aws       `json:"aws,omitempty"`
		GCP       *gcp       `json:"gcp,omitempty"`
		OCI       *oci       `json:"oci,omitempty"`
		Databases []database `json:"databases,omitempty"`
	}

	secretRef := func(s *corev1.SecretKeySelector) string {
		if s == nil {
			return ""
		}
		if _, err := r.GetSecret(ctx, cr, s); err != nil {
			logger.Error(err, "cluster-agent: secret is not available", "secret", s.Name, "key", s.Key)
		}
		return configEnvs.Add(s)
	}

	var cfg config
	if a := spec.AWS; a != nil {
		cfg.AWS = &aws{
			Region:                a.Region,
			RDSTagFilters:         a.RDSTagFilters,
			ElasticacheTagFilters: a.ElasticacheTagFilters,
		}
		if a.AccessKeySecret != nil {
			cfg.AWS.AccessKeyID = secretRef(&corev1.SecretKeySelector{LocalObjectReference: *a.AccessKeySecret, Key: "access_key_id"})
			cfg.AWS.SecretAccessKey = secretRef(&corev1.SecretKeySelector{LocalObjectReference: *a.AccessKeySecret, Key: "secret_access_key"})
		}
	}
	if g := spec.GCP; g != nil {
		cfg.GCP = &gcp{
			ProjectID:               g.ProjectId,
			Region:                  g.Region,
			CloudSQLLabelFilters:    g.CloudSQLLabelFilters,
			MemorystoreLabelFilters: g.MemorystoreLabelFilters,
		}
		if g.CredentialsSecret != nil {
			cfg.GCP.CredentialsJSON = secretRef(g.CredentialsSecret)
		}
	}
	if o := spec.OCI; o != nil {
		cfg.OCI = &oci{
			CompartmentIDs:  o.CompartmentIds,
			Region:          o.Region,
			DBTagFilters:    o.DBTagFilters,
			CacheTagFilters: o.CacheTagFilters,
		}
		if o.ApiKeySecret != nil {
			for _, k := range []struct {
				key string
				dst *string
			}{
				{"tenancy_id", &cfg.OCI.TenancyID}, {"user_id", &cfg.OCI.UserID}, {"fingerprint", &cfg.OCI.Fingerprint}, {"private_key", &cfg.OCI.PrivateKey},
			} {
				*k.dst = secretRef(&corev1.SecretKeySelector{LocalObjectReference: *o.ApiKeySecret, Key: k.key})
			}
		}
	}
	for _, d := range spec.Databases {
		db := database{Type: d.Type, Host: d.Host, Port: d.Port, RDS: d.RDS, Elasticache: d.Elasticache, CloudSQL: d.CloudSQL, Memorystore: d.Memorystore, OCIDB: d.OCIDB, OCICache: d.OCICache, Params: d.Params}
		if c := d.Credentials; c != nil {
			db.Credentials = &credentials{Username: c.Username, Password: c.Password}
			if c.UsernameSecret != nil {
				db.Credentials.Username = secretRef(c.UsernameSecret)
			}
			if c.PasswordSecret != nil {
				db.Credentials.Password = secretRef(c.PasswordSecret)
			}
		}
		cfg.Databases = append(cfg.Databases, db)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		logger.Error(err, "failed to marshal the cluster-agent config")
	}
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-cluster-agent",
			Namespace: cr.Namespace,
			Labels:    Labels(cr, "coroot-cluster-agent"),
		},
		BinaryData: map[string][]byte{"config.yaml": data},
	}
	hash := sha256.New()
	hash.Write(data)
	return cm, hex.EncodeToString(hash.Sum(nil))
}
