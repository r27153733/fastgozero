package security

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/valyala/fasthttp"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/codec"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

const (
	requestUriHeader = "X-Request-Uri"
	signatureField   = "signature"
	timeField        = "time"
)

var (
	// ErrInvalidContentType is an error that indicates invalid content type.
	ErrInvalidContentType = errors.New("invalid content type")
	// ErrInvalidHeader is an error that indicates invalid X-Content-Security header.
	ErrInvalidHeader = errors.New("invalid X-Content-Security header")
	// ErrInvalidKey is an error that indicates invalid key.
	ErrInvalidKey = errors.New("invalid key")
	// ErrInvalidPublicKey is an error that indicates invalid public key.
	ErrInvalidPublicKey = errors.New("invalid public key")
	// ErrInvalidSecret is an error that indicates invalid secret.
	ErrInvalidSecret = errors.New("invalid secret")
)

// A ContentSecurityHeader is a content security header.
type ContentSecurityHeader struct {
	Key         []byte
	Timestamp   string
	ContentType int
	Signature   string
}

// Encrypted checks if it's a crypted request.
func (h *ContentSecurityHeader) Encrypted() bool {
	return h.ContentType == httpx.CryptionType
}

// ParseContentSecurity parses content security settings in give r.
func ParseContentSecurity(decrypters map[string]codec.RsaDecrypter, contentSecurity string) (
	*ContentSecurityHeader, error) {
	attrs := httpx.ParseHeader(contentSecurity)
	fingerprint := attrs[httpx.KeyField]
	secret := attrs[httpx.SecretField]
	signature := attrs[signatureField]

	if len(fingerprint) == 0 || len(secret) == 0 || len(signature) == 0 {
		return nil, ErrInvalidHeader
	}

	decrypter, ok := decrypters[fingerprint]
	if !ok {
		return nil, ErrInvalidPublicKey
	}

	decryptedSecret, err := decrypter.DecryptBase64(secret)
	if err != nil {
		return nil, ErrInvalidSecret
	}

	attrs = httpx.ParseHeader(string(decryptedSecret))
	base64Key := attrs[httpx.KeyField]
	timestamp := attrs[timeField]
	contentType := attrs[httpx.TypeField]

	key, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil {
		return nil, ErrInvalidKey
	}

	cType, err := strconv.Atoi(contentType)
	if err != nil {
		return nil, ErrInvalidContentType
	}

	return &ContentSecurityHeader{
		Key:         key,
		Timestamp:   timestamp,
		ContentType: cType,
		Signature:   signature,
	}, nil
}

// VerifySignature verifies the signature in given r.
func VerifySignature(r *fasthttp.Request, securityHeader *ContentSecurityHeader, tolerance time.Duration) int {
	seconds, err := strconv.ParseInt(securityHeader.Timestamp, 10, 64)
	if err != nil {
		return httpx.CodeSignatureInvalidHeader
	}

	now := time.Now().Unix()
	toleranceSeconds := int64(tolerance.Seconds())
	if seconds+toleranceSeconds < now || now+toleranceSeconds < seconds {
		return httpx.CodeSignatureWrongTime
	}

	reqPath, reqQuery := getPathQuery(r)
	signContent := strings.Join([]string{
		securityHeader.Timestamp,
		string(r.Header.Method()),
		reqPath,
		reqQuery,
		computeBodySignature(r),
	}, "\n")
	actualSignature := codec.HmacBase64(securityHeader.Key, signContent)

	if securityHeader.Signature == actualSignature {
		return httpx.CodeSignaturePass
	}

	logx.Infof("signature different, expect: %s, actual: %s",
		securityHeader.Signature, actualSignature)

	return httpx.CodeSignatureInvalidToken
}

func computeBodySignature(r *fasthttp.Request) string {
	sha := sha256.New()
	sha.Write(r.Body())
	return fmt.Sprintf("%x", sha.Sum(nil))
}

func getPathQuery(r *fasthttp.Request) (string, string) {
	requestUri := string(r.Header.Peek(requestUriHeader))
	if len(requestUri) == 0 {
		return string(r.URI().Path()), string(r.URI().QueryString())
	}

	uri, err := url.Parse(requestUri)
	if err != nil {
		return string(r.URI().Path()), string(r.URI().QueryString())
	}

	return uri.Path, uri.RawQuery
}
