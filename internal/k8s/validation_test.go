package k8s

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	networking "k8s.io/api/networking/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestValidateIngress(t *testing.T) {
	tests := []struct {
		ing            *networking.Ingress
		isPlus         bool
		expectedErrors []string
		msg            string
	}{
		{
			ing: &networking.Ingress{
				Spec: networking.IngressSpec{
					Rules: []networking.IngressRule{
						{
							Host: "example.com",
						},
					},
				},
			},
			isPlus:         false,
			expectedErrors: nil,
			msg:            "valid input",
		},
		{
			ing: &networking.Ingress{
				ObjectMeta: meta_v1.ObjectMeta{
					Annotations: map[string]string{
						"nginx.org/mergeable-ingress-type": "invalid",
					},
				},
				Spec: networking.IngressSpec{
					Rules: []networking.IngressRule{
						{
							Host: "",
						},
					},
				},
			},
			isPlus: false,
			expectedErrors: []string{
				`annotations.nginx.org/mergeable-ingress-type: Invalid value: "invalid": must be one of: 'master' or 'minion'`,
				"spec.rules[0].host: Required value",
			},
			msg: "invalid ingress",
		},
		{
			ing: &networking.Ingress{
				ObjectMeta: meta_v1.ObjectMeta{
					Annotations: map[string]string{
						"nginx.org/mergeable-ingress-type": "master",
					},
				},
				Spec: networking.IngressSpec{
					Rules: []networking.IngressRule{
						{
							Host: "example.com",
							IngressRuleValue: networking.IngressRuleValue{
								HTTP: &networking.HTTPIngressRuleValue{
									Paths: []networking.HTTPIngressPath{
										{
											Path: "/",
										},
									},
								},
							},
						},
					},
				},
			},
			isPlus: false,
			expectedErrors: []string{
				"spec.rules[0].http.paths: Too many: 1: must have at most 0 items",
			},
			msg: "invalid master",
		},
		{
			ing: &networking.Ingress{
				ObjectMeta: meta_v1.ObjectMeta{
					Annotations: map[string]string{
						"nginx.org/mergeable-ingress-type": "minion",
					},
				},
				Spec: networking.IngressSpec{
					Rules: []networking.IngressRule{
						{
							Host:             "example.com",
							IngressRuleValue: networking.IngressRuleValue{},
						},
					},
				},
			},
			isPlus: false,
			expectedErrors: []string{
				"spec.rules[0].http.paths: Required value: must include at least one path",
			},
			msg: "invalid minion",
		},
	}

	for _, test := range tests {
		allErrs := validateIngress(test.ing, test.isPlus)
		assertion := assertErrors("validateIngress()", test.msg, allErrs, test.expectedErrors)
		if assertion != "" {
			t.Error(assertion)
		}
	}
}

