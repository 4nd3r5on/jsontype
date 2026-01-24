package jsontype

import (
	"encoding/base64"
	"encoding/hex"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
)

var (
	// Precompiled regexes (anchored, minimal)
	reUUID        = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	rePhone       = regexp.MustCompile(`^\+[1-9]\d{7,14}$`) // Stricter: require + and min 8 digits
	reMAC         = regexp.MustCompile(`^([0-9a-f]{2}:){5}[0-9a-f]{2}$`)
	reWinPath     = regexp.MustCompile(`^[a-zA-Z]:[/\\]`)
	reDomain      = regexp.MustCompile(`^([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,}$`)
	reHexStrict   = regexp.MustCompile(`^[0-9a-f]+$`)
	reBase64Alpha = regexp.MustCompile(`^[A-Za-z0-9+/]+=*$`)
	reBase64URL   = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
)

type detector func(string) (DetectedType, bool)

// Ordered list of detectors: specific → generic, cheap → expensive
var detectors = []detector{
	// 1. Networking (highest priority, most specific)
	detectIPv6PortPair,
	detectIPv4PortPair,
	detectIPv4WithMask,
	detectIPv6,
	detectIPv4,
	detectMAC,

	// 2. Identifiers (UUID must come before base64url!)
	detectUUID,
	detectEmail,
	detectPhone,

	// 3. URLs and domains
	detectLink,
	detectDomain,

	// 4. Encodings (must come after UUID and networking)
	detectHex,
	detectBase64URL,
	detectBase64,

	// 5. Paths
	detectWindowsPath,
}

// DetectStrType detects the type of a string value
func DetectStrType(s string) DetectedType {
	// Trivial rejection
	if s == "" {
		return TypeString
	}

	// Run detectors in order, first match wins
	for _, detect := range detectors {
		if typ, ok := detect(s); ok {
			return typ
		}
	}

	return TypeString
}

// ============= Networking Detectors =============

func detectIPv6PortPair(s string) (DetectedType, bool) {
	if !strings.HasPrefix(s, "[") {
		return "", false
	}
	host, port, err := net.SplitHostPort(s)
	if err != nil || port == "" {
		return "", false
	}
	ip := net.ParseIP(host)
	if ip != nil && ip.To4() == nil {
		return TypeIPv6PortPair, true
	}
	return "", false
}

func detectIPv4PortPair(s string) (DetectedType, bool) {
	if !strings.Contains(s, ":") {
		return "", false
	}
	host, port, err := net.SplitHostPort(s)
	if err != nil || port == "" {
		return "", false
	}
	ip := net.ParseIP(host)
	if ip != nil && ip.To4() != nil {
		return TypeIPv4PortPair, true
	}
	return "", false
}

func detectIPv4WithMask(s string) (DetectedType, bool) {
	if !strings.Contains(s, "/") {
		return "", false
	}
	_, _, err := net.ParseCIDR(s)
	if err != nil {
		return "", false
	}
	// Ensure it's IPv4 by checking the IP part
	parts := strings.Split(s, "/")
	if len(parts) == 2 {
		ip := net.ParseIP(parts[0])
		if ip != nil && ip.To4() != nil {
			return TypeIPv4WithMask, true
		}
	}
	return "", false
}

func detectIPv6(s string) (DetectedType, bool) {
	if !strings.Contains(s, ":") {
		return "", false
	}
	ip := net.ParseIP(s)
	if ip != nil && ip.To4() == nil {
		return TypeIPv6, true
	}
	return "", false
}

func detectIPv4(s string) (DetectedType, bool) {
	if !strings.Contains(s, ".") {
		return "", false
	}
	ip := net.ParseIP(s)
	if ip != nil && ip.To4() != nil {
		return TypeIPv4, true
	}
	return "", false
}

func detectMAC(s string) (DetectedType, bool) {
	if len(s) != 17 { // Fixed length: xx:xx:xx:xx:xx:xx
		return "", false
	}
	s = strings.ToLower(s)
	if reMAC.MatchString(s) {
		return TypeMAC, true
	}
	return "", false
}

// ============= Identifier Detectors =============

func detectUUID(s string) (DetectedType, bool) {
	if len(s) != 36 { // Fixed length
		return "", false
	}
	s = strings.ToLower(s)
	if reUUID.MatchString(s) {
		return TypeUUID, true
	}
	return "", false
}

func detectEmail(s string) (DetectedType, bool) {
	if !strings.Contains(s, "@") {
		return "", false
	}
	// mail.ParseAddress is permissive, so add basic checks
	addr, err := mail.ParseAddress(s)
	if err != nil {
		return "", false
	}
	// Ensure it's just an email, not "Name <email>"
	if addr.Address == s && strings.Count(s, "@") == 1 {
		return TypeEmail, true
	}
	return "", false
}

