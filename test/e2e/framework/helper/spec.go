package helper

import (
	"fmt"
	"strings"
	"time"

	cmapi "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
	"github.com/jetstack/cert-manager/pkg/util/pki"

	csiapi "github.com/jetstack/cert-manager-csi/pkg/apis/v1alpha1"
	"github.com/jetstack/cert-manager-csi/pkg/util"
)

func (h *Helper) CertificateRequestMatchesSpec(cr *cmapi.CertificateRequest, attr map[string]string) error {
	var errs []string

	issuerName, ok := attr[csiapi.IssuerNameKey]
	if !ok {
		errs = append(errs, fmt.Sprintf("required %q not in volume attributes present", csiapi.IssuerNameKey))
	} else {
		if issuerName != cr.Spec.IssuerRef.Name {
			errs = append(errs, fmt.Sprintf("expected IssuerRef.Name to equal %q, got %q",
				issuerName, cr.Spec.IssuerRef.Name))
		}
	}

	issuerKind, ok := attr[csiapi.IssuerKindKey]
	if !ok {
		issuerKind = "Issuer"
	}
	if issuerKind != cr.Spec.IssuerRef.Kind {
		errs = append(errs, fmt.Sprintf("expected IssuerRef.Kind to equal %q, got %q",
			issuerKind, cr.Spec.IssuerRef.Kind))
	}

	issuerGroup, ok := attr[csiapi.IssuerGroupKey]
	if !ok {
		issuerGroup = "cert-manager.io"
	}
	if issuerGroup != cr.Spec.IssuerRef.Group {
		errs = append(errs, fmt.Sprintf("expected IssuerRef.Group to equal %q, got %q",
			issuerGroup, cr.Spec.IssuerRef.Group))
	}

	isCA, ok := attr[csiapi.IsCAKey]
	if !ok {
		isCA = "false"
	}

	if isCA != "false" && isCA != "true" {
		errs = append(errs,
			fmt.Sprintf("isCA value must be 'true', 'false', or '', got %q",
				isCA))
	} else {
		if (isCA == "true" && !cr.Spec.IsCA) || (isCA == "false" && cr.Spec.IsCA) {
			errs = append(errs,
				fmt.Sprintf("expected IsCA value to be %s, got %t",
					isCA, cr.Spec.IsCA))
		}
	}

	duration, ok := attr[csiapi.DurationKey]
	if ok {
		durationT, err := time.ParseDuration(duration)
		if err != nil {
			errs = append(errs, fmt.Sprintf("failed to parse attribute duration %q: %s",
				duration, err))
		} else if durationT != cr.Spec.Duration.Duration {
			errs = append(errs, fmt.Sprintf("unexpected requested duration, exp=%s got=%s",
				durationT, cr.Spec.Duration.Duration))
		}
	}

	csr, err := pki.DecodeX509CertificateRequestBytes(
		cr.Spec.CSRPEM)
	if err != nil {
		errs = append(errs, fmt.Sprintf("failed to parse certificate request PEM: %s",
			err))
	} else {
		commonName := attr[csiapi.CommonNameKey]
		if commonName != csr.Subject.CommonName {
			errs = append(errs, fmt.Sprintf("common name does not match, exp=%s got=%s",
				commonName, csr.Subject.CommonName))
		}

		dnsNames := util.ParseDNSNames(attr[csiapi.DNSNamesKey])
		if !util.StringsMatch(dnsNames, csr.DNSNames) {
			errs = append(errs, fmt.Sprintf("dns names do not match, exp=%s got=%s",
				dnsNames, csr.DNSNames))
		}

		ips := util.ParseIPAddresses(attr[csiapi.IPSANsKey])
		if !util.IPAddressesMatch(ips, csr.IPAddresses) {
			errs = append(errs, fmt.Sprintf("ip addresses do not match, exp=%v got=%v",
				ips, csr.IPAddresses))
		}

		uris, err := util.ParseURIs(attr[csiapi.URISANsKey])
		if err != nil {
			errs = append(errs, fmt.Sprintf("failed to parse URIs in attributes: %s",
				err))
		} else if !util.URIsMatch(uris, csr.URIs) {
			errs = append(errs, fmt.Sprintf("uris do not match, exp=%v got=%v",
				uris, csr.URIs))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("certificate request %q does not match volume attribute spec: %s",
			cr.Name, strings.Join(errs, ", "))
	}

	return nil
}