func TestValidateNginxIngressAnnotations(t *testing.T) {
	isPlus := false
	tests := []struct {
		annotations    map[string]string
		expectedErrors []string
		msg            string
	}{
		{
			annotations:    map[string]string{},
			expectedErrors: nil,
			msg:            "valid no annotations",
		},

		{
			annotations: map[string]string{
				"nginx.org/lb-method":              "invalid_method",
				"nginx.org/mergeable-ingress-type": "invalid",
			},
			expectedErrors: []string{
				`annotations.nginx.org/lb-method: Invalid value: "invalid_method": Invalid load balancing method: "invalid_method"`,
				`annotations.nginx.org/mergeable-ingress-type: Invalid value: "invalid": must be one of: 'master' or 'minion'`,
			},
			msg: "invalid multiple annotations messages in alphabetical order",
		},

		{
			annotations: map[string]string{
				"nginx.org/mergeable-ingress-type": "master",
			},
			expectedErrors: nil,
			msg:            "valid input with master annotation",
		},
		{
			annotations: map[string]string{
				"nginx.org/mergeable-ingress-type": "minion",
			},
			expectedErrors: nil,
			msg:            "valid input with minion annotation",
		},
		{
			annotations: map[string]string{
				"nginx.org/mergeable-ingress-type": "",
			},
			expectedErrors: []string{
				"annotations.nginx.org/mergeable-ingress-type: Required value",
			},
			msg: "invalid mergeable type annotation 1",
		},
		{
			annotations: map[string]string{
				"nginx.org/mergeable-ingress-type": "abc",
			},
			expectedErrors: []string{
				`annotations.nginx.org/mergeable-ingress-type: Invalid value: "abc": must be one of: 'master' or 'minion'`,
			},
			msg: "invalid mergeable type annotation 2",
		},

		{
			annotations: map[string]string{
				"nginx.org/lb-method": "random",
			},
			expectedErrors: nil,
			msg:            "valid nginx.org/lb-method annotation, nginx normal",
		},
		{
			annotations: map[string]string{
				"nginx.org/lb-method": "least_time header",
			},
			expectedErrors: []string{
				`annotations.nginx.org/lb-method: Invalid value: "least_time header": Invalid load balancing method: "least_time header"`,
			},
			msg: "invalid nginx.org/lb-method annotation, nginx plus only",
		},
		{
			annotations: map[string]string{
				"nginx.org/lb-method": "invalid_method",
			},
			expectedErrors: []string{
				`annotations.nginx.org/lb-method: Invalid value: "invalid_method": Invalid load balancing method: "invalid_method"`,
			},
			msg: "invalid nginx.org/lb-method annotation",
		},

		{
			annotations: map[string]string{
				"nginx.com/health-checks": "true",
			},
			expectedErrors: []string{
				"annotations.nginx.com/health-checks: Forbidden: annotation requires NGINX Plus",
			},
			msg: "invalid nginx.com/health-checks annotation, nginx plus only",
		},

		{
			annotations: map[string]string{
				"nginx.com/health-checks-mandatory": "true",
			},
			expectedErrors: []string{
				"annotations.nginx.com/health-checks-mandatory: Forbidden: annotation requires NGINX Plus",
			},
			msg: "invalid nginx.com/health-checks-mandatory annotation, nginx plus only",
		},

		{
			annotations: map[string]string{
				"nginx.com/health-checks-mandatory-queue": "true",
			},
			expectedErrors: []string{
				"annotations.nginx.com/health-checks-mandatory-queue: Forbidden: annotation requires NGINX Plus",
			},
			msg: "invalid nginx.com/health-checks-mandatory-queue annotation, nginx plus only",
		},

		{
			annotations: map[string]string{
				"nginx.com/slow-start": "true",
			},
			expectedErrors: []string{
				"annotations.nginx.com/slow-start: Forbidden: annotation requires NGINX Plus",
			},
			msg: "invalid nginx.com/slow-start annotation, nginx plus only",
		},

		{
			annotations: map[string]string{
				"nginx.org/server-tokens": "custom_setting",
			},
			expectedErrors: []string{
				`annotations.nginx.org/server-tokens: Invalid value: "custom_setting": must be a valid boolean`,
			},
			msg: "invalid nginx.org/server-tokens annotation, must be a boolean",
		},

		{
			annotations: map[string]string{
				"nginx.org/server-snippets": "snippet-1",
			},
			expectedErrors: nil,
			msg:            "valid nginx.org/server-snippets annotation, single-line",
		},
		{
			annotations: map[string]string{
				"nginx.org/server-snippets": "snippet-1\nsnippet-2\nsnippet-3",
			},
			expectedErrors: nil,
			msg:            "valid nginx.org/server-snippets annotation, multi-line",
		},

		{
			annotations: map[string]string{
				"nginx.org/location-snippets": "snippet-1",
			},
			expectedErrors: nil,
			msg:            "valid nginx.org/location-snippets annotation",
		},
		{
			annotations: map[string]string{
				"nginx.org/location-snippets": "snippet-1\nsnippet-2\nsnippet-3",
			},
			expectedErrors: nil,
			msg:            "valid nginx.org/location-snippets annotation, multi-line",
		},
	}

	for _, test := range tests {
		t.Run(test.msg, func(t *testing.T) {
			allErrs := validateIngressAnnotations(test.annotations, isPlus, field.NewPath("annotations"))
			assertion := assertErrors("validateIngressAnnotations()", test.msg, allErrs, test.expectedErrors)
			if assertion != "" {
				t.Error(assertion)
			}
		})
	}
}

