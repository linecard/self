package manifest

type Computed struct {
	Registry struct {
		Url string
	}

	Repository struct {
		Prefix string
		Name   string
		Path   string
		Url    string
	}

	Resource struct {
		Prefix string
		Name   string
		Policy struct {
			Arn string
		}
		Role struct {
			Arn string
		}
	}

	Resources struct {
		EphemeralStorage int32  `json:"ephemeralStorage"`
		MemorySize       int32  `json:"memorySize"`
		Timeout          int32  `json:"timeout"`
		Http             bool   `json:"http"`
		Public           bool   `json:"public"`
		RouteKey         string `json:"routeKey"`
	}
}
