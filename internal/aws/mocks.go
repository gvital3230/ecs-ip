package aws

// MockClusters returns a list of mocked clusters. This function is used for testing purposes.
func (store *Store) MockClusters() []Cluster {
	res := []Cluster{
		{
			Arn:  "arn:aws:ecs:us-west-2:123456789012:cluster/default",
			Name: "default",
			Services: []Service{
				{
					Name:       "web",
					Image:      "nginx:latest",
					PrivateIPs: []string{"127.0.0.1"},
					PublicIPs:  []string{"8.9.8.8"},
				},
				{
					Name:       "db",
					Image:      "nginx:latest",
					PrivateIPs: []string{"127.0.0.1"},
					PublicIPs:  []string{"8.9.8.8"},
				},
			},
		},
		{
			Arn:  "arn:aws:ecs:us-west-2:123456789012:cluster/default",
			Name: "prod",
			Services: []Service{
				{
					Name:       "web",
					Image:      "nginx:latest",
					PrivateIPs: []string{"127.0.0.1"},
					PublicIPs:  []string{"8.9.8.8"},
				},
				{
					Name:       "db",
					Image:      "nginx:latest",
					PrivateIPs: []string{"127.0.0.1"},
					PublicIPs:  []string{"8.9.8.8"},
				},
			},
		},
	}
	return res
}
