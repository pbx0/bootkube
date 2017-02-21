package asset

import (
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"

	"github.com/kubernetes-incubator/bootkube/pkg/tlsutil"
)

const (
	AssetPathCAKey                       = "tls/ca.key"
	AssetPathCACert                      = "tls/ca.crt"
	AssetPathAPIServerKey                = "tls/apiserver.key"
	AssetPathAPIServerCert               = "tls/apiserver.crt"
	AssetPathServiceAccountPrivKey       = "tls/service-account.key"
	AssetPathServiceAccountPubKey        = "tls/service-account.pub"
	AssetPathAdminKey                    = "tls/admin.key"
	AssetPathAdminCert                   = "tls/admin.crt"
	AssetPathAdminKubeConfig             = "auth/admin-kubeconfig"
	AssetPathBootstrapKubeConfig         = "auth/bootstrap-kubeconfig"
	AssetPathBootstrapAuthToken          = "auth/bootstrap-auth-token"
	AssetPathManifests                   = "manifests"
	AssetPathKubelet                     = "manifests/kubelet.yaml"
	AssetPathKubeletBootstrapRoleBinding = "manifests/kubelet-bootstrap-role-binding.yaml"
	AssetPathKubeSystemSARoleBinding     = "manifests/kube-sa-role-binding.yaml"
	AssetPathProxy                       = "manifests/kube-proxy.yaml"
	AssetPathKubeFlannel                 = "manifests/kube-flannel.yaml"
	AssetPathKubeFlannelCfg              = "manifests/kube-flannel-cfg.yaml"
	AssetPathAPIServerSecret             = "manifests/kube-apiserver-secret.yaml"
	AssetPathAPIServer                   = "manifests/kube-apiserver.yaml"
	AssetPathControllerManager           = "manifests/kube-controller-manager.yaml"
	AssetPathControllerManagerSecret     = "manifests/kube-controller-manager-secret.yaml"
	AssetPathControllerManagerDisruption = "manifests/kube-controller-manager-disruption.yaml"
	AssetPathScheduler                   = "manifests/kube-scheduler.yaml"
	AssetPathSchedulerDisruption         = "manifests/kube-scheduler-disruption.yaml"
	AssetPathKubeDNSDeployment           = "manifests/kube-dns-deployment.yaml"
	AssetPathKubeDNSSvc                  = "manifests/kube-dns-svc.yaml"
	AssetPathCheckpointer                = "manifests/pod-checkpoint-installer.yaml"
	AssetPathEtcdOperator                = "manifests/etcd-operator.yaml"
	AssetPathEtcdSvc                     = "manifests/etcd-service.yaml"
	AssetPathExtraKubeletApprover        = "extra/kubelet-approver.yaml"
)

// AssetConfig holds all configuration needed when generating
// the default set of assets.
type Config struct {
	EtcdServers        []*url.URL
	APIServers         []*url.URL
	CACert             *x509.Certificate
	CAPrivKey          *rsa.PrivateKey
	BootstrapAuthToken string
	AltNames           *tlsutil.AltNames
	SelfHostKubelet    bool
	SelfHostedEtcd     bool
	CloudProvider      string
}

// NewDefaultAssets returns a list of default assets, optionally
// configured via a user provided AssetConfig. Default assets include
// TLS assets (certs, keys and secrets), and k8s component manifests.
func NewDefaultAssets(conf Config) (Assets, error) {
	as := newStaticAssets(conf.SelfHostKubelet, conf.SelfHostedEtcd)
	as = append(as, newDynamicAssets(conf)...)

	// TLS assets
	tlsAssets, err := newTLSAssets(conf.CACert, conf.CAPrivKey, *conf.AltNames)
	if err != nil {
		return Assets{}, err
	}
	as = append(as, tlsAssets...)

	// K8S bootstrap-kubeconfig
	// Used by kubelets to bootstrap their TLS certificate
	bootstrapKubeConfig, err := newBootstrapKubeConfigAsset(as, conf)
	if err != nil {
		return Assets{}, err
	}
	as = append(as, bootstrapKubeConfig)

	// K8S admin-kubeconfig
	// Used by operators to interact with the cluster
	adminKubeConfig, err := newAdminKubeConfigAsset(as, conf)
	if err != nil {
		return Assets{}, err
	}
	as = append(as, adminKubeConfig)

	// K8S APIServer secret
	apiSecret, err := newAPIServerSecretAsset(as)
	if err != nil {
		return Assets{}, err
	}
	as = append(as, apiSecret)

	// K8S ControllerManager secret
	cmSecret, err := newControllerManagerSecretAsset(as)
	if err != nil {
		return Assets{}, err
	}
	as = append(as, cmSecret)

	return as, nil
}

type Asset struct {
	Name string
	Data []byte
}

type Assets []Asset

func (as Assets) Get(name string) (Asset, error) {
	for _, asset := range as {
		if asset.Name == name {
			return asset, nil
		}
	}
	return Asset{}, fmt.Errorf("asset %q does not exist", name)
}

func (as Assets) WriteFiles(path string) error {
	if err := os.Mkdir(path, 0755); err != nil {
		return err
	}
	for _, asset := range as {
		f := filepath.Join(path, asset.Name)
		if err := os.MkdirAll(filepath.Dir(f), 0755); err != nil {
			return err
		}
		fmt.Printf("Writing asset: %s\n", f)
		if err := ioutil.WriteFile(f, asset.Data, 0600); err != nil {
			return err
		}
	}
	return nil
}