func TestValidateNginxPlusIngressAnnotations(t *testing.T) {
	isPlus := true
	tests := []struct {
		annotations    map[string]string
		expectedErrors []string
		msg            string
	}{
		{
			annotations:    map[string]string{},
			expectedErrors: nil,
			msg:            "valid no annotations",
		},

		{
			annotations: map[string]string{
				"nginx.org/lb-method":     "invalid_method",
				"nginx.com/health-checks": "not_a_boolean",
			},
			expectedErrors: []string{
				`annotations.nginx.com/health-checks: Invalid value: "not_a_boolean": must be a valid boolean`,
				`annotations.nginx.org/lb-method: Invalid value: "invalid_method": Invalid load balancing method: "invalid_method"`,
			},
			msg: "invalid multiple annotations messages in alphabetical order",
		},

		{
			annotations: map[string]string{
				"nginx.org/mergeable-ingress-type": "master",
			},
			expectedErrors: nil,
			msg:            "valid input with master annotation",
		},
		{
			annotations: map[string]string{
				"nginx.org/mergeable-ingress-type": "minion",
			},
			expectedErrors: nil,
			msg:            "valid input with minion annotation",
		},
		{
			annotations: map[string]string{
				"nginx.org/mergeable-ingress-type": "",
			},
			expectedErrors: []string{
				"annotations.nginx.org/mergeable-ingress-type: Required value",
			},
			msg: "invalid mergeable type annotation 1",
		},
		{
			annotations: map[string]string{
				"nginx.org/mergeable-ingress-type": "abc",
			},
			expectedErrors: []string{
				`annotations.nginx.org/mergeable-ingress-type: Invalid value: "abc": must be one of: 'master' or 'minion'`,
			},
			msg: "invalid mergeable type annotation 2",
		},

		{
			annotations: map[string]string{
				"nginx.org/lb-method": "least_time header",
			},
			expectedErrors: nil,
			msg:            "valid nginx.org/lb-method annotation",
		},
		{
			annotations: map[string]string{
				"nginx.org/lb-method": "invalid_method",
			},
			expectedErrors: []string{
				`annotations.nginx.org/lb-method: Invalid value: "invalid_method": Invalid load balancing method: "invalid_method"`,
			},
			msg: "invalid nginx.org/lb-method annotation, nginx",
		},

		{
			annotations: map[string]string{
				"nginx.com/health-checks": "true",
			},
			expectedErrors: nil,
			msg:            "valid nginx.com/health-checks annotation",
		},
		{
			annotations: map[string]string{
				"nginx.com/health-checks": "not_a_boolean",
			},
			expectedErrors: []string{
				`annotations.nginx.com/health-checks: Invalid value: "not_a_boolean": must be a valid boolean`,
			},
			msg: "invalid nginx.com/health-checks annotation",
		},

		{
			annotations: map[string]string{
				"nginx.com/health-checks":           "true",
				"nginx.com/health-checks-mandatory": "true",
			},
			expectedErrors: nil,
			msg:            "valid nginx.com/health-checks-mandatory annotation",
		},
		{
			annotations: map[string]string{
				"nginx.com/health-checks":           "true",
				"nginx.com/health-checks-mandatory": "not_a_boolean",
			},
			expectedErrors: []string{
				`annotations.nginx.com/health-checks-mandatory: Invalid value: "not_a_boolean": must be a valid boolean`,
			},
			msg: "invalid nginx.com/health-checks-mandatory, must be a boolean",
		},
		{
			annotations: map[string]string{
				"nginx.com/health-checks-mandatory": "true",
			},
			expectedErrors: []string{
				"annotations.nginx.com/health-checks-mandatory: Forbidden: related annotation nginx.com/health-checks: must be set",
			},
			msg: "invalid nginx.com/health-checks-mandatory, related annotation nginx.com/health-checks not set",
		},
		{
			annotations: map[string]string{
				"nginx.com/health-checks":           "false",
				"nginx.com/health-checks-mandatory": "true",
			},
			expectedErrors: []string{
				"annotations.nginx.com/health-checks-mandatory: Forbidden: related annotation nginx.com/health-checks: must be true",
			},
			msg: "invalid nginx.com/health-checks-mandatory nginx.com/health-checks is not true",
		},

		{
			annotations: map[string]string{
				"nginx.com/health-checks":                 "true",
				"nginx.com/health-checks-mandatory":       "true",
				"nginx.com/health-checks-mandatory-queue": "5",
			},
			expectedErrors: nil,
			msg:            "valid nginx.com/health-checks-mandatory-queue annotation",
		},
		{
			annotations: map[string]string{
				"nginx.com/health-checks":                 "true",
				"nginx.com/health-checks-mandatory":       "true",
				"nginx.com/health-checks-mandatory-queue": "not_a_number",
			},
			expectedErrors: []string{
				`annotations.nginx.com/health-checks-mandatory-queue: Invalid value: "not_a_number": must be a non-negative integer`,
			},
			msg: "invalid nginx.com/health-checks-mandatory-queue, must be a number",
		},
		{
			annotations: map[string]string{
				"nginx.com/health-checks-mandatory-queue": "5",
			},
			expectedErrors: []string{
				"annotations.nginx.com/health-checks-mandatory-queue: Forbidden: related annotation nginx.com/health-checks-mandatory: must be set",
			},
			msg: "invalid nginx.com/health-checks-mandatory-queue, related annotation nginx.com/health-checks-mandatory not set",
		},
		{
			annotations: map[string]string{
				"nginx.com/health-checks":                 "true",
				"nginx.com/health-checks-mandatory":       "false",
				"nginx.com/health-checks-mandatory-queue": "5",
			},
			expectedErrors: []string{
				"annotations.nginx.com/health-checks-mandatory-queue: Forbidden: related annotation nginx.com/health-checks-mandatory: must be true",
			},
			msg: "invalid nginx.com/health-checks-mandatory-queue nginx.com/health-checks-mandatory is not true",
		},

		{
			annotations: map[string]string{
				"nginx.com/slow-start": "60s",
			},
			expectedErrors: nil,
			msg:            "valid nginx.com/slow-start annotation",
		},
		{
			annotations: map[string]string{
				"nginx.com/slow-start": "not_a_time",
			},
			expectedErrors: []string{
				`annotations.nginx.com/slow-start: Invalid value: "not_a_time": must be a valid time`,
			},
			msg: "invalid nginx.com/slow-start annotation",
		},

		{
			annotations: map[string]string{
				"nginx.org/server-tokens": "custom_setting",
			},
			expectedErrors: nil,
			msg:            "valid nginx.org/server-tokens annotation",
		},

		{
			annotations: map[string]string{
				"nginx.org/server-snippets": "snippet-1",
			},
			expectedErrors: nil,
			msg:            "valid nginx.org/server-snippets annotation, single-line",
		},
		{
			annotations: map[string]string{
				"nginx.org/server-snippets": "snippet-1\nsnippet-2\nsnippet-3",
			},
			expectedErrors: nil,
			msg:            "valid nginx.org/server-snippets annotation, multi-line",
		},

		{
			annotations: map[string]string{
				"nginx.org/location-snippets": "snippet-1",
			},
			expectedErrors: nil,
			msg:            "valid nginx.org/location-snippets annotation",
		},
		{
			annotations: map[string]string{
				"nginx.org/location-snippets": "snippet-1\nsnippet-2\nsnippet-3",
			},
			expectedErrors: nil,
			msg:            "valid nginx.org/location-snippets annotation, multi-line",
		},
	}

	for _, test := range tests {
		t.Run(test.msg, func(t *testing.T) {
			allErrs := validateIngressAnnotations(test.annotations, isPlus, field.NewPath("annotations"))
			assertion := assertErrors("validateIngressAnnotations()", test.msg, allErrs, test.expectedErrors)
			if assertion != "" {
				t.Error(assertion)
			}
		})
	}
}

