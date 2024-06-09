package aws

type Cluster struct {
	Arn      string
	Name     string
	Services []Service
}

type Service struct {
	Name       string
	Image      string
	PrivateIPs []string
	PublicIPs  []string
}
