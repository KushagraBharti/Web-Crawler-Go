package crawler

const (
	ErrTimeout       = "timeout"
	ErrDNS           = "dns"
	ErrTLS           = "tls"
	ErrHTTP          = "http"
	ErrStatus        = "status"
	ErrSizeLimit     = "size_limit"
	ErrParse         = "parse"
	ErrUnsupported   = "unsupported"
	ErrRobotsDenied  = "robots_denied"
	ErrCircuitOpen   = "circuit_open"
	ErrMaxDepth      = "max_depth"
	ErrMaxPages      = "max_pages"
	ErrFetch         = "fetch"
)