func TestValidateIngressSpec(t *testing.T) {
	tests := []struct {
		spec           *networking.IngressSpec
		expectedErrors []string
		msg            string
	}{
		{
			spec: &networking.IngressSpec{
				Rules: []networking.IngressRule{
					{
						Host: "foo.example.com",
					},
				},
			},
			expectedErrors: nil,
			msg:            "valid input",
		},
		{
			spec: &networking.IngressSpec{
				Rules: []networking.IngressRule{},
			},
			expectedErrors: []string{
				"spec.rules: Required value",
			},
			msg: "zero rules",
		},
		{
			spec: &networking.IngressSpec{
				Rules: []networking.IngressRule{
					{
						Host: "",
					},
				},
			},
			expectedErrors: []string{
				"spec.rules[0].host: Required value",
			},
			msg: "empty host",
		},
		{
			spec: &networking.IngressSpec{
				Rules: []networking.IngressRule{
					{
						Host: "foo.example.com",
					},
					{
						Host: "foo.example.com",
					},
				},
			},
			expectedErrors: []string{
				`spec.rules[1].host: Duplicate value: "foo.example.com"`,
			},
			msg: "duplicated host",
		},
	}

	for _, test := range tests {
		allErrs := validateIngressSpec(test.spec, field.NewPath("spec"))
		assertion := assertErrors("validateIngressSpec()", test.msg, allErrs, test.expectedErrors)
		if assertion != "" {
			t.Error(assertion)
		}
	}
}

