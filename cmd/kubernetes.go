package cmd

type KubernetesClusterCluster struct {
	CertificateAuthorityData *string `json:"certificate-authority-data,omitempty"`
	Server                   string  `json:"server"`
}

type KubernetesCluster struct {
	Cluster KubernetesClusterCluster `json:"cluster"`
	Name    string                   `json:"name"`
}

type KubernetesContextContext struct {
	Cluster string `json:"cluster"`
	User    string `json:"user"`
}

type KubernetesContext struct {
	Context KubernetesContextContext `json:"context"`
	Name    string                   `json:"name"`
}

type KubernetesUserUser struct {
	ClientCertificateData *string                 `json:"client-certificate-data,omitempty"`
	ClientKeyData         *string                 `json:"client-key-data,omitempty"`
	Exec                  *map[string]interface{} `json:"exec,omitempty"`
}

type KubernetesUser struct {
	Name string             `json:"name"`
	User KubernetesUserUser `json:"user"`
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

func generateKubeconfig(clusterName string, clientAuthenticationAPIVersion string, awsIAMRoleName string) KubernetesConfig {
	return KubernetesConfig{
		ApiVersion:     "v1",
		CurrentContext: clusterName,
		Kind:           "Config",
		Preferences:    map[string]interface{}{},
		Clusters: []KubernetesCluster{
			{
				Cluster: KubernetesClusterCluster{
					Server: "https://api." + clusterName,
				},
				Name: clusterName,
			},
		},
		Contexts: []KubernetesContext{
			{
				Context: KubernetesContextContext{
					Cluster: clusterName,
					User:    clusterName,
				},
				Name: clusterName,
			},
		},
		Users: []KubernetesUser{
			{
				Name: clusterName,
				User: KubernetesUserUser{
					Exec: &map[string]interface{}{
						"apiVersion": clientAuthenticationAPIVersion,
						"args": []string{
							"token",
							"-i",
							clusterName,
							"-r",
							awsIAMRoleName,
						},
						"command":            "aws-iam-authenticator",
						"env":                nil,
						"interactiveMode":    "IfAvailable",
						"provideClusterInfo": false,
					},
				},
			},
		},
	}
}
