package middleware

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/wutong-paas/wutong/pkg/kube"
	httputil "github.com/wutong-paas/wutong/util/http"
	"github.com/wutong-paas/wutong/util/k8s"
)

type LicenseCache struct {
	Content   string
	Valid     bool
	CacheTime time.Time
}

var licenseCache *LicenseCache

var ErrLicenseNotValid = errors.New("wutong region license not valid")

var licensePlaintextKey = "wutong-region-license"

var skipedLicenseMiddlewarePaths = []string{
	"/v2/health",
}

func slicesContaines(slices []string, str string) bool {
	for _, s := range slices {
		if s == str {
			return true
		}
	}
	return false
}

// License
func License(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if slicesContaines(skipedLicenseMiddlewarePaths, r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// 缓存 2 小时
		if licenseCache != nil && time.Since(licenseCache.CacheTime) < 2*time.Hour {
			if licenseCache.Valid {
				next.ServeHTTP(w, r)
				return
			}
			httputil.ReturnError(r, w, http.StatusUnauthorized, ErrLicenseNotValid.Error())
			return
		}

		licenseCache = &LicenseCache{
			CacheTime: time.Now(),
		}

		// 1. get license.crt and license.sig from kubernetes secret
		crt, sig, err := GetLicenseCrtAndSig()
		if err != nil {
			httputil.ReturnError(r, w, http.StatusUnauthorized, ErrLicenseNotValid.Error())
			licenseCache.Valid = false
			return
		}

		// 2. verify license
		err = VerifyLicense(licensePlaintextKey, crt, sig)
		if err != nil {
			httputil.ReturnError(r, w, http.StatusUnauthorized, ErrLicenseNotValid.Error())
			licenseCache.Valid = false
			return
		}
		licenseCache.Valid = true
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

// GetLicenseCrtAndSig 获取证书和签名（从kubernetes secret中获取）
func GetLicenseCrtAndSig() (crt *x509.Certificate, sig []byte, err error) {
	err = ErrLicenseNotValid

	k8scli := k8s.GetClientSet()
	license, getErr := kube.GetCachedResources(k8scli).SecretLister.Secrets("wt-system").Get("wutong-region-license")
	if getErr != nil {
		log.Printf("wutong-region license: get secret error: %v", getErr)
		return
	}
	crtData := license.Data["license.crt"]
	sig = license.Data["license.sig"]
	if crtData == nil && sig == nil {
		log.Printf("wutong-region license: crtData or sigData is nil")
		return
	}

	b, _ := pem.Decode(crtData)
	if b == nil {
		log.Printf("wutong-region license: pem.Decode failed")
		return
	}

	crt, err = x509.ParseCertificate(b.Bytes)
	if err != nil {
		log.Printf("wutong-region license: x509.ParseCertificate error: %v", err)
		return
	}

	err = nil
	return
}

// VerifyLicense 许可证验证：验证证书，并使用证书中的公钥对签名进行验证
func VerifyLicense(licensePlaintextKey string, cert *x509.Certificate, sig []byte) (err error) {
	// 对证书有效期进行验证
	if cert.NotBefore.After(time.Now()) || cert.NotAfter.Before(time.Now()) {
		msg := fmt.Sprintf("wutong-region license: cert is expired, notBefore: %v, notAfter: %v", cert.NotBefore, cert.NotAfter)
		log.Println(msg)
		return errors.New(msg)
	}

	// 打印证书过期时间
	log.Printf("wutong-region license: cert is valid now, and will be expired at %v", cert.NotAfter.Format(time.RFC3339))

	if err != nil {
		log.Printf("wutong-region license: base64 decode error: %v", err)
		return err
	}
	// 对签名进行验证
	h := sha256.New()
	h.Write([]byte(licensePlaintextKey))
	Sha256Code := h.Sum(nil)
	pub := cert.PublicKey.(*rsa.PublicKey)
	err = rsa.VerifyPKCS1v15(pub, crypto.SHA256, Sha256Code, sig)
	if err != nil {
		log.Printf("wutong-region license: verify error: %v", err)
		return err
	}
	return nil
}