func TestValidateMasterSpec(t *testing.T) {
	tests := []struct {
		spec           *networking.IngressSpec
		expectedErrors []string
		msg            string
	}{
		{
			spec: &networking.IngressSpec{
				Rules: []networking.IngressRule{
					{
						Host: "foo.example.com",
						IngressRuleValue: networking.IngressRuleValue{
							HTTP: &networking.HTTPIngressRuleValue{
								Paths: []networking.HTTPIngressPath{},
							},
						},
					},
				},
			},
			expectedErrors: nil,
			msg:            "valid input",
		},
		{
			spec: &networking.IngressSpec{
				Rules: []networking.IngressRule{
					{
						Host: "foo.example.com",
					},
					{
						Host: "bar.example.com",
					},
				},
			},
			expectedErrors: []string{
				"spec.rules: Too many: 2: must have at most 1 items",
			},
			msg: "too many hosts",
		},
		{
			spec: &networking.IngressSpec{
				Rules: []networking.IngressRule{
					{
						Host: "foo.example.com",
						IngressRuleValue: networking.IngressRuleValue{
							HTTP: &networking.HTTPIngressRuleValue{
								Paths: []networking.HTTPIngressPath{
									{
										Path: "/",
									},
								},
							},
						},
					},
				},
			},
			expectedErrors: []string{
				"spec.rules[0].http.paths: Too many: 1: must have at most 0 items",
			},
			msg: "too many paths",
		},
	}

	for _, test := range tests {
		allErrs := validateMasterSpec(test.spec, field.NewPath("spec"))
		assertion := assertErrors("validateMasterSpec()", test.msg, allErrs, test.expectedErrors)
		if assertion != "" {
			t.Error(assertion)
		}
	}
}

func TestValidateMinionSpec(t *testing.T) {
	tests := []struct {
		spec           *networking.IngressSpec
		expectedErrors []string
		msg            string
	}{
		{
			spec: &networking.IngressSpec{
				Rules: []networking.IngressRule{
					{
						Host: "foo.example.com",
						IngressRuleValue: networking.IngressRuleValue{
							HTTP: &networking.HTTPIngressRuleValue{
								Paths: []networking.HTTPIngressPath{
									{
										Path: "/",
									},
								},
							},
						},
					},
				},
			},
			expectedErrors: nil,
			msg:            "valid input",
		},
		{
			spec: &networking.IngressSpec{
				Rules: []networking.IngressRule{
					{
						Host: "foo.example.com",
					},
					{
						Host: "bar.example.com",
					},
				},
			},
			expectedErrors: []string{
				"spec.rules: Too many: 2: must have at most 1 items",
			},
			msg: "too many hosts",
		},
		{
			spec: &networking.IngressSpec{
				Rules: []networking.IngressRule{
					{
						Host: "foo.example.com",
						IngressRuleValue: networking.IngressRuleValue{
							HTTP: &networking.HTTPIngressRuleValue{
								Paths: []networking.HTTPIngressPath{},
							},
						},
					},
				},
			},
			expectedErrors: []string{
				"spec.rules[0].http.paths: Required value: must include at least one path",
			},
			msg: "too few paths",
		},
		{
			spec: &networking.IngressSpec{
				TLS: []networking.IngressTLS{
					{
						Hosts: []string{"foo.example.com"},
					},
				},
				Rules: []networking.IngressRule{
					{
						Host: "foo.example.com",
						IngressRuleValue: networking.IngressRuleValue{
							HTTP: &networking.HTTPIngressRuleValue{
								Paths: []networking.HTTPIngressPath{
									{
										Path: "/",
									},
								},
							},
						},
					},
				},
			},
			expectedErrors: []string{
				"spec.tls: Too many: 1: must have at most 0 items",
			},
			msg: "tls is forbidden",
		},
	}

	for _, test := range tests {
		allErrs := validateMinionSpec(test.spec, field.NewPath("spec"))
		assertion := assertErrors("validateMinionSpec()", test.msg, allErrs, test.expectedErrors)
		if assertion != "" {
			t.Error(assertion)
		}
	}
}

func assertErrors(funcName string, msg string, allErrs field.ErrorList, expectedErrors []string) string {
	errors := errorListToStrings(allErrs)
	if !reflect.DeepEqual(errors, expectedErrors) {
		result := strings.Join(errors, "\n")
		expected := strings.Join(expectedErrors, "\n")

		return fmt.Sprintf("%s returned \n%s \nbut expected \n%s \nfor the case of %s", funcName, result, expected, msg)
	}

	return ""
}

func errorListToStrings(list field.ErrorList) []string {
	var result []string

	for _, e := range list {
		result = append(result, e.Error())
	}

	return result
}
