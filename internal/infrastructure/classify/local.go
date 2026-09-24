package classify

import (
	"context"
	"log"
	"regexp"
	"strings"
	"unicode"

	"go-api/internal/domain/port"
)

const localHitScore = 0.95

type detector func(text string) bool

type LocalDetector struct {
	rules []namedDetector
}

type namedDetector struct {
	name string
	fn   detector
}

func NewLocalDetector() *LocalDetector {
	return &LocalDetector{rules: []namedDetector{
		{name: "email", fn: matchRegexp(reEmail)},
		{name: "iban", fn: matchRegexp(reIBAN)},
		{name: "api_key", fn: hasAPIKey},
		{name: "password", fn: hasPassword},
		{name: "credit_card", fn: hasLuhnNumber},
		{name: "phone", fn: hasPhoneNumber},
		{name: "private_key", fn: matchRegexp(rePrivateKey)},
		{name: "personal_id", fn: matchRegexp(rePersonalID)},
		{name: "connection_string", fn: hasConnectionString},
		{name: "jwt", fn: matchRegexp(reJWT)},
		{name: "wallet_secret", fn: matchRegexp(reWallet)},
		{name: "webhook_secret", fn: hasWebhookSecret},
		{name: "postal_address", fn: matchRegexp(rePostal)},
		{name: "person_name", fn: matchRegexp(rePersonName)},
	}}
}

