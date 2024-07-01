package aws

type Cluster struct {
	Arn      string
	Name     string
	Services []Service
}

type Service struct {
	Name       string
	Image      string
	App        string
	Env        string
	Component  string
	Container  string
	Version    string
	PrivateIPs []string
	PublicIPs  []string
}