func detectPhone(s string) (DetectedType, bool) {
	// E.164 format: must start with +, then 8-15 digits
	if len(s) < 9 || len(s) > 16 {
		return "", false
	}
	if !strings.HasPrefix(s, "+") {
		return "", false
	}
	if rePhone.MatchString(s) {
		return TypePhone, true
	}
	return "", false
}

// ============= URL/Domain Detectors =============

func detectLink(s string) (DetectedType, bool) {
	// Structural pre-check
	lower := strings.ToLower(s)
	hasScheme := strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")

	if !hasScheme {
		// For scheme-less URLs like "google.com/search"
		// Must have both dot and slash
		if !strings.Contains(s, ".") || !strings.Contains(s, "/") {
			return "", false
		}
		// Reject if it looks like a Windows path
		if len(s) >= 3 && s[1] == ':' && (s[2] == '/' || s[2] == '\\') {
			return "", false
		}
	}

	u, err := url.ParseRequestURI(s)
	if err != nil {
		// Try adding scheme for scheme-less URLs
		u, err = url.ParseRequestURI("https://" + s)
		if err != nil {
			return "", false
		}
	}
	if u.Host != "" {
		return TypeLink, true
	}
	return "", false
}

func detectDomain(s string) (DetectedType, bool) {
	if !strings.Contains(s, ".") || strings.Contains(s, "/") || strings.Contains(s, ":") {
		return "", false
	}
	s = strings.ToLower(s)
	if reDomain.MatchString(s) {
		return TypeDomain, true
	}
	return "", false
}

// ============= Encoding Detectors =============

func detectHex(s string) (DetectedType, bool) {
	// Must be even length and at least 8 chars to avoid false positives
	if len(s) < 8 || len(s)%2 != 0 {
		return "", false
	}
	s = strings.ToLower(s)
	if !reHexStrict.MatchString(s) {
		return "", false
	}
	// Reject if it looks like a UUID pattern (has hyphens in original)
	// This is already handled by UUID being checked first

	// Attempt decode to ensure it's valid
	_, err := hex.DecodeString(s)
	return TypeHEX, err == nil
}

func detectBase64URL(s string) (DetectedType, bool) {
	// Minimum 8 chars to reduce false positives
	if len(s) < 8 {
		return "", false
	}

	// Base64URL doesn't use padding and uses - and _
	if strings.Contains(s, "+") || strings.Contains(s, "/") || strings.Contains(s, "=") {
		return "", false
	}

	// Additional check: must contain at least some base64url-specific chars or uppercase
	// to avoid matching simple lowercase strings like "london" or "engineering"
	hasSpecialOrUpper := false
	hasDigit := false

	for _, r := range s {
		if r == '-' || r == '_' {
			hasSpecialOrUpper = true
		} else if r >= 'A' && r <= 'Z' {
			hasSpecialOrUpper = true
		} else if r >= '0' && r <= '9' {
			hasDigit = true
		}
	}

	// Require mixed case or special chars or digits
	// Reject pure lowercase alphabetic strings
	if !hasSpecialOrUpper && !hasDigit {
		return "", false
	}

	if !reBase64URL.MatchString(s) {
		return "", false
	}

	// Attempt decode
	_, err := base64.RawURLEncoding.DecodeString(s)
	return TypeBase64URL, err == nil
}

func detectBase64(s string) (DetectedType, bool) {
	// Minimum 8 chars to reduce false positives
	if len(s) < 8 {
		return "", false
	}

	// Standard base64 alphabet check
	if !reBase64Alpha.MatchString(s) {
		return "", false
	}

	// Must contain at least one uppercase or special char to avoid false positives
	hasUpperOrSpecial := false
	for _, r := range s {
		if (r >= 'A' && r <= 'Z') || r == '+' || r == '/' || r == '=' {
			hasUpperOrSpecial = true
			break
		}
	}

	if !hasUpperOrSpecial {
		return "", false
	}

	// Check padding rules
	padCount := strings.Count(s, "=")
	if padCount > 2 || (padCount > 0 && !strings.HasSuffix(s, strings.Repeat("=", padCount))) {
		return "", false
	}

	// Attempt decode
	_, err := base64.StdEncoding.DecodeString(s)
	return TypeBase64, err == nil
}

// ============= Path Detectors =============

func detectWindowsPath(s string) (DetectedType, bool) {
	if len(s) < 3 {
		return "", false
	}
	if reWinPath.MatchString(s) {
		return TypeFilepathWindows, true
	}
	return "", false
}