var (
	reEmail      = regexp.MustCompile(`(?i)\b[A-Z0-9._%+\-]+@[A-Z0-9.\-]+\.[A-Z]{2,}\b`)
	reIBAN       = regexp.MustCompile(`(?i)\b[A-Z]{2}\d{2}(?:[ ]?[A-Z0-9]){11,30}\b`)
	reIdent      = regexp.MustCompile(`(?i)\$\{[A-Za-z][A-Za-z0-9_]*\}|[A-Za-z][A-Za-z0-9_.-]{1,80}`)
	reAssignment = regexp.MustCompile(`(?i)([A-Za-z][A-Za-z0-9_.-]{1,80})\s*[:=]\s*(\S{2,})`)
	rePassword   = regexp.MustCompile(`(?i)\b(?:password|passwd|pwd|passphrase|mot de passe)\s*[:=]\s*\S+`)
	reTokenShape = regexp.MustCompile(`(?i)\b(?:sk|pk|ghp|gho|xox[baprs]|akia|aiza|whsec)[-_][A-Za-z0-9_-]{12,}`)
	reURI        = regexp.MustCompile(`(?i)\b[a-z][a-z0-9+.-]{1,24}://\S+`)
	reUserPass   = regexp.MustCompile(`[^\s:/@]+:[^\s:/@]+@[A-Za-z0-9_.${}\-]+(?::\d{2,5})?`)
	reAtHostPort = regexp.MustCompile(`@[A-Za-z0-9_.${}\-]+:\d{2,5}\b`)
	reInterpPort = regexp.MustCompile(`\$\{[A-Za-z][A-Za-z0-9_]*\}:\d{2,5}\b`)
	reCard       = regexp.MustCompile(`\b(?:\d[ -]?){13,19}\b`)
	rePrivateKey = regexp.MustCompile(`-----BEGIN (?:RSA |EC |OPENSSH |DSA |ENCRYPTED )?PRIVATE KEY-----`)
	rePersonalID = regexp.MustCompile(`\b(?:\d{3}-\d{2}-\d{4}|[12]\s?\d{2}\s?\d{2}\s?\d{2}\s?\d{3}\s?\d{3}\s?\d{2})\b`)
	reJWT        = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b`)
	reWallet     = regexp.MustCompile(`(?i)(?:\b(?:seed phrase|recovery phrase|mnemonic)\b|0x[a-f0-9]{64}\b)`)
	rePostal     = regexp.MustCompile(`(?i)\b\d{1,4}\s+\S+(?:\s+\S+){0,6}\s+(?:rue|avenue|boulevard|street|road|place|impasse|chemin)\b`)
	rePersonName = regexp.MustCompile(`(?i)\b(?:nom|name|full name)\s*[:=]\s*[A-ZÀ-ÿ][a-zà-ÿ'\-]+\s+[A-ZÀ-ÿ][a-zà-ÿ'\-]+`)

	rePhoneFR   = regexp.MustCompile(`(?i)(?:\+33|0033)[\s./-]*(?:\(0\))?[\s./-]*[1-9](?:[\s./-]*\d{2}){4}|\b0[1-9](?:[\s./-]\d{2}){4}\b|\b0[1-9]\d{8}\b`)
	rePhoneUS   = regexp.MustCompile(`(?:\+1[\s./-]*)?(?:\(?\d{3}\)?[\s./-]\d{3}[\s./-]\d{4})`)
	rePhoneIntl = regexp.MustCompile(`\+\d{1,3}(?:[\s./-]+\d{2,4}){2,5}`)

	reGeoKeyword   = regexp.MustCompile(`(?i)\b(?:lat(?:itude)?|lon(?:gitude)?|lng|gps|coord(?:onnées|inates)?|wgs84)\b`)
	reDecimalCoord = regexp.MustCompile(`(?i)[+\-]?\d{1,3}\.\d{2,}\s*[°º]?\s*[nsewo]?`)
	reDMSCoord     = regexp.MustCompile(`(?i)\d{1,3}\s*[°º]\s*\d{1,2}\s*(?:['′]\s*\d{1,2}\s*(?:["″])?)?\s*[nsewo]`)
	reHemisphere   = regexp.MustCompile(`(?i)[°º]|[nsewo]\b`)
)

var sensitiveParts = map[string][]string{
	"password":       {"pass", "password", "passwd", "pwd", "passphrase", "secret"},
	"api_key":        {"key", "token", "apikey", "auth", "authtoken", "credential", "credentials"},
	"webhook_secret": {"webhook", "hmac", "signing"},
}

func matchRegexp(re *regexp.Regexp) detector {
	return func(text string) bool {
		return re.MatchString(text)
	}
}

func hasPassword(text string) bool {
	return rePassword.MatchString(text) || hasSensitiveIdent(text, "password")
}

func hasAPIKey(text string) bool {
	return reTokenShape.MatchString(text) || hasSensitiveIdent(text, "api_key")
}

func hasWebhookSecret(text string) bool {
	return hasSensitiveIdent(text, "webhook_secret")
}

func hasSensitiveIdent(text, kind string) bool {
	for _, ident := range reIdent.FindAllString(text, -1) {
		if identHasKind(ident, kind) {
			return true
		}
	}
	for _, m := range reAssignment.FindAllStringSubmatch(text, -1) {
		if len(m) < 3 {
			continue
		}
		if identHasKind(m[1], kind) && looksLikeSecretValue(m[2]) {
			return true
		}
	}
	return false
}

func identHasKind(ident, kind string) bool {
	ident = strings.TrimPrefix(ident, "${")
	ident = strings.TrimSuffix(ident, "}")
	for _, part := range splitIdent(ident) {
		for _, word := range sensitiveParts[kind] {
			if part == word {
				return true
			}
		}
	}
	return false
}

func splitIdent(ident string) []string {
	parts := strings.FieldsFunc(strings.ToLower(ident), func(r rune) bool {
		return r == '_' || r == '-' || r == '.'
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func looksLikeSecretValue(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || value == "*" || strings.EqualFold(value, "true") || strings.EqualFold(value, "false") {
		return false
	}
	if strings.HasPrefix(value, "${") {
		return true
	}
	return len(value) >= 3
}

func hasConnectionString(text string) bool {
	if reUserPass.MatchString(text) || reAtHostPort.MatchString(text) || reInterpPort.MatchString(text) {
		return true
	}
	for _, uri := range reURI.FindAllString(text, -1) {
		lower := strings.ToLower(uri)
		if strings.Contains(uri, "@") || strings.Contains(uri, "${") {
			return true
		}
		if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
			return true
		}
	}
	return false
}

func (d *LocalDetector) Classify(_ context.Context, frames []port.ClassifyFrame, threshold float64) ([]port.ClassifyItemResult, error) {
	if threshold <= 0 {
		threshold = 0.7
	}
	out := make([]port.ClassifyItemResult, 0, len(frames))
	for _, frame := range frames {
		text := strings.TrimSpace(frame.Text)
		if text == "" {
			out = append(out, port.ClassifyItemResult{FrameID: frame.FrameID, Status: "skipped"})
			continue
		}
		item := d.evaluate(frame.FrameID, text, threshold)
		log.Printf(
			"classify: local frame=%s categories=%d max=%.3f confidential=%t",
			item.FrameID,
			len(item.Categories),
			item.Probability,
			item.Confidential,
		)
		out = append(out, item)
	}
	return out, nil
}

func (d *LocalDetector) evaluate(frameID, text string, threshold float64) port.ClassifyItemResult {
	categories := make([]port.ClassifyCategory, 0)
	probability := 0.0
	for _, item := range d.rules {
		if !item.fn(text) {
			continue
		}
		categories = append(categories, port.ClassifyCategory{Name: item.name, Probability: localHitScore})
		if localHitScore > probability {
			probability = localHitScore
		}
	}
	return port.ClassifyItemResult{
		FrameID:      frameID,
		Confidential: probability >= threshold,
		Probability:  probability,
		Categories:   categories,
		Status:       "success",
	}
}

func hasPhoneNumber(text string) bool {
	if reGeoKeyword.MatchString(text) || reDMSCoord.MatchString(text) {
		return false
	}
	for _, re := range []*regexp.Regexp{rePhoneFR, rePhoneUS, rePhoneIntl} {
		for _, raw := range re.FindAllString(text, -1) {
			if isPhoneCandidate(raw) {
				return true
			}
		}
	}
	return false
}

func isPhoneCandidate(raw string) bool {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || reHemisphere.MatchString(trimmed) || strings.Contains(trimmed, ",") {
		return false
	}
	if isDecimalCoordinate(trimmed) {
		return false
	}
	if looksLikeGroupedDecimals(trimmed) || hasLongDecimalGroup(trimmed) {
		return false
	}
	digits := countDigits(trimmed)
	if digits < 10 || digits > 15 {
		return false
	}
	return true
}

func hasLongDecimalGroup(s string) bool {
	if !strings.Contains(s, ".") {
		return false
	}
	for _, part := range regexp.MustCompile(`[\s/.-]+`).Split(s, -1) {
		if countDigits(part) >= 5 {
			return true
		}
	}
	return false
}

func isDecimalCoordinate(s string) bool {
	compact := strings.TrimSpace(s)
	compact = strings.TrimRight(compact, "NnSsEeWwOo°º \t")
	dot := strings.Count(compact, ".")
	if dot != 1 {
		return false
	}
	return reDecimalCoord.MatchString(compact)
}

func looksLikeGroupedDecimals(s string) bool {
	parts := regexp.MustCompile(`[\s/.-]+`).Split(strings.TrimSpace(s), -1)
	if len(parts) < 2 {
		return isDecimalCoordinate(s)
	}
	decimals := 0
	for _, part := range parts {
		if strings.Contains(part, ".") || reDecimalCoord.MatchString(part) {
			decimals++
		}
	}
	return decimals >= 1 && !strings.ContainsAny(s, " /") && strings.Count(s, ".") == 1
}

func countDigits(s string) int {
	n := 0
	for _, r := range s {
		if unicode.IsDigit(r) {
			n++
		}
	}
	return n
}

func hasLuhnNumber(text string) bool {
	for _, raw := range reCard.FindAllString(text, -1) {
		if isDecimalCoordinate(raw) || reGeoKeyword.MatchString(text) {
			continue
		}
		digits := make([]int, 0, 19)
		for _, r := range raw {
			if r >= '0' && r <= '9' {
				digits = append(digits, int(r-'0'))
			}
		}
		if luhnValid(digits) {
			return true
		}
	}
	return false
}

func luhnValid(digits []int) bool {
	if len(digits) < 13 || len(digits) > 19 {
		return false
	}
	sum := 0
	alt := false
	for i := len(digits) - 1; i >= 0; i-- {
		n := digits[i]
		if alt {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
		alt = !alt
	}
	return sum%10 == 0
}
