package vault

type AuthN struct {
	Token      TokenAuth      `json:"token" yaml:"token"`
	AppRole    AppRoleAuth    `json:"approle" yaml:"approle"`
	Kubernetes KubernetesAuth `json:"kubernetes" yaml:"kubernetes"`
}

type TokenAuth struct {
	Token string `json:"token" yaml:"token"`
}

type AppRoleAuth struct {
	RoleID           string `json:"role_id" yaml:"role_id" split_words:"true"`
	SecretID         string `json:"secret_id" yaml:"secret_id" split_words:"true"`
	ResponseWrapping bool   `json:"response_wrapping" yaml:"response_wrapping" split_words:"true"`
}

type KubernetesAuth struct {
	Role      string `json:"role" yaml:"role"`
	TokenPath string `json:"token_path" yaml:"token_path" split_words:"true"`
}
