package scanner

// BatchRequest is the request body for batch query to OSV API.
type BatchRequest struct {
	Queries []*Request `json:"queries"`
}

// BatchResult is the result of a scan.
type BatchResult struct {
	Results []RequestResult `json:"results"`
}

// Request is the request body for single query to OSV API.
type Request struct {
	Commit string `json:"commit"`
}

// RequestResult is the result of a scan.
type RequestResult struct {
	Vulnerabilities []Vulnerability `json:"vulns"`
}

// Vulnerability is a vulnerability.
type Vulnerability struct {
	ID       string     `json:"id"`
	Affected []Affected `json:"affected"`
}

// Affected is a package affected by a vulnerability.
type Affected struct {
	Data map[string]interface{} `json:"ecosystem_specific,omitempty"`
}
