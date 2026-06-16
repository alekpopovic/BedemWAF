package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"regexp"
	"strings"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)

func validateSlug(field, value string) error {
	if !slugPattern.MatchString(value) {
		return fmt.Errorf("%s must use lowercase letters, numbers, and hyphens", field)
	}
	return nil
}

func validateHostnames(values []string) error {
	if len(values) == 0 {
		return errors.New("hostnames must contain at least one hostname")
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		host := strings.TrimSpace(strings.ToLower(value))
		if host == "" {
			return errors.New("hostname cannot be empty")
		}
		if strings.Contains(host, "://") || strings.ContainsAny(host, "/?#") {
			return fmt.Errorf("hostname %q must not include scheme, path, or query", value)
		}
		if strings.Contains(host, ":") {
			parsed, _, err := net.SplitHostPort(host)
			if err != nil {
				return fmt.Errorf("hostname %q has invalid port syntax", value)
			}
			host = parsed
		}
		if _, err := netip.ParseAddr(host); err == nil {
			return fmt.Errorf("hostname %q must be a DNS hostname, not an IP address", value)
		}
		if err := validateDNSName(host); err != nil {
			return err
		}
		if _, ok := seen[host]; ok {
			return fmt.Errorf("hostname %q is duplicated", value)
		}
		seen[host] = struct{}{}
	}
	return nil
}

func validateDNSName(host string) error {
	if len(host) > 253 {
		return fmt.Errorf("hostname %q is too long", host)
	}
	labels := strings.Split(host, ".")
	for _, label := range labels {
		if label == "" || len(label) > 63 {
			return fmt.Errorf("hostname %q has an invalid label", host)
		}
		if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return fmt.Errorf("hostname %q labels cannot start or end with hyphen", host)
		}
		for _, r := range label {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
				continue
			}
			return fmt.Errorf("hostname %q contains invalid characters", host)
		}
	}
	return nil
}

func parseOriginURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, errors.New("origin_url scheme must be http or https")
	}
	if parsed.Hostname() == "" {
		return nil, errors.New("origin_url must include host")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("origin_url must not include query or fragment")
	}
	if port := parsed.Port(); port != "" {
		if _, err := net.LookupPort("tcp", port); err != nil {
			return nil, fmt.Errorf("origin_url port is invalid: %w", err)
		}
	}
	return parsed, nil
}

func validateMode(mode string) error {
	switch mode {
	case "count", "block":
		return nil
	default:
		return fmt.Errorf("mode must be count or block")
	}
}

func validateAction(action string) error {
	switch action {
	case "allow", "count", "block", "rate_limit":
		return nil
	default:
		return fmt.Errorf("action must be allow, count, block, or rate_limit")
	}
}

func validateCIDR(value string) error {
	_, err := netip.ParsePrefix(value)
	if err != nil {
		return fmt.Errorf("invalid CIDR %q: %w", value, err)
	}
	return nil
}

func validateJSON(raw json.RawMessage, field string) error {
	if len(raw) == 0 {
		return nil
	}
	if !json.Valid(raw) {
		return fmt.Errorf("%s must be valid JSON", field)
	}
	return nil
}
