package model

type Config struct {
	APIVersion     string       `json:"apiVersion"`
	Kind           string       `json:"kind"`
	CurrentContext string       `json:"current-context"`
	Contexts       ContextList  `json:"contexts"`
	Clusters       ClusterList  `json:"clusters"`
	AuthInfos      AuthInfoList `json:"users"`
}

type ContextList []*ContextItem
type ContextItem struct {
	Name    string   `json:"name"`
	Context *Context `json:"context"`
}

type ClusterList []*ClusterItem
type ClusterItem struct {
	Name    string   `json:"name"`
	Cluster *Cluster `json:"cluster"`
}

type AuthInfoList []*AuthInfoItem
type AuthInfoItem struct {
	Name     string    `json:"name"`
	AuthInfo *AuthInfo `json:"user"`
}

type Context struct {
	Cluster   string `json:"cluster"`
	AuthInfo  string `json:"user"`
	Namespace string `json:"namespace"`
}

type Cluster struct {
	Server                   string `json:"server"`
	CertificateAuthorityData []byte `json:"certificate-authority-data"`
}

type AuthInfo struct {
	Token string `json:"token"`
}
