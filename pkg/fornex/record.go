package fornex

// Record a DNS record.
type Record struct {
	ID    int    `json:"id,omitempty"`
	Host  string `json:"host,omitempty"`
	Type  string `json:"type,omitempty"`
	Value string `json:"value,omitempty"`
	TTL   int    `json:"ttl,omitempty"`
	Prio  int64  `json:"prio,omitempty"`
}
