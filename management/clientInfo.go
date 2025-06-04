package management

type ClientInfo struct {
	HostName    string `json:"HostName"`
	HostAddress string `json:"HostAddress"`
	MACAddress  string `json:"MACAddress"`
	RemoteAddr  string `json:"RemoteAddr"`
}
