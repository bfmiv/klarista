package cmd

type KubernetesCluster struct {
	Cluster struct {
		CertificateAuthorityData *string `json:"certificate-authority-data,omitempty"`
		Server                   string  `json:"server"`
	} `json:"cluster"`
	Name string `json:"name"`
}

type KubernetesContext struct {
	Context struct {
		Cluster string `json:"cluster"`
		User    string `json:"user"`
	} `json:"context"`
	Name string `json:"name"`
}

type KubernetesUser struct {
	Name string `json:"name"`
	User struct {
		ClientCertificateData *string                 `json:"client-certificate-data,omitempty"`
		ClientKeyData         *string                 `json:"client-key-data,omitempty"`
		Exec                  *map[string]interface{} `json:"exec,omitempty"`
	} `json:"user"`
}

type KubernetesConfig struct {
	ApiVersion     string                 `json:"apiVersion"`
	Clusters       []KubernetesCluster    `json:"clusters"`
	Contexts       []KubernetesContext    `json:"contexts"`
	CurrentContext string                 `json:"current-context"`
	Kind           string                 `json:"kind"`
	Preferences    map[string]interface{} `json:"preferences"`
	Users          []KubernetesUser       `json:"users"`
}
