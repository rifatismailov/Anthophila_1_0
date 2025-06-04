package config

type Config struct {
	FileServer     string   `json:"file_server"`
	ManagerServer  *string  `json:"manager_server,omitempty"`
	LogServer      *string  `json:"log_server,omitempty"`
	LogCredentials *string  `json:"log_credentials,omitempty"` // optional: user:pass
	Directories    []string `json:"directories"`
	Extensions     []string `json:"extensions"`
	Hour           int      `json:"hour"`
	Minute         int      `json:"minute"`
	Key            string   `json:"key"`
}
