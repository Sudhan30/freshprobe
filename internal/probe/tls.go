package probe

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ocsp"
)

// TLSResult holds certificate freshness data.
type TLSResult struct {
	Subject       string    `json:"subject"`
	Issuer        string    `json:"issuer"`
	NotBefore     time.Time `json:"not_before"`
	NotAfter      time.Time `json:"not_after"`
	DaysRemaining int       `json:"days_remaining"`
	OCSPStapled   bool      `json:"ocsp_stapled"`
	OCSPStatus    string    `json:"ocsp_status,omitempty"`
	IsValid       bool      `json:"is_valid"`
}

// CheckTLS connects to the host and examines the TLS certificate.
func CheckTLS(ctx context.Context, host string) (*TLSResult, error) {
	// Ensure host has port
	if !strings.Contains(host, ":") {
		host = host + ":443"
	}

	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp", host, &tls.Config{
		InsecureSkipVerify: false,
	})
	if err != nil {
		return nil, fmt.Errorf("tls dial %s: %w", host, err)
	}
	defer conn.Close()

	state := conn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return nil, fmt.Errorf("no peer certificates from %s", host)
	}

	cert := state.PeerCertificates[0]
	now := time.Now()

	result := &TLSResult{
		Subject:       cert.Subject.CommonName,
		Issuer:        cert.Issuer.CommonName,
		NotBefore:     cert.NotBefore,
		NotAfter:      cert.NotAfter,
		DaysRemaining: int(cert.NotAfter.Sub(now).Hours() / 24),
		IsValid:       now.After(cert.NotBefore) && now.Before(cert.NotAfter),
	}

	// Check OCSP stapling
	if len(state.OCSPResponse) > 0 {
		result.OCSPStapled = true
		resp, err := ocsp.ParseResponse(state.OCSPResponse, nil)
		if err == nil {
			switch resp.Status {
			case ocsp.Good:
				result.OCSPStatus = "good"
			case ocsp.Revoked:
				result.OCSPStatus = "revoked"
				result.IsValid = false
			case ocsp.Unknown:
				result.OCSPStatus = "unknown"
			}
		}
	}

	return result, nil
}
